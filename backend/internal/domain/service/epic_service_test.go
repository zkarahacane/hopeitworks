package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockEpicRepo is a mock implementation of port.EpicRepository for testing.
type mockEpicRepo struct {
	epics    map[uuid.UUID]*model.Epic
	createFn func(ctx context.Context, e *model.Epic) (*model.Epic, error)
}

func newMockEpicRepo() *mockEpicRepo {
	return &mockEpicRepo{
		epics: make(map[uuid.UUID]*model.Epic),
	}
}

func (m *mockEpicRepo) Create(ctx context.Context, epic *model.Epic) (*model.Epic, error) {
	if m.createFn != nil {
		return m.createFn(ctx, epic)
	}
	epic.ID = uuid.New()
	m.epics[epic.ID] = epic
	return epic, nil
}

func (m *mockEpicRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Epic, error) {
	e, ok := m.epics[id]
	if !ok {
		return nil, errors.NewNotFound("epic", id)
	}
	return e, nil
}

func (m *mockEpicRepo) ListByProject(_ context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Epic, error) {
	result := make([]*model.Epic, 0)
	i := int32(0)
	for _, e := range m.epics {
		if e.ProjectID == projectID {
			if i >= offset && i < offset+limit {
				result = append(result, e)
			}
			i++
		}
	}
	return result, nil
}

func (m *mockEpicRepo) CountByProject(_ context.Context, projectID uuid.UUID) (int64, error) {
	count := int64(0)
	for _, e := range m.epics {
		if e.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}

func (m *mockEpicRepo) Update(_ context.Context, epic *model.Epic) (*model.Epic, error) {
	m.epics[epic.ID] = epic
	return epic, nil
}

func (m *mockEpicRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.epics, id)
	return nil
}

func TestEpicService_Create(t *testing.T) {
	projectID := uuid.New()

	tests := []struct {
		name    string
		params  CreateEpicParams
		wantErr bool
		errCode string
	}{
		{
			name:    "valid epic with default status",
			params:  CreateEpicParams{ProjectID: projectID, Name: "Epic 1"},
			wantErr: false,
		},
		{
			name:    "valid epic with explicit status",
			params:  CreateEpicParams{ProjectID: projectID, Name: "Epic 2", Status: "in_progress"},
			wantErr: false,
		},
		{
			name:    "empty name",
			params:  CreateEpicParams{ProjectID: projectID, Name: ""},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "name too long",
			params:  CreateEpicParams{ProjectID: projectID, Name: string(make([]byte, 256))},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "description too long",
			params: CreateEpicParams{
				ProjectID:   projectID,
				Name:        "test",
				Description: strPtr(string(make([]byte, 2001))),
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "missing project_id",
			params:  CreateEpicParams{Name: "Epic 1"},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "invalid status",
			params:  CreateEpicParams{ProjectID: projectID, Name: "Epic 1", Status: "invalid"},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "with description",
			params: CreateEpicParams{
				ProjectID:   projectID,
				Name:        "Epic with desc",
				Description: strPtr("A test epic"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockEpicRepo()
			svc := NewEpicService(repo)

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
			if result.ProjectID != tt.params.ProjectID {
				t.Errorf("expected project_id %v, got %v", tt.params.ProjectID, result.ProjectID)
			}
			if tt.params.Status != "" {
				if result.Status != tt.params.Status {
					t.Errorf("expected status %q, got %q", tt.params.Status, result.Status)
				}
			} else {
				if result.Status != "backlog" {
					t.Errorf("expected default status 'backlog', got %q", result.Status)
				}
			}
		})
	}
}

func TestEpicService_GetByID(t *testing.T) {
	repo := newMockEpicRepo()
	svc := NewEpicService(repo)

	id := uuid.New()
	repo.epics[id] = &model.Epic{ID: id, Name: "test-epic", ProjectID: uuid.New(), Status: "backlog"}

	// Get existing epic
	result, err := svc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "test-epic" {
		t.Errorf("expected name 'test-epic', got %q", result.Name)
	}

	// Get non-existent epic
	_, err = svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent epic, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %q", domainErr.Category)
	}
}

func TestEpicService_ListByProject(t *testing.T) {
	repo := newMockEpicRepo()
	svc := NewEpicService(repo)

	projectID := uuid.New()
	otherProjectID := uuid.New()

	// Create epics for the target project
	for i := 0; i < 5; i++ {
		id := uuid.New()
		repo.epics[id] = &model.Epic{
			ID:        id,
			ProjectID: projectID,
			Name:      "epic-" + id.String()[:8],
			Status:    "backlog",
		}
	}
	// Create epic for different project
	otherID := uuid.New()
	repo.epics[otherID] = &model.Epic{
		ID:        otherID,
		ProjectID: otherProjectID,
		Name:      "other-epic",
		Status:    "backlog",
	}

	tests := []struct {
		name      string
		page      int
		perPage   int
		wantTotal int64
	}{
		{name: "default pagination", page: 1, perPage: 20, wantTotal: 5},
		{name: "clamp page to 1", page: 0, perPage: 20, wantTotal: 5},
		{name: "clamp perPage to 20", page: 1, perPage: 0, wantTotal: 5},
		{name: "clamp perPage max to 100", page: 1, perPage: 200, wantTotal: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.ListByProject(context.Background(), projectID, tt.page, tt.perPage)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Total != tt.wantTotal {
				t.Errorf("expected total %d, got %d", tt.wantTotal, result.Total)
			}
		})
	}
}

func TestEpicService_Update(t *testing.T) {
	repo := newMockEpicRepo()
	svc := NewEpicService(repo)

	id := uuid.New()
	repo.epics[id] = &model.Epic{
		ID:        id,
		ProjectID: uuid.New(),
		Name:      "original",
		Status:    "backlog",
	}

	tests := []struct {
		name    string
		params  UpdateEpicParams
		wantErr bool
		errCode string
	}{
		{
			name:    "valid name update",
			params:  UpdateEpicParams{ID: id, Name: strPtr("updated")},
			wantErr: false,
		},
		{
			name:    "valid status update",
			params:  UpdateEpicParams{ID: id, Status: strPtr("in_progress")},
			wantErr: false,
		},
		{
			name:    "not found",
			params:  UpdateEpicParams{ID: uuid.New(), Name: strPtr("test")},
			wantErr: true,
			errCode: "EPIC_NOT_FOUND",
		},
		{
			name:    "empty name",
			params:  UpdateEpicParams{ID: id, Name: strPtr("")},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "name too long",
			params:  UpdateEpicParams{ID: id, Name: strPtr(string(make([]byte, 256)))},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "invalid status",
			params:  UpdateEpicParams{ID: id, Status: strPtr("invalid")},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "description too long",
			params:  UpdateEpicParams{ID: id, Description: strPtr(string(make([]byte, 2001)))},
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

func TestEpicService_Delete(t *testing.T) {
	repo := newMockEpicRepo()
	svc := NewEpicService(repo)

	id := uuid.New()
	repo.epics[id] = &model.Epic{ID: id, Name: "to-delete", ProjectID: uuid.New(), Status: "backlog"}

	// Delete existing epic
	err := svc.Delete(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's gone
	_, err = svc.GetByID(context.Background(), id)
	if err == nil {
		t.Fatal("expected not found error after delete")
	}

	// Delete non-existent epic
	err = svc.Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent epic, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %q", domainErr.Category)
	}
}
