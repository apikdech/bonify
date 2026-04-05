package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/receipt-manager/backend/internal/model"
)

// SplitRepo provides database operations for receipt splits
type SplitRepo struct {
	db *pgxpool.Pool
}

// NewSplitRepo creates a new split repository
func NewSplitRepo(db *pgxpool.Pool) *SplitRepo {
	return &SplitRepo{db: db}
}

// CreateSplit inserts a new receipt split and returns it with ID and timestamp
func (r *SplitRepo) CreateSplit(ctx context.Context, split *model.ReceiptSplit) (*model.ReceiptSplit, error) {
	query := `
		INSERT INTO receipt_splits (receipt_id, user_id, amount, percentage)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	var id uuid.UUID
	var createdAt time.Time

	err := r.db.QueryRow(ctx, query,
		split.ReceiptID,
		split.UserID,
		split.Amount,
		split.Percentage,
	).Scan(&id, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create receipt split: %w", err)
	}

	split.ID = id
	split.CreatedAt = createdAt

	return split, nil
}

// GetSplitsByReceipt retrieves all splits for a receipt
func (r *SplitRepo) GetSplitsByReceipt(ctx context.Context, receiptID uuid.UUID) ([]*model.ReceiptSplit, error) {
	query := `
		SELECT id, receipt_id, user_id, amount, percentage, created_at
		FROM receipt_splits
		WHERE receipt_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt splits: %w", err)
	}
	defer rows.Close()

	var splits []*model.ReceiptSplit
	for rows.Next() {
		split := &model.ReceiptSplit{}
		err := rows.Scan(
			&split.ID,
			&split.ReceiptID,
			&split.UserID,
			&split.Amount,
			&split.Percentage,
			&split.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan receipt split: %w", err)
		}
		splits = append(splits, split)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating receipt splits: %w", err)
	}

	return splits, nil
}

// GetSplitByReceiptAndUser retrieves a specific split for a receipt and user
func (r *SplitRepo) GetSplitByReceiptAndUser(ctx context.Context, receiptID uuid.UUID, userID uuid.UUID) (*model.ReceiptSplit, error) {
	query := `
		SELECT id, receipt_id, user_id, amount, percentage, created_at
		FROM receipt_splits
		WHERE receipt_id = $1 AND user_id = $2
	`

	split := &model.ReceiptSplit{}
	err := r.db.QueryRow(ctx, query, receiptID, userID).Scan(
		&split.ID,
		&split.ReceiptID,
		&split.UserID,
		&split.Amount,
		&split.Percentage,
		&split.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get receipt split: %w", err)
	}

	return split, nil
}

// UpdateSplit updates a receipt split
func (r *SplitRepo) UpdateSplit(ctx context.Context, split *model.ReceiptSplit) error {
	query := `
		UPDATE receipt_splits
		SET amount = $1, percentage = $2
		WHERE id = $3 AND receipt_id = $4
	`

	result, err := r.db.Exec(ctx, query,
		split.Amount,
		split.Percentage,
		split.ID,
		split.ReceiptID,
	)

	if err != nil {
		return fmt.Errorf("failed to update receipt split: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("receipt split not found")
	}

	return nil
}

// DeleteSplit deletes a receipt split by ID
func (r *SplitRepo) DeleteSplit(ctx context.Context, splitID uuid.UUID) error {
	query := `DELETE FROM receipt_splits WHERE id = $1`

	result, err := r.db.Exec(ctx, query, splitID)
	if err != nil {
		return fmt.Errorf("failed to delete receipt split: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("receipt split not found")
	}

	return nil
}

// DeleteSplitsByReceipt deletes all splits for a receipt
func (r *SplitRepo) DeleteSplitsByReceipt(ctx context.Context, receiptID uuid.UUID) error {
	query := `DELETE FROM receipt_splits WHERE receipt_id = $1`

	_, err := r.db.Exec(ctx, query, receiptID)
	if err != nil {
		return fmt.Errorf("failed to delete receipt splits: %w", err)
	}

	return nil
}

// GetUserSplitTotal calculates the total split amount for a user across receipts
func (r *SplitRepo) GetUserSplitTotal(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(rs.amount), 0)
		FROM receipt_splits rs
		JOIN receipts r ON rs.receipt_id = r.id
		WHERE rs.user_id = $1
		  AND r.receipt_date >= $2
		  AND r.receipt_date < $3
		  AND r.status != $4
	`

	var total float64
	err := r.db.QueryRow(ctx, query, userID, fromDate, toDate, model.ReceiptStatusRejected).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get user split total: %w", err)
	}

	return total, nil
}

// GetSettlementSummary calculates settlements between users for a group
// This is a placeholder implementation - full implementation would need
// group membership logic which may be added in future iterations
func (r *SplitRepo) GetSettlementSummary(ctx context.Context, groupID string) ([]*model.Settlement, error) {
	// For now, return empty settlements
	// Full implementation would require:
	// - Group membership table
	// - Calculation of who paid what vs who owes what
	// - Netting out mutual debts
	return []*model.Settlement{}, nil
}
