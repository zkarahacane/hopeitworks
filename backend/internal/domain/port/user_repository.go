package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// UserRepository defines the persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	List(ctx context.Context, limit, offset int32) ([]*model.User, error)
	Count(ctx context.Context) (int64, error)
	Update(ctx context.Context, user *model.User) (*model.User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error
	Delete(ctx context.Context, id uuid.UUID) error
}
