package bot

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/receipt-manager/backend/internal/config"
	"github.com/receipt-manager/backend/internal/model"
	"github.com/receipt-manager/backend/internal/repository"
	"github.com/receipt-manager/backend/internal/service"
	"github.com/receipt-manager/backend/internal/workflow"
	"go.temporal.io/sdk/client"
)

// DiscordBot handles Discord webhook requests and slash commands
type DiscordBot struct {
	cfg            *config.Config
	storageService *service.StorageService
	temporalClient client.Client
	userRepo       *repository.UserRepo
	receiptRepo    *repository.ReceiptRepo
	publicKey      ed25519.PublicKey
	httpClient     *http.Client
}

// NewDiscordBot creates a new Discord bot handler
func NewDiscordBot(
	cfg *config.Config,
	storageService *service.StorageService,
	temporalClient client.Client,
	userRepo *repository.UserRepo,
	receiptRepo *repository.ReceiptRepo,
) (*DiscordBot, error) {
	// Parse the public key from hex
	publicKeyBytes, err := hex.DecodeString(cfg.Bots.DiscordPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Discord public key: %w", err)
	}

	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Discord public key length: expected %d, got %d", ed25519.PublicKeySize, len(publicKeyBytes))
	}

	return &DiscordBot{
		cfg:            cfg,
		storageService: storageService,
		temporalClient: temporalClient,
		userRepo:       userRepo,
		receiptRepo:    receiptRepo,
		publicKey:      ed25519.PublicKey(publicKeyBytes),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// HandleWebhook handles incoming Discord webhook requests
func (b *DiscordBot) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify Ed25519 signature
	if !b.verifySignature(r) {
		slog.Warn("Invalid Discord signature",
			"ip", r.RemoteAddr,
			"path", r.URL.Path,
		)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Parse the interaction
	var interaction Interaction
	if err := json.NewDecoder(r.Body).Decode(&interaction); err != nil {
		slog.Error("Failed to decode Discord interaction", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Handle ping (type 1) for URL verification
	if interaction.Type == InteractionTypePing {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type": 1,
		})
		return
	}

	// Process the interaction
	response := b.processInteraction(r.Context(), &interaction)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode Discord response", "error", err)
	}
}

// verifySignature verifies the Ed25519 signature from Discord
func (b *DiscordBot) verifySignature(r *http.Request) bool {
	signature := r.Header.Get("X-Signature-Ed25519")
	timestamp := r.Header.Get("X-Signature-Timestamp")

	if signature == "" || timestamp == "" {
		return false
	}

	// Decode signature from hex
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		slog.Warn("Failed to decode signature", "error", err)
		return false
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Warn("Failed to read body", "error", err)
		return false
	}
	// Restore body for later use
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// Construct message: timestamp + body
	message := []byte(timestamp + string(body))

	// Verify signature
	return ed25519.Verify(b.publicKey, message, signatureBytes)
}

// processInteraction routes the interaction to the appropriate handler
func (b *DiscordBot) processInteraction(ctx context.Context, interaction *Interaction) *InteractionResponse {
	switch interaction.Type {
	case InteractionTypeApplicationCommand:
		return b.handleApplicationCommand(ctx, interaction)
	case InteractionTypeMessageComponent:
		// Handle message components if needed
		return b.createEphemeralResponse("❌ This interaction type is not supported yet.")
	default:
		return b.createEphemeralResponse("❌ Unknown interaction type.")
	}
}

// handleApplicationCommand processes slash commands
func (b *DiscordBot) handleApplicationCommand(ctx context.Context, interaction *Interaction) *InteractionResponse {
	commandName := interaction.Data.Name

	switch commandName {
	case "receipt":
		return b.processReceiptCommand(ctx, interaction)
	case "summary":
		return b.processSummaryCommand(ctx, interaction)
	case "pending":
		return b.processPendingCommand(ctx, interaction)
	default:
		return b.createEphemeralResponse("❓ Unknown command. Available commands: /receipt upload, /summary, /pending")
	}
}

// processReceiptCommand handles receipt upload command
func (b *DiscordBot) processReceiptCommand(ctx context.Context, interaction *Interaction) *InteractionResponse {
	// Get subcommand
	if len(interaction.Data.Options) == 0 {
		return b.createEphemeralResponse("❌ Please specify a subcommand. Usage: /receipt upload")
	}

	subcommand := interaction.Data.Options[0]
	if subcommand.Name != "upload" {
		return b.createEphemeralResponse("❌ Unknown subcommand. Usage: /receipt upload")
	}

	// Check if user is linked
	user, err := b.getUserByDiscordID(ctx, interaction.Member.User.ID)
	if err != nil {
		slog.Error("Failed to get user by Discord ID", "error", err)
		return b.createEphemeralResponse("❌ Sorry, something went wrong. Please try again later.")
	}

	if user == nil {
		return b.createEphemeralResponse("🔗 Please link your Discord account in the Receipt Manager dashboard first.")
	}

	// Check for attachments in the resolved data
	if interaction.Data.Resolved != nil && len(interaction.Data.Resolved.Attachments) > 0 {
		// Process the first attachment
		for _, attachment := range interaction.Data.Resolved.Attachments {
			return b.processAttachment(ctx, interaction, user, &attachment)
		}
	}

	// If no attachment, instruct user to upload with command
	return b.createEphemeralResponse("📎 Please attach a receipt image when using this command.")
}

// processAttachment downloads and processes a Discord attachment
func (b *DiscordBot) processAttachment(ctx context.Context, interaction *Interaction, user *model.User, attachment *Attachment) *InteractionResponse {
	slog.Info("Processing Discord attachment",
		"user_id", user.ID,
		"filename", attachment.Filename,
		"content_type", attachment.ContentType,
	)

	// Download the attachment
	resp, err := b.httpClient.Get(attachment.URL)
	if err != nil {
		slog.Error("Failed to download attachment", "error", err, "url", attachment.URL)
		return b.createEphemeralResponse("❌ Sorry, failed to download the attachment. Please try again.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Failed to download attachment", "status", resp.StatusCode)
		return b.createEphemeralResponse("❌ Sorry, failed to download the attachment. Please try again.")
	}

	// Read attachment data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read attachment data", "error", err)
		return b.createEphemeralResponse("❌ Sorry, something went wrong. Please try again.")
	}

	// Determine content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = attachment.ContentType
	}
	if contentType == "" {
		contentType = "image/jpeg"
	}

	// Upload to RustFS
	filename := fmt.Sprintf("discord_%s_%d_%s", interaction.Member.User.ID, time.Now().Unix(), attachment.Filename)
	objectKey, err := b.storageService.Upload(ctx, bytes.NewReader(data), int64(len(data)), contentType, filename)
	if err != nil {
		slog.Error("Failed to upload attachment to storage", "error", err)
		return b.createEphemeralResponse("❌ Sorry, failed to save the attachment. Please try again.")
	}

	slog.Info("Attachment uploaded to storage",
		"object_key", objectKey,
		"user_id", user.ID,
	)

	// Generate presigned URL for workflow
	presignedURL, err := b.storageService.GetPresignedURL(ctx, objectKey, 24*time.Hour)
	if err != nil {
		slog.Error("Failed to generate presigned URL", "error", err)
		return b.createEphemeralResponse("❌ Sorry, something went wrong. Please try again.")
	}

	// Start Temporal workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("parse-receipt-discord-%s-%d", interaction.Member.User.ID, time.Now().Unix()),
		TaskQueue: b.cfg.Temporal.TaskQueue,
	}

	input := workflow.ParseReceiptInput{
		UserID:   user.ID.String(),
		ImageURL: presignedURL,
		Source:   "discord",
	}

	we, err := b.temporalClient.ExecuteWorkflow(ctx, workflowOptions, workflow.ParseReceiptWorkflow, input)
	if err != nil {
		slog.Error("Failed to start workflow", "error", err)
		return b.createEphemeralResponse("❌ Sorry, failed to start processing. Please try again.")
	}

	slog.Info("Started ParseReceiptWorkflow",
		"workflow_id", we.GetID(),
		"run_id", we.GetRunID(),
		"user_id", user.ID,
	)

	// Return ephemeral response
	return &InteractionResponse{
		Type: InteractionResponseTypeChannelMessageWithSource,
		Data: &InteractionResponseData{
			Content: "📸 Receipt received! Processing... You'll get a notification when it's ready.",
			Flags:   InteractionResponseFlagsEphemeral,
		},
	}
}

// processSummaryCommand handles the /summary command
func (b *DiscordBot) processSummaryCommand(ctx context.Context, interaction *Interaction) *InteractionResponse {
	// Get user by Discord ID
	user, err := b.getUserByDiscordID(ctx, interaction.Member.User.ID)
	if err != nil {
		slog.Error("Failed to get user by Discord ID", "error", err)
		return b.createEphemeralResponse("❌ Sorry, something went wrong. Please try again later.")
	}

	if user == nil {
		return b.createEphemeralResponse("🔗 Please link your Discord account in the Receipt Manager dashboard first.")
	}

	// Get this month's total
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	total, err := b.receiptRepo.GetMonthlyTotal(ctx, user.ID, startOfMonth, endOfMonth)
	if err != nil {
		slog.Error("Failed to get monthly total", "error", err, "user_id", user.ID)
		return b.createEphemeralResponse("❌ Sorry, something went wrong. Please try again later.")
	}

	// Format the response
	currency := user.HomeCurrency
	if currency == "" {
		currency = "USD"
	}

	response := fmt.Sprintf(
		"📊 **%s %d Spending Summary**\n\n💰 Total: **%.2f %s**",
		now.Month().String(),
		now.Year(),
		total,
		currency,
	)

	return &InteractionResponse{
		Type: InteractionResponseTypeChannelMessageWithSource,
		Data: &InteractionResponseData{
			Content: response,
			Flags:   InteractionResponseFlagsEphemeral,
		},
	}
}

// processPendingCommand handles the /pending command
func (b *DiscordBot) processPendingCommand(ctx context.Context, interaction *Interaction) *InteractionResponse {
	// Get user by Discord ID
	user, err := b.getUserByDiscordID(ctx, interaction.Member.User.ID)
	if err != nil {
		slog.Error("Failed to get user by Discord ID", "error", err)
		return b.createEphemeralResponse("❌ Sorry, something went wrong. Please try again later.")
	}

	if user == nil {
		return b.createEphemeralResponse("🔗 Please link your Discord account in the Receipt Manager dashboard first.")
	}

	// Get pending review count
	count, err := b.receiptRepo.CountByStatus(ctx, user.ID, model.ReceiptStatusPendingReview)
	if err != nil {
		slog.Error("Failed to get pending count", "error", err, "user_id", user.ID)
		return b.createEphemeralResponse("❌ Sorry, something went wrong. Please try again later.")
	}

	// Format the response
	var response string
	if count == 0 {
		response = "✅ **All caught up!**\n\nNo receipts waiting for review."
	} else if count == 1 {
		response = "📝 **1 receipt** is waiting for your review.\n\nCheck it out in your dashboard!"
	} else {
		response = fmt.Sprintf("📝 **%d receipts** are waiting for your review.\n\nCheck them out in your dashboard!", count)
	}

	return &InteractionResponse{
		Type: InteractionResponseTypeChannelMessageWithSource,
		Data: &InteractionResponseData{
			Content: response,
			Flags:   InteractionResponseFlagsEphemeral,
		},
	}
}

// createEphemeralResponse creates a simple ephemeral response
func (b *DiscordBot) createEphemeralResponse(content string) *InteractionResponse {
	return &InteractionResponse{
		Type: InteractionResponseTypeChannelMessageWithSource,
		Data: &InteractionResponseData{
			Content: content,
			Flags:   InteractionResponseFlagsEphemeral,
		},
	}
}

// getUserByDiscordID checks if a Discord ID is linked to a user
func (b *DiscordBot) getUserByDiscordID(ctx context.Context, discordID string) (*model.User, error) {
	return b.userRepo.GetByDiscordID(ctx, discordID)
}
