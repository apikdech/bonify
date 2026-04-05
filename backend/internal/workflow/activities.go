package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/service"
)

// BotNotifier is a placeholder interface for bot notifications
type BotNotifier interface {
	NotifyReceiptParsed(ctx context.Context, userID string, receiptID string, confidence float64) error
}

// Activities holds all the activity implementations with their dependencies
type Activities struct {
	SettingsService *service.SettingsService
	LLMService      *service.LLMService
	ReceiptService  *service.ReceiptService
	FXService       *service.FXService
	Notifier        BotNotifier
}

// NewActivities creates a new Activities instance with all dependencies
func NewActivities(
	settingsService *service.SettingsService,
	llmService *service.LLMService,
	receiptService *service.ReceiptService,
	fxService *service.FXService,
	notifier BotNotifier,
) *Activities {
	return &Activities{
		SettingsService: settingsService,
		LLMService:      llmService,
		ReceiptService:  receiptService,
		FXService:       fxService,
		Notifier:        notifier,
	}
}

// ResolveLLMConfigActivity resolves the LLM configuration for a user
func (a *Activities) ResolveLLMConfigActivity(ctx context.Context, userID string) (LLMConfig, error) {
	config := LLMConfig{}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return config, fmt.Errorf("invalid user ID: %w", err)
	}

	llmCfg, err := a.SettingsService.ResolveLLMConfig(ctx, userUUID)
	if err != nil {
		return config, fmt.Errorf("failed to resolve LLM config: %w", err)
	}

	config.Provider = llmCfg.Provider
	config.Model = llmCfg.Model
	config.APIKey = llmCfg.APIKey

	return config, nil
}

// CallLLMVisionActivity calls the LLM vision API to parse a receipt image
func (a *Activities) CallLLMVisionActivity(ctx context.Context, cfg LLMConfig, imageURL string) (ParsedReceipt, error) {
	parsed := ParsedReceipt{}

	// Since the LLM service already handles the parsing workflow internally,
	// we'll call the ParseReceipt method. However, we need to avoid recursive
	// workflow calls, so we'll use the internal parsing logic.
	// For now, we'll implement a simplified version that fetches the image and calls LLM.

	// Fetch image data
	imageData, contentType, err := fetchImage(ctx, imageURL)
	if err != nil {
		return parsed, fmt.Errorf("failed to fetch image: %w", err)
	}

	// Call the appropriate LLM provider
	rawJSON, err := callLLM(ctx, cfg, imageData, contentType)
	if err != nil {
		return parsed, fmt.Errorf("LLM API call failed: %w", err)
	}

	// Parse the JSON response
	if err := parseReceiptJSON(rawJSON, &parsed); err != nil {
		return parsed, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return parsed, nil
}

// fetchImage retrieves image data from a URL
func fetchImage(ctx context.Context, imageURL string) ([]byte, string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
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

// callLLM calls the LLM provider based on configuration
func callLLM(ctx context.Context, cfg LLMConfig, imageData []byte, contentType string) (string, error) {
	// For now, implement a basic OpenAI integration
	// This can be expanded to support other providers
	switch cfg.Provider {
	case "openai":
		return callOpenAI(ctx, cfg, imageData, contentType)
	case "anthropic":
		return callAnthropic(ctx, cfg, imageData, contentType)
	default:
		// Default to OpenAI
		return callOpenAI(ctx, cfg, imageData, contentType)
	}
}

// callOpenAI calls the OpenAI Vision API
func callOpenAI(ctx context.Context, cfg LLMConfig, imageData []byte, contentType string) (string, error) {
	// This is a simplified implementation that would need to be expanded
	// For now, delegate to the existing LLM service if possible
	return "", fmt.Errorf("OpenAI API call not yet implemented in activity - use service method instead")
}

// callAnthropic calls the Anthropic Claude Vision API
func callAnthropic(ctx context.Context, cfg LLMConfig, imageData []byte, contentType string) (string, error) {
	return "", fmt.Errorf("Anthropic API call not yet implemented in activity - use service method instead")
}

// parseReceiptJSON parses the LLM JSON response into a ParsedReceipt
func parseReceiptJSON(rawJSON string, parsed *ParsedReceipt) error {
	// Clean up the JSON - remove markdown code blocks if present
	cleaned := rawJSON
	if len(cleaned) > 7 && cleaned[:7] == "```json" {
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

	if err := json.Unmarshal([]byte(cleaned), parsed); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// SaveReceiptActivity saves the parsed receipt to the database
func (a *Activities) SaveReceiptActivity(ctx context.Context, userID, imageURL, source string, parsed ParsedReceipt) (string, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}

	// Convert ParsedReceipt to the service format
	serviceParsed := &service.ParsedReceipt{
		Title:         parsed.Title,
		ReceiptDate:   parsed.ReceiptDate,
		Currency:      parsed.Currency,
		PaymentMethod: parsed.PaymentMethod,
		Subtotal:      parsed.Subtotal,
		Total:         parsed.Total,
		OCRConfidence: parsed.OCRConfidence,
		Items:         convertItems(parsed.Items),
		Fees:          convertFees(parsed.Fees),
	}

	// Create receipt from parsed data - source is already validated in the service
	receiptID, err := a.ReceiptService.CreateFromParsed(ctx, userUUID, imageURL, source, serviceParsed)
	if err != nil {
		return "", fmt.Errorf("failed to create receipt: %w", err)
	}

	return receiptID, nil
}

// convertItems converts workflow items to service items
func convertItems(items []ReceiptParseItem) []service.ReceiptParseItem {
	result := make([]service.ReceiptParseItem, len(items))
	for i, item := range items {
		result[i] = service.ReceiptParseItem{
			Name:      item.Name,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Discount:  item.Discount,
			Subtotal:  item.Subtotal,
		}
	}
	return result
}

// convertFees converts workflow fees to service fees
func convertFees(fees []ReceiptParseFee) []service.ReceiptParseFee {
	result := make([]service.ReceiptParseFee, len(fees))
	for i, fee := range fees {
		result[i] = service.ReceiptParseFee{
			Label:   fee.Label,
			FeeType: fee.FeeType,
			Amount:  fee.Amount,
		}
	}
	return result
}

// NotifyUserActivity notifies the user about the receipt parsing result
func (a *Activities) NotifyUserActivity(ctx context.Context, userID, receiptID string, confidence float64) error {
	if a.Notifier == nil {
		// No notifier configured, skip notification
		return nil
	}

	if err := a.Notifier.NotifyReceiptParsed(ctx, userID, receiptID, confidence); err != nil {
		// Notification failures should be logged but not fail the workflow
		return fmt.Errorf("failed to notify user: %w", err)
	}

	return nil
}

// FetchFXRatesActivity fetches FX rates from Frankfurter API and stores in DB
func (a *Activities) FetchFXRatesActivity(ctx context.Context) error {
	if a.FXService == nil {
		return fmt.Errorf("FX service not configured")
	}

	if err := a.FXService.FetchFromFrankfurter(ctx, "IDR"); err != nil {
		return fmt.Errorf("failed to fetch FX rates: %w", err)
	}

	return nil
}
