package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	anyllm "github.com/mozilla-ai/any-llm-go"
	"github.com/mozilla-ai/any-llm-go/providers/anthropic"
	"github.com/mozilla-ai/any-llm-go/providers/gemini"
	"github.com/mozilla-ai/any-llm-go/providers/ollama"
	"github.com/mozilla-ai/any-llm-go/providers/openai"
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

Extract all data from this receipt image. Return the data according to the provided JSON schema.

Rules:
- ocr_confidence reflects how clearly the receipt is readable (1.0 = perfectly clear)
- subtotal is the sum of items before fees
- total is the final amount after all fees and discounts
- If an item has no discount, set discount to 0
- Quantity must be a positive integer
- Do not invent data — use null for genuinely missing fields`

// receiptJSONSchema returns the JSON schema for receipt parsing
func receiptJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "Shop or merchant name",
			},
			"receipt_date": map[string]any{
				"type":        "string",
				"description": "Receipt date in YYYY-MM-DD format or null if not visible",
				"format":      "date",
			},
			"currency": map[string]any{
				"type":        "string",
				"description": "ISO 4217 currency code (e.g., IDR, USD, SGD)",
			},
			"payment_method": map[string]any{
				"type":        "string",
				"description": "Payment method used",
				"enum":        []string{"cash", "card", "qris", "transfer", "unknown"},
			},
			"items": map[string]any{
				"type":        "array",
				"description": "Line items on the receipt",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type":        "string",
							"description": "Item name",
						},
						"quantity": map[string]any{
							"type":        "number",
							"description": "Quantity purchased",
							"minimum":     1,
						},
						"unit_price": map[string]any{
							"type":        "number",
							"description": "Price per unit",
						},
						"discount": map[string]any{
							"type":        "number",
							"description": "Discount amount (0 if none)",
							"minimum":     0,
						},
						"subtotal": map[string]any{
							"type":        "number",
							"description": "Line total (quantity x unit_price - discount)",
						},
					},
					"required": []string{"name", "quantity", "unit_price", "discount", "subtotal"},
				},
			},
			"fees": map[string]any{
				"type":        "array",
				"description": "Additional fees and taxes",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"label": map[string]any{
							"type":        "string",
							"description": "Fee label (e.g., PPN 11%, Service charge, Delivery)",
						},
						"fee_type": map[string]any{
							"type":        "string",
							"description": "Type of fee",
							"enum":        []string{"tax", "service", "delivery", "tip", "other"},
						},
						"amount": map[string]any{
							"type":        "number",
							"description": "Fee amount",
							"minimum":     0,
						},
					},
					"required": []string{"label", "fee_type", "amount"},
				},
			},
			"subtotal": map[string]any{
				"type":        "number",
				"description": "Sum of all items before fees",
				"minimum":     0,
			},
			"total": map[string]any{
				"type":        "number",
				"description": "Final total amount after fees and discounts",
				"minimum":     0,
			},
			"ocr_confidence": map[string]any{
				"type":        "number",
				"description": "Confidence score from 0.0 to 1.0 (1.0 = perfectly clear)",
				"minimum":     0,
				"maximum":     1,
			},
		},
		"required": []string{"title", "currency", "payment_method", "items", "fees", "subtotal", "total", "ocr_confidence"},
	}
}

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

// ParseReceipt parses a receipt image using LLM vision capabilities with structured output
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

	// 3. Create provider based on config
	provider, err := s.newProviderFromConfig(llmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// 4. Call LLM vision API with structured output
	parsedReceipt, err := s.callLLMWithStructuredOutput(ctx, provider, llmConfig, imageData, contentType)
	if err != nil {
		return nil, fmt.Errorf("LLM API call failed: %w", err)
	}

	return parsedReceipt, nil
}

// newProviderFromConfig creates a provider based on the configuration
func (s *LLMService) newProviderFromConfig(config *LLMConfig) (anyllm.Provider, error) {
	switch config.Provider {
	case "anthropic":
		return anthropic.New(anyllm.WithAPIKey(config.APIKey))
	case "openai":
		return openai.New(anyllm.WithAPIKey(config.APIKey))
	case "gemini":
		return gemini.New(anyllm.WithAPIKey(config.APIKey))
	case "ollama":
		return ollama.New()
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.Provider)
	}
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

// callLLMWithStructuredOutput calls the LLM provider with structured JSON output
func (s *LLMService) callLLMWithStructuredOutput(ctx context.Context, provider anyllm.Provider, config *LLMConfig, imageData []byte, contentType string) (*ParsedReceipt, error) {
	// Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Create image URL in data URI format
	imageURL := fmt.Sprintf("data:%s;base64,%s", contentType, base64Image)

	// Build messages with system prompt and user content using ContentPart for multimodal
	messages := []anyllm.Message{
		{
			Role:    anyllm.RoleSystem,
			Content: systemPrompt,
		},
		{
			Role: anyllm.RoleUser,
			Content: []anyllm.ContentPart{
				{
					Type: "text",
					Text: "Extract data from this receipt image.",
				},
				{
					Type: "image_url",
					ImageURL: &anyllm.ImageURL{
						URL: imageURL,
					},
				},
			},
		},
	}

	// Create strict mode for structured output
	strict := true

	// Create completion params with structured JSON output
	params := anyllm.CompletionParams{
		Model:    config.Model,
		Messages: messages,
		ResponseFormat: &anyllm.ResponseFormat{
			Type: "json_schema",
			JSONSchema: &anyllm.JSONSchema{
				Name:        "receipt_extraction",
				Description: "Structured receipt data extracted from an image",
				Schema:      receiptJSONSchema(),
				Strict:      &strict,
			},
		},
	}

	// Call the provider
	resp, err := provider.Completion(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("LLM completion error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Extract the JSON content from the response
	content := resp.Choices[0].Message.Content
	var rawJSON string

	switch v := content.(type) {
	case string:
		rawJSON = v
	case []byte:
		rawJSON = string(v)
	default:
		// If it's already a map (some providers may return parsed JSON), marshal it back
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response content: %w", err)
		}
		rawJSON = string(jsonBytes)
	}

	// Parse the JSON into ParsedReceipt
	var parsedReceipt ParsedReceipt
	if err := json.Unmarshal([]byte(rawJSON), &parsedReceipt); err != nil {
		return nil, fmt.Errorf("failed to parse structured response: %w", err)
	}

	return &parsedReceipt, nil
}
