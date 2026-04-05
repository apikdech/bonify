package workflow

import (
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ParseReceiptInput represents the input to the ParseReceipt workflow
type ParseReceiptInput struct {
	UserID   string `json:"user_id"`
	ImageURL string `json:"image_url"`
	Source   string `json:"source"` // "telegram" or "discord"
}

// ParseReceiptResult represents the result of the ParseReceipt workflow
type ParseReceiptResult struct {
	ReceiptID string `json:"receipt_id"`
}

// LLMConfig represents the resolved LLM configuration
type LLMConfig struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"api_key"`
}

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

// ParseReceiptWorkflow orchestrates the receipt parsing process using Temporal
func ParseReceiptWorkflow(ctx workflow.Context, input ParseReceiptInput) (ParseReceiptResult, error) {
	result := ParseReceiptResult{}

	// Define activity options with retry policy
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Step 1: Resolve LLM configuration for the user
	var llmConfig LLMConfig
	if err := workflow.ExecuteActivity(ctx, "ResolveLLMConfigActivity", input.UserID).Get(ctx, &llmConfig); err != nil {
		return result, err
	}

	// Step 2: Call LLM vision API to parse the receipt
	var parsed ParsedReceipt
	if err := workflow.ExecuteActivity(ctx, "CallLLMVisionActivity", llmConfig, input.ImageURL).Get(ctx, &parsed); err != nil {
		return result, err
	}

	// Step 3: Save the parsed receipt to the database
	var receiptID string
	if err := workflow.ExecuteActivity(ctx, "SaveReceiptActivity", input.UserID, input.ImageURL, input.Source, parsed).Get(ctx, &receiptID); err != nil {
		return result, err
	}

	result.ReceiptID = receiptID

	// Step 4: Notify the user about the parsing result
	confidence := parsed.OCRConfidence
	_ = workflow.ExecuteActivity(ctx, "NotifyUserActivity", input.UserID, receiptID, confidence).Get(ctx, nil)
	// Notification failure should not fail the workflow

	return result, nil
}

// MustParseUUID parses a UUID string, panicking if invalid (for workflow input validation)
func MustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic("invalid UUID: " + s)
	}
	return id
}
