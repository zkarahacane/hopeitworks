package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// envTestMsg is a shared format string for unexpected errors.
const (
	envTestMsg      = "unexpected error"
	envResourceName = "environment"
)

// mockEnvironmentRepo is a hand-written mock of port.EnvironmentRepository.
type mockEnvironmentRepo struct {
	envs      map[uuid.UUID]*model.Environment // keyed by environment ID
	byProject map[uuid.UUID]*model.Environment // keyed by projectID
}

func newMockEnvironmentRepo() *mockEnvironmentRepo {
	return &mockEnvironmentRepo{
		envs:      make(map[uuid.UUID]*model.Environment),
		byProject: make(map[uuid.UUID]*model.Environment),
	}
}

func (m *mockEnvironmentRepo) Create(_ context.Context, e *model.Environment) (*model.Environment, error) {
	if _, exists := m.byProject[e.ProjectID]; exists {
		return nil, errors.NewConflict(envResourceName, e.ProjectID.String())
	}
	e.ID = uuid.New()
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()
	m.envs[e.ID] = e
	m.byProject[e.ProjectID] = e
	return e, nil
}

func (m *mockEnvironmentRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Environment, error) {
	e, ok := m.envs[id]
	if !ok {
		return nil, errors.NewNotFound(envResourceName, id)
	}
	return e, nil
}

func (m *mockEnvironmentRepo) GetByProjectID(_ context.Context, projectID uuid.UUID) (*model.Environment, error) {
	e, ok := m.byProject[projectID]
	if !ok {
		return nil, errors.NewNotFound(envResourceName, projectID)
	}
	return e, nil
}

func (m *mockEnvironmentRepo) Update(_ context.Context, e *model.Environment) (*model.Environment, error) {
	e.UpdatedAt = time.Now()
	m.envs[e.ID] = e
	m.byProject[e.ProjectID] = e
	return e, nil
}

func (m *mockEnvironmentRepo) Delete(_ context.Context, id uuid.UUID) error {
	e, ok := m.envs[id]
	if !ok {
		return errors.NewNotFound(envResourceName, id)
	}
	delete(m.byProject, e.ProjectID)
	delete(m.envs, id)
	return nil
}

func TestEnvironmentService_Upsert_Creates(t *testing.T) {
	repo := newMockEnvironmentRepo()
	svc := NewEnvironmentService(repo)
	projectID := uuid.New()

	input := UpsertEnvironmentInput{
		Stacks:   []string{model.StackKeyGo},
		Services: []model.EnvironmentService{},
		Source:   model.EnvironmentSourceDeclared,
		Commands: map[string]string{"test": "make test"},
	}

	env, err := svc.Upsert(context.Background(), projectID, input)
	if err != nil {
		t.Fatalf("%s: %v", envTestMsg, err)
	}
	if env.ProjectID != projectID {
		t.Errorf("expected project_id %v, got %v", projectID, env.ProjectID)
	}
	if env.Source != model.EnvironmentSourceDeclared {
		t.Errorf("expected source 'declared', got %q", env.Source)
	}
	if len(env.Stacks) != 1 || env.Stacks[0] != model.StackKeyGo {
		t.Errorf("expected stacks=[go], got %v", env.Stacks)
	}
}

func TestEnvironmentService_Upsert_Updates(t *testing.T) {
	repo := newMockEnvironmentRepo()
	svc := NewEnvironmentService(repo)
	projectID := uuid.New()

	// First create
	_, err := svc.Upsert(context.Background(), projectID, UpsertEnvironmentInput{
		Stacks:   []string{model.StackKeyGo},
		Services: []model.EnvironmentService{},
		Source:   model.EnvironmentSourceDeclared,
		Commands: map[string]string{},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Now update
	updated, err := svc.Upsert(context.Background(), projectID, UpsertEnvironmentInput{
		Stacks:   []string{model.StackKeyNode},
		Services: []model.EnvironmentService{{Name: "db", Image: "postgres:16", Env: map[string]string{}}},
		Source:   model.EnvironmentSourceCompose,
		Commands: map[string]string{"test": "npm test"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Source != model.EnvironmentSourceCompose {
		t.Errorf("expected source 'compose', got %q", updated.Source)
	}
	if len(updated.Stacks) != 1 || updated.Stacks[0] != model.StackKeyNode {
		t.Errorf("expected stacks=[node], got %v", updated.Stacks)
	}
	if len(updated.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(updated.Services))
	}
}

func TestEnvironmentService_Delete(t *testing.T) {
	repo := newMockEnvironmentRepo()
	svc := NewEnvironmentService(repo)
	projectID := uuid.New()

	// Create an environment first
	_, err := svc.Upsert(context.Background(), projectID, UpsertEnvironmentInput{
		Stacks:   []string{model.StackKeyPython},
		Services: []model.EnvironmentService{},
		Source:   model.EnvironmentSourceDeclared,
		Commands: map[string]string{},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Delete
	if err := svc.Delete(context.Background(), projectID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Should be gone
	_, err = svc.GetByProject(context.Background(), projectID)
	if err == nil {
		t.Fatal("expected NotFound after delete, got nil error")
	}
	domErr, ok := err.(*errors.DomainError)
	if !ok || domErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found, got %v", err)
	}
}

func TestEnvironmentService_Delete_NotFound(t *testing.T) {
	repo := newMockEnvironmentRepo()
	svc := NewEnvironmentService(repo)

	err := svc.Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent environment")
	}
	domErr, ok := err.(*errors.DomainError)
	if !ok || domErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found, got %v", err)
	}
}

func TestEnvironmentService_Validation_BadSource(t *testing.T) {
	repo := newMockEnvironmentRepo()
	svc := NewEnvironmentService(repo)

	_, err := svc.Upsert(context.Background(), uuid.New(), UpsertEnvironmentInput{
		Stacks:   []string{model.StackKeyGo},
		Services: []model.EnvironmentService{},
		Source:   "unknown-source",
		Commands: map[string]string{},
	})
	if err == nil {
		t.Fatal("expected validation error for unknown source")
	}
	domErr, ok := err.(*errors.DomainError)
	if !ok || domErr.Category != errors.CategoryValidation {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestEnvironmentService_Validation_BadStack(t *testing.T) {
	repo := newMockEnvironmentRepo()
	svc := NewEnvironmentService(repo)

	_, err := svc.Upsert(context.Background(), uuid.New(), UpsertEnvironmentInput{
		Stacks:   []string{"ruby"},
		Services: []model.EnvironmentService{},
		Source:   model.EnvironmentSourceDeclared,
		Commands: map[string]string{},
	})
	if err == nil {
		t.Fatal("expected validation error for unknown stack key")
	}
	domErr, ok := err.(*errors.DomainError)
	if !ok || domErr.Category != errors.CategoryValidation {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestEnvironmentService_EmptySource_DefaultsDeclared(t *testing.T) {
	repo := newMockEnvironmentRepo()
	svc := NewEnvironmentService(repo)

	env, err := svc.Upsert(context.Background(), uuid.New(), UpsertEnvironmentInput{
		Stacks:   []string{model.StackKeyGoNode},
		Services: []model.EnvironmentService{},
		Source:   "", // empty should default to "declared"
		Commands: map[string]string{},
	})
	if err != nil {
		t.Fatalf("%s: %v", envTestMsg, err)
	}
	if env.Source != model.EnvironmentSourceDeclared {
		t.Errorf("expected source 'declared' (default), got %q", env.Source)
	}
}
