package service

import (
	"context"
	"regexp"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// storyKeyPattern validates story key format: uppercase letters/digits, dash, one or more digits.
var storyKeyPattern = regexp.MustCompile(`^[A-Z0-9]+-\d+$`)

// StoryService provides business logic for story operations.
type StoryService struct {
	repo port.StoryRepository
}

// NewStoryService creates a new StoryService.
func NewStoryService(repo port.StoryRepository) *StoryService {
	return &StoryService{repo: repo}
}

// CreateStoryParams holds parameters for creating a story.
type CreateStoryParams struct {
	ProjectID          uuid.UUID
	EpicID             *uuid.UUID
	Key                string
	Title              string
	Objective          *string
	TargetFiles        []string
	DependsOn          []string
	Scope              *string
	Status             string
	AcceptanceCriteria *string
}

// Create validates inputs and creates a new story.
func (s *StoryService) Create(ctx context.Context, params CreateStoryParams) (*model.Story, error) {
	if params.Key == "" {
		return nil, errors.NewValidation("key", "is required")
	}
	if len(params.Key) > 50 {
		return nil, errors.NewValidation("key", "must be 50 characters or less")
	}
	if !storyKeyPattern.MatchString(params.Key) {
		return nil, errors.NewValidation("key", "must match format [A-Z0-9]+-N (e.g., S-14, STORY-123)")
	}
	if params.Title == "" {
		return nil, errors.NewValidation("title", "is required")
	}
	if len(params.Title) > 255 {
		return nil, errors.NewValidation("title", "must be 255 characters or less")
	}
	if params.ProjectID == uuid.Nil {
		return nil, errors.NewValidation("project_id", "is required")
	}

	status := params.Status
	if status == "" {
		status = model.StoryStatusBacklog
	}
	if !isValidStoryStatus(status) {
		return nil, errors.NewValidation("status", "must be one of: backlog, running, done, failed")
	}

	if params.Scope != nil && !isValidStoryScope(*params.Scope) {
		return nil, errors.NewValidation("scope", "must be one of: backend, frontend, shared")
	}

	story := &model.Story{
		ProjectID:          params.ProjectID,
		EpicID:             params.EpicID,
		Key:                params.Key,
		Title:              params.Title,
		Objective:          params.Objective,
		TargetFiles:        params.TargetFiles,
		DependsOn:          params.DependsOn,
		Scope:              params.Scope,
		Status:             status,
		AcceptanceCriteria: params.AcceptanceCriteria,
	}

	return s.repo.Create(ctx, story)
}

// GetByID retrieves a story by ID.
func (s *StoryService) GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByKey retrieves a story by project ID and key.
func (s *StoryService) GetByKey(ctx context.Context, projectID uuid.UUID, key string) (*model.Story, error) {
	return s.repo.GetByKey(ctx, projectID, key)
}

// StoryListResult holds the result of a paginated list operation.
type StoryListResult struct {
	Stories []*model.Story
	Total   int64
}

// ListByProject retrieves a paginated list of stories for a project.
func (s *StoryService) ListByProject(ctx context.Context, projectID uuid.UUID, page, perPage int) (*StoryListResult, error) {
	limit, offset := paginationToLimitOffset(page, perPage)

	stories, err := s.repo.ListByProject(ctx, projectID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &StoryListResult{
		Stories: stories,
		Total:   total,
	}, nil
}

// ListByStatus retrieves a paginated list of stories filtered by status.
func (s *StoryService) ListByStatus(ctx context.Context, projectID uuid.UUID, statuses []string, page, perPage int) (*StoryListResult, error) {
	limit, offset := paginationToLimitOffset(page, perPage)

	stories, err := s.repo.ListByStatus(ctx, projectID, statuses, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountByStatus(ctx, projectID, statuses)
	if err != nil {
		return nil, err
	}

	return &StoryListResult{
		Stories: stories,
		Total:   total,
	}, nil
}

// UpdateStoryParams holds parameters for updating a story.
type UpdateStoryParams struct {
	ID                 uuid.UUID
	Title              *string
	Objective          *string
	TargetFiles        *[]string
	DependsOn          *[]string
	Scope              *string
	Status             *string
	AcceptanceCriteria *string
	EpicID             *uuid.UUID
}

// Update validates inputs and updates an existing story.
func (s *StoryService) Update(ctx context.Context, params UpdateStoryParams) (*model.Story, error) {
	existing, err := s.repo.GetByID(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if params.Title != nil {
		if *params.Title == "" {
			return nil, errors.NewValidation("title", "must not be empty")
		}
		if len(*params.Title) > 255 {
			return nil, errors.NewValidation("title", "must be 255 characters or less")
		}
		existing.Title = *params.Title
	}
	if params.Objective != nil {
		existing.Objective = params.Objective
	}
	if params.TargetFiles != nil {
		existing.TargetFiles = *params.TargetFiles
	}
	if params.DependsOn != nil {
		existing.DependsOn = *params.DependsOn
	}
	if params.Scope != nil {
		if *params.Scope != "" && !isValidStoryScope(*params.Scope) {
			return nil, errors.NewValidation("scope", "must be one of: backend, frontend, shared")
		}
		existing.Scope = params.Scope
	}
	if params.Status != nil {
		if !isValidStoryStatus(*params.Status) {
			return nil, errors.NewValidation("status", "must be one of: backlog, running, done, failed")
		}
		existing.Status = *params.Status
	}
	if params.AcceptanceCriteria != nil {
		existing.AcceptanceCriteria = params.AcceptanceCriteria
	}
	if params.EpicID != nil {
		existing.EpicID = params.EpicID
	}

	return s.repo.Update(ctx, existing)
}

// Delete removes a story by ID.
func (s *StoryService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func isValidStoryStatus(status string) bool {
	switch status {
	case model.StoryStatusBacklog, model.StoryStatusRunning, model.StoryStatusDone, model.StoryStatusFailed:
		return true
	default:
		return false
	}
}

func isValidStoryScope(scope string) bool {
	switch scope {
	case model.StoryScopeBackend, model.StoryScopeFrontend, model.StoryScopeShared:
		return true
	default:
		return false
	}
}

