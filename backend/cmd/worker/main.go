package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/db"
	"github.com/receipt-manager/backend/internal/repository"
	"github.com/receipt-manager/backend/internal/service"
	"github.com/receipt-manager/backend/internal/workflow"
)

func main() {
	// Initialize structured logging with JSON handler
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Receipt Manager Temporal Worker")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	logger.Info("Configuration loaded", "env", cfg.Server.Env)

	// Connect to PostgreSQL
	database, err := db.New(context.Background(), cfg)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

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
	settingsRepo := repository.NewSettingsRepo(database.Pool)
	fxRepo := repository.NewFXRepo(database.Pool)
	logger.Info("Repositories initialized")

	// Initialize all services
	settingsService := service.NewSettingsService(cfg, settingsRepo, userRepo)
	storageService, err := service.NewStorageService(cfg)
	if err != nil {
		logger.Error("Failed to create storage service", "error", err)
		os.Exit(1)
	}
	llmService := service.NewLLMService(settingsService, storageService)
	receiptService := service.NewReceiptService(receiptRepo, tagRepo)
	fxService := service.NewFXService(fxRepo)
	logger.Info("Services initialized")

	// Create Temporal client
	temporalClient, err := client.Dial(client.Options{
		HostPort:  cfg.Temporal.Host,
		Namespace: cfg.Temporal.Namespace,
	})
	if err != nil {
		logger.Error("Failed to create Temporal client", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()
	logger.Info("Temporal client connected", "host", cfg.Temporal.Host, "namespace", cfg.Temporal.Namespace)

	// Create activities with all dependencies
	activities := workflow.NewActivities(
		settingsService,
		llmService,
		receiptService,
		fxService,
		nil, // Notifier - placeholder for now
	)

	// Create worker
	w := worker.New(temporalClient, cfg.Temporal.TaskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(workflow.ParseReceiptWorkflow)
	w.RegisterWorkflow(workflow.FXSyncWorkflow)

	// Register activities
	w.RegisterActivity(activities.ResolveLLMConfigActivity)
	w.RegisterActivity(activities.CallLLMVisionActivity)
	w.RegisterActivity(activities.SaveReceiptActivity)
	w.RegisterActivity(activities.NotifyUserActivity)
	w.RegisterActivity(activities.FetchFXRatesActivity)

	// Schedule FX sync workflow to run daily at 2 AM
	scheduleOptions := client.ScheduleOptions{
		ID: "fx-sync-daily",
		Spec: client.ScheduleSpec{
			CronExpressions: []string{"0 2 * * *"}, // 2 AM daily
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "fx-sync-" + time.Now().Format("20060102-150405"),
			Workflow:  workflow.FXSyncWorkflow,
			TaskQueue: cfg.Temporal.TaskQueue,
		},
	}

	scheduleClient := temporalClient.ScheduleClient()
	_, err = scheduleClient.Create(context.Background(), scheduleOptions)
	if err != nil {
		// Schedule might already exist, log but don't fail
		logger.Info("FX sync schedule may already exist or failed to create", "error", err)
	} else {
		logger.Info("FX sync scheduled workflow created", "schedule", "0 2 * * *")
	}

	logger.Info("Worker starting", "taskQueue", cfg.Temporal.TaskQueue)

	// Start worker
	if err := w.Run(worker.InterruptCh()); err != nil {
		logger.Error("Worker failed", "error", err)
		os.Exit(1)
	}
}
