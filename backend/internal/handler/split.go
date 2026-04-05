package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/middleware"
	"github.com/receipt-manager/backend/internal/model"
	"github.com/receipt-manager/backend/internal/repository"
)

// SplitHandler handles receipt split HTTP requests
type SplitHandler struct {
	cfg         *config.Config
	splitRepo   *repository.SplitRepo
	receiptRepo *repository.ReceiptRepo
}

// NewSplitHandler creates a new split handler
func NewSplitHandler(cfg *config.Config, splitRepo *repository.SplitRepo, receiptRepo *repository.ReceiptRepo) *SplitHandler {
	return &SplitHandler{
		cfg:         cfg,
		splitRepo:   splitRepo,
		receiptRepo: receiptRepo,
	}
}

// CreateSplits handles POST /api/v1/receipts/:id/splits
func (h *SplitHandler) CreateSplits(w http.ResponseWriter, r *http.Request) {
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

	// Get receipt ID from URL
	receiptIDStr := chi.URLParam(r, "id")
	receiptID, err := uuid.Parse(receiptIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid receipt ID"}`, http.StatusBadRequest)
		return
	}

	// Verify receipt ownership
	receipt, err := h.receiptRepo.GetByID(r.Context(), receiptID)
	if err != nil {
		http.Error(w, `{"error": "failed to get receipt"}`, http.StatusInternalServerError)
		return
	}
	if receipt == nil {
		http.Error(w, `{"error": "receipt not found"}`, http.StatusNotFound)
		return
	}
	if receipt.UserID != userID {
		http.Error(w, `{"error": "receipt not found"}`, http.StatusNotFound)
		return
	}

	// Parse request body
	var req model.CreateSplitsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate splits
	if len(req.Splits) == 0 {
		http.Error(w, `{"error": "at least one split is required"}`, http.StatusBadRequest)
		return
	}

	// Validate each split and calculate total
	var totalPercentage float64
	var hasPercentage bool
	for _, split := range req.Splits {
		if split.UserID == "" {
			http.Error(w, `{"error": "user_id is required for each split"}`, http.StatusBadRequest)
			return
		}
		if _, err := uuid.Parse(split.UserID); err != nil {
			http.Error(w, `{"error": "invalid user_id format"}`, http.StatusBadRequest)
			return
		}
		if split.Amount < 0 {
			http.Error(w, `{"error": "amount cannot be negative"}`, http.StatusBadRequest)
			return
		}
		if split.Percentage < 0 || split.Percentage > 100 {
			http.Error(w, `{"error": "percentage must be between 0 and 100"}`, http.StatusBadRequest)
			return
		}
		if split.Percentage > 0 {
			hasPercentage = true
			totalPercentage += split.Percentage
		}
	}

	// If percentages are used, they should sum to approximately 100
	if hasPercentage && (totalPercentage < 99.99 || totalPercentage > 100.01) {
		http.Error(w, `{"error": "percentages must sum to 100"}`, http.StatusBadRequest)
		return
	}

	// Delete existing splits for this receipt
	if err := h.splitRepo.DeleteSplitsByReceipt(r.Context(), receiptID); err != nil {
		http.Error(w, `{"error": "failed to delete existing splits"}`, http.StatusInternalServerError)
		return
	}

	// Create new splits
	createdSplits := make([]*model.ReceiptSplit, 0, len(req.Splits))
	for _, splitReq := range req.Splits {
		userUUID, _ := uuid.Parse(splitReq.UserID)

		var percentage *float64
		if splitReq.Percentage > 0 {
			p := splitReq.Percentage
			percentage = &p
		}

		split := &model.ReceiptSplit{
			ReceiptID:  receiptID,
			UserID:     userUUID,
			Amount:     splitReq.Amount,
			Percentage: percentage,
		}

		created, err := h.splitRepo.CreateSplit(r.Context(), split)
		if err != nil {
			http.Error(w, `{"error": "failed to create split"}`, http.StatusInternalServerError)
			return
		}

		createdSplits = append(createdSplits, created)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"splits": createdSplits,
	})
}

// GetSplits handles GET /api/v1/receipts/:id/splits
func (h *SplitHandler) GetSplits(w http.ResponseWriter, r *http.Request) {
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

	// Get receipt ID from URL
	receiptIDStr := chi.URLParam(r, "id")
	receiptID, err := uuid.Parse(receiptIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid receipt ID"}`, http.StatusBadRequest)
		return
	}

	// Verify receipt ownership
	receipt, err := h.receiptRepo.GetByID(r.Context(), receiptID)
	if err != nil {
		http.Error(w, `{"error": "failed to get receipt"}`, http.StatusInternalServerError)
		return
	}
	if receipt == nil {
		http.Error(w, `{"error": "receipt not found"}`, http.StatusNotFound)
		return
	}
	if receipt.UserID != userID {
		http.Error(w, `{"error": "receipt not found"}`, http.StatusNotFound)
		return
	}

	// Get splits
	splits, err := h.splitRepo.GetSplitsByReceipt(r.Context(), receiptID)
	if err != nil {
		http.Error(w, `{"error": "failed to get splits"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"splits": splits,
	})
}

// GetSettlements handles GET /api/v1/splits/settlements
func (h *SplitHandler) GetSettlements(w http.ResponseWriter, r *http.Request) {
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

	// Placeholder: Return empty settlements
	// Full implementation would require group membership logic
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"settlements": []model.Settlement{},
		"message":     "settlements feature coming soon",
	})
}
