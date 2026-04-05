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

// TagRepo provides database operations for tags
type TagRepo struct {
	db *pgxpool.Pool
}

// NewTagRepo creates a new tag repository
func NewTagRepo(db *pgxpool.Pool) *TagRepo {
	return &TagRepo{db: db}
}

// Create inserts a new tag and returns it with ID and timestamps
func (r *TagRepo) Create(ctx context.Context, tag *model.Tag) (*model.Tag, error) {
	query := `
		INSERT INTO tags (user_id, name, color)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	var id uuid.UUID
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, query,
		tag.UserID,
		tag.Name,
		tag.Color,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	tag.ID = id
	tag.CreatedAt = createdAt
	tag.UpdatedAt = updatedAt

	return tag, nil
}

// ListByUser retrieves all tags for a user
func (r *TagRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.Tag, error) {
	query := `
		SELECT id, user_id, name, color, created_at, updated_at
		FROM tags
		WHERE user_id = $1
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
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

// GetByID retrieves a tag by ID
func (r *TagRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Tag, error) {
	query := `
		SELECT id, user_id, name, color, created_at, updated_at
		FROM tags
		WHERE id = $1
	`

	tag := &model.Tag{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&tag.ID,
		&tag.UserID,
		&tag.Name,
		&tag.Color,
		&tag.CreatedAt,
		&tag.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get tag by ID: %w", err)
	}

	return tag, nil
}

// GetByReceiptID retrieves all tags for a receipt via receipt_tags join
func (r *TagRepo) GetByReceiptID(ctx context.Context, receiptID uuid.UUID) ([]*model.Tag, error) {
	query := `
		SELECT t.id, t.user_id, t.name, t.color, t.created_at, t.updated_at
		FROM tags t
		JOIN receipt_tags rt ON t.id = rt.tag_id
		WHERE rt.receipt_id = $1
		ORDER BY t.name ASC
	`

	rows, err := r.db.Query(ctx, query, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags by receipt ID: %w", err)
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

// Update updates a tag's name and color with ownership check
func (r *TagRepo) Update(ctx context.Context, tag *model.Tag) (*model.Tag, error) {
	query := `
		UPDATE tags
		SET name = $1, color = $2, updated_at = now()
		WHERE id = $3 AND user_id = $4
		RETURNING updated_at
	`

	result, err := r.db.Exec(ctx, query,
		tag.Name,
		tag.Color,
		tag.ID,
		tag.UserID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}

	if result.RowsAffected() == 0 {
		return nil, fmt.Errorf("tag not found or access denied")
	}

	// Get the updated_at timestamp
	err = r.db.QueryRow(ctx, "SELECT updated_at FROM tags WHERE id = $1", tag.ID).Scan(&tag.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated timestamp: %w", err)
	}

	return tag, nil
}

// Delete deletes a tag by ID with user ownership check
func (r *TagRepo) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM tags WHERE id = $1 AND user_id = $2`

	result, err := r.db.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("tag not found or access denied")
	}

	return nil
}

// CountReceipts counts the number of receipts using a specific tag
func (r *TagRepo) CountReceipts(ctx context.Context, tagID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM receipt_tags WHERE tag_id = $1`

	var count int64
	err := r.db.QueryRow(ctx, query, tagID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count receipts for tag: %w", err)
	}

	return count, nil
}
