package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// EpicService provides business logic for epic operations.
type EpicService struct {
	repo port.EpicRepository
}

// NewEpicService creates a new EpicService.
func NewEpicService(repo port.EpicRepository) *EpicService {
	return &EpicService{repo: repo}
}

// CreateEpicParams holds parameters for creating an epic.
type CreateEpicParams struct {
	ProjectID   uuid.UUID
	Name        string
	Description *string
	Status      string
}

// Create validates inputs and creates a new epic.
func (s *EpicService) Create(ctx context.Context, params CreateEpicParams) (*model.Epic, error) {
	if params.Name == "" {
		return nil, errors.NewValidation("name", "is required")
	}
	if len(params.Name) > 255 {
		return nil, errors.NewValidation("name", "must be 255 characters or less")
	}
	if params.Description != nil && len(*params.Description) > 2000 {
		return nil, errors.NewValidation("description", "must be 2000 characters or less")
	}
	if params.ProjectID == uuid.Nil {
		return nil, errors.NewValidation("project_id", "is required")
	}

	status := params.Status
	if status == "" {
		status = model.EpicStatusBacklog
	}
	if !isValidEpicStatus(status) {
		return nil, errors.NewValidation("status", "must be one of: backlog, in_progress, done")
	}

	epic := &model.Epic{
		ProjectID:   params.ProjectID,
		Name:        params.Name,
		Description: params.Description,
		Status:      status,
	}

	return s.repo.Create(ctx, epic)
}

// GetByID retrieves an epic by ID.
func (s *EpicService) GetByID(ctx context.Context, id uuid.UUID) (*model.Epic, error) {
	return s.repo.GetByID(ctx, id)
}

// EpicListResult holds the result of a paginated list operation.
type EpicListResult struct {
	Epics []*model.Epic
	Total int64
}

// ListByProject retrieves a paginated list of epics for a project.
func (s *EpicService) ListByProject(ctx context.Context, projectID uuid.UUID, page, perPage int) (*EpicListResult, error) {
	limit, offset := paginationToLimitOffset(page, perPage)

	epics, err := s.repo.ListByProject(ctx, projectID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &EpicListResult{
		Epics: epics,
		Total: total,
	}, nil
}

// UpdateEpicParams holds parameters for updating an epic.
type UpdateEpicParams struct {
	ID          uuid.UUID
	Name        *string
	Description *string
	Status      *string
}

// Update validates inputs and updates an existing epic.
func (s *EpicService) Update(ctx context.Context, params UpdateEpicParams) (*model.Epic, error) {
	existing, err := s.repo.GetByID(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if params.Name != nil {
		if *params.Name == "" {
			return nil, errors.NewValidation("name", "must not be empty")
		}
		if len(*params.Name) > 255 {
			return nil, errors.NewValidation("name", "must be 255 characters or less")
		}
		existing.Name = *params.Name
	}
	if params.Description != nil {
		if len(*params.Description) > 2000 {
			return nil, errors.NewValidation("description", "must be 2000 characters or less")
		}
		existing.Description = params.Description
	}
	if params.Status != nil {
		if !isValidEpicStatus(*params.Status) {
			return nil, errors.NewValidation("status", "must be one of: backlog, in_progress, done")
		}
		existing.Status = *params.Status
	}

	return s.repo.Update(ctx, existing)
}

// Delete removes an epic by ID.
func (s *EpicService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func isValidEpicStatus(status string) bool {
	switch status {
	case model.EpicStatusBacklog, model.EpicStatusInProgress, model.EpicStatusDone:
		return true
	default:
		return false
	}
}
