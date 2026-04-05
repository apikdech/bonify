package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/model"
	"github.com/receipt-manager/backend/internal/repository"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// AuthService provides authentication functionality
type AuthService struct {
	cfg      *config.Config
	userRepo *repository.UserRepo
	redis    *redis.Client
}

// NewAuthService creates a new authentication service
func NewAuthService(cfg *config.Config, userRepo *repository.UserRepo, redisClient *redis.Client) *AuthService {
	return &AuthService{
		cfg:      cfg,
		userRepo: userRepo,
		redis:    redisClient,
	}
}

// Login authenticates a user and returns a token pair
func (s *AuthService) Login(ctx context.Context, email, password string) (*model.TokenPair, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("authentication failed")
	}
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate tokens
	tokenPair, err := s.generateTokens(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokenPair, nil
}

// Refresh validates a refresh token and issues a new token pair
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	// Parse and validate the refresh token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWT.JWTSecret), nil
	}, jwt.WithExpirationRequired())

	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Verify token type
	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}

	// Check if token is blacklisted using SHA-256 hash of token
	tokenHash := sha256.Sum256([]byte(refreshToken))
	blacklistKey := fmt.Sprintf("blacklist:refresh:%s", hex.EncodeToString(tokenHash[:]))
	exists, err := s.redis.Exists(ctx, blacklistKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	if exists > 0 {
		return nil, fmt.Errorf("token has been revoked")
	}

	// Extract user ID
	userIDStr, _ := claims["sub"].(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	// Verify user still exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify user")
	}
	if user == nil {
		return nil, fmt.Errorf("user no longer exists")
	}

	// Generate new token pair
	tokenPair, err := s.generateTokens(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokenPair, nil
}

// Logout blacklists a refresh token
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	// Parse the token to get expiration
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWT.JWTSecret), nil
	}, jwt.WithExpirationRequired())

	// Use SHA-256 hash of token for Redis blacklist key
	tokenHash := sha256.Sum256([]byte(refreshToken))
	blacklistKey := fmt.Sprintf("blacklist:refresh:%s", hex.EncodeToString(tokenHash[:]))

	if err != nil {
		// If we can't parse it, just blacklist it anyway with the default TTL
		return s.redis.Set(ctx, blacklistKey, "1", s.cfg.JWT.JWTRefreshTTL).Err()
	}

	if !token.Valid {
		return s.redis.Set(ctx, blacklistKey, "1", s.cfg.JWT.JWTRefreshTTL).Err()
	}

	// Extract expiration time
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return s.redis.Set(ctx, blacklistKey, "1", s.cfg.JWT.JWTRefreshTTL).Err()
	}

	// Validate that exp claim exists before calculating TTL
	exp, ok := claims["exp"].(float64)
	if !ok {
		// Token has no expiration, use default TTL
		return s.redis.Set(ctx, blacklistKey, "1", s.cfg.JWT.JWTRefreshTTL).Err()
	}

	expTime := time.Unix(int64(exp), 0)
	ttl := time.Until(expTime)
	if ttl <= 0 {
		ttl = time.Second // Token already expired, minimal TTL
	}

	return s.redis.Set(ctx, blacklistKey, "1", ttl).Err()
}

// generateTokens creates access and refresh JWT tokens
func (s *AuthService) generateTokens(ctx context.Context, userID uuid.UUID) (*model.TokenPair, error) {
	now := time.Now()

	// Generate access token
	accessClaims := jwt.MapClaims{
		"sub":  userID.String(),
		"iat":  now.Unix(),
		"exp":  now.Add(s.cfg.JWT.JWTAccessTTL).Unix(),
		"type": "access",
		"iss":  "receipt-manager",
		"aud":  "receipt-manager-api",
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.cfg.JWT.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshClaims := jwt.MapClaims{
		"sub":  userID.String(),
		"iat":  now.Unix(),
		"exp":  now.Add(s.cfg.JWT.JWTRefreshTTL).Unix(),
		"type": "refresh",
		"iss":  "receipt-manager",
		"aud":  "receipt-manager-api",
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.cfg.JWT.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}

// HashPassword creates a bcrypt hash of the password
func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// CreateInitialAdmin creates the first admin user if no users exist
func (s *AuthService) CreateInitialAdmin(ctx context.Context, name, email, password string) (*model.User, error) {
	// Check if any users exist
	users, err := s.userRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing users: %w", err)
	}
	if len(users) > 0 {
		return nil, fmt.Errorf("users already exist, cannot create initial admin")
	}

	// Hash password
	passwordHash, err := s.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Create admin user
	user := &model.User{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         "admin",
		HomeCurrency: "IDR",
	}

	createdUser, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial admin: %w", err)
	}

	return createdUser, nil
}
