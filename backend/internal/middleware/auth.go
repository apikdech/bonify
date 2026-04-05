package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/config"
)

// contextKey is a custom type for context keys (unexported struct to prevent collisions)
type contextKey struct{}

var userIDKey = &contextKey{}

// JWTAuth creates a middleware that validates JWT tokens
func JWTAuth(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Check for Bearer prefix
			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				http.Error(w, `{"error": "invalid authorization header format"}`, http.StatusUnauthorized)
				return
			}

			// Extract token
			tokenString := strings.TrimPrefix(authHeader, bearerPrefix)

			// Parse and validate JWT
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Validate signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.JWT.JWTSecret), nil
			}, jwt.WithExpirationRequired())

			if err != nil {
				http.Error(w, `{"error": "invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error": "invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			// Verify token type
			tokenType, _ := claims["type"].(string)
			if tokenType != "access" {
				http.Error(w, `{"error": "invalid token type"}`, http.StatusUnauthorized)
				return
			}

			// Extract user ID from subject claim
			userIDStr, ok := claims["sub"].(string)
			if !ok {
				http.Error(w, `{"error": "invalid user ID in token"}`, http.StatusUnauthorized)
				return
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				http.Error(w, `{"error": "invalid user ID format"}`, http.StatusUnauthorized)
				return
			}

			// Store userID in context
			ctx := context.WithValue(r.Context(), userIDKey, userID)

			// Continue with the next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the user ID from the context
// Returns uuid.Nil if no user ID is found
func GetUserID(ctx context.Context) uuid.UUID {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return userID
}

// RequireAuth is a convenience function that combines JWTAuth and returns 401 if no user
func RequireAuth(cfg *config.Config) func(http.Handler) http.Handler {
	return JWTAuth(cfg)
}
