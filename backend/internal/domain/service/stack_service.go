package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// StackService provides read access to the stack catalogue. Stacks are seeded
// platform reference data; the API exposes reads so an agent editor can pick a
// catalogued stack instead of typing a free-form image string.
type StackService struct {
	repo port.StackRepository
}

// NewStackService creates a new StackService.
func NewStackService(repo port.StackRepository) *StackService {
	return &StackService{repo: repo}
}

// StackListResult holds the result of a list operation.
type StackListResult struct {
	Stacks []*model.Stack
	Total  int
}

// List returns the full stack catalogue.
func (s *StackService) List(ctx context.Context) (*StackListResult, error) {
	stacks, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return &StackListResult{Stacks: stacks, Total: len(stacks)}, nil
}

// GetByID returns a single stack by id.
func (s *StackService) GetByID(ctx context.Context, id uuid.UUID) (*model.Stack, error) {
	return s.repo.GetByID(ctx, id)
}
