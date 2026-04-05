package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/receipt-manager/backend/internal/model"
)

// BudgetRepo provides database operations for budgets
type BudgetRepo struct {
	db *pgxpool.Pool
}

// NewBudgetRepo creates a new budget repository
func NewBudgetRepo(db *pgxpool.Pool) *BudgetRepo {
	return &BudgetRepo{db: db}
}

// Create inserts a new budget and returns it with ID
func (r *BudgetRepo) Create(ctx context.Context, budget *model.Budget) (*model.Budget, error) {
	query := `
		INSERT INTO budgets (user_id, tag_id, month, amount_limit)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		budget.UserID,
		budget.TagID,
		budget.Month,
		budget.AmountLimit,
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to create budget: %w", err)
	}

	budget.ID = id
	return budget, nil
}

// GetByUserAndMonth retrieves all budgets for a user in a specific month
func (r *BudgetRepo) GetByUserAndMonth(ctx context.Context, userID uuid.UUID, month string) ([]*model.Budget, error) {
	query := `
		SELECT id, user_id, tag_id, month, amount_limit
		FROM budgets
		WHERE user_id = $1 AND month = $2
		ORDER BY id ASC
	`

	rows, err := r.db.Query(ctx, query, userID, month)
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets by user and month: %w", err)
	}
	defer rows.Close()

	var budgets []*model.Budget
	for rows.Next() {
		budget := &model.Budget{}
		err := rows.Scan(
			&budget.ID,
			&budget.UserID,
			&budget.TagID,
			&budget.Month,
			&budget.AmountLimit,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget: %w", err)
		}
		budgets = append(budgets, budget)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating budgets: %w", err)
	}

	return budgets, nil
}

// GetSpentByTag calculates total spent for a specific tag in a month
// Uses receipts that have this tag via receipt_tags join table
func (r *BudgetRepo) GetSpentByTag(ctx context.Context, userID uuid.UUID, tagID uuid.UUID, month string) (float64, error) {
	// Parse month to get date range
	// Month format is YYYY-MM, so we construct start and end dates
	query := `
		SELECT COALESCE(SUM(r.total), 0)
		FROM receipts r
		JOIN receipt_tags rt ON r.id = rt.receipt_id
		WHERE r.user_id = $1
		  AND rt.tag_id = $2
		  AND r.status != $3
		  AND r.receipt_date >= $4::date
		  AND r.receipt_date < ($4::date + INTERVAL '1 month')
	`

	// month is in YYYY-MM format, append -01 for the first day of month
	monthStart := month + "-01"

	var total float64
	err := r.db.QueryRow(ctx, query,
		userID,
		tagID,
		model.ReceiptStatusRejected,
		monthStart,
	).Scan(&total)

	if err != nil {
		return 0, fmt.Errorf("failed to get spent by tag: %w", err)
	}

	return total, nil
}

// Update updates an existing budget (with user ownership check)
func (r *BudgetRepo) Update(ctx context.Context, budget *model.Budget) (*model.Budget, error) {
	query := `
		UPDATE budgets
		SET tag_id = $1, month = $2, amount_limit = $3
		WHERE id = $4 AND user_id = $5
		RETURNING id
	`

	result, err := r.db.Exec(ctx, query,
		budget.TagID,
		budget.Month,
		budget.AmountLimit,
		budget.ID,
		budget.UserID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update budget: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil, fmt.Errorf("budget not found or access denied")
	}

	return budget, nil
}

// Delete deletes a budget by ID with user ownership check
func (r *BudgetRepo) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM budgets WHERE id = $1 AND user_id = $2`

	result, err := r.db.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("budget not found or access denied")
	}

	return nil
}

// GetByID retrieves a budget by ID
func (r *BudgetRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Budget, error) {
	query := `
		SELECT id, user_id, tag_id, month, amount_limit
		FROM budgets
		WHERE id = $1
	`

	budget := &model.Budget{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&budget.ID,
		&budget.UserID,
		&budget.TagID,
		&budget.Month,
		&budget.AmountLimit,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get budget by ID: %w", err)
	}

	return budget, nil
}
