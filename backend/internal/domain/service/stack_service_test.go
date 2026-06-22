package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockStackRepo is a hand-written port.StackRepository for service/run tests.
type mockStackRepo struct {
	stacks    map[uuid.UUID]*model.Stack
	listFn    func() ([]*model.Stack, error)
	upsertFn  func(s *model.Stack) (*model.Stack, error)
	upsertLog []model.Stack
}

func newMockStackRepo() *mockStackRepo {
	return &mockStackRepo{stacks: make(map[uuid.UUID]*model.Stack)}
}

func (m *mockStackRepo) List(_ context.Context) ([]*model.Stack, error) {
	if m.listFn != nil {
		return m.listFn()
	}
	out := make([]*model.Stack, 0, len(m.stacks))
	for _, s := range m.stacks {
		out = append(out, s)
	}
	return out, nil
}

func (m *mockStackRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Stack, error) {
	if s, ok := m.stacks[id]; ok {
		return s, nil
	}
	return nil, errors.NewNotFound("stack", id)
}

func (m *mockStackRepo) GetByKey(_ context.Context, key string) (*model.Stack, error) {
	for _, s := range m.stacks {
		if s.Key == key {
			return s, nil
		}
	}
	return nil, errors.NewNotFound("stack", key)
}

func (m *mockStackRepo) Upsert(_ context.Context, s *model.Stack) (*model.Stack, error) {
	m.upsertLog = append(m.upsertLog, *s)
	if m.upsertFn != nil {
		return m.upsertFn(s)
	}
	out := *s
	if out.ID == uuid.Nil {
		out.ID = uuid.New()
	}
	return &out, nil
}

func TestStackService_List(t *testing.T) {
	repo := newMockStackRepo()
	id := uuid.New()
	repo.stacks[id] = &model.Stack{ID: id, Key: model.StackKeyGo, ImageRef: "ghcr.io/x/agent-go:latest"}

	svc := NewStackService(repo)
	result, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Total != 1 || len(result.Stacks) != 1 {
		t.Fatalf("expected 1 stack, got total=%d len=%d", result.Total, len(result.Stacks))
	}
	if result.Stacks[0].Key != model.StackKeyGo {
		t.Errorf("expected key %q, got %q", model.StackKeyGo, result.Stacks[0].Key)
	}
}

func TestStackService_GetByID(t *testing.T) {
	repo := newMockStackRepo()
	id := uuid.New()
	repo.stacks[id] = &model.Stack{ID: id, Key: model.StackKeyNode, ImageRef: "ghcr.io/x/agent-node:latest"}

	svc := NewStackService(repo)
	got, err := svc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ImageRef != "ghcr.io/x/agent-node:latest" {
		t.Errorf("unexpected image_ref %q", got.ImageRef)
	}

	if _, err := svc.GetByID(context.Background(), uuid.New()); err == nil {
		t.Error("expected not-found error for unknown id")
	}
}
