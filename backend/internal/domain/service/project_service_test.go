package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockProjectRepo is a mock implementation of port.ProjectRepository for testing.
type mockProjectRepo struct {
	projects map[uuid.UUID]*model.Project
	createFn func(ctx context.Context, p *model.Project) (*model.Project, error)
}

func newMockProjectRepoForService() *mockProjectRepo {
	return &mockProjectRepo{
		projects: make(map[uuid.UUID]*model.Project),
	}
}

func (m *mockProjectRepo) Create(ctx context.Context, project *model.Project) (*model.Project, error) {
	if m.createFn != nil {
		return m.createFn(ctx, project)
	}
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

func TestProjectService_Create(t *testing.T) {
	tests := []struct {
		name    string
		params  CreateProjectParams
		wantErr bool
		errCode string
	}{
		{
			name:    "valid project",
			params:  CreateProjectParams{Name: "test-project"},
			wantErr: false,
		},
		{
			name:    "empty name",
			params:  CreateProjectParams{Name: ""},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "name too long",
			params:  CreateProjectParams{Name: string(make([]byte, 256))},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "description too long",
			params: CreateProjectParams{
				Name:        "test",
				Description: strPtr(string(make([]byte, 1001))),
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "with description",
			params: CreateProjectParams{
				Name:        "test-project",
				Description: strPtr("A test project"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockProjectRepoForService()
			svc := NewProjectService(repo)

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
			if result.GitProvider != "github" {
				t.Errorf("expected git_provider 'github', got %q", result.GitProvider)
			}
			if result.AgentRuntime != "docker" {
				t.Errorf("expected agent_runtime 'docker', got %q", result.AgentRuntime)
			}
		})
	}
}

func TestProjectService_List(t *testing.T) {
	repo := newMockProjectRepoForService()
	svc := NewProjectService(repo)

	// Create some projects
	for i := 0; i < 5; i++ {
		id := uuid.New()
		repo.projects[id] = &model.Project{
			ID:   id,
			Name: "project-" + id.String()[:8],
		}
	}

	tests := []struct {
		name    string
		page    int
		perPage int
		wantLen int
	}{
		{name: "default pagination", page: 1, perPage: 20, wantLen: 5},
		{name: "clamp page to 1", page: 0, perPage: 20, wantLen: 5},
		{name: "clamp perPage to 20", page: 1, perPage: 0, wantLen: 5},
		{name: "clamp perPage max to 100", page: 1, perPage: 200, wantLen: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.List(context.Background(), tt.page, tt.perPage)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Total != 5 {
				t.Errorf("expected total 5, got %d", result.Total)
			}
		})
	}
}

func TestProjectService_Update(t *testing.T) {
	repo := newMockProjectRepoForService()
	svc := NewProjectService(repo)

	// Create a project first
	id := uuid.New()
	repo.projects[id] = &model.Project{
		ID:           id,
		Name:         "original",
		GitProvider:  "github",
		AgentRuntime: "docker",
	}

	tests := []struct {
		name    string
		params  UpdateProjectParams
		wantErr bool
		errCode string
	}{
		{
			name:    "valid update",
			params:  UpdateProjectParams{ID: id, Name: strPtr("updated")},
			wantErr: false,
		},
		{
			name:    "not found",
			params:  UpdateProjectParams{ID: uuid.New(), Name: strPtr("test")},
			wantErr: true,
			errCode: "PROJECT_NOT_FOUND",
		},
		{
			name:    "empty name",
			params:  UpdateProjectParams{ID: id, Name: strPtr("")},
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

func TestProjectService_Delete(t *testing.T) {
	repo := newMockProjectRepoForService()
	svc := NewProjectService(repo)

	id := uuid.New()
	repo.projects[id] = &model.Project{ID: id, Name: "to-delete"}

	// Delete existing project
	err := svc.Delete(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Delete non-existent project
	err = svc.Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent project, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %q", domainErr.Category)
	}
}

func strPtr(s string) *string {
	return &s
}
