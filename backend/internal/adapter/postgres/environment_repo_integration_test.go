package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// assertCategory fails the test unless err is a *DomainError of the given category.
func assertCategory(t *testing.T, err error, want apperrors.ErrorCategory) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error of category %q, got nil", want)
	}
	var de *apperrors.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("expected *DomainError, got %T: %v", err, err)
	}
	if de.Category != want {
		t.Fatalf("expected category %q, got %q (%v)", want, de.Category, err)
	}
}

func newTestEnvironment(projectID uuid.UUID) *model.Environment {
	return &model.Environment{
		ProjectID: projectID,
		Stacks:    []string{model.StackKeyGo, model.StackKeyNode},
		Services: []model.EnvironmentService{
			{
				Name:  "postgres",
				Image: "postgres:16-alpine",
				Env:   map[string]string{"POSTGRES_PASSWORD": "test"},
			},
			{
				Name:  "redis",
				Image: "redis:7-alpine",
				Env:   map[string]string{},
			},
		},
		Source:   model.EnvironmentSourceDeclared,
		Commands: map[string]string{"test": "make test", "migrate": "make migrate"},
	}
}

func TestIntegration_EnvironmentRepo_CreateAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	repo := postgres.NewEnvironmentRepo(postgres.New(db.pool))
	projectID := createTestProject(t, db.pool)

	created, err := repo.Create(ctx, newTestEnvironment(projectID))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("expected generated id")
	}
	if created.ProjectID != projectID {
		t.Errorf("expected project id %v, got %v", projectID, created.ProjectID)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Error("expected created_at and updated_at to be set")
	}

	// Round-trip TEXT[] (stacks) and JSONB (services + commands) via GetByID.
	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Stacks) != 2 || got.Stacks[0] != model.StackKeyGo || got.Stacks[1] != model.StackKeyNode {
		t.Errorf("stacks round-trip mismatch: %v", got.Stacks)
	}
	if len(got.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(got.Services))
	}
	if got.Services[0].Name != "postgres" || got.Services[0].Image != "postgres:16-alpine" {
		t.Errorf("service[0] round-trip mismatch: %+v", got.Services[0])
	}
	if got.Services[0].Env["POSTGRES_PASSWORD"] != "test" {
		t.Errorf("service[0] env round-trip mismatch: %v", got.Services[0].Env)
	}
	if got.Source != model.EnvironmentSourceDeclared {
		t.Errorf("expected source %q, got %q", model.EnvironmentSourceDeclared, got.Source)
	}
	if got.Commands["test"] != "make test" || got.Commands["migrate"] != "make migrate" {
		t.Errorf("commands round-trip mismatch: %v", got.Commands)
	}
}

func TestIntegration_EnvironmentRepo_GetByProjectID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	repo := postgres.NewEnvironmentRepo(postgres.New(db.pool))
	projectID := createTestProject(t, db.pool)

	// NotFound before any environment exists.
	if _, err := repo.GetByProjectID(ctx, projectID); err == nil {
		t.Fatal("expected NotFound for project without environment")
	} else {
		assertCategory(t, err, apperrors.CategoryNotFound)
	}

	created, err := repo.Create(ctx, newTestEnvironment(projectID))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByProjectID(ctx, projectID)
	if err != nil {
		t.Fatalf("GetByProjectID: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected environment id %v, got %v", created.ID, got.ID)
	}

	// Unknown project id → NotFound.
	if _, err := repo.GetByProjectID(ctx, uuid.New()); err == nil {
		t.Fatal("expected NotFound for unknown project id")
	} else {
		assertCategory(t, err, apperrors.CategoryNotFound)
	}
}

func TestIntegration_EnvironmentRepo_UniqueProjectConstraint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	repo := postgres.NewEnvironmentRepo(postgres.New(db.pool))
	projectID := createTestProject(t, db.pool)

	if _, err := repo.Create(ctx, newTestEnvironment(projectID)); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	// One environment per project: a second Create on the same project → Conflict.
	_, err := repo.Create(ctx, newTestEnvironment(projectID))
	assertCategory(t, err, apperrors.CategoryConflict)
}

func TestIntegration_EnvironmentRepo_CreateUnknownProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	repo := postgres.NewEnvironmentRepo(postgres.New(db.pool))

	// FK violation on a non-existent project → NotFound("project").
	_, err := repo.Create(ctx, newTestEnvironment(uuid.New()))
	assertCategory(t, err, apperrors.CategoryNotFound)
}

func TestIntegration_EnvironmentRepo_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	repo := postgres.NewEnvironmentRepo(postgres.New(db.pool))
	projectID := createTestProject(t, db.pool)

	created, err := repo.Create(ctx, newTestEnvironment(projectID))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	created.Stacks = []string{model.StackKeyPython}
	created.Services = []model.EnvironmentService{
		{Name: "mailhog", Image: "mailhog/mailhog:latest", Env: map[string]string{}},
	}
	created.Source = model.EnvironmentSourceCompose
	created.Commands = map[string]string{"seed": "make seed"}

	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(updated.Stacks) != 1 || updated.Stacks[0] != model.StackKeyPython {
		t.Errorf("stacks not updated: %v", updated.Stacks)
	}
	if len(updated.Services) != 1 || updated.Services[0].Name != "mailhog" {
		t.Errorf("services not updated: %+v", updated.Services)
	}
	if updated.Source != model.EnvironmentSourceCompose {
		t.Errorf("source not updated: %q", updated.Source)
	}
	if updated.Commands["seed"] != "make seed" || len(updated.Commands) != 1 {
		t.Errorf("commands not updated: %v", updated.Commands)
	}

	// Persisted: re-read confirms the update.
	reread, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if len(reread.Stacks) != 1 || reread.Stacks[0] != model.StackKeyPython {
		t.Errorf("reread stacks mismatch: %v", reread.Stacks)
	}

	// Updating a non-existent environment → NotFound.
	created.ID = uuid.New()
	if _, err := repo.Update(ctx, created); err == nil {
		t.Fatal("expected NotFound updating unknown environment")
	} else {
		assertCategory(t, err, apperrors.CategoryNotFound)
	}
}

func TestIntegration_EnvironmentRepo_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	repo := postgres.NewEnvironmentRepo(postgres.New(db.pool))
	projectID := createTestProject(t, db.pool)

	created, err := repo.Create(ctx, newTestEnvironment(projectID))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := repo.GetByID(ctx, created.ID); err == nil {
		t.Fatal("expected NotFound after delete")
	} else {
		assertCategory(t, err, apperrors.CategoryNotFound)
	}

	// Deleting an already-absent environment is a no-op (DELETE affects zero rows).
	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Errorf("Delete of absent environment should be no-op, got: %v", err)
	}
}
