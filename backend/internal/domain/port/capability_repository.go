package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// CapabilityRepository persists runtime-agnostic capabilities and their composition
// onto agents (the agent_capabilities join). It is the data source the BundleService
// reads when assembling an agent's RuntimeBundle.
type CapabilityRepository interface {
	Create(ctx context.Context, c *model.Capability) (*model.Capability, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Capability, error)
	// ListByScope returns all global capabilities plus, when projectID is non-nil,
	// the capabilities owned by that project.
	ListByScope(ctx context.Context, projectID *uuid.UUID) ([]*model.Capability, error)
	Delete(ctx context.Context, id uuid.UUID) error

	AttachToAgent(ctx context.Context, agentID, capabilityID uuid.UUID) error
	DetachFromAgent(ctx context.Context, agentID, capabilityID uuid.UUID) error
	// ListForAgent returns the capabilities composed onto an agent.
	ListForAgent(ctx context.Context, agentID uuid.UUID) ([]*model.Capability, error)
}
