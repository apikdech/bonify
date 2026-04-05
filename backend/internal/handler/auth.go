package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/middleware"
	"github.com/receipt-manager/backend/internal/model"
	"github.com/receipt-manager/backend/internal/service"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	cfg         *config.Config
	authService *service.AuthService
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(cfg *config.Config, authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		cfg:         cfg,
		authService: authService,
	}
}

// LoginRequest represents a login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error": "email and password are required"}`, http.StatusBadRequest)
		return
	}

	tokenPair, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, `{"error": "invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenPair)
}

// RefreshRequest represents a refresh token request body
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh handles POST /auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		http.Error(w, `{"error": "refresh_token is required"}`, http.StatusBadRequest)
		return
	}

	tokenPair, err := h.authService.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, `{"error": "invalid or expired refresh token"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenPair)
}

// LogoutRequest represents a logout request body
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout handles POST /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		http.Error(w, `{"error": "refresh_token is required"}`, http.StatusBadRequest)
		return
	}

	if err := h.authService.Logout(r.Context(), req.RefreshToken); err != nil {
		// Even if logout fails (e.g., token already expired), we return success
		// to prevent information leakage
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "logged out successfully"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "logged out successfully"})
}

// MeResponse represents the current user response
type MeResponse struct {
	User *model.User `json:"user"`
}

// Me handles GET /auth/me (protected)
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context (set by JWT middleware)
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// TODO: Fetch full user details from the user service in the future
	// For now, return a minimal response with the user ID only
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"user_id": userID.String(),
		"message": "authenticated",
	})
}
