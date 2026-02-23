package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure AgentRepo implements port.AgentRepository at compile time.
var _ port.AgentRepository = (*AgentRepo)(nil)

// AgentRepo implements port.AgentRepository using sqlc-generated queries.
type AgentRepo struct {
	queries *Queries
}

// NewAgentRepo creates a new AgentRepo.
func NewAgentRepo(queries *Queries) *AgentRepo {
	return &AgentRepo{queries: queries}
}

// CreateAgent inserts a new agent.
func (r *AgentRepo) CreateAgent(ctx context.Context, agent *model.Agent) (*model.Agent, error) {
	params := CreateAgentParams{
		ID:              agent.ID,
		Name:            agent.Name,
		Model:           textFromString(agent.Model),
		Image:           textFromString(agent.Image),
		TemplateContent: agent.TemplateContent,
		Scope:           agent.Scope,
		ProjectID:       uuidFromPtr(agent.ProjectID),
	}

	if params.ID == uuid.Nil {
		params.ID = uuid.New()
	}

	row, err := r.queries.CreateAgent(ctx, params)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("agent", agent.Name)
		}
		return nil, apperrors.NewInternal("failed to create agent", err)
	}
	return toDomainAgent(row), nil
}

// GetAgent retrieves an agent by ID.
func (r *AgentRepo) GetAgent(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	row, err := r.queries.GetAgent(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("agent", id)
		}
		return nil, apperrors.NewInternal("failed to get agent", err)
	}
	return toDomainAgent(row), nil
}

// ListAgentsByProject retrieves agents scoped to a specific project.
func (r *AgentRepo) ListAgentsByProject(ctx context.Context, projectID uuid.UUID) ([]*model.Agent, error) {
	rows, err := r.queries.ListAgentsByProject(ctx, pgtype.UUID{Bytes: projectID, Valid: true})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list agents by project", err)
	}
	agents := make([]*model.Agent, len(rows))
	for i, row := range rows {
		agents[i] = toDomainAgent(row)
	}
	return agents, nil
}

// ListGlobalAgents returns all agents with scope = "global".
func (r *AgentRepo) ListGlobalAgents(ctx context.Context) ([]*model.Agent, error) {
	rows, err := r.queries.ListGlobalAgents(ctx)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list global agents", err)
	}
	agents := make([]*model.Agent, len(rows))
	for i, row := range rows {
		agents[i] = toDomainAgent(row)
	}
	return agents, nil
}

// ListAgentsByProjectMerged returns all agents scoped to projectID plus all global agents.
func (r *AgentRepo) ListAgentsByProjectMerged(ctx context.Context, projectID uuid.UUID) ([]*model.Agent, error) {
	rows, err := r.queries.ListAgentsByProjectMerged(ctx, pgtype.UUID{Bytes: projectID, Valid: true})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list merged agents", err)
	}
	agents := make([]*model.Agent, len(rows))
	for i, row := range rows {
		agents[i] = toDomainAgent(row)
	}
	return agents, nil
}

// UpdateAgent updates an existing agent.
func (r *AgentRepo) UpdateAgent(ctx context.Context, agent *model.Agent) (*model.Agent, error) {
	params := UpdateAgentParams{
		ID:              agent.ID,
		Name:            agent.Name,
		Model:           textFromString(agent.Model),
		Image:           textFromString(agent.Image),
		TemplateContent: agent.TemplateContent,
	}

	row, err := r.queries.UpdateAgent(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("agent", agent.ID)
		}
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("agent", agent.Name)
		}
		return nil, apperrors.NewInternal("failed to update agent", err)
	}
	return toDomainAgent(row), nil
}

// DeleteAgent removes an agent by ID.
func (r *AgentRepo) DeleteAgent(ctx context.Context, id uuid.UUID) error {
	err := r.queries.DeleteAgent(ctx, id)
	if err != nil {
		return apperrors.NewInternal("failed to delete agent", err)
	}
	return nil
}

// toDomainAgent maps a sqlc-generated Agent to a domain Agent.
func toDomainAgent(a Agent) *model.Agent {
	agent := &model.Agent{
		ID:              a.ID,
		Name:            a.Name,
		TemplateContent: a.TemplateContent,
		Scope:           a.Scope,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}

	if a.ProjectID.Valid {
		id := uuid.UUID(a.ProjectID.Bytes)
		agent.ProjectID = &id
	}

	if a.Model.Valid {
		agent.Model = a.Model.String
	}

	if a.Image.Valid {
		agent.Image = a.Image.String
	}

	return agent
}

// textFromString converts a Go string to pgtype.Text. Empty strings are stored as NULL.
func textFromString(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{String: "", Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}
