package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"

	"context"
)

// mockProjectRepo is a mock implementation of port.ProjectRepository for handler tests.
type mockProjectRepo struct {
	projects map[uuid.UUID]*model.Project
}

// Compile-time check that mockProjectRepo implements port.ProjectRepository.
var _ port.ProjectRepository = (*mockProjectRepo)(nil)

func newMockRepo() *mockProjectRepo {
	return &mockProjectRepo{
		projects: make(map[uuid.UUID]*model.Project),
	}
}

func (m *mockProjectRepo) Create(_ context.Context, project *model.Project) (*model.Project, error) {
	project.ID = uuid.New()
	m.projects[project.ID] = project
	return project, nil
}

func (m *mockProjectRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return nil, errors.NewNotFound("project", id)
	}
	return p, nil
}

func (m *mockProjectRepo) List(_ context.Context, limit, offset int32) ([]*model.Project, error) {
	result := make([]*model.Project, 0)
	i := int32(0)
	for _, p := range m.projects {
		if i >= offset && i < offset+limit {
			result = append(result, p)
		}
		i++
	}
	return result, nil
}

func (m *mockProjectRepo) Count(_ context.Context) (int64, error) {
	return int64(len(m.projects)), nil
}

func (m *mockProjectRepo) Update(_ context.Context, project *model.Project) (*model.Project, error) {
	m.projects[project.ID] = project
	return project, nil
}

func (m *mockProjectRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.projects, id)
	return nil
}

func setupHandler() (*ProjectHandler, *mockProjectRepo) {
	repo := newMockRepo()
	svc := service.NewProjectService(repo)
	handler := NewProjectHandler(svc)
	return handler, repo
}

func TestCreateProject_AdminOnly(t *testing.T) {
	h, _ := setupHandler()

	tests := []struct {
		name       string
		role       string
		body       string
		wantStatus int
	}{
		{
			name:       "admin can create",
			role:       "admin",
			body:       `{"name":"test-project"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "non-admin gets 403",
			role:       "member",
			body:       `{"name":"test-project"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no role gets 403",
			role:       "",
			body:       `{"name":"test-project"}`,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.role != "" {
				ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.CreateProject(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateProject_Validation(t *testing.T) {
	h, _ := setupHandler()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid",
			body:       `{"name":"test-project"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty name",
			body:       `{"name":""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := middleware.SetUserContext(req.Context(), uuid.New(), "admin")
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			h.CreateProject(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestListProjects(t *testing.T) {
	h, repo := setupHandler()

	// Seed data
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.projects[id] = &model.Project{
			ID:           id,
			Name:         "project-" + id.String()[:8],
			GitProvider:  "github",
			AgentRuntime: "docker",
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec := httptest.NewRecorder()

	h.ListProjects(rec, req, ListProjectsParams{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp ProjectList
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Errorf("expected 3 projects, got %d", len(resp.Data))
	}
	if resp.Pagination.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Pagination.Total)
	}
}

func TestGetProject_NotFound(t *testing.T) {
	h, _ := setupHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()

	h.GetProject(rec, req, uuid.New())

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateProject_AdminOnly(t *testing.T) {
	h, repo := setupHandler()

	id := uuid.New()
	repo.projects[id] = &model.Project{
		ID:           id,
		Name:         "original",
		GitProvider:  "github",
		AgentRuntime: "docker",
	}

	// Non-admin should get 403
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+id.String(),
		bytes.NewBufferString(`{"name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), "member")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.UpdateProject(rec, req, id)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rec.Code)
	}

	// Admin should succeed
	req = httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+id.String(),
		bytes.NewBufferString(`{"name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx = middleware.SetUserContext(req.Context(), uuid.New(), "admin")
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.UpdateProject(rec, req, id)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteProject_AdminOnly(t *testing.T) {
	h, repo := setupHandler()

	id := uuid.New()
	repo.projects[id] = &model.Project{
		ID:           id,
		Name:         "to-delete",
		GitProvider:  "github",
		AgentRuntime: "docker",
	}

	// Non-admin should get 403
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+id.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), "member")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.DeleteProject(rec, req, id)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rec.Code)
	}

	// Admin should succeed
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+id.String(), nil)
	ctx = middleware.SetUserContext(req.Context(), uuid.New(), "admin")
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.DeleteProject(rec, req, id)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for admin, got %d", rec.Code)
	}
}
