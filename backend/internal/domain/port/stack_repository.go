package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// StackRepository provides read access to the stack catalogue: the curated,
// platform-owned set of runtime images an agent can reference instead of a
// free-form image string. The catalogue is seeded via migration; P2a exposes
// reads only (no create/update from the API).
type StackRepository interface {
	// List returns all catalogued stacks ordered by key.
	List(ctx context.Context) ([]*model.Stack, error)
	// GetByID returns a stack by its id.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Stack, error)
	// GetByKey returns a stack by its unique key (go | node | python | go-node).
	GetByKey(ctx context.Context, key string) (*model.Stack, error)
}
