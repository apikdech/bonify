package model

import (
	"time"

	"github.com/google/uuid"
)

// ReceiptSplit represents a split of a receipt among users
type ReceiptSplit struct {
	ID         uuid.UUID `json:"id"`
	ReceiptID  uuid.UUID `json:"receipt_id"`
	UserID     uuid.UUID `json:"user_id"`
	Amount     float64   `json:"amount"`
	Percentage *float64  `json:"percentage,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// Settlement represents a settlement between two users
type Settlement struct {
	FromUserID uuid.UUID `json:"from_user_id"`
	ToUserID   uuid.UUID `json:"to_user_id"`
	Amount     float64   `json:"amount"`
}

// CreateSplitRequest represents a request to create a receipt split
type CreateSplitRequest struct {
	UserID     string  `json:"user_id"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage,omitempty"`
}

// CreateSplitsRequest represents a request to create multiple receipt splits
type CreateSplitsRequest struct {
	Splits []CreateSplitRequest `json:"splits"`
}

// UpdateSplitRequest represents a request to update a receipt split
type UpdateSplitRequest struct {
	Amount     *float64 `json:"amount,omitempty"`
	Percentage *float64 `json:"percentage,omitempty"`
}
