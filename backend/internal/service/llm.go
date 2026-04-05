package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
)

// ReceiptParseItem represents an item extracted from a receipt
type ReceiptParseItem struct {
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Discount  float64 `json:"discount"`
	Subtotal  float64 `json:"subtotal"`
}

// ReceiptParseFee represents a fee extracted from a receipt
type ReceiptParseFee struct {
	Label   string  `json:"label"`
	FeeType string  `json:"fee_type"`
	Amount  float64 `json:"amount"`
}

// ParsedReceipt represents the structured data extracted from a receipt image
type ParsedReceipt struct {
	Title         string             `json:"title"`
	ReceiptDate   *string            `json:"receipt_date"`
	Currency      string             `json:"currency"`
	PaymentMethod string             `json:"payment_method"`
	Items         []ReceiptParseItem `json:"items"`
	Fees          []ReceiptParseFee  `json:"fees"`
	Subtotal      float64            `json:"subtotal"`
	Total         float64            `json:"total"`
	OCRConfidence float64            `json:"ocr_confidence"`
}

const systemPrompt = `You are a receipt data extraction engine.
Your only job is to extract structured data from receipt images and return valid JSON.
Never add commentary, explanations, or markdown.
If a field cannot be determined from the image, use null.
All monetary values must be numbers (not strings), in the receipt's original currency unit.

Extract all data from this receipt image and return ONLY a JSON object with this exact structure:

{
  "title": "string — shop or merchant name",
  "receipt_date": "YYYY-MM-DD or null",
  "currency": "ISO 4217 code e.g. IDR, USD, SGD",
  "payment_method": "cash | card | qris | transfer | unknown",
  "items": [
    {
      "name": "string",
      "quantity": number,
      "unit_price": number,
      "discount": number or 0,
      "subtotal": number
    }
  ],
  "fees": [
    {
      "label": "string e.g. PPN 11%, Service charge, Delivery",
      "fee_type": "tax | service | delivery | tip | other",
      "amount": number
    }
  ],
  "subtotal": number,
  "total": number,
  "ocr_confidence": number between 0.0 and 1.0
}

Rules:
- ocr_confidence reflects how clearly the receipt is readable (1.0 = perfectly clear)
- subtotal is the sum of items before fees
- total is the final amount after all fees and discounts
- If an item has no discount, set discount to 0
- Quantity must be a positive integer
- Do not invent data — use null for genuinely missing fields`

// LLMService provides LLM-based receipt parsing
type LLMService struct {
	settingsService *SettingsService
	storageService  *StorageService
	httpClient      *http.Client
}

// NewLLMService creates a new LLM service
func NewLLMService(
	settingsService *SettingsService,
	storageService *StorageService,
) *LLMService {
	return &LLMService{
		settingsService: settingsService,
		storageService:  storageService,
		httpClient:      &http.Client{},
	}
}

// ParseReceipt parses a receipt image using LLM vision capabilities
func (s *LLMService) ParseReceipt(ctx context.Context, userID uuid.UUID, imageURL string) (*ParsedReceipt, error) {
	// 1. Resolve LLM config
	llmConfig, err := s.settingsService.ResolveLLMConfig(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve LLM config: %w", err)
	}

	// 2. Fetch image data
	imageData, contentType, err := s.fetchImage(ctx, imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}

	// 3. Call LLM vision API based on provider
	var rawJSON string
	switch llmConfig.Provider {
	case "anthropic":
		rawJSON, err = s.callAnthropic(ctx, llmConfig, imageData, contentType)
	case "openai":
		rawJSON, err = s.callOpenAI(ctx, llmConfig, imageData, contentType)
	case "gemini":
		rawJSON, err = s.callGemini(ctx, llmConfig, imageData, contentType)
	case "ollama":
		rawJSON, err = s.callOllama(ctx, llmConfig, imageData, contentType)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", llmConfig.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("LLM API call failed: %w", err)
	}

	// 4. Parse JSON response
	parsedReceipt, err := s.parseReceiptJSON(rawJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return parsedReceipt, nil
}

// fetchImage retrieves image data from a URL (supports presigned URLs and direct URLs)
func (s *LLMService) fetchImage(ctx context.Context, imageURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to fetch image: status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %w", err)
	}

	return imageData, contentType, nil
}

// callOpenAI calls the OpenAI Vision API
func (s *LLMService) callOpenAI(ctx context.Context, config *LLMConfig, imageData []byte, contentType string) (string, error) {
	client := openai.NewClient(config.APIKey)

	// Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	imageURL := fmt.Sprintf("data:%s;base64,%s", contentType, base64Image)

	req := openai.ChatCompletionRequest{
		Model: config.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Extract data from this receipt image.",
			},
		},
	}

	// Add vision content using the proper content part type
	req.Messages[1].MultiContent = []openai.ChatMessagePart{
		{
			Type: openai.ChatMessagePartTypeText,
			Text: "Extract data from this receipt image.",
		},
		{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: imageURL,
			},
		},
	}
	req.Messages[1].Content = "" // Clear content when using MultiContent

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// callAnthropic calls the Anthropic Claude Vision API
func (s *LLMService) callAnthropic(ctx context.Context, config *LLMConfig, imageData []byte, contentType string) (string, error) {
	// Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Determine media type from content type
	mediaType := contentType
	if mediaType == "" {
		mediaType = "image/jpeg"
	}

	reqBody := map[string]interface{}{
		"model": config.Model,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": systemPrompt + "\n\nExtract data from this receipt image.",
			},
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image",
						"source": map[string]string{
							"type":       "base64",
							"media_type": mediaType,
							"data":       base64Image,
						},
					},
				},
			},
		},
		"max_tokens": 4096,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("no content in Anthropic response")
	}

	return result.Content[0].Text, nil
}

// callGemini calls the Google Gemini Vision API
func (s *LLMService) callGemini(ctx context.Context, config *LLMConfig, imageData []byte, contentType string) (string, error) {
	// Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Determine MIME type
	mimeType := contentType
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": systemPrompt + "\n\nExtract data from this receipt image.",
					},
					{
						"inline_data": map[string]string{
							"mime_type": mimeType,
							"data":      base64Image,
						},
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"responseMimeType": "application/json",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", config.Model, config.APIKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in Gemini response")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

// callOllama calls a local Ollama instance
func (s *LLMService) callOllama(ctx context.Context, config *LLMConfig, imageData []byte, contentType string) (string, error) {
	// Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	reqBody := map[string]interface{}{
		"model": config.Model,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": "Extract data from this receipt image.",
				"images":  []string{base64Image},
			},
		},
		"stream": false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:11434/api/chat", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Message.Content, nil
}

// parseReceiptJSON parses the LLM JSON response into a ParsedReceipt
func (s *LLMService) parseReceiptJSON(rawJSON string) (*ParsedReceipt, error) {
	// Clean up the JSON - remove markdown code blocks if present
	cleaned := rawJSON
	if len(cleaned) > 7 && cleaned[:7] == "```json" {
		// Find the closing ```
		endIdx := len(cleaned) - 3
		if endIdx > 0 && cleaned[endIdx:] == "```" {
			cleaned = cleaned[7:endIdx]
		}
	}
	if len(cleaned) > 3 && cleaned[:3] == "```" {
		cleaned = cleaned[3:]
		if len(cleaned) > 3 && cleaned[len(cleaned)-3:] == "```" {
			cleaned = cleaned[:len(cleaned)-3]
		}
	}

	var parsed ParsedReceipt
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &parsed, nil
}
