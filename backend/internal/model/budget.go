package model

import (
	"github.com/google/uuid"
)

// Budget represents a spending limit for a specific tag in a specific month
type Budget struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	TagID       *uuid.UUID `json:"tag_id,omitempty"`
	Month       string     `json:"month"` // YYYY-MM format
	AmountLimit float64    `json:"amount_limit"`
}
