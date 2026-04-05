package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/repository"
)

// LLMConfig represents resolved LLM configuration
type LLMConfig struct {
	Provider string
	Model    string
	APIKey   string
}

// SettingsService provides operations for managing system settings
type SettingsService struct {
	cfg          *config.Config
	settingsRepo *repository.SettingsRepo
	userRepo     *repository.UserRepo
}

// allowedSettings defines which keys can be updated via the API
var allowedSettings = []string{
	"llm_provider",
	"llm_model",
	"ocr_threshold",
	"fx_base_currency",
	"fx_provider",
}

// NewSettingsService creates a new settings service
func NewSettingsService(
	cfg *config.Config,
	settingsRepo *repository.SettingsRepo,
	userRepo *repository.UserRepo,
) *SettingsService {
	return &SettingsService{
		cfg:          cfg,
		settingsRepo: settingsRepo,
		userRepo:     userRepo,
	}
}

// ResolveLLMConfig resolves LLM configuration using the following order of precedence:
// 1. User-level override (users.llm_provider, users.llm_model)
// 2. System setting (system_settings table)
// 3. Environment variable fallback (cfg.LLM.Provider, cfg.LLM.Model)
func (s *SettingsService) ResolveLLMConfig(ctx context.Context, userID uuid.UUID) (*LLMConfig, error) {
	// 1. Try user-level override
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user != nil && user.LLMProvider != nil && user.LLMModel != nil {
		return &LLMConfig{
			Provider: *user.LLMProvider,
			Model:    *user.LLMModel,
			APIKey:   s.getAPIKeyForProvider(*user.LLMProvider),
		}, nil
	}

	// 2. Try system settings
	provider, err := s.settingsRepo.Get(ctx, "llm_provider")
	if err != nil {
		return nil, fmt.Errorf("failed to get llm_provider setting: %w", err)
	}

	model, err := s.settingsRepo.Get(ctx, "llm_model")
	if err != nil {
		return nil, fmt.Errorf("failed to get llm_model setting: %w", err)
	}

	if provider != "" && model != "" {
		return &LLMConfig{
			Provider: provider,
			Model:    model,
			APIKey:   s.getAPIKeyForProvider(provider),
		}, nil
	}

	// 3. Fall back to environment variables
	return &LLMConfig{
		Provider: s.cfg.LLM.Provider,
		Model:    s.cfg.LLM.Model,
		APIKey:   s.getAPIKeyForProvider(s.cfg.LLM.Provider),
	}, nil
}

// getAPIKeyForProvider returns the appropriate API key for a given provider
func (s *SettingsService) getAPIKeyForProvider(provider string) string {
	switch provider {
	case "anthropic":
		if s.cfg.LLM.AnthropicAPIKey != "" {
			return s.cfg.LLM.AnthropicAPIKey
		}
		return s.cfg.LLM.APIKey
	case "openai":
		if s.cfg.LLM.OpenAIAPIKey != "" {
			return s.cfg.LLM.OpenAIAPIKey
		}
		return s.cfg.LLM.APIKey
	case "gemini":
		if s.cfg.LLM.GeminiAPIKey != "" {
			return s.cfg.LLM.GeminiAPIKey
		}
		return s.cfg.LLM.APIKey
	case "ollama":
		return "" // Ollama is local, no API key needed
	default:
		return s.cfg.LLM.APIKey
	}
}

// GetAllSettings retrieves all system settings
func (s *SettingsService) GetAllSettings(ctx context.Context) (map[string]string, error) {
	return s.settingsRepo.GetAll(ctx)
}

// UpdateSetting updates a system setting (validates key is allowed)
func (s *SettingsService) UpdateSetting(ctx context.Context, key, value string, updatedBy string) error {
	// Validate key is in allowed list
	if !slices.Contains(allowedSettings, key) {
		return fmt.Errorf("setting key '%s' is not allowed to be updated", key)
	}

	return s.settingsRepo.Update(ctx, key, value, updatedBy)
}
