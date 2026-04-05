package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/model"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	// NotificationTypeParse is sent when a receipt is parsed
	NotificationTypeParse NotificationType = "parse"
	// NotificationTypePendingReview is sent for pending review reminders
	NotificationTypePendingReview NotificationType = "pending_review"
	// NotificationTypeBudgetAlert is sent when budget threshold is exceeded
	NotificationTypeBudgetAlert NotificationType = "budget_alert"
)

// BotNotifier defines the interface for sending notifications via bots
type BotNotifier interface {
	// SendMessage sends a message to a user via their linked bot
	SendMessage(ctx context.Context, userID uuid.UUID, message string) error
	// IsAvailable returns true if the notifier can send messages to this user
	IsAvailable(user *model.User) bool
}

// UserRepository defines the interface for user data operations needed by notification service
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

// NotificationService provides notification business logic
type NotificationService struct {
	userRepo    UserRepository
	botNotifier BotNotifier
}

// NewNotificationService creates a new notification service
func NewNotificationService(userRepo UserRepository, botNotifier BotNotifier) *NotificationService {
	return &NotificationService{
		userRepo:    userRepo,
		botNotifier: botNotifier,
	}
}

// ShouldNotify checks if a user should receive a notification of the given type
func (s *NotificationService) ShouldNotify(userID string, notificationType string) bool {
	ctx := context.Background()

	// Parse UUID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false
	}

	// Get user to check preferences
	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil || user == nil {
		return false
	}

	// Check notification preferences based on type
	switch NotificationType(notificationType) {
	case NotificationTypeParse:
		return user.NotifyOnParse
	case NotificationTypePendingReview:
		return user.NotifyOnPendingReview
	case NotificationTypeBudgetAlert:
		return user.NotifyBudgetAlerts
	default:
		return false
	}
}

// SendBudgetAlert sends a budget alert notification if the user has enabled them
func (s *NotificationService) SendBudgetAlert(ctx context.Context, userID string, budget *BudgetStatus) error {
	// Parse UUID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if user wants budget alerts
	if !s.ShouldNotify(userID, string(NotificationTypeBudgetAlert)) {
		return nil // Silently skip if notifications disabled
	}

	// Get user details
	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Check if bot notifier is available for this user
	if s.botNotifier == nil || !s.botNotifier.IsAvailable(user) {
		return nil // No bot linked, skip silently
	}

	// Build alert message
	var alertEmoji string
	if budget.Percentage >= 100 {
		alertEmoji = "🚨"
	} else {
		alertEmoji = "⚠️"
	}

	message := fmt.Sprintf(
		"%s *Budget Alert*\n\n"+
			"You've spent *%.1f%%* of your budget limit.\n"+
			"Limit: *%.2f*\n"+
			"Spent: *%.2f*\n"+
			"Remaining: *%.2f*",
		alertEmoji,
		budget.Percentage,
		budget.AmountLimit,
		budget.Spent,
		budget.Remaining,
	)

	// Send the notification
	if err := s.botNotifier.SendMessage(ctx, uid, message); err != nil {
		return fmt.Errorf("failed to send budget alert: %w", err)
	}

	return nil
}

// SendPendingReminder sends a pending review reminder if the user has enabled them
func (s *NotificationService) SendPendingReminder(ctx context.Context, userID string, count int) error {
	// Skip if no pending receipts
	if count <= 0 {
		return nil
	}

	// Parse UUID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if user wants pending review reminders
	if !s.ShouldNotify(userID, string(NotificationTypePendingReview)) {
		return nil // Silently skip if notifications disabled
	}

	// Get user details
	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Check if bot notifier is available for this user
	if s.botNotifier == nil || !s.botNotifier.IsAvailable(user) {
		return nil // No bot linked, skip silently
	}

	// Build reminder message
	var message string
	if count == 1 {
		message = "📝 *Pending Review Reminder*\n\nYou have *1 receipt* waiting for your review."
	} else {
		message = fmt.Sprintf(
			"📝 *Pending Review Reminder*\n\nYou have *%d receipts* waiting for your review.",
			count,
		)
	}

	// Send the notification
	if err := s.botNotifier.SendMessage(ctx, uid, message); err != nil {
		return fmt.Errorf("failed to send pending reminder: %w", err)
	}

	return nil
}

// SendParseNotification sends a receipt parsed notification if the user has enabled them
func (s *NotificationService) SendParseNotification(ctx context.Context, userID string, receiptTitle string) error {
	// Parse UUID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if user wants parse notifications
	if !s.ShouldNotify(userID, string(NotificationTypeParse)) {
		return nil // Silently skip if notifications disabled
	}

	// Get user details
	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Check if bot notifier is available for this user
	if s.botNotifier == nil || !s.botNotifier.IsAvailable(user) {
		return nil // No bot linked, skip silently
	}

	// Build notification message
	message := fmt.Sprintf(
		"✅ *Receipt Processed*\n\n"+
			"Your receipt *%s* has been successfully parsed and saved.",
		receiptTitle,
	)

	// Send the notification
	if err := s.botNotifier.SendMessage(ctx, uid, message); err != nil {
		return fmt.Errorf("failed to send parse notification: %w", err)
	}

	return nil
}
