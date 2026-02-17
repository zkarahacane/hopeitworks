package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockStoryRepo is a mock implementation of port.StoryRepository for testing.
type mockStoryRepo struct {
	stories  map[uuid.UUID]*model.Story
	createFn func(ctx context.Context, s *model.Story) (*model.Story, error)
}

func newMockStoryRepo() *mockStoryRepo {
	return &mockStoryRepo{
		stories: make(map[uuid.UUID]*model.Story),
	}
}

func (m *mockStoryRepo) Create(ctx context.Context, story *model.Story) (*model.Story, error) {
	if m.createFn != nil {
		return m.createFn(ctx, story)
	}
	for _, s := range m.stories {
		if s.ProjectID == story.ProjectID && s.Key == story.Key {
			return nil, errors.NewConflict("story", story.Key)
		}
	}
	story.ID = uuid.New()
	m.stories[story.ID] = story
	return story, nil
}

func (m *mockStoryRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Story, error) {
	s, ok := m.stories[id]
	if !ok {
		return nil, errors.NewNotFound("story", id)
	}
	return s, nil
}

func (m *mockStoryRepo) GetByKey(_ context.Context, projectID uuid.UUID, key string) (*model.Story, error) {
	for _, s := range m.stories {
		if s.ProjectID == projectID && s.Key == key {
			return s, nil
		}
	}
	return nil, errors.NewNotFound("story", key)
}

func (m *mockStoryRepo) ListByProject(_ context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Story, error) {
	result := make([]*model.Story, 0)
	i := int32(0)
	for _, s := range m.stories {
		if s.ProjectID == projectID {
			if i >= offset && i < offset+limit {
				result = append(result, s)
			}
			i++
		}
	}
	return result, nil
}

func (m *mockStoryRepo) ListByStatus(_ context.Context, projectID uuid.UUID, statuses []string, limit, offset int32) ([]*model.Story, error) {
	statusSet := make(map[string]bool)
	for _, st := range statuses {
		statusSet[st] = true
	}
	result := make([]*model.Story, 0)
	i := int32(0)
	for _, s := range m.stories {
		if s.ProjectID == projectID && statusSet[s.Status] {
			if i >= offset && i < offset+limit {
				result = append(result, s)
			}
			i++
		}
	}
	return result, nil
}

func (m *mockStoryRepo) ListByEpic(_ context.Context, epicID uuid.UUID, limit, offset int32) ([]*model.Story, error) {
	result := make([]*model.Story, 0)
	i := int32(0)
	for _, s := range m.stories {
		if s.EpicID != nil && *s.EpicID == epicID {
			if i >= offset && i < offset+limit {
				result = append(result, s)
			}
			i++
		}
	}
	return result, nil
}

func (m *mockStoryRepo) CountByProject(_ context.Context, projectID uuid.UUID) (int64, error) {
	count := int64(0)
	for _, s := range m.stories {
		if s.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}

func (m *mockStoryRepo) CountByStatus(_ context.Context, projectID uuid.UUID, statuses []string) (int64, error) {
	statusSet := make(map[string]bool)
	for _, st := range statuses {
		statusSet[st] = true
	}
	count := int64(0)
	for _, s := range m.stories {
		if s.ProjectID == projectID && statusSet[s.Status] {
			count++
		}
	}
	return count, nil
}

func (m *mockStoryRepo) Update(_ context.Context, story *model.Story) (*model.Story, error) {
	m.stories[story.ID] = story
	return story, nil
}

func (m *mockStoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.stories, id)
	return nil
}

func TestStoryService_Create(t *testing.T) {
	projectID := uuid.New()

	tests := []struct {
		name    string
		params  CreateStoryParams
		wantErr bool
		errCode string
	}{
		{
			name:    "valid story with default status",
			params:  CreateStoryParams{ProjectID: projectID, Key: "S-01", Title: "Story 1"},
			wantErr: false,
		},
		{
			name:    "valid story with explicit status",
			params:  CreateStoryParams{ProjectID: projectID, Key: "S-02", Title: "Story 2", Status: "running"},
			wantErr: false,
		},
		{
			name:    "valid story with all fields",
			params:  CreateStoryParams{ProjectID: projectID, Key: "S-03", Title: "Story 3", Objective: storyStrPtr("An objective"), TargetFiles: []string{"main.go"}, DependsOn: []string{"S-01"}, Scope: storyStrPtr("backend"), AcceptanceCriteria: storyStrPtr("It works")},
			wantErr: false,
		},
		{
			name:    "empty key",
			params:  CreateStoryParams{ProjectID: projectID, Key: "", Title: "Story"},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "key too long",
			params:  CreateStoryParams{ProjectID: projectID, Key: string(make([]byte, 51)), Title: "Story"},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "empty title",
			params:  CreateStoryParams{ProjectID: projectID, Key: "S-01", Title: ""},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "title too long",
			params:  CreateStoryParams{ProjectID: projectID, Key: "S-01", Title: string(make([]byte, 256))},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "missing project_id",
			params:  CreateStoryParams{Key: "S-01", Title: "Story"},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "invalid status",
			params:  CreateStoryParams{ProjectID: projectID, Key: "S-01", Title: "Story", Status: "invalid"},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "invalid scope",
			params:  CreateStoryParams{ProjectID: projectID, Key: "S-01", Title: "Story", Scope: storyStrPtr("invalid")},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "valid scope backend",
			params:  CreateStoryParams{ProjectID: projectID, Key: "S-04", Title: "Story", Scope: storyStrPtr("backend")},
			wantErr: false,
		},
		{
			name:    "valid scope frontend",
			params:  CreateStoryParams{ProjectID: projectID, Key: "S-05", Title: "Story", Scope: storyStrPtr("frontend")},
			wantErr: false,
		},
		{
			name:    "valid scope shared",
			params:  CreateStoryParams{ProjectID: projectID, Key: "S-06", Title: "Story", Scope: storyStrPtr("shared")},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockStoryRepo()
			svc := NewStoryService(repo)

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
			if result.Key != tt.params.Key {
				t.Errorf("expected key %q, got %q", tt.params.Key, result.Key)
			}
			if result.Title != tt.params.Title {
				t.Errorf("expected title %q, got %q", tt.params.Title, result.Title)
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

func TestStoryService_GetByID(t *testing.T) {
	repo := newMockStoryRepo()
	svc := NewStoryService(repo)

	id := uuid.New()
	repo.stories[id] = &model.Story{ID: id, Key: "S-01", Title: "test-story", ProjectID: uuid.New(), Status: "backlog"}

	result, err := svc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Key != "S-01" {
		t.Errorf("expected key 'S-01', got %q", result.Key)
	}

	_, err = svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent story, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %q", domainErr.Category)
	}
}

func TestStoryService_GetByKey(t *testing.T) {
	repo := newMockStoryRepo()
	svc := NewStoryService(repo)

	projectID := uuid.New()
	id := uuid.New()
	repo.stories[id] = &model.Story{ID: id, Key: "S-14", Title: "test-story", ProjectID: projectID, Status: "backlog"}

	result, err := svc.GetByKey(context.Background(), projectID, "S-14")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Key != "S-14" {
		t.Errorf("expected key 'S-14', got %q", result.Key)
	}

	_, err = svc.GetByKey(context.Background(), projectID, "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for non-existent key, got nil")
	}
}

func TestStoryService_ListByProject(t *testing.T) {
	repo := newMockStoryRepo()
	svc := NewStoryService(repo)

	projectID := uuid.New()
	otherProjectID := uuid.New()

	for i := 0; i < 5; i++ {
		id := uuid.New()
		repo.stories[id] = &model.Story{
			ID:        id,
			ProjectID: projectID,
			Key:       "S-" + id.String()[:4],
			Title:     "story-" + id.String()[:8],
			Status:    "backlog",
		}
	}
	otherID := uuid.New()
	repo.stories[otherID] = &model.Story{
		ID:        otherID,
		ProjectID: otherProjectID,
		Key:       "OTHER-1",
		Title:     "other-story",
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

func TestStoryService_ListByStatus(t *testing.T) {
	repo := newMockStoryRepo()
	svc := NewStoryService(repo)

	projectID := uuid.New()

	// Create stories with different statuses
	statuses := []string{"backlog", "running", "done", "failed", "backlog"}
	for i, st := range statuses {
		id := uuid.New()
		repo.stories[id] = &model.Story{
			ID:        id,
			ProjectID: projectID,
			Key:       "S-" + uuid.New().String()[:4],
			Title:     "story-" + string(rune('A'+i)),
			Status:    st,
		}
	}

	// Filter by backlog only
	result, err := svc.ListByStatus(context.Background(), projectID, []string{"backlog"}, 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 backlog stories, got %d", result.Total)
	}

	// Filter by backlog and running
	result, err = svc.ListByStatus(context.Background(), projectID, []string{"backlog", "running"}, 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("expected 3 backlog+running stories, got %d", result.Total)
	}
}

func TestStoryService_Update(t *testing.T) {
	repo := newMockStoryRepo()
	svc := NewStoryService(repo)

	id := uuid.New()
	repo.stories[id] = &model.Story{
		ID:        id,
		ProjectID: uuid.New(),
		Key:       "S-01",
		Title:     "original",
		Status:    "backlog",
	}

	tests := []struct {
		name    string
		params  UpdateStoryParams
		wantErr bool
		errCode string
	}{
		{
			name:    "valid title update",
			params:  UpdateStoryParams{ID: id, Title: storyStrPtr("updated")},
			wantErr: false,
		},
		{
			name:    "valid status update",
			params:  UpdateStoryParams{ID: id, Status: storyStrPtr("running")},
			wantErr: false,
		},
		{
			name:    "valid scope update",
			params:  UpdateStoryParams{ID: id, Scope: storyStrPtr("backend")},
			wantErr: false,
		},
		{
			name:    "not found",
			params:  UpdateStoryParams{ID: uuid.New(), Title: storyStrPtr("test")},
			wantErr: true,
			errCode: "STORY_NOT_FOUND",
		},
		{
			name:    "empty title",
			params:  UpdateStoryParams{ID: id, Title: storyStrPtr("")},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "title too long",
			params:  UpdateStoryParams{ID: id, Title: storyStrPtr(string(make([]byte, 256)))},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "invalid status",
			params:  UpdateStoryParams{ID: id, Status: storyStrPtr("invalid")},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "invalid scope",
			params:  UpdateStoryParams{ID: id, Scope: storyStrPtr("invalid")},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "update target_files",
			params: UpdateStoryParams{
				ID:          id,
				TargetFiles: &[]string{"main.go", "handler.go"},
			},
			wantErr: false,
		},
		{
			name: "update depends_on",
			params: UpdateStoryParams{
				ID:        id,
				DependsOn: &[]string{"S-02", "S-03"},
			},
			wantErr: false,
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

func TestStoryService_Delete(t *testing.T) {
	repo := newMockStoryRepo()
	svc := NewStoryService(repo)

	id := uuid.New()
	repo.stories[id] = &model.Story{ID: id, Key: "S-01", Title: "to-delete", ProjectID: uuid.New(), Status: "backlog"}

	err := svc.Delete(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.GetByID(context.Background(), id)
	if err == nil {
		t.Fatal("expected not found error after delete")
	}

	err = svc.Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent story, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %q", domainErr.Category)
	}
}

func storyStrPtr(s string) *string {
	return &s
}
