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

// validStepsYAML returns a YAML string using the new step format (action_type).
func validStepsYAML() string {
	return "steps:\n" +
		"  - id: 880e8400-e29b-41d4-a716-446655440001\n" +
		"    name: implement\n" +
		"    action_type: implement\n" +
		"    model: claude-opus-4-6\n" +
		"    auto_approve: false\n" +
		"    retry_policy:\n" +
		"      max_retries: 2\n" +
		"      retry_type: on-failure\n"
}

// validStepsRequest returns a valid UpdatePipelineConfigRequest using the new API shape.
func validStepsRequest() UpdatePipelineConfigRequest {
	stepID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440001")
	return UpdatePipelineConfigRequest{
		Steps: []PipelineStep{
			{
				Id:          stepID,
				Name:        "implement",
				ActionType:  PipelineStepActionTypeImplement,
				Model:       ClaudeOpus46,
				AutoApprove: false,
				RetryPolicy: RetryPolicy{
					MaxRetries: 2,
					RetryType:  OnFailure,
				},
			},
		},
	}
}

func TestGetPipelineConfig_Found(t *testing.T) {
	h, repo := setupPipelineConfigHandler()
	projectID := uuid.New()

	repo.configs[projectID] = &model.PipelineConfig{
		ID:         uuid.New(),
		ProjectID:  projectID,
		ConfigYAML: validStepsYAML(),
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
	if len(resp.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(resp.Steps))
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
		ConfigYAML: validStepsYAML(),
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

	validBody, _ := json.Marshal(validStepsRequest())

	tests := []struct {
		name       string
		role       model.Role
		body       string
		wantStatus int
	}{
		{
			name:       "admin can update",
			role:       model.RoleAdmin,
			body:       string(validBody),
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			body:       string(validBody),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no role gets 403",
			role:       "",
			body:       string(validBody),
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

func TestUpdatePipelineConfig_RoundTrip(t *testing.T) {
	h, repo := setupPipelineConfigHandler()
	projectID := uuid.New()

	// Seed an existing config
	repo.configs[projectID] = &model.PipelineConfig{
		ID:         uuid.New(),
		ProjectID:  projectID,
		ConfigYAML: validStepsYAML(),
		Version:    1,
	}

	body, _ := json.Marshal(validStepsRequest())
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
	if resp.ProjectId != projectID {
		t.Errorf("expected project_id %v, got %v", projectID, resp.ProjectId)
	}
	if len(resp.Steps) != 1 {
		t.Errorf("expected 1 step in response, got %d", len(resp.Steps))
	}
	if resp.Steps[0].ActionType != PipelineStepActionTypeImplement {
		t.Errorf("expected action_type 'implement', got %s", resp.Steps[0].ActionType)
	}
}
