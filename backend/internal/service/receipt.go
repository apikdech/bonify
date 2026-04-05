package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/model"
	"github.com/receipt-manager/backend/internal/repository"
)

// Sentinel errors for the service layer
var (
	ErrReceiptNotFound = errors.New("receipt not found")
)

// ReceiptService provides receipt business logic
type ReceiptService struct {
	receiptRepo *repository.ReceiptRepo
	tagRepo     *repository.TagRepo
}

// NewReceiptService creates a new receipt service
func NewReceiptService(receiptRepo *repository.ReceiptRepo, tagRepo *repository.TagRepo) *ReceiptService {
	return &ReceiptService{
		receiptRepo: receiptRepo,
		tagRepo:     tagRepo,
	}
}

// CreateManual creates a receipt from manual entry
func (s *ReceiptService) CreateManual(ctx context.Context, userID uuid.UUID, req *model.CreateReceiptRequest) (*model.Receipt, error) {
	// Calculate subtotals and totals
	items, itemsSubtotal := calculateItems(req.Items)
	fees, feesTotal := calculateFees(req.Fees)

	// Create the receipt
	receipt := &model.Receipt{
		UserID:        userID,
		Title:         stringPtrIfNotEmpty(req.Title),
		Source:        model.ReceiptSourceManual,
		Currency:      req.Currency,
		PaymentMethod: stringPtrIfNotEmpty(req.PaymentMethod),
		Subtotal:      itemsSubtotal,
		Total:         itemsSubtotal + feesTotal,
		Status:        model.ReceiptStatusPendingReview,
		Notes:         stringPtrIfNotEmpty(req.Notes),
		ReceiptDate:   req.ReceiptDate,
		PaidBy:        stringPtrIfNotEmpty(req.PaidBy),
	}

	// Save the receipt
	createdReceipt, err := s.receiptRepo.Create(ctx, receipt)
	if err != nil {
		return nil, fmt.Errorf("failed to create receipt: %w", err)
	}

	// Add items
	for _, item := range items {
		item.ReceiptID = createdReceipt.ID
		_, err := s.receiptRepo.AddItem(ctx, &item)
		if err != nil {
			return nil, fmt.Errorf("failed to add receipt item: %w", err)
		}
	}

	// Add fees
	for _, fee := range fees {
		fee.ReceiptID = createdReceipt.ID
		_, err := s.receiptRepo.AddFee(ctx, &fee)
		if err != nil {
			return nil, fmt.Errorf("failed to add receipt fee: %w", err)
		}
	}

	// Set tags if provided
	if len(req.TagIDs) > 0 {
		tagUUIDs := make([]uuid.UUID, 0, len(req.TagIDs))
		for _, tagID := range req.TagIDs {
			tagUUID, err := uuid.Parse(tagID)
			if err != nil {
				return nil, fmt.Errorf("invalid tag ID: %w", err)
			}
			tagUUIDs = append(tagUUIDs, tagUUID)
		}

		err := s.receiptRepo.SetTags(ctx, createdReceipt.ID, tagUUIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to set tags: %w", err)
		}
	}

	// Load items, fees, and tags for the response
	createdReceipt.Items, err = s.receiptRepo.GetItems(ctx, createdReceipt.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load items: %w", err)
	}
	createdReceipt.Fees, err = s.receiptRepo.GetFees(ctx, createdReceipt.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load fees: %w", err)
	}
	createdReceipt.Tags, err = s.receiptRepo.GetTags(ctx, createdReceipt.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tags: %w", err)
	}

	return createdReceipt, nil
}

// GetByID retrieves a receipt by ID with ownership check
func (s *ReceiptService) GetByID(ctx context.Context, receiptID uuid.UUID, userID uuid.UUID) (*model.Receipt, error) {
	receipt, err := s.receiptRepo.GetByID(ctx, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt: %w", err)
	}
	if receipt == nil {
		return nil, ErrReceiptNotFound
	}

	// Check ownership
	if receipt.UserID != userID {
		return nil, ErrReceiptNotFound
	}

	// Load related data
	receipt.Items, err = s.receiptRepo.GetItems(ctx, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to load items: %w", err)
	}
	receipt.Fees, err = s.receiptRepo.GetFees(ctx, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to load fees: %w", err)
	}
	receipt.Tags, err = s.receiptRepo.GetTags(ctx, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tags: %w", err)
	}

	return receipt, nil
}

// List lists receipts for a user with filters
func (s *ReceiptService) List(ctx context.Context, filter *model.ListReceiptsFilter) (*model.ReceiptListResponse, error) {
	// Set default pagination
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	receipts, total, err := s.receiptRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list receipts: %w", err)
	}

	// Load items, fees, and tags for each receipt
	for _, receipt := range receipts {
		var err error
		receipt.Items, err = s.receiptRepo.GetItems(ctx, receipt.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load items for receipt %s: %w", receipt.ID, err)
		}
		receipt.Fees, err = s.receiptRepo.GetFees(ctx, receipt.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load fees for receipt %s: %w", receipt.ID, err)
		}
		receipt.Tags, err = s.receiptRepo.GetTags(ctx, receipt.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load tags for receipt %s: %w", receipt.ID, err)
		}
	}

	return &model.ReceiptListResponse{
		Receipts: receipts,
		Total:    total,
		Page:     filter.Page,
		Limit:    filter.Limit,
	}, nil
}

// Update updates a receipt with recalculation
func (s *ReceiptService) Update(ctx context.Context, receiptID uuid.UUID, userID uuid.UUID, req *model.UpdateReceiptRequest) (*model.Receipt, error) {
	// Get the existing receipt with ownership check
	existing, err := s.GetByID(ctx, receiptID, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrReceiptNotFound
	}

	// Update fields
	if req.Title != nil {
		existing.Title = req.Title
	}
	if req.Currency != nil {
		existing.Currency = *req.Currency
	}
	if req.PaymentMethod != nil {
		existing.PaymentMethod = req.PaymentMethod
	}
	if req.Notes != nil {
		existing.Notes = req.Notes
	}
	if req.ReceiptDate != nil {
		existing.ReceiptDate = req.ReceiptDate
	}
	if req.PaidBy != nil {
		existing.PaidBy = req.PaidBy
	}

	// Track whether items or fees were updated
	itemsUpdated := req.Items != nil
	feesUpdated := req.Fees != nil

	// Update items if provided
	if itemsUpdated {
		// Delete existing items
		if err := s.receiptRepo.DeleteAllItems(ctx, receiptID); err != nil {
			return nil, fmt.Errorf("failed to delete existing items: %w", err)
		}

		// Add new items
		items, itemsSubtotal := calculateItems(*req.Items)
		for _, item := range items {
			item.ReceiptID = receiptID
			_, err := s.receiptRepo.AddItem(ctx, &item)
			if err != nil {
				return nil, fmt.Errorf("failed to add receipt item: %w", err)
			}
		}
		existing.Subtotal = itemsSubtotal
	}

	// Update fees if provided
	if feesUpdated {
		// Delete existing fees
		if err := s.receiptRepo.DeleteAllFees(ctx, receiptID); err != nil {
			return nil, fmt.Errorf("failed to delete existing fees: %w", err)
		}

		// Add new fees
		fees, feesTotal := calculateFees(*req.Fees)
		for _, fee := range fees {
			fee.ReceiptID = receiptID
			_, err := s.receiptRepo.AddFee(ctx, &fee)
			if err != nil {
				return nil, fmt.Errorf("failed to add receipt fee: %w", err)
			}
		}
		existing.Total = existing.Subtotal + feesTotal
	}

	// Recalculate total if items were updated but fees weren't
	// This ensures total reflects: new subtotal + existing fees
	if itemsUpdated && !feesUpdated {
		_, feesTotal := calculateFeesFromModel(existing.Fees)
		existing.Total = existing.Subtotal + feesTotal
	}

	// Update tags if provided
	if req.TagIDs != nil {
		tagUUIDs := make([]uuid.UUID, 0, len(*req.TagIDs))
		for _, tagID := range *req.TagIDs {
			tagUUID, err := uuid.Parse(tagID)
			if err != nil {
				return nil, fmt.Errorf("invalid tag ID: %w", err)
			}
			tagUUIDs = append(tagUUIDs, tagUUID)
		}

		if err := s.receiptRepo.SetTags(ctx, receiptID, tagUUIDs); err != nil {
			return nil, fmt.Errorf("failed to set tags: %w", err)
		}
	}

	// Save the receipt
	updated, err := s.receiptRepo.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update receipt: %w", err)
	}

	// Load items, fees, and tags for the response
	updated.Items, err = s.receiptRepo.GetItems(ctx, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to load items: %w", err)
	}
	updated.Fees, err = s.receiptRepo.GetFees(ctx, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to load fees: %w", err)
	}
	updated.Tags, err = s.receiptRepo.GetTags(ctx, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tags: %w", err)
	}

	return updated, nil
}

// UpdateStatus updates the status of a receipt
func (s *ReceiptService) UpdateStatus(ctx context.Context, receiptID uuid.UUID, userID uuid.UUID, status model.ReceiptStatus) error {
	// Verify ownership first
	receipt, err := s.receiptRepo.GetByID(ctx, receiptID)
	if err != nil {
		return fmt.Errorf("failed to get receipt: %w", err)
	}
	if receipt == nil {
		return ErrReceiptNotFound
	}
	if receipt.UserID != userID {
		return ErrReceiptNotFound
	}

	// Validate status
	if status != model.ReceiptStatusPendingReview &&
		status != model.ReceiptStatusConfirmed &&
		status != model.ReceiptStatusRejected {
		return fmt.Errorf("invalid status")
	}

	return s.receiptRepo.UpdateStatus(ctx, receiptID, status)
}

// Delete deletes a receipt with ownership check
func (s *ReceiptService) Delete(ctx context.Context, receiptID uuid.UUID, userID uuid.UUID) error {
	// Verify ownership first
	receipt, err := s.receiptRepo.GetByID(ctx, receiptID)
	if err != nil {
		return fmt.Errorf("failed to get receipt: %w", err)
	}
	if receipt == nil {
		return ErrReceiptNotFound
	}
	if receipt.UserID != userID {
		return ErrReceiptNotFound
	}

	return s.receiptRepo.Delete(ctx, receiptID)
}

// calculateItems calculates subtotals for items
func calculateItems(inputs []model.ReceiptItemInput) ([]model.ReceiptItem, float64) {
	items := make([]model.ReceiptItem, 0, len(inputs))
	totalSubtotal := 0.0

	for _, input := range inputs {
		subtotal := input.Quantity*input.UnitPrice - input.Discount
		if subtotal < 0 {
			subtotal = 0
		}
		totalSubtotal += subtotal

		items = append(items, model.ReceiptItem{
			Name:      input.Name,
			Quantity:  input.Quantity,
			UnitPrice: input.UnitPrice,
			Discount:  input.Discount,
			Subtotal:  subtotal,
		})
	}

	return items, totalSubtotal
}

// calculateFees processes fee inputs
func calculateFees(inputs []model.ReceiptFeeInput) ([]model.ReceiptFee, float64) {
	fees := make([]model.ReceiptFee, 0, len(inputs))
	totalFees := 0.0

	for _, input := range inputs {
		totalFees += input.Amount
		fees = append(fees, model.ReceiptFee{
			Label:   input.Label,
			FeeType: input.FeeType,
			Amount:  input.Amount,
		})
	}

	return fees, totalFees
}

// calculateFeesFromModel processes existing fees
func calculateFeesFromModel(existing []*model.ReceiptFee) ([]model.ReceiptFee, float64) {
	fees := make([]model.ReceiptFee, 0, len(existing))
	totalFees := 0.0

	for _, fee := range existing {
		totalFees += fee.Amount
		fees = append(fees, *fee)
	}

	return fees, totalFees
}

// stringPtrIfNotEmpty returns a string pointer if the string is not empty, otherwise nil
func stringPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// CreateFromParsed creates a receipt from LLM-parsed data
func (s *ReceiptService) CreateFromParsed(ctx context.Context, userID uuid.UUID, imageURL string, source string, parsed *ParsedReceipt) (string, error) {
	// Determine receipt source
	receiptSource := model.ReceiptSource(source)
	if receiptSource != "telegram" && receiptSource != "discord" {
		receiptSource = model.ReceiptSourceOCR
	}

	// Determine status based on OCR confidence
	confidence := parsed.OCRConfidence
	status := model.ReceiptStatusPendingReview
	// Note: The threshold should ideally come from config, using 0.85 as default
	if confidence >= 0.85 {
		status = model.ReceiptStatusConfirmed
	}

	// Convert parsed items to model items
	items := make([]model.ReceiptItem, 0, len(parsed.Items))
	var itemsSubtotal float64
	for _, parsedItem := range parsed.Items {
		subtotal := parsedItem.Quantity*parsedItem.UnitPrice - parsedItem.Discount
		if subtotal < 0 {
			subtotal = 0
		}
		itemsSubtotal += subtotal
		items = append(items, model.ReceiptItem{
			Name:      parsedItem.Name,
			Quantity:  parsedItem.Quantity,
			UnitPrice: parsedItem.UnitPrice,
			Discount:  parsedItem.Discount,
			Subtotal:  subtotal,
		})
	}

	// Convert parsed fees to model fees
	fees := make([]model.ReceiptFee, 0, len(parsed.Fees))
	var feesTotal float64
	for _, parsedFee := range parsed.Fees {
		feesTotal += parsedFee.Amount
		fees = append(fees, model.ReceiptFee{
			Label:   parsedFee.Label,
			FeeType: model.FeeType(parsedFee.FeeType),
			Amount:  parsedFee.Amount,
		})
	}

	// Parse receipt date if provided
	var receiptDate *string
	if parsed.ReceiptDate != nil {
		receiptDate = parsed.ReceiptDate
	}

	// Create the receipt
	receipt := &model.Receipt{
		UserID:        userID,
		Title:         stringPtrIfNotEmpty(parsed.Title),
		Source:        receiptSource,
		ImageURL:      stringPtrIfNotEmpty(imageURL),
		OCRConfidence: &confidence,
		Currency:      parsed.Currency,
		PaymentMethod: stringPtrIfNotEmpty(parsed.PaymentMethod),
		Subtotal:      itemsSubtotal,
		Total:         itemsSubtotal + feesTotal,
		Status:        status,
		ReceiptDate:   nil, // Will be set if parsed
		PaidBy:        nil,
	}

	// Parse receipt date
	if receiptDate != nil && *receiptDate != "" {
		// Try to parse the date - could be in various formats
		// For now, store as is and let the repository handle it
		// A proper implementation would try multiple date formats
	}

	// Save the receipt
	createdReceipt, err := s.receiptRepo.Create(ctx, receipt)
	if err != nil {
		return "", fmt.Errorf("failed to create receipt: %w", err)
	}

	// Add items
	for _, item := range items {
		item.ReceiptID = createdReceipt.ID
		_, err := s.receiptRepo.AddItem(ctx, &item)
		if err != nil {
			return "", fmt.Errorf("failed to add receipt item: %w", err)
		}
	}

	// Add fees
	for _, fee := range fees {
		fee.ReceiptID = createdReceipt.ID
		_, err := s.receiptRepo.AddFee(ctx, &fee)
		if err != nil {
			return "", fmt.Errorf("failed to add receipt fee: %w", err)
		}
	}

	return createdReceipt.ID.String(), nil
}
