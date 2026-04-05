package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/model"
)

// BudgetRepository defines the interface for budget data operations
type BudgetRepository interface {
	GetByUserAndMonth(ctx context.Context, userID uuid.UUID, month string) ([]*model.Budget, error)
	GetSpentByTag(ctx context.Context, userID uuid.UUID, tagID uuid.UUID, month string) (float64, error)
}

// AlertType represents the severity of a budget alert
type AlertType string

const (
	// AlertTypeWarning indicates budget is approaching limit (80%+)
	AlertTypeWarning AlertType = "warning"
	// AlertTypeCritical indicates budget has exceeded limit (100%+)
	AlertTypeCritical AlertType = "critical"
)

// BudgetStatus represents the current status of a budget including spent amount
type BudgetStatus struct {
	BudgetID    uuid.UUID  `json:"budget_id"`
	TagID       *uuid.UUID `json:"tag_id,omitempty"`
	Month       string     `json:"month"`
	AmountLimit float64    `json:"amount_limit"`
	Spent       float64    `json:"spent"`
	Percentage  float64    `json:"percentage"`
	Remaining   float64    `json:"remaining"`
}

// BudgetAlert represents an alert when a budget threshold is exceeded
type BudgetAlert struct {
	AlertType   AlertType  `json:"alert_type"`
	BudgetID    uuid.UUID  `json:"budget_id"`
	TagID       *uuid.UUID `json:"tag_id,omitempty"`
	Month       string     `json:"month"`
	AmountLimit float64    `json:"amount_limit"`
	Spent       float64    `json:"spent"`
	Percentage  float64    `json:"percentage"`
	Message     string     `json:"message"`
}

// BudgetService provides budget business logic
type BudgetService struct {
	budgetRepo BudgetRepository
}

// NewBudgetService creates a new budget service
func NewBudgetService(budgetRepo BudgetRepository) *BudgetService {
	return &BudgetService{
		budgetRepo: budgetRepo,
	}
}

// GetBudgetStatus returns budget vs actual spending for a specific month
func (s *BudgetService) GetBudgetStatus(ctx context.Context, userID uuid.UUID, month string) ([]BudgetStatus, error) {
	// Get all budgets for the user in the specified month
	budgets, err := s.budgetRepo.GetByUserAndMonth(ctx, userID, month)
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets: %w", err)
	}

	// Build status for each budget
	statusList := make([]BudgetStatus, 0, len(budgets))
	for _, budget := range budgets {
		var spent float64

		// If budget has a tag, calculate spent for that tag
		if budget.TagID != nil {
			spent, err = s.budgetRepo.GetSpentByTag(ctx, userID, *budget.TagID, month)
			if err != nil {
				return nil, fmt.Errorf("failed to get spent for tag %s: %w", *budget.TagID, err)
			}
		}
		// If TagID is nil, spent remains 0 (global budgets need different handling)

		// Calculate percentage and remaining
		percentage := 0.0
		if budget.AmountLimit > 0 {
			percentage = (spent / budget.AmountLimit) * 100
		}
		remaining := budget.AmountLimit - spent

		status := BudgetStatus{
			BudgetID:    budget.ID,
			TagID:       budget.TagID,
			Month:       budget.Month,
			AmountLimit: budget.AmountLimit,
			Spent:       spent,
			Percentage:  percentage,
			Remaining:   remaining,
		}
		statusList = append(statusList, status)
	}

	return statusList, nil
}

// CheckBudgetAlerts checks all active budgets and returns alerts for those over threshold
func (s *BudgetService) CheckBudgetAlerts(ctx context.Context, userID uuid.UUID) ([]BudgetAlert, error) {
	// Get current month in YYYY-MM format
	now := time.Now()
	month := now.Format("2006-01")

	// Get budget status for current month
	statusList, err := s.GetBudgetStatus(ctx, userID, month)
	if err != nil {
		return nil, err
	}

	// Generate alerts for budgets exceeding thresholds
	alerts := make([]BudgetAlert, 0)
	for _, status := range statusList {
		var alert *BudgetAlert

		// Check for critical threshold (100%+)
		if status.Percentage >= 100 {
			alert = &BudgetAlert{
				AlertType:   AlertTypeCritical,
				BudgetID:    status.BudgetID,
				TagID:       status.TagID,
				Month:       status.Month,
				AmountLimit: status.AmountLimit,
				Spent:       status.Spent,
				Percentage:  status.Percentage,
				Message:     fmt.Sprintf("Budget exceeded: spent %.1f%% of limit", status.Percentage),
			}
		} else if status.Percentage >= 80 {
			// Check for warning threshold (80%+)
			alert = &BudgetAlert{
				AlertType:   AlertTypeWarning,
				BudgetID:    status.BudgetID,
				TagID:       status.TagID,
				Month:       status.Month,
				AmountLimit: status.AmountLimit,
				Spent:       status.Spent,
				Percentage:  status.Percentage,
				Message:     fmt.Sprintf("Budget warning: spent %.1f%% of limit", status.Percentage),
			}
		}

		if alert != nil {
			alerts = append(alerts, *alert)
		}
	}

	return alerts, nil
}
