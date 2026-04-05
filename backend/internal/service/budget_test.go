package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBudgetRepo is a mock implementation of the budget repository
type MockBudgetRepo struct {
	mock.Mock
}

func (m *MockBudgetRepo) Create(ctx context.Context, budget *model.Budget) (*model.Budget, error) {
	args := m.Called(ctx, budget)
	return args.Get(0).(*model.Budget), args.Error(1)
}

func (m *MockBudgetRepo) GetByUserAndMonth(ctx context.Context, userID uuid.UUID, month string) ([]*model.Budget, error) {
	args := m.Called(ctx, userID, month)
	return args.Get(0).([]*model.Budget), args.Error(1)
}

func (m *MockBudgetRepo) GetSpentByTag(ctx context.Context, userID uuid.UUID, tagID uuid.UUID, month string) (float64, error) {
	args := m.Called(ctx, userID, tagID, month)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockBudgetRepo) Update(ctx context.Context, budget *model.Budget) (*model.Budget, error) {
	args := m.Called(ctx, budget)
	return args.Get(0).(*model.Budget), args.Error(1)
}

func (m *MockBudgetRepo) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockBudgetRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Budget, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Budget), args.Error(1)
}

func TestBudgetService_GetBudgetStatus(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tagID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	month := "2024-01"

	t.Run("returns budget status for single budget", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		budget := &model.Budget{
			ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			UserID:      userID,
			TagID:       &tagID,
			Month:       month,
			AmountLimit: 1000.00,
		}

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{budget}, nil)
		mockRepo.On("GetSpentByTag", ctx, userID, tagID, month).Return(600.00, nil)

		status, err := service.GetBudgetStatus(ctx, userID, month)

		assert.NoError(t, err)
		assert.Len(t, status, 1)
		assert.Equal(t, budget.ID, status[0].BudgetID)
		assert.Equal(t, tagID, *status[0].TagID)
		assert.Equal(t, 1000.00, status[0].AmountLimit)
		assert.Equal(t, 600.00, status[0].Spent)
		assert.Equal(t, 60.0, status[0].Percentage)
		assert.Equal(t, 400.00, status[0].Remaining)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns budget status for multiple budgets", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		tagID1 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		tagID2 := uuid.MustParse("44444444-4444-4444-4444-444444444444")

		budget1 := &model.Budget{
			ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			UserID:      userID,
			TagID:       &tagID1,
			Month:       month,
			AmountLimit: 1000.00,
		}
		budget2 := &model.Budget{
			ID:          uuid.MustParse("55555555-5555-5555-5555-555555555555"),
			UserID:      userID,
			TagID:       &tagID2,
			Month:       month,
			AmountLimit: 500.00,
		}

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{budget1, budget2}, nil)
		mockRepo.On("GetSpentByTag", ctx, userID, tagID1, month).Return(800.00, nil)
		mockRepo.On("GetSpentByTag", ctx, userID, tagID2, month).Return(200.00, nil)

		status, err := service.GetBudgetStatus(ctx, userID, month)

		assert.NoError(t, err)
		assert.Len(t, status, 2)

		// First budget: 80% spent
		assert.Equal(t, 800.00, status[0].Spent)
		assert.Equal(t, 80.0, status[0].Percentage)
		assert.Equal(t, 200.00, status[0].Remaining)

		// Second budget: 40% spent
		assert.Equal(t, 200.00, status[1].Spent)
		assert.Equal(t, 40.0, status[1].Percentage)
		assert.Equal(t, 300.00, status[1].Remaining)

		mockRepo.AssertExpectations(t)
	})

	t.Run("handles budget with zero spending", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		budget := &model.Budget{
			ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			UserID:      userID,
			TagID:       &tagID,
			Month:       month,
			AmountLimit: 1000.00,
		}

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{budget}, nil)
		mockRepo.On("GetSpentByTag", ctx, userID, tagID, month).Return(0.00, nil)

		status, err := service.GetBudgetStatus(ctx, userID, month)

		assert.NoError(t, err)
		assert.Len(t, status, 1)
		assert.Equal(t, 0.00, status[0].Spent)
		assert.Equal(t, 0.0, status[0].Percentage)
		assert.Equal(t, 1000.00, status[0].Remaining)
		mockRepo.AssertExpectations(t)
	})

	t.Run("handles budget with no tag (global budget)", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		budget := &model.Budget{
			ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			UserID:      userID,
			TagID:       nil,
			Month:       month,
			AmountLimit: 2000.00,
		}

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{budget}, nil)
		// For global budgets with nil TagID, GetSpentByTag should not be called

		status, err := service.GetBudgetStatus(ctx, userID, month)

		assert.NoError(t, err)
		assert.Len(t, status, 1)
		assert.Nil(t, status[0].TagID)
		assert.Equal(t, 0.00, status[0].Spent)
		assert.Equal(t, 0.0, status[0].Percentage)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns empty slice when no budgets exist", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{}, nil)

		status, err := service.GetBudgetStatus(ctx, userID, month)

		assert.NoError(t, err)
		assert.Empty(t, status)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{}, errors.New("db error"))

		status, err := service.GetBudgetStatus(ctx, userID, month)

		assert.Error(t, err)
		assert.Nil(t, status)
		assert.Contains(t, err.Error(), "failed to get budgets")
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error when GetSpentByTag fails", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		budget := &model.Budget{
			ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			UserID:      userID,
			TagID:       &tagID,
			Month:       month,
			AmountLimit: 1000.00,
		}

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{budget}, nil)
		mockRepo.On("GetSpentByTag", ctx, userID, tagID, month).Return(0.00, errors.New("spent error"))

		status, err := service.GetBudgetStatus(ctx, userID, month)

		assert.Error(t, err)
		assert.Nil(t, status)
		assert.Contains(t, err.Error(), "failed to get spent")
		mockRepo.AssertExpectations(t)
	})
}

func TestBudgetService_CheckBudgetAlerts(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	// Use current month since CheckBudgetAlerts uses time.Now()
	month := time.Now().Format("2006-01")

	t.Run("returns warning alert when budget exceeds 80% threshold", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		tagID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		budget := &model.Budget{
			ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			UserID:      userID,
			TagID:       &tagID,
			Month:       month,
			AmountLimit: 1000.00,
		}

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{budget}, nil)
		mockRepo.On("GetSpentByTag", ctx, userID, tagID, month).Return(850.00, nil)

		alerts, err := service.CheckBudgetAlerts(ctx, userID)

		assert.NoError(t, err)
		assert.Len(t, alerts, 1)
		assert.Equal(t, AlertTypeWarning, alerts[0].AlertType)
		assert.Equal(t, budget.ID, alerts[0].BudgetID)
		assert.Contains(t, alerts[0].Message, "85.0%")
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns critical alert when budget exceeds 100% threshold", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		tagID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		budget := &model.Budget{
			ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			UserID:      userID,
			TagID:       &tagID,
			Month:       month,
			AmountLimit: 1000.00,
		}

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{budget}, nil)
		mockRepo.On("GetSpentByTag", ctx, userID, tagID, month).Return(1200.00, nil)

		alerts, err := service.CheckBudgetAlerts(ctx, userID)

		assert.NoError(t, err)
		assert.Len(t, alerts, 1)
		assert.Equal(t, AlertTypeCritical, alerts[0].AlertType)
		assert.Equal(t, budget.ID, alerts[0].BudgetID)
		assert.Contains(t, alerts[0].Message, "120.0%")
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns no alerts when budget is below 80% threshold", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		tagID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		budget := &model.Budget{
			ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			UserID:      userID,
			TagID:       &tagID,
			Month:       month,
			AmountLimit: 1000.00,
		}

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{budget}, nil)
		mockRepo.On("GetSpentByTag", ctx, userID, tagID, month).Return(500.00, nil)

		alerts, err := service.CheckBudgetAlerts(ctx, userID)

		assert.NoError(t, err)
		assert.Empty(t, alerts)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns multiple alerts for different budgets", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		tagID1 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		tagID2 := uuid.MustParse("44444444-4444-4444-4444-444444444444")
		tagID3 := uuid.MustParse("66666666-6666-6666-6666-666666666666")

		budget1 := &model.Budget{
			ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			UserID:      userID,
			TagID:       &tagID1,
			Month:       month,
			AmountLimit: 1000.00,
		}
		budget2 := &model.Budget{
			ID:          uuid.MustParse("55555555-5555-5555-5555-555555555555"),
			UserID:      userID,
			TagID:       &tagID2,
			Month:       month,
			AmountLimit: 500.00,
		}
		budget3 := &model.Budget{
			ID:          uuid.MustParse("77777777-7777-7777-7777-777777777777"),
			UserID:      userID,
			TagID:       &tagID3,
			Month:       month,
			AmountLimit: 2000.00,
		}

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{budget1, budget2, budget3}, nil)
		mockRepo.On("GetSpentByTag", ctx, userID, tagID1, month).Return(900.00, nil)  // 90% - warning
		mockRepo.On("GetSpentByTag", ctx, userID, tagID2, month).Return(600.00, nil)  // 120% - critical
		mockRepo.On("GetSpentByTag", ctx, userID, tagID3, month).Return(1000.00, nil) // 50% - no alert

		alerts, err := service.CheckBudgetAlerts(ctx, userID)

		assert.NoError(t, err)
		assert.Len(t, alerts, 2)

		// Should have one warning and one critical
		alertTypes := make([]AlertType, len(alerts))
		for i, alert := range alerts {
			alertTypes[i] = alert.AlertType
		}
		assert.Contains(t, alertTypes, AlertTypeWarning)
		assert.Contains(t, alertTypes, AlertTypeCritical)

		mockRepo.AssertExpectations(t)
	})

	t.Run("returns empty alerts when no budgets exist", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{}, nil)

		alerts, err := service.CheckBudgetAlerts(ctx, userID)

		assert.NoError(t, err)
		assert.Empty(t, alerts)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		mockRepo := new(MockBudgetRepo)
		service := NewBudgetService(mockRepo)

		mockRepo.On("GetByUserAndMonth", ctx, userID, month).Return([]*model.Budget{}, errors.New("db error"))

		alerts, err := service.CheckBudgetAlerts(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, alerts)
		assert.Contains(t, err.Error(), "failed to get budgets")
		mockRepo.AssertExpectations(t)
	})
}
