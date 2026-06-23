package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

const envResource = "environment"

// mockEnvRepo is a hand-written mock of port.EnvironmentRepository for handler tests.
type mockEnvRepo struct {
	envs      map[uuid.UUID]*model.Environment
	byProject map[uuid.UUID]*model.Environment
}

var _ port.EnvironmentRepository = (*mockEnvRepo)(nil)

func newMockEnvRepo() *mockEnvRepo {
	return &mockEnvRepo{
		envs:      make(map[uuid.UUID]*model.Environment),
		byProject: make(map[uuid.UUID]*model.Environment),
	}
}

func (m *mockEnvRepo) Create(_ context.Context, e *model.Environment) (*model.Environment, error) {
	if _, exists := m.byProject[e.ProjectID]; exists {
		return nil, errors.NewConflict(envResource, e.ProjectID.String())
	}
	e.ID = uuid.New()
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()
	m.envs[e.ID] = e
	m.byProject[e.ProjectID] = e
	return e, nil
}

func (m *mockEnvRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Environment, error) {
	e, ok := m.envs[id]
	if !ok {
		return nil, errors.NewNotFound(envResource, id)
	}
	return e, nil
}

func (m *mockEnvRepo) GetByProjectID(_ context.Context, projectID uuid.UUID) (*model.Environment, error) {
	e, ok := m.byProject[projectID]
	if !ok {
		return nil, errors.NewNotFound(envResource, projectID)
	}
	return e, nil
}

func (m *mockEnvRepo) Update(_ context.Context, e *model.Environment) (*model.Environment, error) {
	e.UpdatedAt = time.Now()
	m.envs[e.ID] = e
	m.byProject[e.ProjectID] = e
	return e, nil
}

func (m *mockEnvRepo) Delete(_ context.Context, id uuid.UUID) error {
	e, ok := m.envs[id]
	if !ok {
		return errors.NewNotFound(envResource, id)
	}
	delete(m.byProject, e.ProjectID)
	delete(m.envs, id)
	return nil
}

func setupEnvironmentHandler() (*EnvironmentHandler, *mockEnvRepo) {
	repo := newMockEnvRepo()
	svc := service.NewEnvironmentService(repo)
	h := NewEnvironmentHandler(svc)
	return h, repo
}

func TestGetProjectEnvironment_Found(t *testing.T) {
	h, repo := setupEnvironmentHandler()
	projectID := uuid.New()

	env := &model.Environment{
		ID:        uuid.New(),
		ProjectID: projectID,
		Stacks:    []string{model.StackKeyGo},
		Services:  []model.EnvironmentService{},
		Source:    model.EnvironmentSourceDeclared,
		Commands:  map[string]string{"test": "make test"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.envs[env.ID] = env
	repo.byProject[projectID] = env

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/environment", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetProjectEnvironment(rec, req, projectID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Environment
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ProjectId != projectID {
		t.Errorf("expected project_id %v, got %v", projectID, resp.ProjectId)
	}
	if resp.Source != EnvironmentSourceDeclared {
		t.Errorf("expected source 'declared', got %q", resp.Source)
	}
}

func TestGetProjectEnvironment_NotFound(t *testing.T) {
	h, _ := setupEnvironmentHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/environment", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetProjectEnvironment(rec, req, projectID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestPutProjectEnvironment_Upsert(t *testing.T) {
	h, _ := setupEnvironmentHandler()
	projectID := uuid.New()

	body := EnvironmentInput{
		Stacks:   []string{model.StackKeyNode},
		Services: []EnvironmentService{{Name: "db", Image: "postgres:16", Env: map[string]string{"POSTGRES_DB": "app"}}},
		Source:   EnvironmentInputSourceDeclared,
		Commands: map[string]string{"test": "npm test"},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/environment",
		bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.PutProjectEnvironment(rec, req, projectID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Environment
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ProjectId != projectID {
		t.Errorf("expected project_id %v, got %v", projectID, resp.ProjectId)
	}
	if len(resp.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(resp.Services))
	}
	if len(resp.Stacks) != 1 || resp.Stacks[0] != model.StackKeyNode {
		t.Errorf("expected stacks=[node], got %v", resp.Stacks)
	}
}

func TestPutProjectEnvironment_InvalidJSON(t *testing.T) {
	h, _ := setupEnvironmentHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/environment",
		bytes.NewBufferString(`{invalid json}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.PutProjectEnvironment(rec, req, projectID)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteProjectEnvironment_NoContent(t *testing.T) {
	h, repo := setupEnvironmentHandler()
	projectID := uuid.New()

	env := &model.Environment{
		ID:        uuid.New(),
		ProjectID: projectID,
		Stacks:    []string{model.StackKeyGo},
		Services:  []model.EnvironmentService{},
		Source:    model.EnvironmentSourceDeclared,
		Commands:  map[string]string{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.envs[env.ID] = env
	repo.byProject[projectID] = env

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/environment", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.DeleteProjectEnvironment(rec, req, projectID)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteProjectEnvironment_NotFound(t *testing.T) {
	h, _ := setupEnvironmentHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/environment", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.DeleteProjectEnvironment(rec, req, projectID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
