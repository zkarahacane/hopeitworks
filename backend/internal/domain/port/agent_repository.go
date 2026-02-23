package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// AgentRepository defines persistence operations for Agent entities.
type AgentRepository interface {
	CreateAgent(ctx context.Context, agent *model.Agent) (*model.Agent, error)
	GetAgent(ctx context.Context, id uuid.UUID) (*model.Agent, error)
	ListAgentsByProject(ctx context.Context, projectID uuid.UUID) ([]*model.Agent, error)
	// ListGlobalAgents returns all agents with scope = "global".
	ListGlobalAgents(ctx context.Context) ([]*model.Agent, error)
	// ListAgentsByProjectMerged returns all agents scoped to projectID plus all global agents.
	ListAgentsByProjectMerged(ctx context.Context, projectID uuid.UUID) ([]*model.Agent, error)
	UpdateAgent(ctx context.Context, agent *model.Agent) (*model.Agent, error)
	DeleteAgent(ctx context.Context, id uuid.UUID) error
}
