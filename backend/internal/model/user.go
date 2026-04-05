package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID                    uuid.UUID `json:"id"`
	Name                  string    `json:"name"`
	Email                 string    `json:"email"`
	PasswordHash          string    `json:"-"`
	TelegramID            *string   `json:"telegram_id,omitempty"`
	DiscordID             *string   `json:"discord_id,omitempty"`
	Role                  string    `json:"role"`
	LLMProvider           *string   `json:"llm_provider,omitempty"`
	LLMModel              *string   `json:"llm_model,omitempty"`
	HomeCurrency          string    `json:"home_currency"`
	NotifyOnParse         bool      `json:"notify_on_parse"`
	NotifyOnPendingReview bool      `json:"notify_on_pending_review"`
	NotifyBudgetAlerts    bool      `json:"notify_budget_alerts"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// TokenPair holds access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
