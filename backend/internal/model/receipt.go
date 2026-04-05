package model

import (
	"time"

	"github.com/google/uuid"
)

// ReceiptStatus represents the status of a receipt
type ReceiptStatus string

const (
	ReceiptStatusPendingReview ReceiptStatus = "pending_review"
	ReceiptStatusConfirmed     ReceiptStatus = "confirmed"
	ReceiptStatusRejected      ReceiptStatus = "rejected"
)

// ReceiptSource represents how the receipt was created
type ReceiptSource string

const (
	ReceiptSourceManual ReceiptSource = "manual"
	ReceiptSourceOCR    ReceiptSource = "ocr"
	ReceiptSourceAPI    ReceiptSource = "api"
)

// Receipt represents a receipt in the system
type Receipt struct {
	ID            uuid.UUID     `json:"id"`
	UserID        uuid.UUID     `json:"user_id"`
	Title         *string       `json:"title,omitempty"`
	Source        ReceiptSource `json:"source"`
	ImageURL      *string       `json:"image_url,omitempty"`
	OCRConfidence *float64      `json:"ocr_confidence,omitempty"`
	Currency      string        `json:"currency"`
	PaymentMethod *string       `json:"payment_method,omitempty"`
	Subtotal      float64       `json:"subtotal"`
	Total         float64       `json:"total"`
	Status        ReceiptStatus `json:"status"`
	Notes         *string       `json:"notes,omitempty"`
	ReceiptDate   *time.Time    `json:"receipt_date,omitempty"`
	PaidBy        *string       `json:"paid_by,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`

	// Relations (not stored in DB)
	Items []*ReceiptItem `json:"items,omitempty"`
	Fees  []*ReceiptFee  `json:"fees,omitempty"`
	Tags  []*Tag         `json:"tags,omitempty"`
}

// ReceiptItem represents an item in a receipt
type ReceiptItem struct {
	ID        uuid.UUID `json:"id"`
	ReceiptID uuid.UUID `json:"receipt_id"`
	Name      string    `json:"name"`
	Quantity  float64   `json:"quantity"`
	UnitPrice float64   `json:"unit_price"`
	Discount  float64   `json:"discount"`
	Subtotal  float64   `json:"subtotal"`
}

// ReceiptFee represents a fee or tax associated with a receipt
type ReceiptFee struct {
	ID        uuid.UUID `json:"id"`
	ReceiptID uuid.UUID `json:"receipt_id"`
	Label     string    `json:"label"`
	FeeType   FeeType   `json:"fee_type"`
	Amount    float64   `json:"amount"`
}

// FeeType represents the type of fee
type FeeType string

const (
	FeeTypeTax    FeeType = "tax"
	FeeTypeTip    FeeType = "tip"
	FeeTypeCharge FeeType = "charge"
	FeeTypeOther  FeeType = "other"
)

// Tag represents a tag that can be associated with receipts
type Tag struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReceiptItemInput represents input data for creating/updating a receipt item
type ReceiptItemInput struct {
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Discount  float64 `json:"discount,omitempty"`
}

// ReceiptFeeInput represents input data for creating/updating a receipt fee
type ReceiptFeeInput struct {
	Label   string  `json:"label"`
	FeeType FeeType `json:"fee_type"`
	Amount  float64 `json:"amount"`
}

// CreateReceiptRequest represents a request to create a receipt
type CreateReceiptRequest struct {
	Title         string             `json:"title,omitempty"`
	Currency      string             `json:"currency"`
	PaymentMethod string             `json:"payment_method,omitempty"`
	ReceiptDate   *time.Time         `json:"receipt_date,omitempty"`
	Notes         string             `json:"notes,omitempty"`
	Items         []ReceiptItemInput `json:"items"`
	Fees          []ReceiptFeeInput  `json:"fees,omitempty"`
	TagIDs        []string           `json:"tag_ids,omitempty"`
	PaidBy        string             `json:"paid_by,omitempty"`
}

// UpdateReceiptRequest represents a request to update a receipt
type UpdateReceiptRequest struct {
	Title         *string             `json:"title,omitempty"`
	Currency      *string             `json:"currency,omitempty"`
	PaymentMethod *string             `json:"payment_method,omitempty"`
	ReceiptDate   *time.Time          `json:"receipt_date,omitempty"`
	Notes         *string             `json:"notes,omitempty"`
	Items         *[]ReceiptItemInput `json:"items,omitempty"`
	Fees          *[]ReceiptFeeInput  `json:"fees,omitempty"`
	TagIDs        *[]string           `json:"tag_ids,omitempty"`
	PaidBy        *string             `json:"paid_by,omitempty"`
}

// ListReceiptsFilter represents filters for listing receipts
type ListReceiptsFilter struct {
	UserID   uuid.UUID
	Status   *ReceiptStatus
	TagID    *uuid.UUID
	FromDate *time.Time
	ToDate   *time.Time
	Query    *string
	Page     int
	Limit    int
}

// ReceiptListResponse represents a paginated list of receipts
type ReceiptListResponse struct {
	Receipts []*Receipt `json:"receipts"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	Limit    int        `json:"limit"`
}

// UpdateStatusRequest represents a request to update receipt status
type UpdateStatusRequest struct {
	Status ReceiptStatus `json:"status"`
}
