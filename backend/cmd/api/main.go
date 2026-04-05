package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/client"

	"github.com/receipt-manager/backend/internal/bot"
	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/db"
	"github.com/receipt-manager/backend/internal/handler"
	appmiddleware "github.com/receipt-manager/backend/internal/middleware"
	"github.com/receipt-manager/backend/internal/repository"
	"github.com/receipt-manager/backend/internal/service"
)

func main() {
	// Initialize structured logging with JSON handler
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Receipt Manager API server")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	logger.Info("Configuration loaded", "env", cfg.Server.Env, "port", cfg.Server.Port)

	// Connect to PostgreSQL
	database, err := db.New(context.Background(), cfg)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Run database migrations
	if err := database.RunMigrations(context.Background()); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Connect to Redis
	redisOpts, err := redis.ParseURL(cfg.Redis.RedisURL)
	if err != nil {
		logger.Error("Failed to parse Redis URL", "error", err)
		os.Exit(1)
	}
	redisClient := redis.NewClient(redisOpts)

	// Test Redis connection
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	logger.Info("Redis connection established")

	// Initialize all repositories
	userRepo := repository.NewUserRepo(database.Pool)
	receiptRepo := repository.NewReceiptRepo(database.Pool)
	tagRepo := repository.NewTagRepo(database.Pool)
	logger.Info("Repositories initialized")

	// Initialize all services
	authService := service.NewAuthService(cfg, userRepo, redisClient)
	receiptService := service.NewReceiptService(receiptRepo, tagRepo)
	tagService := service.NewTagService(tagRepo)
	storageService, err := service.NewStorageService(cfg)
	if err != nil {
		logger.Error("Failed to initialize storage service", "error", err)
		os.Exit(1)
	}
	logger.Info("Services initialized")

	// Create Temporal client
	var temporalClient client.Client
	if cfg.Temporal.Host != "" {
		temporalClient, err = client.Dial(client.Options{
			HostPort:  cfg.Temporal.Host,
			Namespace: cfg.Temporal.Namespace,
		})
		if err != nil {
			logger.Error("Failed to create Temporal client", "error", err)
			os.Exit(1)
		}
		defer temporalClient.Close()
		logger.Info("Temporal client connected", "host", cfg.Temporal.Host, "namespace", cfg.Temporal.Namespace)
	} else {
		logger.Warn("Temporal host not configured, workflow functionality disabled")
	}

	// Initialize Telegram bot (if configured)
	var telegramBot *bot.TelegramBot
	if cfg.Bots.TelegramBotToken != "" && temporalClient != nil {
		telegramBot = bot.NewTelegramBot(cfg, storageService, temporalClient, userRepo, receiptRepo)
		logger.Info("Telegram bot initialized")
	} else {
		if cfg.Bots.TelegramBotToken == "" {
			logger.Info("Telegram bot token not configured, bot disabled")
		}
		if temporalClient == nil {
			logger.Info("Temporal client not available, Telegram bot disabled")
		}
	}

	// Initialize Discord bot (if configured)
	var discordBot *bot.DiscordBot
	if cfg.Bots.DiscordBotToken != "" && cfg.Bots.DiscordPublicKey != "" && temporalClient != nil {
		discordBot, err = bot.NewDiscordBot(cfg, storageService, temporalClient, userRepo, receiptRepo)
		if err != nil {
			logger.Error("Failed to initialize Discord bot", "error", err)
		} else {
			logger.Info("Discord bot initialized")
		}
	} else {
		if cfg.Bots.DiscordBotToken == "" || cfg.Bots.DiscordPublicKey == "" {
			logger.Info("Discord bot token or public key not configured, bot disabled")
		}
		if temporalClient == nil {
			logger.Info("Temporal client not available, Discord bot disabled")
		}
	}

	// Initialize all handlers
	authHandler := handler.NewAuthHandler(cfg, authService)
	receiptHandler := handler.NewReceiptHandler(cfg, receiptService)
	tagHandler := handler.NewTagHandler(cfg, tagService)
	logger.Info("Handlers initialized")

	// Setup Chi router with middleware
	r := chi.NewRouter()

	// Basic middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS middleware (allow all origins for now, configure for production)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check endpoint (protected, with rate limiting)
	r.Group(func(r chi.Router) {
		r.Use(appmiddleware.JWTAuth(cfg))
		r.Use(appmiddleware.RateLimit(redisClient, 60, time.Minute))
		r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		})
	})

	// Public auth routes (no auth required)
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)

		// Protected auth routes (require JWT)
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.JWTAuth(cfg))
			r.Use(appmiddleware.RateLimit(redisClient, 60, time.Minute))
			r.Get("/me", authHandler.Me)
		})
	})

	// Receipt routes (protected + rate limited)
	r.Route("/api/v1/receipts", func(r chi.Router) {
		r.Use(appmiddleware.JWTAuth(cfg))
		r.Use(appmiddleware.RateLimit(redisClient, 60, time.Minute))

		r.Get("/", receiptHandler.List)
		r.Post("/", receiptHandler.Create)
		r.Get("/{id}", receiptHandler.Get)
		r.Patch("/{id}", receiptHandler.Update)
		r.Delete("/{id}", receiptHandler.Delete)
		r.Patch("/{id}/confirm", receiptHandler.Confirm)
		r.Patch("/{id}/reject", receiptHandler.Reject)
	})

	// Tag routes (protected + rate limited)
	r.Route("/api/v1/tags", func(r chi.Router) {
		r.Use(appmiddleware.JWTAuth(cfg))
		r.Use(appmiddleware.RateLimit(redisClient, 60, time.Minute))

		r.Get("/", tagHandler.List)
		r.Post("/", tagHandler.Create)
		r.Patch("/{id}", tagHandler.Update)
		r.Delete("/{id}", tagHandler.Delete)
	})

	// Telegram webhook route (public - uses secret token for auth)
	if telegramBot != nil {
		r.Post("/webhooks/telegram", telegramBot.HandleWebhook)
		logger.Info("Telegram webhook route registered", "path", "/webhooks/telegram")
	}

	// Discord webhook route (public - uses Ed25519 signature verification)
	if discordBot != nil {
		r.Post("/webhooks/discord", discordBot.HandleWebhook)
		logger.Info("Discord webhook route registered", "path", "/webhooks/discord")
	}

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	if addr == ":" {
		addr = ":8080"
	}

	logger.Info("Server starting", "address", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
