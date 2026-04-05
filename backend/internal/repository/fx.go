package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FXRepo provides database operations for FX rates
type FXRepo struct {
	db *pgxpool.Pool
}

// NewFXRepo creates a new FX repository
func NewFXRepo(db *pgxpool.Pool) *FXRepo {
	return &FXRepo{db: db}
}

// GetRate retrieves the FX rate from base to target currency
func (r *FXRepo) GetRate(ctx context.Context, base, target string) (float64, time.Time, error) {
	query := `
		SELECT rate, updated_at
		FROM fx_rates
		WHERE base_currency = $1 AND target_currency = $2
	`

	var rate float64
	var updatedAt time.Time

	err := r.db.QueryRow(ctx, query, base, target).Scan(&rate, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, time.Time{}, nil
		}
		return 0, time.Time{}, fmt.Errorf("failed to get FX rate: %w", err)
	}

	return rate, updatedAt, nil
}

// SaveRate saves or updates an FX rate
func (r *FXRepo) SaveRate(ctx context.Context, base, target string, rate float64) error {
	query := `
		INSERT INTO fx_rates (base_currency, target_currency, rate, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (base_currency, target_currency) DO UPDATE
		SET rate = EXCLUDED.rate,
		    updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.Exec(ctx, query, base, target, rate)
	if err != nil {
		return fmt.Errorf("failed to save FX rate: %w", err)
	}

	return nil
}

// GetRatesForBase retrieves all rates for a base currency
func (r *FXRepo) GetRatesForBase(ctx context.Context, base string) (map[string]float64, error) {
	query := `
		SELECT target_currency, rate
		FROM fx_rates
		WHERE base_currency = $1
	`

	rows, err := r.db.Query(ctx, query, base)
	if err != nil {
		return nil, fmt.Errorf("failed to get FX rates for base: %w", err)
	}
	defer rows.Close()

	rates := make(map[string]float64)
	for rows.Next() {
		var target string
		var rate float64
		if err := rows.Scan(&target, &rate); err != nil {
			return nil, fmt.Errorf("failed to scan FX rate: %w", err)
		}
		rates[target] = rate
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating FX rates: %w", err)
	}

	return rates, nil
}
