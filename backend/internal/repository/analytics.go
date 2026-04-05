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

// AnalyticsRepo provides database operations for analytics queries
type AnalyticsRepo struct {
	db *pgxpool.Pool
}

// NewAnalyticsRepo creates a new analytics repository
func NewAnalyticsRepo(db *pgxpool.Pool) *AnalyticsRepo {
	return &AnalyticsRepo{db: db}
}

// ReceiptWithCurrency represents a receipt with its currency for conversion
type ReceiptWithCurrency struct {
	ID       uuid.UUID
	Title    string
	Total    float64
	Currency string
	Date     time.Time
}

// SummaryResult holds the summary statistics result
type SummaryResult struct {
	TotalSpend    float64
	ReceiptCount  int
	AvgPerReceipt float64
}

// GetSummary retrieves summary stats (total, count, average) for a date range
func (r *AnalyticsRepo) GetSummary(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) (*SummaryResult, error) {
	query := `
		SELECT 
			COALESCE(SUM(total), 0) as total_spend,
			COUNT(*) as receipt_count,
			COALESCE(AVG(total), 0) as avg_per_receipt
		FROM receipts
		WHERE user_id = $1 
		  AND receipt_date >= $2 
		  AND receipt_date < $3
		  AND status != $4
	`

	var result SummaryResult
	err := r.db.QueryRow(ctx, query, userID, fromDate, toDate, model.ReceiptStatusRejected).Scan(
		&result.TotalSpend,
		&result.ReceiptCount,
		&result.AvgPerReceipt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	return &result, nil
}

// MonthDataResult holds monthly aggregation data
type MonthDataResult struct {
	Month string
	Total float64
	Count int
}

// GetMonthlyTrends retrieves monthly aggregation for trends
func (r *AnalyticsRepo) GetMonthlyTrends(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) ([]*MonthDataResult, error) {
	query := `
		SELECT 
			TO_CHAR(receipt_date, 'YYYY-MM') as month,
			COALESCE(SUM(total), 0) as total,
			COUNT(*) as count
		FROM receipts
		WHERE user_id = $1 
		  AND receipt_date >= $2 
		  AND receipt_date < $3
		  AND status != $4
		GROUP BY TO_CHAR(receipt_date, 'YYYY-MM')
		ORDER BY month ASC
	`

	rows, err := r.db.Query(ctx, query, userID, fromDate, toDate, model.ReceiptStatusRejected)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly trends: %w", err)
	}
	defer rows.Close()

	var results []*MonthDataResult
	for rows.Next() {
		var data MonthDataResult
		err := rows.Scan(&data.Month, &data.Total, &data.Count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monthly data: %w", err)
		}
		results = append(results, &data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating monthly data: %w", err)
	}

	return results, nil
}

// TagSpendResult holds tag-based aggregation data
type TagSpendResult struct {
	TagID uuid.UUID
	Name  string
	Color *string
	Total float64
	Count int
}

// GetByTag retrieves tag-based aggregation for a date range
func (r *AnalyticsRepo) GetByTag(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) ([]*TagSpendResult, error) {
	query := `
		SELECT 
			t.id as tag_id,
			t.name,
			t.color,
			COALESCE(SUM(r.total), 0) as total,
			COUNT(r.id) as count
		FROM tags t
		LEFT JOIN receipt_tags rt ON t.id = rt.tag_id
		LEFT JOIN receipts r ON rt.receipt_id = r.id 
			AND r.receipt_date >= $2 
			AND r.receipt_date < $3
			AND r.status != $4
		WHERE t.user_id = $1
		GROUP BY t.id, t.name, t.color
		HAVING COUNT(r.id) > 0
		ORDER BY total DESC
	`

	rows, err := r.db.Query(ctx, query, userID, fromDate, toDate, model.ReceiptStatusRejected)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag spend: %w", err)
	}
	defer rows.Close()

	var results []*TagSpendResult
	for rows.Next() {
		var data TagSpendResult
		err := rows.Scan(&data.TagID, &data.Name, &data.Color, &data.Total, &data.Count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag data: %w", err)
		}
		results = append(results, &data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tag data: %w", err)
	}

	return results, nil
}

// ShopSpendResult holds shop-based aggregation data
type ShopSpendResult struct {
	Name       string
	Total      float64
	VisitCount int
	AvgTicket  float64
	LastVisit  time.Time
}

// GetByShop retrieves shop-based aggregation with visit counts for a date range
func (r *AnalyticsRepo) GetByShop(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) ([]*ShopSpendResult, error) {
	query := `
		SELECT 
			COALESCE(NULLIF(title, ''), 'Unknown') as name,
			COALESCE(SUM(total), 0) as total,
			COUNT(*) as visit_count,
			COALESCE(AVG(total), 0) as avg_ticket,
			MAX(receipt_date) as last_visit
		FROM receipts
		WHERE user_id = $1 
		  AND receipt_date >= $2 
		  AND receipt_date < $3
		  AND status != $4
		GROUP BY COALESCE(NULLIF(title, ''), 'Unknown')
		ORDER BY total DESC
	`

	rows, err := r.db.Query(ctx, query, userID, fromDate, toDate, model.ReceiptStatusRejected)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop spend: %w", err)
	}
	defer rows.Close()

	var results []*ShopSpendResult
	for rows.Next() {
		var data ShopSpendResult
		err := rows.Scan(&data.Name, &data.Total, &data.VisitCount, &data.AvgTicket, &data.LastVisit)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shop data: %w", err)
		}
		results = append(results, &data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shop data: %w", err)
	}

	return results, nil
}

// BiggestReceiptResult holds the biggest receipt info
type BiggestReceiptResult struct {
	ID    uuid.UUID
	Title string
	Total float64
	Date  time.Time
}

// GetBiggestReceipt retrieves the biggest receipt in the date range
func (r *AnalyticsRepo) GetBiggestReceipt(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) (*BiggestReceiptResult, error) {
	query := `
		SELECT 
			id,
			COALESCE(NULLIF(title, ''), 'Unknown') as title,
			total,
			receipt_date
		FROM receipts
		WHERE user_id = $1 
		  AND receipt_date >= $2 
		  AND receipt_date < $3
		  AND status != $4
		ORDER BY total DESC
		LIMIT 1
	`

	var result BiggestReceiptResult
	err := r.db.QueryRow(ctx, query, userID, fromDate, toDate, model.ReceiptStatusRejected).Scan(
		&result.ID,
		&result.Title,
		&result.Total,
		&result.Date,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get biggest receipt: %w", err)
	}

	return &result, nil
}

// MostVisitedShopResult holds the most visited shop info
type MostVisitedShopResult struct {
	Name   string
	Visits int
}

// GetMostVisitedShop retrieves the most visited shop in the date range
func (r *AnalyticsRepo) GetMostVisitedShop(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) (*MostVisitedShopResult, error) {
	query := `
		SELECT 
			COALESCE(NULLIF(title, ''), 'Unknown') as name,
			COUNT(*) as visits
		FROM receipts
		WHERE user_id = $1 
		  AND receipt_date >= $2 
		  AND receipt_date < $3
		  AND status != $4
		GROUP BY COALESCE(NULLIF(title, ''), 'Unknown')
		ORDER BY visits DESC, SUM(total) DESC
		LIMIT 1
	`

	var result MostVisitedShopResult
	err := r.db.QueryRow(ctx, query, userID, fromDate, toDate, model.ReceiptStatusRejected).Scan(
		&result.Name,
		&result.Visits,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get most visited shop: %w", err)
	}

	return &result, nil
}

// GetTotalForPeriod retrieves the total spend for a specific period (for MoM calculation)
func (r *AnalyticsRepo) GetTotalForPeriod(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) (float64, error) {
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
		return 0, fmt.Errorf("failed to get total for period: %w", err)
	}

	return total, nil
}

// BusiestDayResult holds the busiest day info
type BusiestDayResult struct {
	Day   string
	Total float64
}

// GetBusiestDay retrieves the day of week with highest total spend
func (r *AnalyticsRepo) GetBusiestDay(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) (*BusiestDayResult, error) {
	query := `
		SELECT 
			TO_CHAR(receipt_date, 'Day') as day,
			COALESCE(SUM(total), 0) as total
		FROM receipts
		WHERE user_id = $1 
		  AND receipt_date >= $2 
		  AND receipt_date < $3
		  AND status != $4
		GROUP BY TO_CHAR(receipt_date, 'Day'), EXTRACT(DOW FROM receipt_date)
		ORDER BY total DESC
		LIMIT 1
	`

	var result BusiestDayResult
	err := r.db.QueryRow(ctx, query, userID, fromDate, toDate, model.ReceiptStatusRejected).Scan(
		&result.Day,
		&result.Total,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get busiest day: %w", err)
	}

	// Trim whitespace from day name
	result.Day = trimDayName(result.Day)

	return &result, nil
}

// trimDayName removes extra whitespace from PostgreSQL day names
func trimDayName(day string) string {
	// PostgreSQL TO_CHAR with 'Day' adds padding to 9 characters
	// We need to trim it
	dayMap := map[string]string{
		"Sunday   ": "Sunday",
		"Monday   ": "Monday",
		"Tuesday  ": "Tuesday",
		"Wednesday": "Wednesday",
		"Thursday ": "Thursday",
		"Friday   ": "Friday",
		"Saturday ": "Saturday",
	}

	// Try direct map lookup first
	if clean, ok := dayMap[day]; ok {
		return clean
	}

	// Fallback: trim all whitespace
	clean := ""
	for _, c := range day {
		if c != ' ' {
			clean += string(c)
		}
	}
	return clean
}

// GetReceiptsWithCurrency retrieves all receipts with their currencies for a date range
// This is used for currency conversion in analytics
func (r *AnalyticsRepo) GetReceiptsWithCurrency(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) ([]*ReceiptWithCurrency, error) {
	query := `
		SELECT 
			id,
			COALESCE(NULLIF(title, ''), 'Unknown') as title,
			total,
			currency,
			receipt_date
		FROM receipts
		WHERE user_id = $1 
		  AND receipt_date >= $2 
		  AND receipt_date < $3
		  AND status != $4
		ORDER BY receipt_date DESC
	`

	rows, err := r.db.Query(ctx, query, userID, fromDate, toDate, model.ReceiptStatusRejected)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipts with currency: %w", err)
	}
	defer rows.Close()

	var results []*ReceiptWithCurrency
	for rows.Next() {
		var receipt ReceiptWithCurrency
		err := rows.Scan(&receipt.ID, &receipt.Title, &receipt.Total, &receipt.Currency, &receipt.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to scan receipt: %w", err)
		}
		results = append(results, &receipt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating receipts: %w", err)
	}

	return results, nil
}
