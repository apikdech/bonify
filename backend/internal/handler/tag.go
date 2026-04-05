package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/middleware"
	"github.com/receipt-manager/backend/internal/service"
)

// TagHandler handles tag HTTP requests
type TagHandler struct {
	cfg        *config.Config
	tagService *service.TagService
}

// NewTagHandler creates a new tag handler
func NewTagHandler(cfg *config.Config, tagService *service.TagService) *TagHandler {
	return &TagHandler{
		cfg:        cfg,
		tagService: tagService,
	}
}

// List handles GET /tags
func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
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

	// Call service
	tags, err := h.tagService.List(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error": "failed to list tags"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tags": tags,
	})
}

// Create handles POST /tags
func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
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
	var req service.CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Call service
	tag, err := h.tagService.Create(r.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidTagName) {
			http.Error(w, `{"error": "tag name is required"}`, http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrInvalidTagColor) {
			http.Error(w, `{"error": "tag color must be a valid hex color (e.g., #6366f1)"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"error": "failed to create tag"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tag)
}

// Update handles PATCH /tags/:id
func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	// Get tag ID from URL
	tagIDStr := chi.URLParam(r, "id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid tag ID"}`, http.StatusBadRequest)
		return
	}

	// Parse request body
	var req service.UpdateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Call service
	tag, err := h.tagService.Update(r.Context(), tagID, userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrTagNotFound) {
			http.Error(w, `{"error": "tag not found"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrInvalidTagName) {
			http.Error(w, `{"error": "tag name is required"}`, http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrInvalidTagColor) {
			http.Error(w, `{"error": "tag color must be a valid hex color (e.g., #6366f1)"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, `{"error": "failed to update tag"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tag)
}

// Delete handles DELETE /tags/:id
func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	// Get tag ID from URL
	tagIDStr := chi.URLParam(r, "id")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		http.Error(w, `{"error": "invalid tag ID"}`, http.StatusBadRequest)
		return
	}

	// Call service
	err = h.tagService.Delete(r.Context(), tagID, userID)
	if err != nil {
		if errors.Is(err, service.ErrTagNotFound) {
			http.Error(w, `{"error": "tag not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "failed to delete tag"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "tag deleted successfully"})
}
