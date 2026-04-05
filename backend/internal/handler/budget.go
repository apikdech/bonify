package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/middleware"
	"github.com/receipt-manager/backend/internal/model"
	"github.com/receipt-manager/backend/internal/service"
)

// BudgetHandler handles budget HTTP requests
type BudgetHandler struct {
	cfg           *config.Config
	budgetService *service.BudgetService
	budgetRepo    BudgetRepository
}

// BudgetRepository defines the budget repository interface needed by the handler
type BudgetRepository interface {
	Create(ctx context.Context, budget *model.Budget) (*model.Budget, error)
	GetByUserAndMonth(ctx context.Context, userID uuid.UUID, month string) ([]*model.Budget, error)
	Update(ctx context.Context, budget *model.Budget) (*model.Budget, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Budget, error)
}

// NewBudgetHandler creates a new budget handler
func NewBudgetHandler(cfg *config.Config, budgetService *service.BudgetService, budgetRepo BudgetRepository) *BudgetHandler {
	return &BudgetHandler{
		cfg:           cfg,
		budgetService: budgetService,
		budgetRepo:    budgetRepo,
	}
}

// CreateBudgetRequest represents a request to create a budget
type CreateBudgetRequest struct {
	TagID       *uuid.UUID `json:"tag_id,omitempty"`
	Month       string     `json:"month"`
	AmountLimit float64    `json:"amount_limit"`
}

// UpdateBudgetRequest represents a request to update a budget
type UpdateBudgetRequest struct {
	TagID       *uuid.UUID `json:"tag_id,omitempty"`
	Month       string     `json:"month,omitempty"`
	AmountLimit float64    `json:"amount_limit,omitempty"`
}

// List handles GET /budgets
func (h *BudgetHandler) List(w http.ResponseWriter, r *http.Request) {
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

	// Default to current month if not provided
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}

	// Call repository to get budgets
	budgets, err := h.budgetRepo.GetByUserAndMonth(r.Context(), userID, month)
	if err != nil {
		http.Error(w, `{"error": "failed to list budgets"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"budgets": budgets,
		"month":   month,
	})
}

// Create handles POST /budgets
func (h *BudgetHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req CreateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate month format (YYYY-MM)
	if req.Month == "" {
		http.Error(w, `{"error": "month is required"}`, http.StatusBadRequest)
		return
	}
	_, err := time.Parse("2006-01", req.Month)
	if err != nil {
		http.Error(w, `{"error": "month must be in YYYY-MM format"}`, http.StatusBadRequest)
		return
	}

	// Validate amount limit
	if req.AmountLimit <= 0 {
		http.Error(w, `{"error": "amount_limit must be greater than 0"}`, http.StatusBadRequest)
		return
	}

	// Create budget model
	budget := &model.Budget{
		UserID:      userID,
		TagID:       req.TagID,
		Month:       req.Month,
		AmountLimit: req.AmountLimit,
	}

	// Call repository
	created, err := h.budgetRepo.Create(r.Context(), budget)
	if err != nil {
		http.Error(w, `{"error": "failed to create budget"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// Update handles PATCH /budgets/:id
func (h *BudgetHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Get budget ID from URL
	budgetIDStr := chi.URLParam(r, "id")
	budgetID, err := uuid.Parse(budgetIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid budget ID"}`, http.StatusBadRequest)
		return
	}

	// Check if budget exists and belongs to user
	existing, err := h.budgetRepo.GetByID(r.Context(), budgetID)
	if err != nil {
		http.Error(w, `{"error": "failed to get budget"}`, http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, `{"error": "budget not found"}`, http.StatusNotFound)
		return
	}
	if existing.UserID != userID {
		http.Error(w, `{"error": "access denied"}`, http.StatusForbidden)
		return
	}

	// Parse request body
	var req UpdateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Update fields if provided
	if req.TagID != nil {
		existing.TagID = req.TagID
	}
	if req.Month != "" {
		_, err := time.Parse("2006-01", req.Month)
		if err != nil {
			http.Error(w, `{"error": "month must be in YYYY-MM format"}`, http.StatusBadRequest)
			return
		}
		existing.Month = req.Month
	}
	if req.AmountLimit > 0 {
		existing.AmountLimit = req.AmountLimit
	}

	// Call repository
	updated, err := h.budgetRepo.Update(r.Context(), existing)
	if err != nil {
		if errors.Is(err, errors.New("budget not found or access denied")) {
			http.Error(w, `{"error": "budget not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "failed to update budget"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updated)
}

// Delete handles DELETE /budgets/:id
func (h *BudgetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Get budget ID from URL
	budgetIDStr := chi.URLParam(r, "id")
	budgetID, err := uuid.Parse(budgetIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid budget ID"}`, http.StatusBadRequest)
		return
	}

	// Call repository
	err = h.budgetRepo.Delete(r.Context(), budgetID, userID)
	if err != nil {
		if err.Error() == "budget not found or access denied" {
			http.Error(w, `{"error": "budget not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "failed to delete budget"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "budget deleted successfully"})
}

// GetStatus handles GET /budgets/status?month=
func (h *BudgetHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
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

	// Get month from query param, default to current month
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}

	// Validate month format
	_, err := time.Parse("2006-01", month)
	if err != nil {
		http.Error(w, `{"error": "month must be in YYYY-MM format"}`, http.StatusBadRequest)
		return
	}

	// Call service to get budget status
	status, err := h.budgetService.GetBudgetStatus(r.Context(), userID, month)
	if err != nil {
		http.Error(w, `{"error": "failed to get budget status"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"month":  month,
	})
}
