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

// mockStoryRepo is a mock implementation of port.StoryRepository for handler tests.
type mockStoryRepo struct {
	stories map[uuid.UUID]*model.Story
}

var _ port.StoryRepository = (*mockStoryRepo)(nil)

func newMockStoryRepo() *mockStoryRepo {
	return &mockStoryRepo{
		stories: make(map[uuid.UUID]*model.Story),
	}
}

func (m *mockStoryRepo) Create(_ context.Context, story *model.Story) (*model.Story, error) {
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

func setupStoryHandler() (*StoryHandler, *mockStoryRepo) {
	repo := newMockStoryRepo()
	svc := service.NewStoryService(repo)
	h := NewStoryHandler(svc)
	return h, repo
}

func TestCreateStory_AdminOnly(t *testing.T) {
	h, _ := setupStoryHandler()
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
			body:       `{"key":"S-01","title":"Story 1"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			body:       `{"key":"S-01","title":"Story 1"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no role gets 403",
			role:       "",
			body:       `{"key":"S-01","title":"Story 1"}`,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/stories",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.role != "" {
				ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.CreateStory(rec, req, projectID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateStory_Validation(t *testing.T) {
	h, _ := setupStoryHandler()
	projectID := uuid.New()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid minimal",
			body:       `{"key":"S-01","title":"Story 1"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "with all fields",
			body:       `{"key":"S-02","title":"Story 2","objective":"Obj","target_files":["main.go"],"depends_on":["S-01"],"scope":"backend","status":"running","acceptance_criteria":"It works"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "with epic_id",
			body:       `{"key":"S-03","title":"Story 3","epic_id":"` + uuid.New().String() + `"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty key",
			body:       `{"key":"","title":"Story"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty title",
			body:       `{"key":"S-01","title":""}`,
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/stories",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			h.CreateStory(rec, req, projectID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateStory_Conflict(t *testing.T) {
	h, repo := setupStoryHandler()
	projectID := uuid.New()

	id := uuid.New()
	repo.stories[id] = &model.Story{
		ID:        id,
		ProjectID: projectID,
		Key:       "S-01",
		Title:     "Existing Story",
		Status:    "backlog",
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/stories",
		bytes.NewBufferString(`{"key":"S-01","title":"Duplicate"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.CreateStory(rec, req, projectID)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListStories(t *testing.T) {
	h, repo := setupStoryHandler()
	projectID := uuid.New()

	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.stories[id] = &model.Story{
			ID:        id,
			ProjectID: projectID,
			Key:       "S-" + id.String()[:4],
			Title:     "story-" + id.String()[:8],
			Status:    "backlog",
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/stories", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListStories(rec, req, projectID, ListStoriesParams{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp StoryList
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Errorf("expected 3 stories, got %d", len(resp.Data))
	}
	if resp.Pagination.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Pagination.Total)
	}
}

func TestListStories_StatusFilter(t *testing.T) {
	h, repo := setupStoryHandler()
	projectID := uuid.New()

	statuses := []string{"backlog", "running", "done"}
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

	statusFilter := "backlog,running"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/stories?status=backlog,running", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListStories(rec, req, projectID, ListStoriesParams{Status: &statusFilter})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp StoryList
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Errorf("expected 2 filtered stories, got %d", len(resp.Data))
	}
}

func TestListStories_KeyLookup(t *testing.T) {
	h, repo := setupStoryHandler()
	projectID := uuid.New()

	id := uuid.New()
	repo.stories[id] = &model.Story{
		ID:        id,
		ProjectID: projectID,
		Key:       "S-14",
		Title:     "Story 14",
		Status:    "backlog",
	}

	keyParam := "S-14"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/stories?key=S-14", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListStories(rec, req, projectID, ListStoriesParams{Key: &keyParam})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Story
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Key != "S-14" {
		t.Errorf("expected key 'S-14', got %q", resp.Key)
	}
}

func TestListStories_KeyLookupNotFound(t *testing.T) {
	h, _ := setupStoryHandler()
	projectID := uuid.New()

	keyParam := "NONEXISTENT"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/stories?key=NONEXISTENT", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListStories(rec, req, projectID, ListStoriesParams{Key: &keyParam})

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestListStories_NonAdmin(t *testing.T) {
	h, repo := setupStoryHandler()
	projectID := uuid.New()

	id := uuid.New()
	repo.stories[id] = &model.Story{
		ID:        id,
		ProjectID: projectID,
		Key:       "S-01",
		Title:     "story-1",
		Status:    "backlog",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/stories", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListStories(rec, req, projectID, ListStoriesParams{})

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for non-admin list, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetStory_Found(t *testing.T) {
	h, repo := setupStoryHandler()
	projectID := uuid.New()
	storyID := uuid.New()
	repo.stories[storyID] = &model.Story{
		ID:        storyID,
		ProjectID: projectID,
		Key:       "S-01",
		Title:     "test-story",
		Status:    "backlog",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/stories/"+storyID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetStory(rec, req, projectID, storyID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Story
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Key != "S-01" {
		t.Errorf("expected key 'S-01', got %q", resp.Key)
	}
}

func TestGetStory_NotFound(t *testing.T) {
	h, _ := setupStoryHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/stories/"+uuid.New().String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetStory(rec, req, projectID, uuid.New())

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateStory_AdminOnly(t *testing.T) {
	h, repo := setupStoryHandler()
	projectID := uuid.New()
	storyID := uuid.New()
	repo.stories[storyID] = &model.Story{
		ID:        storyID,
		ProjectID: projectID,
		Key:       "S-01",
		Title:     "original",
		Status:    "backlog",
	}

	// Non-admin should get 403
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/stories/"+storyID.String(),
		bytes.NewBufferString(`{"title":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.UpdateStory(rec, req, projectID, storyID)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rec.Code)
	}

	// Admin should succeed
	req = httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/stories/"+storyID.String(),
		bytes.NewBufferString(`{"title":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx = middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.UpdateStory(rec, req, projectID, storyID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateStory_StatusChange(t *testing.T) {
	h, repo := setupStoryHandler()
	projectID := uuid.New()
	storyID := uuid.New()
	repo.stories[storyID] = &model.Story{
		ID:        storyID,
		ProjectID: projectID,
		Key:       "S-01",
		Title:     "story-1",
		Status:    "backlog",
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/stories/"+storyID.String(),
		bytes.NewBufferString(`{"status":"running"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateStory(rec, req, projectID, storyID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Story
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "running" {
		t.Errorf("expected status 'running', got %q", resp.Status)
	}
}

func TestDeleteStory_AdminOnly(t *testing.T) {
	h, repo := setupStoryHandler()
	projectID := uuid.New()
	storyID := uuid.New()
	repo.stories[storyID] = &model.Story{
		ID:        storyID,
		ProjectID: projectID,
		Key:       "S-01",
		Title:     "to-delete",
		Status:    "backlog",
	}

	// Non-admin should get 403
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/stories/"+storyID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.DeleteStory(rec, req, projectID, storyID)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rec.Code)
	}

	// Admin should succeed
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/stories/"+storyID.String(), nil)
	ctx = middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.DeleteStory(rec, req, projectID, storyID)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for admin, got %d", rec.Code)
	}
}

func TestDeleteStory_NotFound(t *testing.T) {
	h, _ := setupStoryHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/stories/"+uuid.New().String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.DeleteStory(rec, req, projectID, uuid.New())

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestCreateStory_WithJSONBFields(t *testing.T) {
	h, _ := setupStoryHandler()
	projectID := uuid.New()

	body := `{"key":"S-01","title":"Story with JSONB","target_files":["backend/main.go","backend/handler.go"],"depends_on":["S-00"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/stories",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.CreateStory(rec, req, projectID)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Story
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.TargetFiles == nil || len(*resp.TargetFiles) != 2 {
		t.Errorf("expected 2 target_files, got %v", resp.TargetFiles)
	}
	if resp.DependsOn == nil || len(*resp.DependsOn) != 1 {
		t.Errorf("expected 1 depends_on, got %v", resp.DependsOn)
	}
}
