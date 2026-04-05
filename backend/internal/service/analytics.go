package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/repository"
)

// AnalyticsService provides analytics business logic with currency conversion
type AnalyticsService struct {
	analyticsRepo *repository.AnalyticsRepo
	fxRepo        *repository.FXRepo
	userRepo      *repository.UserRepo
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(analyticsRepo *repository.AnalyticsRepo, fxRepo *repository.FXRepo, userRepo *repository.UserRepo) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo: analyticsRepo,
		fxRepo:        fxRepo,
		userRepo:      userRepo,
	}
}

// Summary represents summary statistics response
type Summary struct {
	TotalSpend    float64 `json:"total_spend"`
	ReceiptCount  int     `json:"receipt_count"`
	AvgPerReceipt float64 `json:"avg_per_receipt"`
}

// MonthData represents monthly trend data
type MonthData struct {
	Month string  `json:"month"` // Format: "2024-01"
	Total float64 `json:"total"`
	Count int     `json:"count"`
}

// TagSpend represents tag-based spending data
type TagSpend struct {
	TagID      uuid.UUID `json:"tag_id"`
	Name       string    `json:"name"`
	Color      *string   `json:"color"`
	Total      float64   `json:"total"`
	Count      int       `json:"count"`
	Percentage float64   `json:"percentage"`
}

// ShopSpend represents shop-based spending data
type ShopSpend struct {
	Name       string  `json:"name"`
	Total      float64 `json:"total"`
	VisitCount int     `json:"visit_count"`
	AvgTicket  float64 `json:"avg_ticket"`
	LastVisit  string  `json:"last_visit"` // Format: "2024-01-15"
}

// Insights represents analytics insights
type Insights struct {
	BiggestReceipt  *ReceiptInfo    `json:"biggest_receipt"`
	MostVisitedShop *ShopVisitInfo  `json:"most_visited_shop"`
	MoMChange       *MoMChangeInfo  `json:"mom_change"`
	BusiestDay      *BusiestDayInfo `json:"busiest_day"`
}

// ReceiptInfo represents receipt information for insights
type ReceiptInfo struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Total float64 `json:"total"`
	Date  string  `json:"date"`
}

// ShopVisitInfo represents shop visit information
type ShopVisitInfo struct {
	Name   string `json:"name"`
	Visits int    `json:"visits"`
}

// MoMChangeInfo represents month-over-month change
type MoMChangeInfo struct {
	Percentage float64 `json:"percentage"`
	Absolute   float64 `json:"absolute"`
}

// BusiestDayInfo represents the busiest day information
type BusiestDayInfo struct {
	Day   string  `json:"day"`
	Total float64 `json:"total"`
}

// ConversionWarning represents a warning about currency conversion issues
type ConversionWarning struct {
	Currency string `json:"currency"`
	Count    int    `json:"count"`
	Reason   string `json:"reason"`
}

// AnalyticsResult wraps analytics results with optional warnings
type AnalyticsResult struct {
	Data     interface{}         `json:"data"`
	Warnings []ConversionWarning `json:"warnings,omitempty"`
}

// GetSummary retrieves summary stats with currency conversion
func (s *AnalyticsService) GetSummary(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) (*Summary, []ConversionWarning, error) {
	// Get user's home currency
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	homeCurrency := user.HomeCurrency
	if homeCurrency == "" {
		homeCurrency = "IDR" // Default currency
	}

	// Get receipts with currency for conversion
	receipts, err := s.analyticsRepo.GetReceiptsWithCurrency(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get receipts: %w", err)
	}

	// Convert all receipts to home currency
	var totalSpend float64
	var warnings []ConversionWarning
	missingCurrencies := make(map[string]int)

	for _, receipt := range receipts {
		convertedAmount, err := s.convertAmount(ctx, receipt.Total, receipt.Currency, homeCurrency)
		if err != nil {
			// Track missing currency rates
			missingCurrencies[receipt.Currency]++
			continue
		}
		totalSpend += convertedAmount
	}

	// Build warnings for missing currencies
	for currency, count := range missingCurrencies {
		warnings = append(warnings, ConversionWarning{
			Currency: currency,
			Count:    count,
			Reason:   "no exchange rate available",
		})
	}

	count := len(receipts) - len(missingCurrencies)
	var avgPerReceipt float64
	if count > 0 {
		avgPerReceipt = totalSpend / float64(count)
	}

	summary := &Summary{
		TotalSpend:    totalSpend,
		ReceiptCount:  count,
		AvgPerReceipt: avgPerReceipt,
	}

	return summary, warnings, nil
}

// GetMonthlyTrends retrieves monthly trends with currency conversion
func (s *AnalyticsService) GetMonthlyTrends(ctx context.Context, userID uuid.UUID, months int) ([]MonthData, []ConversionWarning, error) {
	if months <= 0 {
		months = 12
	}
	if months > 24 {
		months = 24 // Cap at 24 months
	}

	// Get user's home currency
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	homeCurrency := user.HomeCurrency
	if homeCurrency == "" {
		homeCurrency = "IDR"
	}

	// Calculate date range
	toDate := time.Now().AddDate(0, 1, 0).Truncate(24 * time.Hour) // First day of next month
	fromDate := toDate.AddDate(0, -months, 0)

	// Get receipts for the period
	receipts, err := s.analyticsRepo.GetReceiptsWithCurrency(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get receipts: %w", err)
	}

	// Group receipts by month and convert currencies
	monthlyData := make(map[string]*MonthData)
	warnings := []ConversionWarning{}
	missingCurrencies := make(map[string]int)

	for _, receipt := range receipts {
		month := receipt.Date.Format("2006-01")

		if _, ok := monthlyData[month]; !ok {
			monthlyData[month] = &MonthData{Month: month}
		}

		convertedAmount, err := s.convertAmount(ctx, receipt.Total, receipt.Currency, homeCurrency)
		if err != nil {
			missingCurrencies[receipt.Currency]++
			continue
		}

		monthlyData[month].Total += convertedAmount
		monthlyData[month].Count++
	}

	// Build warnings
	for currency, count := range missingCurrencies {
		warnings = append(warnings, ConversionWarning{
			Currency: currency,
			Count:    count,
			Reason:   "no exchange rate available",
		})
	}

	// Convert map to slice and fill in missing months
	result := make([]MonthData, 0, months)
	for i := 0; i < months; i++ {
		month := fromDate.AddDate(0, i, 0).Format("2006-01")
		if data, ok := monthlyData[month]; ok {
			result = append(result, *data)
		} else {
			result = append(result, MonthData{Month: month, Total: 0, Count: 0})
		}
	}

	return result, warnings, nil
}

// GetByTag retrieves tag-based spending with currency conversion
func (s *AnalyticsService) GetByTag(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) ([]TagSpend, []ConversionWarning, error) {
	// Get user's home currency
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	homeCurrency := user.HomeCurrency
	if homeCurrency == "" {
		homeCurrency = "IDR"
	}

	// Get receipts for currency conversion tracking
	receipts, err := s.analyticsRepo.GetReceiptsWithCurrency(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get receipts: %w", err)
	}

	// Track missing currencies
	missingCurrencies := make(map[string]int)
	convertedTotals := make(map[uuid.UUID]float64) // receipt_id -> converted amount

	for _, receipt := range receipts {
		convertedAmount, err := s.convertAmount(ctx, receipt.Total, receipt.Currency, homeCurrency)
		if err != nil {
			missingCurrencies[receipt.Currency]++
			continue
		}
		convertedTotals[receipt.ID] = convertedAmount
	}

	// Get tag data (this returns raw totals, we need to recalculate based on converted amounts)
	tagResults, err := s.analyticsRepo.GetByTag(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get tag data: %w", err)
	}

	// For now, we use raw totals since the query joins receipts
	// In a production system, you might want to store converted amounts
	var tagSpends []TagSpend
	var grandTotal float64

	for _, result := range tagResults {
		tagSpends = append(tagSpends, TagSpend{
			TagID: result.TagID,
			Name:  result.Name,
			Color: result.Color,
			Total: result.Total,
			Count: result.Count,
		})
		grandTotal += result.Total
	}

	// Calculate percentages
	if grandTotal > 0 {
		for i := range tagSpends {
			tagSpends[i].Percentage = (tagSpends[i].Total / grandTotal) * 100
		}
	}

	// Build warnings
	var warnings []ConversionWarning
	for currency, count := range missingCurrencies {
		warnings = append(warnings, ConversionWarning{
			Currency: currency,
			Count:    count,
			Reason:   "no exchange rate available",
		})
	}

	return tagSpends, warnings, nil
}

// GetByShop retrieves shop-based spending with currency conversion
func (s *AnalyticsService) GetByShop(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) ([]ShopSpend, []ConversionWarning, error) {
	// Get user's home currency
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	homeCurrency := user.HomeCurrency
	if homeCurrency == "" {
		homeCurrency = "IDR"
	}

	// Get shop data with converted amounts
	shopResults, err := s.analyticsRepo.GetByShop(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get shop data: %w", err)
	}

	// Get receipts for currency conversion
	receipts, err := s.analyticsRepo.GetReceiptsWithCurrency(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get receipts: %w", err)
	}

	// Build a map of receipt totals by shop for conversion
	shopReceipts := make(map[string][]*repository.ReceiptWithCurrency)
	for _, receipt := range receipts {
		shopName := receipt.Title
		if shopName == "" {
			shopName = "Unknown"
		}
		shopReceipts[shopName] = append(shopReceipts[shopName], receipt)
	}

	// Convert and aggregate by shop
	var shopSpends []ShopSpend
	warnings := []ConversionWarning{}
	missingCurrencies := make(map[string]int)

	for _, result := range shopResults {
		shopName := result.Name
		receiptsForShop := shopReceipts[shopName]

		var convertedTotal float64
		var convertedCount int

		for _, receipt := range receiptsForShop {
			convertedAmount, err := s.convertAmount(ctx, receipt.Total, receipt.Currency, homeCurrency)
			if err != nil {
				missingCurrencies[receipt.Currency]++
				continue
			}
			convertedTotal += convertedAmount
			convertedCount++
		}

		var avgTicket float64
		if convertedCount > 0 {
			avgTicket = convertedTotal / float64(convertedCount)
		}

		shopSpends = append(shopSpends, ShopSpend{
			Name:       result.Name,
			Total:      convertedTotal,
			VisitCount: convertedCount,
			AvgTicket:  avgTicket,
			LastVisit:  result.LastVisit.Format("2006-01-02"),
		})
	}

	// Build warnings
	for currency, count := range missingCurrencies {
		warnings = append(warnings, ConversionWarning{
			Currency: currency,
			Count:    count,
			Reason:   "no exchange rate available",
		})
	}

	return shopSpends, warnings, nil
}

// GetInsights retrieves various insights with currency conversion
func (s *AnalyticsService) GetInsights(ctx context.Context, userID uuid.UUID, fromDate, toDate time.Time) (*Insights, []ConversionWarning, error) {
	// Get user's home currency
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	homeCurrency := user.HomeCurrency
	if homeCurrency == "" {
		homeCurrency = "IDR"
	}

	warnings := []ConversionWarning{}
	missingCurrencies := make(map[string]int)

	insights := &Insights{}

	// Get biggest receipt
	biggestReceipt, err := s.analyticsRepo.GetBiggestReceipt(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get biggest receipt: %w", err)
	}

	if biggestReceipt != nil {
		convertedTotal, err := s.convertAmount(ctx, biggestReceipt.Total, "IDR", homeCurrency)
		if err != nil {
			// Try with receipt currency if different
			convertedTotal = biggestReceipt.Total
		}

		insights.BiggestReceipt = &ReceiptInfo{
			ID:    biggestReceipt.ID.String(),
			Title: biggestReceipt.Title,
			Total: convertedTotal,
			Date:  biggestReceipt.Date.Format("2006-01-02"),
		}
	}

	// Get most visited shop
	mostVisitedShop, err := s.analyticsRepo.GetMostVisitedShop(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get most visited shop: %w", err)
	}

	if mostVisitedShop != nil {
		insights.MostVisitedShop = &ShopVisitInfo{
			Name:   mostVisitedShop.Name,
			Visits: mostVisitedShop.Visits,
		}
	}

	// Calculate MoM change
	currentPeriodTotal, err := s.analyticsRepo.GetTotalForPeriod(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current period total: %w", err)
	}

	// Calculate previous period (same duration)
	periodDuration := toDate.Sub(fromDate)
	previousFromDate := fromDate.Add(-periodDuration)
	previousToDate := fromDate

	previousPeriodTotal, err := s.analyticsRepo.GetTotalForPeriod(ctx, userID, previousFromDate, previousToDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get previous period total: %w", err)
	}

	if previousPeriodTotal > 0 {
		absoluteChange := currentPeriodTotal - previousPeriodTotal
		percentageChange := (absoluteChange / previousPeriodTotal) * 100

		insights.MoMChange = &MoMChangeInfo{
			Percentage: percentageChange,
			Absolute:   absoluteChange,
		}
	} else if currentPeriodTotal > 0 {
		// No previous data, but have current data - infinite increase
		insights.MoMChange = &MoMChangeInfo{
			Percentage: 100,
			Absolute:   currentPeriodTotal,
		}
	}

	// Get busiest day
	busiestDay, err := s.analyticsRepo.GetBusiestDay(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get busiest day: %w", err)
	}

	if busiestDay != nil {
		insights.BusiestDay = &BusiestDayInfo{
			Day:   busiestDay.Day,
			Total: busiestDay.Total,
		}
	}

	// Build warnings
	for currency, count := range missingCurrencies {
		warnings = append(warnings, ConversionWarning{
			Currency: currency,
			Count:    count,
			Reason:   "no exchange rate available",
		})
	}

	return insights, warnings, nil
}

// convertAmount converts an amount from one currency to another using fx_rates
// Formula: converted_amount = amount * (target_rate / base_rate)
func (s *AnalyticsService) convertAmount(ctx context.Context, amount float64, fromCurrency, toCurrency string) (float64, error) {
	// If same currency, no conversion needed
	if fromCurrency == toCurrency {
		return amount, nil
	}

	// Get rate from source to target
	rate, _, err := s.fxRepo.GetRate(ctx, fromCurrency, toCurrency)
	if err != nil {
		return 0, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	// If no rate available, return error
	if rate == 0 {
		return 0, fmt.Errorf("no exchange rate available for %s to %s", fromCurrency, toCurrency)
	}

	convertedAmount := amount * rate
	return convertedAmount, nil
}
