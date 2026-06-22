package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// StackRepository provides access to the stack catalogue: the curated,
// platform-owned set of runtime images an agent can reference instead of a
// free-form image string. Reads back the API (P2a, read-only). Upsert backs the
// idempotent boot-time seeder that re-applies the versioned config catalogue,
// making that config the source of truth for image_ref/toolchain.
type StackRepository interface {
	// List returns all catalogued stacks ordered by key.
	List(ctx context.Context) ([]*model.Stack, error)
	// GetByID returns a stack by its id.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Stack, error)
	// GetByKey returns a stack by its unique key (go | node | python | go-node).
	GetByKey(ctx context.Context, key string) (*model.Stack, error)
	// Upsert inserts a stack or, on key conflict, updates its image_ref and
	// toolchain. Idempotent: safe to re-apply on every boot from the config
	// catalogue. Returns the persisted row.
	Upsert(ctx context.Context, s *model.Stack) (*model.Stack, error)
}
