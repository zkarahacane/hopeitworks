package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// ContainerTokenStore manages short-lived bearer tokens for agent containers.
type ContainerTokenStore interface {
	// Create generates and stores a new token for a run step bound to an agent.
	// agentID may be uuid.Nil when no agent is bound (the bundle then resolves empty).
	// Returns the token string.
	Create(ctx context.Context, runID, stepID, agentID uuid.UUID, ttl time.Duration) (string, error)

	// Validate checks if a token is valid and returns the associated container token.
	// Returns an error if the token is invalid or expired.
	Validate(ctx context.Context, token string) (*model.ContainerToken, error)

	// Revoke invalidates a token.
	Revoke(ctx context.Context, token string) error
}
