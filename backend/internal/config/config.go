package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	MinIO    MinIOConfig
	RustFS   RustFSConfig
	LLM      LLMConfig
	Temporal TemporalConfig
	FX       FXConfig
	Bots     BotsConfig
	App      AppConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port string
	Env  string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	DatabaseURL string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	RedisURL string
}

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
}

// MinIOConfig holds MinIO/S3 configuration
type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// RustFSConfig holds RustFS S3-compatible configuration
type RustFSConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	Provider        string
	Model           string
	APIKey          string
	AnthropicAPIKey string
	OpenAIAPIKey    string
	GeminiAPIKey    string
}

// TemporalConfig holds Temporal workflow configuration
type TemporalConfig struct {
	Host      string
	Namespace string
	TaskQueue string
}

// FXConfig holds foreign exchange rate configuration
type FXConfig struct {
	Provider     string
	BaseCurrency string
	RefreshCron  string
}

// BotsConfig holds bot integration configuration
type BotsConfig struct {
	TelegramBotToken      string
	TelegramWebhookSecret string
	DiscordBotToken       string
	DiscordPublicKey      string
}

// AppConfig holds general application configuration
type AppConfig struct {
	AppURL                 string
	OCRConfidenceThreshold float64
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (optional)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Env:  getEnv("SERVER_ENV", "development"),
		},
		Database: DatabaseConfig{
			DatabaseURL: getEnv("DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		JWT: JWTConfig{
			JWTSecret:     getEnv("JWT_SECRET", ""),
			JWTAccessTTL:  getDuration("JWT_ACCESS_TTL", "15m"),
			JWTRefreshTTL: getDuration("JWT_REFRESH_TTL", "168h"),
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", ""),
			SecretKey: getEnv("MINIO_SECRET_KEY", ""),
			Bucket:    getEnv("MINIO_BUCKET", "receipts"),
			UseSSL:    getBool("MINIO_USE_SSL", false),
		},
		RustFS: RustFSConfig{
			Endpoint:  getEnv("RUSTFS_ENDPOINT", "http://localhost:9000"),
			AccessKey: getEnv("RUSTFS_ACCESS_KEY", ""),
			SecretKey: getEnv("RUSTFS_SECRET_KEY", ""),
			Bucket:    getEnv("RUSTFS_BUCKET", "receipts"),
			Region:    getEnv("RUSTFS_REGION", "us-east-1"),
			UseSSL:    getBool("RUSTFS_USE_SSL", false),
		},
		LLM: LLMConfig{
			Provider:        getEnv("LLM_PROVIDER", "openai"),
			Model:           getEnv("LLM_MODEL", "gpt-4"),
			APIKey:          getEnv("LLM_API_KEY", ""),
			AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
			OpenAIAPIKey:    getEnv("OPENAI_API_KEY", ""),
			GeminiAPIKey:    getEnv("GEMINI_API_KEY", ""),
		},
		Temporal: TemporalConfig{
			Host:      getEnv("TEMPORAL_HOST", "localhost:7233"),
			Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
			TaskQueue: getEnv("TEMPORAL_TASK_QUEUE", "receipt-tasks"),
		},
		FX: FXConfig{
			Provider:     getEnv("FX_PROVIDER", "exchangerate-api"),
			BaseCurrency: getEnv("FX_BASE_CURRENCY", "USD"),
			RefreshCron:  getEnv("FX_REFRESH_CRON", "0 0 * * *"),
		},
		Bots: BotsConfig{
			TelegramBotToken:      getEnv("TELEGRAM_BOT_TOKEN", ""),
			TelegramWebhookSecret: getEnv("TELEGRAM_WEBHOOK_SECRET", ""),
			DiscordBotToken:       getEnv("DISCORD_BOT_TOKEN", ""),
			DiscordPublicKey:      getEnv("DISCORD_PUBLIC_KEY", ""),
		},
		App: AppConfig{
			AppURL:                 getEnv("APP_URL", "http://localhost:8080"),
			OCRConfidenceThreshold: getFloat64("OCR_CONFIDENCE_THRESHOLD", "0.85"),
		},
	}

	cfg.RustFS.Endpoint = normalizeRustFSEndpoint(cfg.RustFS.Endpoint, cfg.RustFS.UseSSL)

	// Validate required fields
	if cfg.JWT.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required but not set")
	}

	return cfg, nil
}

// normalizeRustFSEndpoint ensures a scheme so AWS SDK custom endpoints parse as URIs (e.g. host:port is invalid).
func normalizeRustFSEndpoint(endpoint string, useSSL bool) string {
	if endpoint == "" {
		return "http://localhost:9000"
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	return scheme + "://" + endpoint
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDuration retrieves a duration environment variable or returns a default value
func getDuration(key, defaultValue string) time.Duration {
	value := getEnv(key, defaultValue)
	duration, err := time.ParseDuration(value)
	if err != nil {
		return mustParseDuration(defaultValue)
	}
	return duration
}

// getBool retrieves a boolean environment variable or returns a default value
func getBool(key string, defaultValue bool) bool {
	value := getEnv(key, "")
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// getFloat64 retrieves a float64 environment variable or returns a default value
func getFloat64(key, defaultValue string) float64 {
	value := getEnv(key, defaultValue)
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		parsed, _ = strconv.ParseFloat(defaultValue, 64)
		return parsed
	}
	return parsed
}

// mustParseDuration parses a duration string or panics
func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(fmt.Sprintf("invalid default duration: %s", s))
	}
	return d
}
