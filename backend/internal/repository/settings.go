package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SettingsRepo provides database operations for system settings
type SettingsRepo struct {
	db *pgxpool.Pool
}

// NewSettingsRepo creates a new settings repository
func NewSettingsRepo(db *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{db: db}
}

// Get retrieves a single setting value by key
func (r *SettingsRepo) Get(ctx context.Context, key string) (string, error) {
	query := `
		SELECT value
		FROM system_settings
		WHERE key = $1
	`

	var value string
	err := r.db.QueryRow(ctx, query, key).Scan(&value)

	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get setting %s: %w", key, err)
	}

	return value, nil
}

// GetAll retrieves all settings as a map
func (r *SettingsRepo) GetAll(ctx context.Context) (map[string]string, error) {
	query := `
		SELECT key, value
		FROM system_settings
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		err := rows.Scan(&key, &value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settings: %w", err)
	}

	return settings, nil
}

// Update updates a setting value
func (r *SettingsRepo) Update(ctx context.Context, key, value string, updatedBy string) error {
	query := `
		INSERT INTO system_settings (key, value, updated_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value,
		    updated_at = now(),
		    updated_by = EXCLUDED.updated_by
	`

	_, err := r.db.Exec(ctx, query, key, value, updatedBy)
	if err != nil {
		return fmt.Errorf("failed to update setting %s: %w", key, err)
	}

	return nil
}
