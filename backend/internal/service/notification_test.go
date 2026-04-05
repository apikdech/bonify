package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepo is a mock implementation of the user repository
type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

// MockBotNotifier is a mock implementation of the BotNotifier interface
type MockBotNotifier struct {
	mock.Mock
}

func (m *MockBotNotifier) SendMessage(ctx context.Context, userID uuid.UUID, message string) error {
	args := m.Called(ctx, userID, message)
	return args.Error(0)
}

func (m *MockBotNotifier) IsAvailable(user *model.User) bool {
	args := m.Called(user)
	return args.Bool(0)
}

func TestNotificationService_ShouldNotify(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	t.Run("returns true for budget alerts when enabled", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                 userID,
			NotifyBudgetAlerts: true,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil)

		result := service.ShouldNotify(userID.String(), string(NotificationTypeBudgetAlert))

		assert.True(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns false for budget alerts when disabled", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                 userID,
			NotifyBudgetAlerts: false,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil)

		result := service.ShouldNotify(userID.String(), string(NotificationTypeBudgetAlert))

		assert.False(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns true for pending review when enabled", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                    userID,
			NotifyOnPendingReview: true,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil)

		result := service.ShouldNotify(userID.String(), string(NotificationTypePendingReview))

		assert.True(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns false for pending review when disabled", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                    userID,
			NotifyOnPendingReview: false,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil)

		result := service.ShouldNotify(userID.String(), string(NotificationTypePendingReview))

		assert.False(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns true for parse notifications when enabled", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:            userID,
			NotifyOnParse: true,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil)

		result := service.ShouldNotify(userID.String(), string(NotificationTypeParse))

		assert.True(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns false for parse notifications when disabled", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:            userID,
			NotifyOnParse: false,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil)

		result := service.ShouldNotify(userID.String(), string(NotificationTypeParse))

		assert.False(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns false for unknown notification type", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID: userID,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil)

		result := service.ShouldNotify(userID.String(), "unknown_type")

		assert.False(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns false when user not found", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		mockRepo.On("GetByID", ctx, userID).Return(nil, nil)

		result := service.ShouldNotify(userID.String(), string(NotificationTypeBudgetAlert))

		assert.False(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns false when repository fails", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		mockRepo.On("GetByID", ctx, userID).Return(nil, errors.New("db error"))

		result := service.ShouldNotify(userID.String(), string(NotificationTypeBudgetAlert))

		assert.False(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns false for invalid user ID", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		result := service.ShouldNotify("invalid-uuid", string(NotificationTypeBudgetAlert))

		assert.False(t, result)
	})
}

func TestNotificationService_SendBudgetAlert(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	budgetID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	t.Run("sends budget alert when enabled and bot available", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                 userID,
			NotifyBudgetAlerts: true,
			TelegramID:         strPtr("123456"),
		}

		budget := &BudgetStatus{
			BudgetID:    budgetID,
			AmountLimit: 1000.00,
			Spent:       850.00,
			Percentage:  85.0,
			Remaining:   150.00,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Twice()
		mockNotifier.On("IsAvailable", user).Return(true)
		mockNotifier.On("SendMessage", ctx, userID, mock.AnythingOfType("string")).Return(nil)

		err := service.SendBudgetAlert(ctx, userID.String(), budget)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)
	})

	t.Run("sends critical alert when percentage >= 100", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                 userID,
			NotifyBudgetAlerts: true,
			TelegramID:         strPtr("123456"),
		}

		budget := &BudgetStatus{
			BudgetID:    budgetID,
			AmountLimit: 1000.00,
			Spent:       1200.00,
			Percentage:  120.0,
			Remaining:   -200.00,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Twice()
		mockNotifier.On("IsAvailable", user).Return(true)
		mockNotifier.On("SendMessage", ctx, userID, mock.MatchedBy(func(msg string) bool {
			return contains(msg, "🚨")
		})).Return(nil)

		err := service.SendBudgetAlert(ctx, userID.String(), budget)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)
	})

	t.Run("skips when budget alerts disabled", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                 userID,
			NotifyBudgetAlerts: false,
		}

		budget := &BudgetStatus{
			BudgetID:    budgetID,
			AmountLimit: 1000.00,
			Spent:       900.00,
			Percentage:  90.0,
			Remaining:   100.00,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Once()

		err := service.SendBudgetAlert(ctx, userID.String(), budget)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockNotifier.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("skips silently when no bot linked", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                 userID,
			NotifyBudgetAlerts: true,
			TelegramID:         nil,
			DiscordID:          nil,
		}

		budget := &BudgetStatus{
			BudgetID:    budgetID,
			AmountLimit: 1000.00,
			Spent:       900.00,
			Percentage:  90.0,
			Remaining:   100.00,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Twice()
		mockNotifier.On("IsAvailable", user).Return(false)

		err := service.SendBudgetAlert(ctx, userID.String(), budget)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)
	})

	t.Run("skips silently when user not found", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		budget := &BudgetStatus{
			BudgetID:    budgetID,
			AmountLimit: 1000.00,
			Spent:       900.00,
			Percentage:  90.0,
			Remaining:   100.00,
		}

		mockRepo.On("GetByID", ctx, userID).Return(nil, nil).Once()

		err := service.SendBudgetAlert(ctx, userID.String(), budget)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error for invalid user ID", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		budget := &BudgetStatus{
			BudgetID:    budgetID,
			AmountLimit: 1000.00,
			Spent:       900.00,
			Percentage:  90.0,
			Remaining:   100.00,
		}

		err := service.SendBudgetAlert(ctx, "invalid-uuid", budget)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("returns error when notifier fails", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                 userID,
			NotifyBudgetAlerts: true,
			TelegramID:         strPtr("123456"),
		}

		budget := &BudgetStatus{
			BudgetID:    budgetID,
			AmountLimit: 1000.00,
			Spent:       900.00,
			Percentage:  90.0,
			Remaining:   100.00,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Twice()
		mockNotifier.On("IsAvailable", user).Return(true)
		mockNotifier.On("SendMessage", ctx, userID, mock.Anything).Return(errors.New("telegram error"))

		err := service.SendBudgetAlert(ctx, userID.String(), budget)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send budget alert")
		mockRepo.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)
	})
}

func TestNotificationService_SendPendingReminder(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	t.Run("sends reminder when enabled and count > 0", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                    userID,
			NotifyOnPendingReview: true,
			TelegramID:            strPtr("123456"),
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Twice()
		mockNotifier.On("IsAvailable", user).Return(true)
		mockNotifier.On("SendMessage", ctx, userID, mock.MatchedBy(func(msg string) bool {
			return contains(msg, "5 receipts")
		})).Return(nil)

		err := service.SendPendingReminder(ctx, userID.String(), 5)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)
	})

	t.Run("sends singular reminder when count is 1", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                    userID,
			NotifyOnPendingReview: true,
			TelegramID:            strPtr("123456"),
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Twice()
		mockNotifier.On("IsAvailable", user).Return(true)
		mockNotifier.On("SendMessage", ctx, userID, mock.MatchedBy(func(msg string) bool {
			return contains(msg, "1 receipt") && !contains(msg, "receipts")
		})).Return(nil)

		err := service.SendPendingReminder(ctx, userID.String(), 1)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)
	})

	t.Run("skips when count is 0", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		err := service.SendPendingReminder(ctx, userID.String(), 0)

		assert.NoError(t, err)
		mockRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
		mockNotifier.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("skips when count is negative", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		err := service.SendPendingReminder(ctx, userID.String(), -1)

		assert.NoError(t, err)
		mockRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
		mockNotifier.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("skips when pending review notifications disabled", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                    userID,
			NotifyOnPendingReview: false,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Once()

		err := service.SendPendingReminder(ctx, userID.String(), 5)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockNotifier.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("skips silently when no bot linked", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                    userID,
			NotifyOnPendingReview: true,
			TelegramID:            nil,
			DiscordID:             nil,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Twice()
		mockNotifier.On("IsAvailable", user).Return(false)

		err := service.SendPendingReminder(ctx, userID.String(), 5)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)
	})

	t.Run("skips silently when user not found", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		mockRepo.On("GetByID", ctx, userID).Return(nil, nil).Once()

		err := service.SendPendingReminder(ctx, userID.String(), 5)

		assert.NoError(t, err)
	})

	t.Run("returns error for invalid user ID", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		err := service.SendPendingReminder(ctx, "invalid-uuid", 5)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("returns error when notifier fails", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:                    userID,
			NotifyOnPendingReview: true,
			TelegramID:            strPtr("123456"),
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Twice()
		mockNotifier.On("IsAvailable", user).Return(true)
		mockNotifier.On("SendMessage", ctx, userID, mock.Anything).Return(errors.New("telegram error"))

		err := service.SendPendingReminder(ctx, userID.String(), 5)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send pending reminder")
	})
}

func TestNotificationService_SendParseNotification(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	t.Run("sends parse notification when enabled", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:            userID,
			NotifyOnParse: true,
			TelegramID:    strPtr("123456"),
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Twice()
		mockNotifier.On("IsAvailable", user).Return(true)
		mockNotifier.On("SendMessage", ctx, userID, mock.MatchedBy(func(msg string) bool {
			return contains(msg, "Receipt Processed") && contains(msg, "Grocery Store")
		})).Return(nil)

		err := service.SendParseNotification(ctx, userID.String(), "Grocery Store")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)
	})

	t.Run("skips when parse notifications disabled", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:            userID,
			NotifyOnParse: false,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Once()

		err := service.SendParseNotification(ctx, userID.String(), "Grocery Store")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockNotifier.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("skips silently when no bot linked", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		mockNotifier := new(MockBotNotifier)
		service := NewNotificationService(mockRepo, mockNotifier)

		user := &model.User{
			ID:            userID,
			NotifyOnParse: true,
			TelegramID:    nil,
			DiscordID:     nil,
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Twice()
		mockNotifier.On("IsAvailable", user).Return(false)

		err := service.SendParseNotification(ctx, userID.String(), "Grocery Store")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockNotifier.AssertExpectations(t)
	})

	t.Run("handles nil bot notifier gracefully", func(t *testing.T) {
		mockRepo := new(MockUserRepo)
		service := NewNotificationService(mockRepo, nil)

		user := &model.User{
			ID:            userID,
			NotifyOnParse: true,
			TelegramID:    strPtr("123456"),
		}

		mockRepo.On("GetByID", ctx, userID).Return(user, nil).Twice()

		err := service.SendParseNotification(ctx, userID.String(), "Grocery Store")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
