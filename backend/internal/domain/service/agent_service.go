package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// AgentService provides business logic for agent operations.
type AgentService struct {
	repo port.AgentRepository
}

// NewAgentService creates a new AgentService.
func NewAgentService(repo port.AgentRepository) *AgentService {
	return &AgentService{repo: repo}
}

// CreateAgentParams holds parameters for creating an agent.
type CreateAgentParams struct {
	ProjectID       *uuid.UUID
	Name            string
	Model           string
	Image           string
	TemplateContent string
	Scope           string
	Provider        string
}

// Create validates inputs and creates a new agent.
func (s *AgentService) Create(ctx context.Context, params CreateAgentParams) (*model.Agent, error) {
	if params.Name == "" {
		return nil, errors.NewValidation("name", "is required")
	}
	if len(params.Name) > model.MaxNameLength {
		return nil, errors.NewValidation("name", "must be 255 characters or less")
	}
	if params.TemplateContent == "" {
		return nil, errors.NewValidation("template_content", "is required")
	}

	scope := params.Scope
	if scope == "" {
		scope = model.AgentScopeProject
	}
	if scope != model.AgentScopeGlobal && scope != model.AgentScopeProject {
		return nil, errors.NewValidation("scope", "must be 'global' or 'project'")
	}

	if scope == model.AgentScopeProject && (params.ProjectID == nil || *params.ProjectID == uuid.Nil) {
		return nil, errors.NewValidation("project_id", "is required for project-scoped agents")
	}

	provider := params.Provider
	if provider == "" {
		provider = model.ProviderClaude
	}
	if provider != model.ProviderClaude && provider != model.ProviderOpenCode {
		return nil, errors.NewValidation("provider", "must be 'claude' or 'opencode'")
	}

	agent := &model.Agent{
		ID:              uuid.New(),
		Name:            params.Name,
		Model:           params.Model,
		Image:           params.Image,
		TemplateContent: params.TemplateContent,
		Scope:           scope,
		Provider:        provider,
		ProjectID:       params.ProjectID,
	}

	return s.repo.CreateAgent(ctx, agent)
}

// GetByID retrieves an agent by ID.
func (s *AgentService) GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	return s.repo.GetAgent(ctx, id)
}

// AgentListResult holds the result of a list operation.
type AgentListResult struct {
	Agents []*model.Agent
	Total  int
}

// ListByProject retrieves agents scoped to a project.
func (s *AgentService) ListByProject(ctx context.Context, projectID uuid.UUID) (*AgentListResult, error) {
	agents, err := s.repo.ListAgentsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &AgentListResult{Agents: agents, Total: len(agents)}, nil
}

// ListGlobal retrieves all global agents.
func (s *AgentService) ListGlobal(ctx context.Context) (*AgentListResult, error) {
	agents, err := s.repo.ListGlobalAgents(ctx)
	if err != nil {
		return nil, err
	}
	return &AgentListResult{Agents: agents, Total: len(agents)}, nil
}

// ListMerged retrieves project + global agents for a project.
func (s *AgentService) ListMerged(ctx context.Context, projectID uuid.UUID) (*AgentListResult, error) {
	agents, err := s.repo.ListAgentsByProjectMerged(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &AgentListResult{Agents: agents, Total: len(agents)}, nil
}

// UpdateAgentParams holds parameters for updating an agent.
type UpdateAgentParams struct {
	ID              uuid.UUID
	Name            *string
	Model           *string
	Image           *string
	TemplateContent *string
	Provider        *string
}

// Update validates inputs and updates an existing agent.
func (s *AgentService) Update(ctx context.Context, params UpdateAgentParams) (*model.Agent, error) {
	existing, err := s.repo.GetAgent(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if params.Name != nil {
		if *params.Name == "" {
			return nil, errors.NewValidation("name", "must not be empty")
		}
		if len(*params.Name) > model.MaxNameLength {
			return nil, errors.NewValidation("name", "must be 255 characters or less")
		}
		existing.Name = *params.Name
	}
	if params.Model != nil {
		existing.Model = *params.Model
	}
	if params.Image != nil {
		existing.Image = *params.Image
	}
	if params.TemplateContent != nil {
		if *params.TemplateContent == "" {
			return nil, errors.NewValidation("template_content", "must not be empty")
		}
		existing.TemplateContent = *params.TemplateContent
	}
	if params.Provider != nil {
		p := *params.Provider
		if p != model.ProviderClaude && p != model.ProviderOpenCode {
			return nil, errors.NewValidation("provider", "must be 'claude' or 'opencode'")
		}
		existing.Provider = p
	}

	return s.repo.UpdateAgent(ctx, existing)
}

// Delete removes an agent by ID.
func (s *AgentService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetAgent(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.DeleteAgent(ctx, id)
}
