package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// ProjectService provides business logic for project operations.
type ProjectService struct {
	repo                  port.ProjectRepository
	pipelineConfigService *PipelineConfigService
}

// NewProjectService creates a new ProjectService.
func NewProjectService(repo port.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

// SetPipelineConfigService sets the pipeline config service for seeding on project creation.
func (s *ProjectService) SetPipelineConfigService(pcs *PipelineConfigService) {
	s.pipelineConfigService = pcs
}

// CreateProjectParams holds parameters for creating a project.
type CreateProjectParams struct {
	Name         string
	Description  *string
	OwnerID      *uuid.UUID
	RepoURL      *string
	GitProvider  *string
	GitTokenEnv  *string
	AgentRuntime *string
	DefaultModel *string
}

// Create validates inputs and creates a new project.
func (s *ProjectService) Create(ctx context.Context, params CreateProjectParams) (*model.Project, error) {
	if params.Name == "" {
		return nil, errors.NewValidation("name", "is required")
	}
	if len(params.Name) > 255 {
		return nil, errors.NewValidation("name", "must be 255 characters or less")
	}
	if params.Description != nil && len(*params.Description) > 1000 {
		return nil, errors.NewValidation("description", "must be 1000 characters or less")
	}

	project := &model.Project{
		Name:         params.Name,
		Description:  params.Description,
		OwnerID:      params.OwnerID,
		RepoURL:      params.RepoURL,
		GitProvider:  "github",
		AgentRuntime: "docker",
		GitTokenEnv:  params.GitTokenEnv,
		DefaultModel: params.DefaultModel,
	}
	if params.GitProvider != nil && *params.GitProvider != "" {
		project.GitProvider = *params.GitProvider
	}
	if params.AgentRuntime != nil && *params.AgentRuntime != "" {
		project.AgentRuntime = *params.AgentRuntime
	}

	created, err := s.repo.Create(ctx, project)
	if err != nil {
		return nil, err
	}

	// Seed default pipeline config for the new project
	if s.pipelineConfigService != nil {
		if _, err := s.pipelineConfigService.SeedDefault(ctx, created.ID); err != nil {
			return nil, fmt.Errorf("seeding default pipeline config: %w", err)
		}
	}

	return created, nil
}

// GetByID retrieves a project by ID.
func (s *ProjectService) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	return s.repo.GetByID(ctx, id)
}

// ListResult holds the result of a paginated list operation.
type ListResult struct {
	Projects []*model.Project
	Total    int64
}

// List retrieves a paginated list of projects.
func (s *ProjectService) List(ctx context.Context, page, perPage int) (*ListResult, error) {
	limit, offset := paginationToLimitOffset(page, perPage)

	projects, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, err
	}

	return &ListResult{
		Projects: projects,
		Total:    total,
	}, nil
}

// UpdateProjectParams holds parameters for updating a project.
type UpdateProjectParams struct {
	ID           uuid.UUID
	Name         *string
	Description  *string
	MaxBudget    *float64
	SetBudget    bool // true when max_budget field is explicitly set (allows setting to nil)
	RepoURL      *string
	SetRepoURL   bool // true when repo_url field is explicitly set (allows clearing)
	GitProvider  *string
	GitTokenEnv  *string
	SetTokenEnv  bool // true when git_token_env field is explicitly set (allows clearing)
	AgentRuntime *string
	DefaultModel *string
	SetModel     bool // true when default_model field is explicitly set (allows clearing)
}

// Update validates inputs and updates an existing project.
func (s *ProjectService) Update(ctx context.Context, params UpdateProjectParams) (*model.Project, error) {
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
		if len(*params.Description) > 1000 {
			return nil, errors.NewValidation("description", "must be 1000 characters or less")
		}
		existing.Description = params.Description
	}
	if params.SetBudget {
		existing.MaxBudget = params.MaxBudget
	}
	if params.GitProvider != nil {
		existing.GitProvider = *params.GitProvider
	}
	if params.AgentRuntime != nil {
		existing.AgentRuntime = *params.AgentRuntime
	}
	if params.SetRepoURL {
		existing.RepoURL = params.RepoURL
	}
	if params.SetTokenEnv {
		existing.GitTokenEnv = params.GitTokenEnv
	}
	if params.SetModel {
		existing.DefaultModel = params.DefaultModel
	}

	return s.repo.Update(ctx, existing)
}

// Delete removes a project by ID.
func (s *ProjectService) Delete(ctx context.Context, id uuid.UUID) error {
	// Verify the project exists before deleting
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
