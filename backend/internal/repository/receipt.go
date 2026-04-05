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

// ReceiptRepo provides database operations for receipts
type ReceiptRepo struct {
	db *pgxpool.Pool
}

// NewReceiptRepo creates a new receipt repository
func NewReceiptRepo(db *pgxpool.Pool) *ReceiptRepo {
	return &ReceiptRepo{db: db}
}

// Create inserts a new receipt and returns it with ID and timestamps
func (r *ReceiptRepo) Create(ctx context.Context, receipt *model.Receipt) (*model.Receipt, error) {
	query := `
		INSERT INTO receipts (user_id, title, source, image_url, ocr_confidence, currency, payment_method, 
		                      subtotal, total, status, notes, receipt_date, paid_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`

	var id uuid.UUID
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, query,
		receipt.UserID,
		receipt.Title,
		receipt.Source,
		receipt.ImageURL,
		receipt.OCRConfidence,
		receipt.Currency,
		receipt.PaymentMethod,
		receipt.Subtotal,
		receipt.Total,
		receipt.Status,
		receipt.Notes,
		receipt.ReceiptDate,
		receipt.PaidBy,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create receipt: %w", err)
	}

	receipt.ID = id
	receipt.CreatedAt = createdAt
	receipt.UpdatedAt = updatedAt

	return receipt, nil
}

// GetByID retrieves a receipt by ID
func (r *ReceiptRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Receipt, error) {
	query := `
		SELECT id, user_id, title, source, image_url, ocr_confidence, currency, payment_method,
		       subtotal, total, status, notes, receipt_date, paid_by, created_at, updated_at
		FROM receipts
		WHERE id = $1
	`

	receipt := &model.Receipt{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&receipt.ID,
		&receipt.UserID,
		&receipt.Title,
		&receipt.Source,
		&receipt.ImageURL,
		&receipt.OCRConfidence,
		&receipt.Currency,
		&receipt.PaymentMethod,
		&receipt.Subtotal,
		&receipt.Total,
		&receipt.Status,
		&receipt.Notes,
		&receipt.ReceiptDate,
		&receipt.PaidBy,
		&receipt.CreatedAt,
		&receipt.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get receipt by ID: %w", err)
	}

	return receipt, nil
}

// List retrieves receipts with filters and pagination
func (r *ReceiptRepo) List(ctx context.Context, filter *model.ListReceiptsFilter) ([]*model.Receipt, int64, error) {
	// Build the WHERE clause
	whereClause := "WHERE user_id = $1"
	args := []interface{}{filter.UserID}
	argCount := 1

	if filter.Status != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
	}

	if filter.FromDate != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND receipt_date >= $%d", argCount)
		args = append(args, *filter.FromDate)
	}

	if filter.ToDate != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND receipt_date <= $%d", argCount)
		args = append(args, *filter.ToDate)
	}

	if filter.Query != nil && *filter.Query != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND (title ILIKE $%d OR notes ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+*filter.Query+"%")
	}

	if filter.TagID != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND id IN (SELECT receipt_id FROM receipt_tags WHERE tag_id = $%d)", argCount)
		args = append(args, *filter.TagID)
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM receipts " + whereClause
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count receipts: %w", err)
	}

	// Data query with pagination
	argCount++
	limitOffset := fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)

	dataQuery := `
		SELECT id, user_id, title, source, image_url, ocr_confidence, currency, payment_method,
		       subtotal, total, status, notes, receipt_date, paid_by, created_at, updated_at
		FROM receipts
	` + whereClause + limitOffset

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list receipts: %w", err)
	}
	defer rows.Close()

	var receipts []*model.Receipt
	for rows.Next() {
		receipt := &model.Receipt{}
		err := rows.Scan(
			&receipt.ID,
			&receipt.UserID,
			&receipt.Title,
			&receipt.Source,
			&receipt.ImageURL,
			&receipt.OCRConfidence,
			&receipt.Currency,
			&receipt.PaymentMethod,
			&receipt.Subtotal,
			&receipt.Total,
			&receipt.Status,
			&receipt.Notes,
			&receipt.ReceiptDate,
			&receipt.PaidBy,
			&receipt.CreatedAt,
			&receipt.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan receipt: %w", err)
		}
		receipts = append(receipts, receipt)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating receipts: %w", err)
	}

	return receipts, total, nil
}

// Update updates a receipt's fields
func (r *ReceiptRepo) Update(ctx context.Context, receipt *model.Receipt) (*model.Receipt, error) {
	query := `
		UPDATE receipts
		SET title = $1, currency = $2, payment_method = $3, subtotal = $4, 
		    total = $5, notes = $6, receipt_date = $7, paid_by = $8, updated_at = now()
		WHERE id = $9
		RETURNING updated_at
	`

	result, err := r.db.Exec(ctx, query,
		receipt.Title,
		receipt.Currency,
		receipt.PaymentMethod,
		receipt.Subtotal,
		receipt.Total,
		receipt.Notes,
		receipt.ReceiptDate,
		receipt.PaidBy,
		receipt.ID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update receipt: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil, fmt.Errorf("receipt not found")
	}

	// Get the updated_at timestamp
	err = r.db.QueryRow(ctx, "SELECT updated_at FROM receipts WHERE id = $1", receipt.ID).Scan(&receipt.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated timestamp: %w", err)
	}

	return receipt, nil
}

// UpdateStatus updates just the status of a receipt
func (r *ReceiptRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.ReceiptStatus) error {
	query := `
		UPDATE receipts
		SET status = $1, updated_at = now()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update receipt status: %w", err)
	}

	return nil
}

// Delete deletes a receipt by ID
func (r *ReceiptRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM receipts WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete receipt: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("receipt not found")
	}

	return nil
}

// AddItem adds an item to a receipt
func (r *ReceiptRepo) AddItem(ctx context.Context, item *model.ReceiptItem) (*model.ReceiptItem, error) {
	query := `
		INSERT INTO receipt_items (receipt_id, name, quantity, unit_price, discount, subtotal)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		item.ReceiptID,
		item.Name,
		item.Quantity,
		item.UnitPrice,
		item.Discount,
		item.Subtotal,
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to add receipt item: %w", err)
	}

	item.ID = id
	return item, nil
}

// GetItems retrieves all items for a receipt
func (r *ReceiptRepo) GetItems(ctx context.Context, receiptID uuid.UUID) ([]*model.ReceiptItem, error) {
	query := `
		SELECT id, receipt_id, name, quantity, unit_price, discount, subtotal
		FROM receipt_items
		WHERE receipt_id = $1
		ORDER BY id ASC
	`

	rows, err := r.db.Query(ctx, query, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt items: %w", err)
	}
	defer rows.Close()

	var items []*model.ReceiptItem
	for rows.Next() {
		item := &model.ReceiptItem{}
		err := rows.Scan(
			&item.ID,
			&item.ReceiptID,
			&item.Name,
			&item.Quantity,
			&item.UnitPrice,
			&item.Discount,
			&item.Subtotal,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan receipt item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating receipt items: %w", err)
	}

	return items, nil
}

// UpdateItem updates a receipt item
func (r *ReceiptRepo) UpdateItem(ctx context.Context, item *model.ReceiptItem) error {
	query := `
		UPDATE receipt_items
		SET name = $1, quantity = $2, unit_price = $3, discount = $4, subtotal = $5
		WHERE id = $6 AND receipt_id = $7
	`

	_, err := r.db.Exec(ctx, query,
		item.Name,
		item.Quantity,
		item.UnitPrice,
		item.Discount,
		item.Subtotal,
		item.ID,
		item.ReceiptID,
	)

	if err != nil {
		return fmt.Errorf("failed to update receipt item: %w", err)
	}

	return nil
}

// DeleteItem deletes a receipt item
func (r *ReceiptRepo) DeleteItem(ctx context.Context, itemID uuid.UUID, receiptID uuid.UUID) error {
	query := `DELETE FROM receipt_items WHERE id = $1 AND receipt_id = $2`

	_, err := r.db.Exec(ctx, query, itemID, receiptID)
	if err != nil {
		return fmt.Errorf("failed to delete receipt item: %w", err)
	}

	return nil
}

// AddFee adds a fee to a receipt
func (r *ReceiptRepo) AddFee(ctx context.Context, fee *model.ReceiptFee) (*model.ReceiptFee, error) {
	query := `
		INSERT INTO receipt_fees (receipt_id, label, fee_type, amount)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		fee.ReceiptID,
		fee.Label,
		fee.FeeType,
		fee.Amount,
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to add receipt fee: %w", err)
	}

	fee.ID = id
	return fee, nil
}

// GetFees retrieves all fees for a receipt
func (r *ReceiptRepo) GetFees(ctx context.Context, receiptID uuid.UUID) ([]*model.ReceiptFee, error) {
	query := `
		SELECT id, receipt_id, label, fee_type, amount
		FROM receipt_fees
		WHERE receipt_id = $1
		ORDER BY id ASC
	`

	rows, err := r.db.Query(ctx, query, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt fees: %w", err)
	}
	defer rows.Close()

	var fees []*model.ReceiptFee
	for rows.Next() {
		fee := &model.ReceiptFee{}
		err := rows.Scan(
			&fee.ID,
			&fee.ReceiptID,
			&fee.Label,
			&fee.FeeType,
			&fee.Amount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan receipt fee: %w", err)
		}
		fees = append(fees, fee)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating receipt fees: %w", err)
	}

	return fees, nil
}

// DeleteAllItems deletes all items for a receipt
func (r *ReceiptRepo) DeleteAllItems(ctx context.Context, receiptID uuid.UUID) error {
	query := `DELETE FROM receipt_items WHERE receipt_id = $1`

	_, err := r.db.Exec(ctx, query, receiptID)
	if err != nil {
		return fmt.Errorf("failed to delete all receipt items: %w", err)
	}

	return nil
}

// DeleteAllFees deletes all fees for a receipt
func (r *ReceiptRepo) DeleteAllFees(ctx context.Context, receiptID uuid.UUID) error {
	query := `DELETE FROM receipt_fees WHERE receipt_id = $1`

	_, err := r.db.Exec(ctx, query, receiptID)
	if err != nil {
		return fmt.Errorf("failed to delete all receipt fees: %w", err)
	}

	return nil
}

// SetTags sets the tags for a receipt (replaces existing tags)
func (r *ReceiptRepo) SetTags(ctx context.Context, receiptID uuid.UUID, tagIDs []uuid.UUID) error {
	// Start a transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete existing tags
	_, err = tx.Exec(ctx, `DELETE FROM receipt_tags WHERE receipt_id = $1`, receiptID)
	if err != nil {
		return fmt.Errorf("failed to delete existing tags: %w", err)
	}

	// Insert new tags
	if len(tagIDs) > 0 {
		for _, tagID := range tagIDs {
			_, err := tx.Exec(ctx,
				`INSERT INTO receipt_tags (receipt_id, tag_id) VALUES ($1, $2)`,
				receiptID, tagID,
			)
			if err != nil {
				return fmt.Errorf("failed to add tag: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetTags retrieves all tags for a receipt
func (r *ReceiptRepo) GetTags(ctx context.Context, receiptID uuid.UUID) ([]*model.Tag, error) {
	query := `
		SELECT t.id, t.user_id, t.name, t.color, t.created_at, t.updated_at
		FROM tags t
		JOIN receipt_tags rt ON t.id = rt.tag_id
		WHERE rt.receipt_id = $1
		ORDER BY t.name ASC
	`

	rows, err := r.db.Query(ctx, query, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt tags: %w", err)
	}
	defer rows.Close()

	var tags []*model.Tag
	for rows.Next() {
		tag := &model.Tag{}
		err := rows.Scan(
			&tag.ID,
			&tag.UserID,
			&tag.Name,
			&tag.Color,
			&tag.CreatedAt,
			&tag.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	return tags, nil
}

// GetMonthlyTotal calculates the total spending for a user within a date range
func (r *ReceiptRepo) GetMonthlyTotal(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(total), 0)
		FROM receipts
		WHERE user_id = $1 
		  AND receipt_date >= $2 
		  AND receipt_date < $3
		  AND status != $4
	`

	var total float64
	err := r.db.QueryRow(ctx, query, userID, fromDate, toDate, model.ReceiptStatusRejected).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get monthly total: %w", err)
	}

	return total, nil
}

// CountByStatus counts receipts with a specific status for a user
func (r *ReceiptRepo) CountByStatus(ctx context.Context, userID uuid.UUID, status model.ReceiptStatus) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM receipts
		WHERE user_id = $1 AND status = $2
	`

	var count int64
	err := r.db.QueryRow(ctx, query, userID, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count receipts by status: %w", err)
	}

	return count, nil
}
