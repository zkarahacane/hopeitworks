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

// mockAgentRepo is a mock implementation of port.AgentRepository for handler tests.
type mockAgentRepo struct {
	agents map[uuid.UUID]*model.Agent
}

var _ port.AgentRepository = (*mockAgentRepo)(nil)

func newMockAgentRepo() *mockAgentRepo {
	return &mockAgentRepo{
		agents: make(map[uuid.UUID]*model.Agent),
	}
}

func (m *mockAgentRepo) CreateAgent(_ context.Context, agent *model.Agent) (*model.Agent, error) {
	for _, a := range m.agents {
		if a.ProjectID != nil && agent.ProjectID != nil && *a.ProjectID == *agent.ProjectID && a.Name == agent.Name {
			return nil, errors.NewConflict("agent", agent.Name)
		}
	}
	m.agents[agent.ID] = agent
	return agent, nil
}

func (m *mockAgentRepo) GetAgent(_ context.Context, id uuid.UUID) (*model.Agent, error) {
	a, ok := m.agents[id]
	if !ok {
		return nil, errors.NewNotFound("agent", id)
	}
	return a, nil
}

func (m *mockAgentRepo) ListAgentsByProject(_ context.Context, projectID uuid.UUID) ([]*model.Agent, error) {
	var result []*model.Agent
	for _, a := range m.agents {
		if a.ProjectID != nil && *a.ProjectID == projectID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAgentRepo) ListGlobalAgents(_ context.Context) ([]*model.Agent, error) {
	var result []*model.Agent
	for _, a := range m.agents {
		if a.Scope == "global" {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAgentRepo) ListAgentsByProjectMerged(_ context.Context, projectID uuid.UUID) ([]*model.Agent, error) {
	var result []*model.Agent
	for _, a := range m.agents {
		if a.Scope == "global" || (a.ProjectID != nil && *a.ProjectID == projectID) {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAgentRepo) UpdateAgent(_ context.Context, agent *model.Agent) (*model.Agent, error) {
	m.agents[agent.ID] = agent
	return agent, nil
}

func (m *mockAgentRepo) DeleteAgent(_ context.Context, id uuid.UUID) error {
	delete(m.agents, id)
	return nil
}

func setupAgentHandler() (*AgentHandler, *mockAgentRepo) {
	repo := newMockAgentRepo()
	svc := service.NewAgentService(repo)
	h := NewAgentHandler(svc)
	return h, repo
}

func TestCreateAgent_AdminOnly(t *testing.T) {
	h, _ := setupAgentHandler()
	projectID := uuid.New()

	tests := []struct {
		name       string
		role       model.Role
		body       string
		wantStatus int
	}{
		{
			name:       "admin can create",
			role:       model.RoleAdmin,
			body:       `{"name":"Agent 1","template_content":"content","model":"claude-sonnet-4-6","image":"agent:latest"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			body:       `{"name":"Agent 1","template_content":"content","model":"claude-sonnet-4-6","image":"agent:latest"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no role gets 403",
			role:       "",
			body:       `{"name":"Agent 1","template_content":"content","model":"claude-sonnet-4-6","image":"agent:latest"}`,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/agents",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.role != "" {
				ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.CreateAgent(rec, req, projectID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateAgent_Validation(t *testing.T) {
	h, _ := setupAgentHandler()
	projectID := uuid.New()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid",
			body:       `{"name":"Agent 1","template_content":"content","model":"claude-sonnet-4-6","image":"agent:latest"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty name",
			body:       `{"name":"","template_content":"content","model":"claude-sonnet-4-6","image":"agent:latest"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty template_content",
			body:       `{"name":"Test","template_content":"","model":"claude-sonnet-4-6","image":"agent:latest"}`,
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/agents",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			h.CreateAgent(rec, req, projectID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestGetAgent_Found(t *testing.T) {
	h, repo := setupAgentHandler()
	projectID := uuid.New()
	agentID := uuid.New()
	repo.agents[agentID] = &model.Agent{
		ID:              agentID,
		ProjectID:       &projectID,
		Name:            "test-agent",
		TemplateContent: "test content",
		Model:           "claude-sonnet-4-6",
		Image:           "agent:latest",
		Scope:           "project",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/agents/"+agentID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetAgent(rec, req, projectID, agentID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Agent
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got %q", resp.Name)
	}
	if resp.Model != "claude-sonnet-4-6" {
		t.Errorf("expected model 'claude-sonnet-4-6', got %q", resp.Model)
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	h, _ := setupAgentHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/agents/"+uuid.New().String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetAgent(rec, req, projectID, uuid.New())

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateAgent_AdminOnly(t *testing.T) {
	h, repo := setupAgentHandler()
	projectID := uuid.New()
	agentID := uuid.New()
	repo.agents[agentID] = &model.Agent{
		ID:              agentID,
		ProjectID:       &projectID,
		Name:            "original",
		TemplateContent: "original content",
		Model:           "claude-sonnet-4-6",
		Scope:           "project",
	}

	// Non-admin should get 403
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/agents/"+agentID.String(),
		bytes.NewBufferString(`{"name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.UpdateAgent(rec, req, projectID, agentID)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rec.Code)
	}

	// Admin should succeed
	req = httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/agents/"+agentID.String(),
		bytes.NewBufferString(`{"name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx = middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.UpdateAgent(rec, req, projectID, agentID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteAgent_AdminOnly(t *testing.T) {
	h, repo := setupAgentHandler()
	projectID := uuid.New()
	agentID := uuid.New()
	repo.agents[agentID] = &model.Agent{
		ID:              agentID,
		ProjectID:       &projectID,
		Name:            "to-delete",
		TemplateContent: "content",
		Scope:           "project",
	}

	// Non-admin should get 403
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/agents/"+agentID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.DeleteAgent(rec, req, projectID, agentID)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rec.Code)
	}

	// Admin should succeed
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/agents/"+agentID.String(), nil)
	ctx = middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.DeleteAgent(rec, req, projectID, agentID)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for admin, got %d", rec.Code)
	}
}

func TestDeleteAgent_NotFound(t *testing.T) {
	h, _ := setupAgentHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/agents/"+uuid.New().String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.DeleteAgent(rec, req, projectID, uuid.New())

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestListGlobalAgents(t *testing.T) {
	h, repo := setupAgentHandler()

	// Add global agents
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.agents[id] = &model.Agent{
			ID:              id,
			Name:            "global-" + id.String()[:8],
			TemplateContent: "content",
			Scope:           "global",
		}
	}

	// Add a project agent (should not appear in global list)
	projectID := uuid.New()
	projectAgentID := uuid.New()
	repo.agents[projectAgentID] = &model.Agent{
		ID:              projectAgentID,
		Name:            "project-agent",
		ProjectID:       &projectID,
		TemplateContent: "content",
		Scope:           "project",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListGlobalAgents(rec, req, ListGlobalAgentsParams{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Data       []Agent    `json:"data"`
		Pagination Pagination `json:"pagination"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Errorf("expected 3 global agents, got %d", len(resp.Data))
	}
	if resp.Pagination.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Pagination.Total)
	}
}

func TestListProjectAgents(t *testing.T) {
	h, repo := setupAgentHandler()
	projectID := uuid.New()

	// Add project agents
	for i := 0; i < 2; i++ {
		id := uuid.New()
		repo.agents[id] = &model.Agent{
			ID:              id,
			Name:            "project-" + id.String()[:8],
			ProjectID:       &projectID,
			TemplateContent: "content",
			Scope:           "project",
		}
	}

	// Add a global agent (should also appear in merged list)
	globalID := uuid.New()
	repo.agents[globalID] = &model.Agent{
		ID:              globalID,
		Name:            "global-agent",
		TemplateContent: "content",
		Scope:           "global",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/agents", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListProjectAgents(rec, req, projectID, ListProjectAgentsParams{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Data       []Agent    `json:"data"`
		Pagination Pagination `json:"pagination"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Errorf("expected 3 agents (2 project + 1 global), got %d", len(resp.Data))
	}
	if resp.Pagination.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Pagination.Total)
	}
}
