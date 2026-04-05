package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/receipt-manager/backend/internal/model"
	"github.com/receipt-manager/backend/internal/repository"
)

// Sentinel errors for the tag service
var (
	ErrTagNotFound     = errors.New("tag not found")
	ErrInvalidTagName  = errors.New("tag name is required")
	ErrInvalidTagColor = errors.New("tag color must be a valid hex color (e.g., #6366f1)")
)

// hexColorRegex matches hex color codes like #6366f1 or #FFF
var hexColorRegex = regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)

// TagService provides tag business logic
type TagService struct {
	tagRepo *repository.TagRepo
}

// NewTagService creates a new tag service
func NewTagService(tagRepo *repository.TagRepo) *TagService {
	return &TagService{
		tagRepo: tagRepo,
	}
}

// CreateRequest represents a request to create a tag
type CreateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// UpdateRequest represents a request to update a tag
type UpdateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Create creates a new tag for a user
func (s *TagService) Create(ctx context.Context, userID uuid.UUID, req *CreateTagRequest) (*model.Tag, error) {
	// Validate request
	if err := validateTagRequest(req.Name, req.Color); err != nil {
		return nil, err
	}

	// Create the tag
	tag := &model.Tag{
		UserID: userID,
		Name:   req.Name,
		Color:  req.Color,
	}

	createdTag, err := s.tagRepo.Create(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	return createdTag, nil
}

// List lists all tags for a user
func (s *TagService) List(ctx context.Context, userID uuid.UUID) ([]*model.Tag, error) {
	tags, err := s.tagRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	return tags, nil
}

// GetByID retrieves a tag by ID with ownership check
func (s *TagService) GetByID(ctx context.Context, tagID uuid.UUID, userID uuid.UUID) (*model.Tag, error) {
	tag, err := s.tagRepo.GetByID(ctx, tagID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}
	if tag == nil {
		return nil, ErrTagNotFound
	}

	// Check ownership
	if tag.UserID != userID {
		return nil, ErrTagNotFound
	}

	return tag, nil
}

// Update updates a tag with ownership check
func (s *TagService) Update(ctx context.Context, tagID uuid.UUID, userID uuid.UUID, req *UpdateTagRequest) (*model.Tag, error) {
	// Verify ownership first
	existing, err := s.GetByID(ctx, tagID, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrTagNotFound
	}

	// Validate request
	if err := validateTagRequest(req.Name, req.Color); err != nil {
		return nil, err
	}

	// Update fields
	existing.Name = req.Name
	existing.Color = req.Color

	// Save the tag
	updated, err := s.tagRepo.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}

	return updated, nil
}

// Delete deletes a tag with ownership check
func (s *TagService) Delete(ctx context.Context, tagID uuid.UUID, userID uuid.UUID) error {
	// Verify ownership first
	tag, err := s.tagRepo.GetByID(ctx, tagID)
	if err != nil {
		return fmt.Errorf("failed to get tag: %w", err)
	}
	if tag == nil {
		return ErrTagNotFound
	}
	if tag.UserID != userID {
		return ErrTagNotFound
	}

	return s.tagRepo.Delete(ctx, tagID, userID)
}

// validateTagRequest validates tag name and color
func validateTagRequest(name, color string) error {
	if name == "" {
		return ErrInvalidTagName
	}

	if color == "" {
		return ErrInvalidTagColor
	}

	if !hexColorRegex.MatchString(color) {
		return ErrInvalidTagColor
	}

	return nil
}
