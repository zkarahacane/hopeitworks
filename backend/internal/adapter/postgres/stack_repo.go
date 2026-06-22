package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure StackRepo implements port.StackRepository at compile time.
var _ port.StackRepository = (*StackRepo)(nil)

// StackRepo implements port.StackRepository using sqlc-generated queries.
type StackRepo struct {
	queries *Queries
}

// NewStackRepo creates a new StackRepo.
func NewStackRepo(queries *Queries) *StackRepo {
	return &StackRepo{queries: queries}
}

// List returns all catalogued stacks ordered by key.
func (r *StackRepo) List(ctx context.Context) ([]*model.Stack, error) {
	rows, err := r.queries.ListStacks(ctx)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list stacks", err)
	}
	stacks := make([]*model.Stack, len(rows))
	for i, row := range rows {
		stacks[i] = toDomainStack(row)
	}
	return stacks, nil
}

// GetByID returns a stack by its id.
func (r *StackRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Stack, error) {
	row, err := r.queries.GetStack(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("stack", id)
		}
		return nil, apperrors.NewInternal("failed to get stack", err)
	}
	return toDomainStack(row), nil
}

// GetByKey returns a stack by its unique key.
func (r *StackRepo) GetByKey(ctx context.Context, key string) (*model.Stack, error) {
	row, err := r.queries.GetStackByKey(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("stack", key)
		}
		return nil, apperrors.NewInternal("failed to get stack by key", err)
	}
	return toDomainStack(row), nil
}

// toDomainStack maps a sqlc-generated Stack to the domain model.
func toDomainStack(s Stack) *model.Stack {
	return &model.Stack{
		ID:        s.ID,
		Key:       s.Key,
		ImageRef:  s.ImageRef,
		Toolchain: s.Toolchain,
		CreatedAt: s.CreatedAt,
	}
}
