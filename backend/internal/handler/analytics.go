package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/middleware"
	"github.com/receipt-manager/backend/internal/service"
)

// AnalyticsHandler handles analytics HTTP requests
type AnalyticsHandler struct {
	cfg              *config.Config
	analyticsService *service.AnalyticsService
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(cfg *config.Config, analyticsService *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		cfg:              cfg,
		analyticsService: analyticsService,
	}
}

// parseDateRange parses from and to query parameters with defaults
func parseDateRange(r *http.Request) (fromDate, toDate time.Time, err error) {
	// Default: last 30 days
	now := time.Now()
	toDate = now.Add(24 * time.Hour).Truncate(24 * time.Hour) // Tomorrow for inclusive today
	fromDate = now.AddDate(0, 0, -30).Truncate(24 * time.Hour)

	// Parse from date
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			// Try YYYY-MM-DD format
			t, err = time.Parse("2006-01-02", fromStr)
			if err != nil {
				return time.Time{}, time.Time{}, err
			}
		}
		fromDate = t
	}

	// Parse to date
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			// Try YYYY-MM-DD format
			t, err = time.Parse("2006-01-02", toStr)
			if err != nil {
				return time.Time{}, time.Time{}, err
			}
		}
		toDate = t.Add(24 * time.Hour) // Add one day to make it inclusive
	}

	return fromDate, toDate, nil
}

// Summary handles GET /api/v1/analytics/summary
func (h *AnalyticsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context (JWT auth is applied via middleware)
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Parse date range
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, `{"error": "invalid date format, use RFC3339 or YYYY-MM-DD"}`, http.StatusBadRequest)
		return
	}

	// Validate date range
	if fromDate.After(toDate) {
		http.Error(w, `{"error": "from date must be before to date"}`, http.StatusBadRequest)
		return
	}

	// Call service
	summary, warnings, err := h.analyticsService.GetSummary(r.Context(), userID, fromDate, toDate)
	if err != nil {
		http.Error(w, `{"error": "failed to get summary"}`, http.StatusInternalServerError)
		return
	}

	// Build response
	response := struct {
		Data     *service.Summary            `json:"data"`
		Warnings []service.ConversionWarning `json:"warnings,omitempty"`
	}{
		Data:     summary,
		Warnings: warnings,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Monthly handles GET /api/v1/analytics/monthly
func (h *AnalyticsHandler) Monthly(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Parse months parameter (default: 12)
	months := 12
	if monthsStr := r.URL.Query().Get("months"); monthsStr != "" {
		if m, err := strconv.Atoi(monthsStr); err == nil && m > 0 && m <= 24 {
			months = m
		}
	}

	// Call service
	data, warnings, err := h.analyticsService.GetMonthlyTrends(r.Context(), userID, months)
	if err != nil {
		http.Error(w, `{"error": "failed to get monthly trends"}`, http.StatusInternalServerError)
		return
	}

	// Build response
	response := struct {
		Data     []service.MonthData         `json:"data"`
		Warnings []service.ConversionWarning `json:"warnings,omitempty"`
	}{
		Data:     data,
		Warnings: warnings,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ByTag handles GET /api/v1/analytics/by-tag
func (h *AnalyticsHandler) ByTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Parse date range
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, `{"error": "invalid date format, use RFC3339 or YYYY-MM-DD"}`, http.StatusBadRequest)
		return
	}

	// Validate date range
	if fromDate.After(toDate) {
		http.Error(w, `{"error": "from date must be before to date"}`, http.StatusBadRequest)
		return
	}

	// Call service
	data, warnings, err := h.analyticsService.GetByTag(r.Context(), userID, fromDate, toDate)
	if err != nil {
		http.Error(w, `{"error": "failed to get tag analytics"}`, http.StatusInternalServerError)
		return
	}

	// Build response
	response := struct {
		Data     []service.TagSpend          `json:"data"`
		Warnings []service.ConversionWarning `json:"warnings,omitempty"`
	}{
		Data:     data,
		Warnings: warnings,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ByShop handles GET /api/v1/analytics/by-shop
func (h *AnalyticsHandler) ByShop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Parse date range
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, `{"error": "invalid date format, use RFC3339 or YYYY-MM-DD"}`, http.StatusBadRequest)
		return
	}

	// Validate date range
	if fromDate.After(toDate) {
		http.Error(w, `{"error": "from date must be before to date"}`, http.StatusBadRequest)
		return
	}

	// Call service
	data, warnings, err := h.analyticsService.GetByShop(r.Context(), userID, fromDate, toDate)
	if err != nil {
		http.Error(w, `{"error": "failed to get shop analytics"}`, http.StatusInternalServerError)
		return
	}

	// Build response
	response := struct {
		Data     []service.ShopSpend         `json:"data"`
		Warnings []service.ConversionWarning `json:"warnings,omitempty"`
	}{
		Data:     data,
		Warnings: warnings,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Insights handles GET /api/v1/analytics/insights
func (h *AnalyticsHandler) Insights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Parse date range
	fromDate, toDate, err := parseDateRange(r)
	if err != nil {
		http.Error(w, `{"error": "invalid date format, use RFC3339 or YYYY-MM-DD"}`, http.StatusBadRequest)
		return
	}

	// Validate date range
	if fromDate.After(toDate) {
		http.Error(w, `{"error": "from date must be before to date"}`, http.StatusBadRequest)
		return
	}

	// Call service
	data, warnings, err := h.analyticsService.GetInsights(r.Context(), userID, fromDate, toDate)
	if err != nil {
		http.Error(w, `{"error": "failed to get insights"}`, http.StatusInternalServerError)
		return
	}

	// Build response
	response := struct {
		Data     *service.Insights           `json:"data"`
		Warnings []service.ConversionWarning `json:"warnings,omitempty"`
	}{
		Data:     data,
		Warnings: warnings,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
