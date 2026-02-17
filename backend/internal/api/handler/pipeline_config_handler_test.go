package handler

import (
	"bytes"
	"context"
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
)

// mockPipelineConfigRepo is a mock implementation of port.PipelineConfigRepository for handler tests.
type mockPipelineConfigRepo struct {
	configs map[uuid.UUID]*model.PipelineConfig
}

var _ port.PipelineConfigRepository = (*mockPipelineConfigRepo)(nil)

func newMockPipelineConfigRepo() *mockPipelineConfigRepo {
	return &mockPipelineConfigRepo{
		configs: make(map[uuid.UUID]*model.PipelineConfig),
	}
}

func (m *mockPipelineConfigRepo) GetByProjectID(_ context.Context, projectID uuid.UUID) (*model.PipelineConfig, error) {
	c, ok := m.configs[projectID]
	if !ok {
		return nil, errors.NewNotFound("pipeline_config", projectID)
	}
	return c, nil
}

func (m *mockPipelineConfigRepo) Upsert(_ context.Context, config *model.PipelineConfig) (*model.PipelineConfig, error) {
	existing, ok := m.configs[config.ProjectID]
	if ok {
		existing.ConfigYAML = config.ConfigYAML
		existing.Version++
		return existing, nil
	}
	config.ID = uuid.New()
	config.Version = 1
	m.configs[config.ProjectID] = config
	return config, nil
}

func setupPipelineConfigHandler() (*PipelineConfigHandler, *mockPipelineConfigRepo) {
	repo := newMockPipelineConfigRepo()
	svc := service.NewPipelineConfigService(repo)
	h := NewPipelineConfigHandler(svc)
	return h, repo
}

func TestGetPipelineConfig_Found(t *testing.T) {
	h, repo := setupPipelineConfigHandler()
	projectID := uuid.New()

	repo.configs[projectID] = &model.PipelineConfig{
		ID:         uuid.New(),
		ProjectID:  projectID,
		ConfigYAML: "steps:\n  - name: agent_run\n    action: agent_run\n",
		Version:    1,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/pipeline", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetPipelineConfig(rec, req, projectID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp PipelineConfig
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ProjectId != projectID {
		t.Errorf("expected project_id %v, got %v", projectID, resp.ProjectId)
	}
	if resp.Version != 1 {
		t.Errorf("expected version 1, got %d", resp.Version)
	}
}

func TestGetPipelineConfig_NotFound(t *testing.T) {
	h, _ := setupPipelineConfigHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/pipeline", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetPipelineConfig(rec, req, projectID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetPipelineConfig_NonAdminCanRead(t *testing.T) {
	h, repo := setupPipelineConfigHandler()
	projectID := uuid.New()

	repo.configs[projectID] = &model.PipelineConfig{
		ID:         uuid.New(),
		ProjectID:  projectID,
		ConfigYAML: "steps:\n  - name: agent_run\n    action: agent_run\n",
		Version:    1,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/pipeline", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetPipelineConfig(rec, req, projectID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for non-admin read, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdatePipelineConfig_AdminOnly(t *testing.T) {
	h, _ := setupPipelineConfigHandler()
	projectID := uuid.New()

	validYAML := `{"config_yaml":"steps:\n  - name: agent_run\n    action: agent_run\n"}`

	tests := []struct {
		name       string
		role       model.Role
		body       string
		wantStatus int
	}{
		{
			name:       "admin can update",
			role:       model.RoleAdmin,
			body:       validYAML,
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			body:       validYAML,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no role gets 403",
			role:       "",
			body:       validYAML,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/pipeline",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.role != "" {
				ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.UpdatePipelineConfig(rec, req, projectID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestUpdatePipelineConfig_InvalidYAML(t *testing.T) {
	h, _ := setupPipelineConfigHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/pipeline",
		bytes.NewBufferString(`{"config_yaml":"steps:\n  - name: bad\n    action: invalid_action\n"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdatePipelineConfig(rec, req, projectID)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdatePipelineConfig_InvalidJSON(t *testing.T) {
	h, _ := setupPipelineConfigHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/pipeline",
		bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdatePipelineConfig(rec, req, projectID)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdatePipelineConfig_VersionIncrement(t *testing.T) {
	h, repo := setupPipelineConfigHandler()
	projectID := uuid.New()

	configYAML := "steps:\n  - name: agent_run\n    action: agent_run\n"

	// Seed an existing config
	repo.configs[projectID] = &model.PipelineConfig{
		ID:         uuid.New(),
		ProjectID:  projectID,
		ConfigYAML: configYAML,
		Version:    1,
	}

	body, _ := json.Marshal(UpdatePipelineConfigRequest{ConfigYaml: configYAML})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/pipeline",
		bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdatePipelineConfig(rec, req, projectID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp PipelineConfig
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Version != 2 {
		t.Errorf("expected version 2, got %d", resp.Version)
	}
}
