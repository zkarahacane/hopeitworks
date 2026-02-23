package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockAgentRepo is a mock implementation of port.AgentRepository for testing.
type mockAgentRepo struct {
	agents   map[uuid.UUID]*model.Agent
	createFn func(ctx context.Context, a *model.Agent) (*model.Agent, error)
}

func newMockAgentRepo() *mockAgentRepo {
	return &mockAgentRepo{
		agents: make(map[uuid.UUID]*model.Agent),
	}
}

func (m *mockAgentRepo) CreateAgent(ctx context.Context, agent *model.Agent) (*model.Agent, error) {
	if m.createFn != nil {
		return m.createFn(ctx, agent)
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

func TestAgentService_Create(t *testing.T) {
	projectID := uuid.New()

	tests := []struct {
		name    string
		params  CreateAgentParams
		wantErr bool
		errCode string
	}{
		{
			name: "valid project agent",
			params: CreateAgentParams{
				ProjectID:       &projectID,
				Name:            "dev-agent",
				Model:           "claude-opus-4-6",
				Image:           "hopeitworks/agent:latest",
				TemplateContent: "You are an agent...",
				Scope:           "project",
			},
			wantErr: false,
		},
		{
			name: "valid global agent",
			params: CreateAgentParams{
				Name:            "global-agent",
				Model:           "claude-sonnet-4-6",
				Image:           "hopeitworks/agent:latest",
				TemplateContent: "Global agent template",
				Scope:           "global",
			},
			wantErr: false,
		},
		{
			name: "default scope is project",
			params: CreateAgentParams{
				ProjectID:       &projectID,
				Name:            "default-scope",
				TemplateContent: "content",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			params: CreateAgentParams{
				ProjectID:       &projectID,
				Name:            "",
				TemplateContent: "content",
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "name too long",
			params: CreateAgentParams{
				ProjectID:       &projectID,
				Name:            string(make([]byte, 256)),
				TemplateContent: "content",
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "empty template_content",
			params: CreateAgentParams{
				ProjectID:       &projectID,
				Name:            "Test",
				TemplateContent: "",
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "missing project_id for project scope",
			params: CreateAgentParams{
				Name:            "Test",
				TemplateContent: "content",
				Scope:           "project",
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "invalid scope",
			params: CreateAgentParams{
				ProjectID:       &projectID,
				Name:            "Test",
				TemplateContent: "content",
				Scope:           "invalid",
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockAgentRepo()
			svc := NewAgentService(repo)

			result, err := svc.Create(context.Background(), tt.params)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				domainErr, ok := err.(*errors.DomainError)
				if !ok {
					t.Fatalf("expected DomainError, got %T", err)
				}
				if domainErr.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, domainErr.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Name != tt.params.Name {
				t.Errorf("expected name %q, got %q", tt.params.Name, result.Name)
			}
		})
	}
}

func TestAgentService_GetByID(t *testing.T) {
	repo := newMockAgentRepo()
	svc := NewAgentService(repo)

	id := uuid.New()
	projectID := uuid.New()
	repo.agents[id] = &model.Agent{
		ID:              id,
		Name:            "test-agent",
		ProjectID:       &projectID,
		TemplateContent: "content",
		Scope:           "project",
	}

	result, err := svc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got %q", result.Name)
	}

	// Get non-existent agent
	_, err = svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent agent, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %q", domainErr.Category)
	}
}

func TestAgentService_ListGlobal(t *testing.T) {
	repo := newMockAgentRepo()
	svc := NewAgentService(repo)

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

	// Add project agent
	projectID := uuid.New()
	projectAgentID := uuid.New()
	repo.agents[projectAgentID] = &model.Agent{
		ID:              projectAgentID,
		Name:            "project-agent",
		ProjectID:       &projectID,
		TemplateContent: "content",
		Scope:           "project",
	}

	result, err := svc.ListGlobal(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("expected 3 global agents, got %d", result.Total)
	}
}

func TestAgentService_Update(t *testing.T) {
	repo := newMockAgentRepo()
	svc := NewAgentService(repo)

	id := uuid.New()
	projectID := uuid.New()
	repo.agents[id] = &model.Agent{
		ID:              id,
		ProjectID:       &projectID,
		Name:            "original",
		Model:           "claude-sonnet-4-6",
		TemplateContent: "original content",
		Scope:           "project",
	}

	tests := []struct {
		name    string
		params  UpdateAgentParams
		wantErr bool
		errCode string
	}{
		{
			name:    "valid name update",
			params:  UpdateAgentParams{ID: id, Name: strPtr("updated")},
			wantErr: false,
		},
		{
			name:    "valid model update",
			params:  UpdateAgentParams{ID: id, Model: strPtr("claude-opus-4-6")},
			wantErr: false,
		},
		{
			name:    "valid content update",
			params:  UpdateAgentParams{ID: id, TemplateContent: strPtr("new content")},
			wantErr: false,
		},
		{
			name:    "not found",
			params:  UpdateAgentParams{ID: uuid.New(), Name: strPtr("test")},
			wantErr: true,
			errCode: "AGENT_NOT_FOUND",
		},
		{
			name:    "empty name",
			params:  UpdateAgentParams{ID: id, Name: strPtr("")},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "name too long",
			params:  UpdateAgentParams{ID: id, Name: strPtr(string(make([]byte, 256)))},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "empty template_content",
			params:  UpdateAgentParams{ID: id, TemplateContent: strPtr("")},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Update(context.Background(), tt.params)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				domainErr, ok := err.(*errors.DomainError)
				if !ok {
					t.Fatalf("expected DomainError, got %T", err)
				}
				if domainErr.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, domainErr.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAgentService_Delete(t *testing.T) {
	repo := newMockAgentRepo()
	svc := NewAgentService(repo)

	id := uuid.New()
	projectID := uuid.New()
	repo.agents[id] = &model.Agent{
		ID:              id,
		Name:            "to-delete",
		ProjectID:       &projectID,
		TemplateContent: "content",
		Scope:           "project",
	}

	err := svc.Delete(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's gone
	_, err = svc.GetByID(context.Background(), id)
	if err == nil {
		t.Fatal("expected not found error after delete")
	}

	// Delete non-existent agent
	err = svc.Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent agent, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %q", domainErr.Category)
	}
}
