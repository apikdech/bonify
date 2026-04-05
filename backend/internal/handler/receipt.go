package handler

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/middleware"
	"github.com/receipt-manager/backend/internal/model"
	"github.com/receipt-manager/backend/internal/service"
)

// ReceiptHandler handles receipt HTTP requests
type ReceiptHandler struct {
	cfg            *config.Config
	receiptService *service.ReceiptService
}

// NewReceiptHandler creates a new receipt handler
func NewReceiptHandler(cfg *config.Config, receiptService *service.ReceiptService) *ReceiptHandler {
	return &ReceiptHandler{
		cfg:            cfg,
		receiptService: receiptService,
	}
}

// List handles GET /receipts
func (h *ReceiptHandler) List(w http.ResponseWriter, r *http.Request) {
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

	// Parse query parameters
	filter := &model.ListReceiptsFilter{
		UserID: userID,
		Page:   1,
		Limit:  20,
	}

	// Parse status filter
	if status := r.URL.Query().Get("status"); status != "" {
		s := model.ReceiptStatus(status)
		filter.Status = &s
	}

	// Parse date range
	if from := r.URL.Query().Get("from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			http.Error(w, `{"error": "invalid from date format"}`, http.StatusBadRequest)
			return
		}
		filter.FromDate = &t
	}
	if to := r.URL.Query().Get("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			http.Error(w, `{"error": "invalid to date format"}`, http.StatusBadRequest)
			return
		}
		filter.ToDate = &t
	}

	// Parse search query
	if q := r.URL.Query().Get("q"); q != "" {
		filter.Query = &q
	}

	// Parse tag filter
	if tagID := r.URL.Query().Get("tag_id"); tagID != "" {
		if tid, err := uuid.Parse(tagID); err == nil {
			filter.TagID = &tid
		}
	}

	// Parse pagination
	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filter.Page = p
		}
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	// Call service
	response, err := h.receiptService.List(r.Context(), filter)
	if err != nil {
		http.Error(w, `{"error": "failed to list receipts"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Create handles POST /receipts
func (h *ReceiptHandler) Create(w http.ResponseWriter, r *http.Request) {
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
	var req model.CreateReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Currency == "" {
		http.Error(w, `{"error": "currency is required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, `{"error": "at least one item is required"}`, http.StatusBadRequest)
		return
	}

	// Validate items
	for _, item := range req.Items {
		if item.Name == "" {
			http.Error(w, `{"error": "item name is required"}`, http.StatusBadRequest)
			return
		}
		if item.Quantity <= 0 {
			http.Error(w, `{"error": "item quantity must be positive"}`, http.StatusBadRequest)
			return
		}
		if item.UnitPrice < 0 {
			http.Error(w, `{"error": "item unit price cannot be negative"}`, http.StatusBadRequest)
			return
		}
		if item.Discount < 0 {
			http.Error(w, `{"error": "item discount cannot be negative"}`, http.StatusBadRequest)
			return
		}
		if item.Discount > item.Quantity*item.UnitPrice {
			http.Error(w, `{"error": "item discount cannot exceed item total"}`, http.StatusBadRequest)
			return
		}
	}

	// Validate fees
	for _, fee := range req.Fees {
		if fee.Label == "" {
			http.Error(w, `{"error": "fee label is required"}`, http.StatusBadRequest)
			return
		}
		if fee.Amount < 0 {
			http.Error(w, `{"error": "fee amount cannot be negative"}`, http.StatusBadRequest)
			return
		}
		if !isValidFeeType(fee.FeeType) {
			http.Error(w, `{"error": "invalid fee type"}`, http.StatusBadRequest)
			return
		}
	}

	// Call service
	receipt, err := h.receiptService.CreateManual(r.Context(), userID, &req)
	if err != nil {
		http.Error(w, `{"error": "failed to create receipt"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(receipt)
}

// Get handles GET /receipts/:id
func (h *ReceiptHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	// Call service
	receipt, err := h.receiptService.GetByID(r.Context(), receiptID, userID)
	if err != nil {
		http.Error(w, `{"error": "failed to get receipt"}`, http.StatusInternalServerError)
		return
	}
	if receipt == nil {
		http.Error(w, `{"error": "receipt not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(receipt)
}

// Update handles PATCH /receipts/:id
func (h *ReceiptHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	// Get receipt ID from URL
	receiptIDStr := chi.URLParam(r, "id")
	receiptID, err := uuid.Parse(receiptIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid receipt ID"}`, http.StatusBadRequest)
		return
	}

	// Parse request body
	var req model.UpdateReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate items if provided
	if req.Items != nil {
		for _, item := range *req.Items {
			if item.Name == "" {
				http.Error(w, `{"error": "item name is required"}`, http.StatusBadRequest)
				return
			}
			if item.Quantity <= 0 {
				http.Error(w, `{"error": "item quantity must be positive"}`, http.StatusBadRequest)
				return
			}
			if item.UnitPrice < 0 {
				http.Error(w, `{"error": "item unit price cannot be negative"}`, http.StatusBadRequest)
				return
			}
			if item.Discount < 0 {
				http.Error(w, `{"error": "item discount cannot be negative"}`, http.StatusBadRequest)
				return
			}
			if item.Discount > item.Quantity*item.UnitPrice {
				http.Error(w, `{"error": "item discount cannot exceed item total"}`, http.StatusBadRequest)
				return
			}
		}
	}

	// Validate fees if provided
	if req.Fees != nil {
		for _, fee := range *req.Fees {
			if fee.Label == "" {
				http.Error(w, `{"error": "fee label is required"}`, http.StatusBadRequest)
				return
			}
			if fee.Amount < 0 {
				http.Error(w, `{"error": "fee amount cannot be negative"}`, http.StatusBadRequest)
				return
			}
			if !isValidFeeType(fee.FeeType) {
				http.Error(w, `{"error": "invalid fee type"}`, http.StatusBadRequest)
				return
			}
		}
	}

	// Call service
	receipt, err := h.receiptService.Update(r.Context(), receiptID, userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrReceiptNotFound) {
			http.Error(w, `{"error": "receipt not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "failed to update receipt"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(receipt)
}

// Delete handles DELETE /receipts/:id
func (h *ReceiptHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	// Get receipt ID from URL
	receiptIDStr := chi.URLParam(r, "id")
	receiptID, err := uuid.Parse(receiptIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid receipt ID"}`, http.StatusBadRequest)
		return
	}

	// Call service
	err = h.receiptService.Delete(r.Context(), receiptID, userID)
	if err != nil {
		if errors.Is(err, service.ErrReceiptNotFound) {
			http.Error(w, `{"error": "receipt not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "failed to delete receipt"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "receipt deleted successfully"})
}

// Confirm handles PATCH /receipts/:id/confirm
func (h *ReceiptHandler) Confirm(w http.ResponseWriter, r *http.Request) {
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

	// Get receipt ID from URL
	receiptIDStr := chi.URLParam(r, "id")
	receiptID, err := uuid.Parse(receiptIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid receipt ID"}`, http.StatusBadRequest)
		return
	}

	// Call service
	err = h.receiptService.UpdateStatus(r.Context(), receiptID, userID, model.ReceiptStatusConfirmed)
	if err != nil {
		if errors.Is(err, service.ErrReceiptNotFound) {
			http.Error(w, `{"error": "receipt not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "failed to confirm receipt"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "receipt confirmed", "status": "confirmed"})
}

// Reject handles PATCH /receipts/:id/reject
func (h *ReceiptHandler) Reject(w http.ResponseWriter, r *http.Request) {
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

	// Get receipt ID from URL
	receiptIDStr := chi.URLParam(r, "id")
	receiptID, err := uuid.Parse(receiptIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid receipt ID"}`, http.StatusBadRequest)
		return
	}

	// Call service
	err = h.receiptService.UpdateStatus(r.Context(), receiptID, userID, model.ReceiptStatusRejected)
	if err != nil {
		if errors.Is(err, service.ErrReceiptNotFound) {
			http.Error(w, `{"error": "receipt not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "failed to reject receipt"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "receipt rejected", "status": "rejected"})
}

// isValidFeeType checks if the fee type is valid
func isValidFeeType(feeType model.FeeType) bool {
	switch feeType {
	case model.FeeTypeTax, model.FeeTypeTip, model.FeeTypeCharge, model.FeeTypeOther:
		return true
	default:
		return false
	}
}

// ExportCSV handles GET /receipts/export?from=&to=&format=csv
func (h *ReceiptHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
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

	// Build filter
	filter := &model.ListReceiptsFilter{
		UserID: userID,
	}

	// Parse date range
	if from := r.URL.Query().Get("from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			http.Error(w, `{"error": "invalid from date format"}`, http.StatusBadRequest)
			return
		}
		filter.FromDate = &t
	}
	if to := r.URL.Query().Get("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			http.Error(w, `{"error": "invalid to date format"}`, http.StatusBadRequest)
			return
		}
		filter.ToDate = &t
	}

	// Get all receipts (no pagination)
	receipts, err := h.receiptService.ListAll(r.Context(), filter)
	if err != nil {
		http.Error(w, `{"error": "failed to export receipts"}`, http.StatusInternalServerError)
		return
	}

	// Buffer CSV in memory first to handle errors properly
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	headers := []string{"date", "shop", "items", "total", "currency", "tags", "source", "status"}
	if err := writer.Write(headers); err != nil {
		http.Error(w, `{"error": "failed to write CSV header"}`, http.StatusInternalServerError)
		return
	}

	// Write data rows
	for _, receipt := range receipts {
		row := h.formatReceiptForCSV(receipt)
		if err := writer.Write(row); err != nil {
			http.Error(w, `{"error": "failed to write CSV data"}`, http.StatusInternalServerError)
			return
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		http.Error(w, `{"error": "failed to flush CSV data"}`, http.StatusInternalServerError)
		return
	}

	// Set headers for CSV download only after successful write
	w.Header().Set("Content-Type", "text/csv")
	filename := fmt.Sprintf("receipts_%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Write the buffered CSV to response
	w.Write(buf.Bytes())
}

// formatReceiptForCSV formats a receipt as a CSV row
func (h *ReceiptHandler) formatReceiptForCSV(receipt *model.Receipt) []string {
	// Date - use receipt_date if available, otherwise created_at
	date := receipt.CreatedAt.Format("2006-01-02")
	if receipt.ReceiptDate != nil {
		date = receipt.ReceiptDate.Format("2006-01-02")
	}

	// Shop - use title if available
	shop := ""
	if receipt.Title != nil {
		shop = *receipt.Title
	}

	// Items - flatten all item names with quantities
	items := ""
	if len(receipt.Items) > 0 {
		itemParts := make([]string, 0, len(receipt.Items))
		for _, item := range receipt.Items {
			itemParts = append(itemParts, fmt.Sprintf("%s (%.2f x %.2f)", item.Name, item.Quantity, item.UnitPrice))
		}
		items = strings.Join(itemParts, "; ")
	}

	// Total
	total := fmt.Sprintf("%.2f", receipt.Total)

	// Currency
	currency := receipt.Currency

	// Tags - comma-separated tag names
	tags := ""
	if len(receipt.Tags) > 0 {
		tagNames := make([]string, 0, len(receipt.Tags))
		for _, tag := range receipt.Tags {
			tagNames = append(tagNames, tag.Name)
		}
		tags = strings.Join(tagNames, ", ")
	}

	// Source
	source := string(receipt.Source)

	// Status
	status := string(receipt.Status)

	return []string{date, shop, items, total, currency, tags, source, status}
}
