package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// UserRepository implements port.UserRepository using sqlc-generated Queries.
type UserRepository struct {
	q *Queries
}

var _ port.UserRepository = (*UserRepository)(nil)

func NewUserRepository(db DBTX) *UserRepository {
	return &UserRepository{q: New(db)}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	row, err := r.q.CreateUser(ctx, CreateUserParams{
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Name:         user.Name,
		Role:         string(user.Role),
	})
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *UserRepository) List(ctx context.Context, limit, offset int32) ([]*model.User, error) {
	rows, err := r.q.ListUsers(ctx, ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	users := make([]*model.User, len(rows))
	for i, row := range rows {
		users[i] = toDomainUser(row)
	}
	return users, nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountUsers(ctx)
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) (*model.User, error) {
	row, err := r.q.UpdateUser(ctx, UpdateUserParams{
		ID:    user.ID,
		Name:  pgtype.Text{String: user.Name, Valid: user.Name != ""},
		Email: pgtype.Text{String: user.Email, Valid: user.Email != ""},
		Role:  pgtype.Text{String: string(user.Role), Valid: user.Role != ""},
	})
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *UserRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	return r.q.UpdateUserPasswordHash(ctx, UpdateUserPasswordHashParams{
		ID:           id,
		PasswordHash: hash,
	})
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteUser(ctx, id)
}

func toDomainUser(u User) *model.User {
	var deletedAt *time.Time
	if u.DeletedAt.Valid {
		deletedAt = &u.DeletedAt.Time
	}
	return &model.User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		Role:         model.Role(u.Role),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		DeletedAt:    deletedAt,
	}
}
