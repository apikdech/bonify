package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/model"
	"github.com/receipt-manager/backend/internal/repository"
	"github.com/receipt-manager/backend/internal/service"
	"github.com/receipt-manager/backend/internal/workflow"
	"go.temporal.io/sdk/client"
)

// TelegramBot handles Telegram webhook requests and commands
type TelegramBot struct {
	cfg            *config.Config
	storageService *service.StorageService
	temporalClient client.Client
	userRepo       *repository.UserRepo
	receiptRepo    *repository.ReceiptRepo
	httpClient     *http.Client
}

// NewTelegramBot creates a new Telegram bot handler
func NewTelegramBot(
	cfg *config.Config,
	storageService *service.StorageService,
	temporalClient client.Client,
	userRepo *repository.UserRepo,
	receiptRepo *repository.ReceiptRepo,
) *TelegramBot {
	return &TelegramBot{
		cfg:            cfg,
		storageService: storageService,
		temporalClient: temporalClient,
		userRepo:       userRepo,
		receiptRepo:    receiptRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HandleWebhook handles incoming Telegram webhook requests
func (b *TelegramBot) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify secret token
	secretToken := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
	if secretToken != b.cfg.Bots.TelegramWebhookSecret {
		slog.Warn("Invalid or missing webhook secret token",
			"ip", r.RemoteAddr,
			"path", r.URL.Path,
		)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Parse the update
	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		slog.Error("Failed to decode Telegram update", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Process the update
	b.processUpdate(r.Context(), &update)

	// Always return 200 OK to Telegram to avoid retries
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}

// processUpdate routes the update to the appropriate handler
func (b *TelegramBot) processUpdate(ctx context.Context, update *Update) {
	if update.Message == nil {
		return
	}

	message := update.Message
	chatID := message.Chat.ID

	// Handle commands
	if message.Text != "" && message.Text[0] == '/' {
		b.handleCommand(ctx, chatID, message)
		return
	}

	// Handle photos
	if len(message.Photo) > 0 {
		b.handlePhoto(ctx, chatID, message)
		return
	}

	// Default: send help message
	b.sendMessage(chatID, "👋 Hello! Send me a photo of a receipt, or use /start for help.")
}

// handleCommand processes bot commands
func (b *TelegramBot) handleCommand(ctx context.Context, chatID int64, message *Message) {
	command := message.Text

	switch command {
	case "/start":
		b.handleStart(chatID)
	case "/summary":
		b.handleSummary(ctx, chatID, message.From.ID)
	case "/pending":
		b.handlePending(ctx, chatID, message.From.ID)
	default:
		b.sendMessage(chatID, "❓ Unknown command. Try /start for available commands.")
	}
}

// handleStart sends welcome message with instructions
func (b *TelegramBot) handleStart(chatID int64) {
	welcomeMsg := `👋 *Welcome to Receipt Manager Bot!*

Here's what I can do:

📸 *Send me a photo* of any receipt and I'll:
   • Extract merchant name, date, items, and total
   • Save it to your account
   • Notify you when processing is complete

📊 */summary* — View this month's total spending
📝 */pending* — Check receipts awaiting your review

*Getting started:*
1. Log in to your dashboard
2. Link your Telegram in Settings → Integrations
3. Start sending receipts!

Need help? Contact support anytime.`

	b.sendMessage(chatID, welcomeMsg)
}

// handleSummary returns this month's total spending
func (b *TelegramBot) handleSummary(ctx context.Context, chatID int64, telegramUserID int64) {
	// Get user by Telegram ID
	user, err := b.getUserByTelegramID(ctx, strconv.FormatInt(telegramUserID, 10))
	if err != nil {
		slog.Error("Failed to get user by Telegram ID", "error", err)
		b.sendMessage(chatID, "❌ Sorry, something went wrong. Please try again later.")
		return
	}

	if user == nil {
		b.sendMessage(chatID, "🔗 Please link your Telegram account in the Receipt Manager dashboard first.")
		return
	}

	// Get this month's total
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	total, err := b.receiptRepo.GetMonthlyTotal(ctx, user.ID, startOfMonth, endOfMonth)
	if err != nil {
		slog.Error("Failed to get monthly total", "error", err, "user_id", user.ID)
		b.sendMessage(chatID, "❌ Sorry, something went wrong. Please try again later.")
		return
	}

	// Format the response
	currency := user.HomeCurrency
	if currency == "" {
		currency = "USD"
	}

	response := fmt.Sprintf(
		"📊 *%s %d Spending Summary*\n\n💰 Total: *%.2f %s*",
		now.Month().String(),
		now.Year(),
		total,
		currency,
	)

	b.sendMessage(chatID, response)
}

// handlePending returns the count of unreviewed receipts
func (b *TelegramBot) handlePending(ctx context.Context, chatID int64, telegramUserID int64) {
	// Get user by Telegram ID
	user, err := b.getUserByTelegramID(ctx, strconv.FormatInt(telegramUserID, 10))
	if err != nil {
		slog.Error("Failed to get user by Telegram ID", "error", err)
		b.sendMessage(chatID, "❌ Sorry, something went wrong. Please try again later.")
		return
	}

	if user == nil {
		b.sendMessage(chatID, "🔗 Please link your Telegram account in the Receipt Manager dashboard first.")
		return
	}

	// Get pending review count
	count, err := b.receiptRepo.CountByStatus(ctx, user.ID, model.ReceiptStatusPendingReview)
	if err != nil {
		slog.Error("Failed to get pending count", "error", err, "user_id", user.ID)
		b.sendMessage(chatID, "❌ Sorry, something went wrong. Please try again later.")
		return
	}

	// Format the response
	var response string
	if count == 0 {
		response = "✅ *All caught up!*\n\nNo receipts waiting for review."
	} else if count == 1 {
		response = "📝 *1 receipt* is waiting for your review.\n\nCheck it out in your dashboard!"
	} else {
		response = fmt.Sprintf("📝 *%d receipts* are waiting for your review.\n\nCheck them out in your dashboard!", count)
	}

	b.sendMessage(chatID, response)
}

// handlePhoto processes receipt photos
func (b *TelegramBot) handlePhoto(ctx context.Context, chatID int64, message *Message) {
	telegramUserID := strconv.FormatInt(message.From.ID, 10)

	// Check if user is linked
	user, err := b.getUserByTelegramID(ctx, telegramUserID)
	if err != nil {
		slog.Error("Failed to get user by Telegram ID", "error", err)
		b.sendMessage(chatID, "❌ Sorry, something went wrong. Please try again later.")
		return
	}

	if user == nil {
		b.sendMessage(chatID, "🔗 Please link your Telegram account in the Receipt Manager dashboard before sending receipts.")
		return
	}

	// Get largest photo (last in array)
	largestPhoto := message.Photo[len(message.Photo)-1]
	fileID := largestPhoto.FileID

	slog.Info("Processing photo receipt",
		"chat_id", chatID,
		"user_id", user.ID,
		"file_id", fileID,
	)

	// Get file URL from Telegram
	fileURL, err := b.getFileURL(fileID)
	if err != nil {
		slog.Error("Failed to get file URL", "error", err, "file_id", fileID)
		b.sendMessage(chatID, "❌ Sorry, failed to get the photo. Please try again.")
		return
	}

	// Download the photo
	resp, err := b.httpClient.Get(fileURL)
	if err != nil {
		slog.Error("Failed to download photo", "error", err, "url", fileURL)
		b.sendMessage(chatID, "❌ Sorry, failed to download the photo. Please try again.")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Failed to download photo", "status", resp.StatusCode)
		b.sendMessage(chatID, "❌ Sorry, failed to download the photo. Please try again.")
		return
	}

	// Read photo data
	photoData, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read photo data", "error", err)
		b.sendMessage(chatID, "❌ Sorry, something went wrong. Please try again.")
		return
	}

	// Upload to RustFS
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	filename := fmt.Sprintf("telegram_%d_%d.jpg", message.From.ID, time.Now().Unix())
	objectKey, err := b.storageService.Upload(ctx, bytes.NewReader(photoData), int64(len(photoData)), contentType, filename)
	if err != nil {
		slog.Error("Failed to upload photo to storage", "error", err)
		b.sendMessage(chatID, "❌ Sorry, failed to save the photo. Please try again.")
		return
	}

	slog.Info("Photo uploaded to storage",
		"object_key", objectKey,
		"user_id", user.ID,
	)

	// Generate presigned URL for workflow
	presignedURL, err := b.storageService.GetPresignedURL(ctx, objectKey, 24*time.Hour)
	if err != nil {
		slog.Error("Failed to generate presigned URL", "error", err)
		b.sendMessage(chatID, "❌ Sorry, something went wrong. Please try again.")
		return
	}

	// Start Temporal workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("parse-receipt-telegram-%d-%d", message.From.ID, time.Now().Unix()),
		TaskQueue: b.cfg.Temporal.TaskQueue,
	}

	input := workflow.ParseReceiptInput{
		UserID:   user.ID.String(),
		ImageURL: presignedURL,
		Source:   "telegram",
	}

	we, err := b.temporalClient.ExecuteWorkflow(ctx, workflowOptions, workflow.ParseReceiptWorkflow, input)
	if err != nil {
		slog.Error("Failed to start workflow", "error", err)
		b.sendMessage(chatID, "❌ Sorry, failed to start processing. Please try again.")
		return
	}

	slog.Info("Started ParseReceiptWorkflow",
		"workflow_id", we.GetID(),
		"run_id", we.GetRunID(),
		"user_id", user.ID,
	)

	// Reply to user
	b.sendMessage(chatID, "📸 Receipt received! Processing... You'll get a notification when it's ready.")
}

// sendMessage sends a text message to a chat via Telegram API
func (b *TelegramBot) sendMessage(chatID int64, text string) {
	if b.cfg.Bots.TelegramBotToken == "" {
		slog.Warn("Telegram bot token not configured, skipping message send")
		return
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.cfg.Bots.TelegramBotToken)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal message payload", "error", err)
		return
	}

	resp, err := b.httpClient.Post(apiURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		slog.Error("Failed to send Telegram message", "error", err, "chat_id", chatID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("Telegram API error", "status", resp.StatusCode, "body", string(body))
		return
	}

	slog.Debug("Message sent to Telegram", "chat_id", chatID)
}

// getFileURL gets the download URL for a file from Telegram
func (b *TelegramBot) getFileURL(fileID string) (string, error) {
	if b.cfg.Bots.TelegramBotToken == "" {
		return "", fmt.Errorf("telegram bot token not configured")
	}

	// Get file info from Telegram
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", b.cfg.Bots.TelegramBotToken, fileID)

	resp, err := b.httpClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	var result struct {
		Ok     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode file info: %w", err)
	}

	if !result.Ok || result.Result.FilePath == "" {
		return "", fmt.Errorf("failed to get file path from Telegram")
	}

	// Construct download URL
	downloadURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.cfg.Bots.TelegramBotToken, result.Result.FilePath)

	return downloadURL, nil
}

// getUserByTelegramID checks if a Telegram ID is linked to a user
func (b *TelegramBot) getUserByTelegramID(ctx context.Context, telegramID string) (*model.User, error) {
	return b.userRepo.GetByTelegramID(ctx, telegramID)
}
