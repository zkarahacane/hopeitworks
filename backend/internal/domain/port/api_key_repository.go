package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// APIKeyRepository defines the persistence interface for user API keys.
type APIKeyRepository interface {
	Create(ctx context.Context, key *model.UserAPIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.UserAPIKey, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.UserAPIKey, error)
	GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*model.UserAPIKey, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
