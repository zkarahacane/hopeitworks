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
	planningadapter "github.com/zakari/hopeitworks/backend/internal/adapter/planning"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockStoryRepo is a mock implementation of port.StoryRepository for handler tests.
type mockStoryRepo struct {
	stories map[uuid.UUID]*model.Story
	// countErr, when set, is returned by CountByProject to exercise the
	// best-effort graceful-degradation path of callers (#289).
	countErr error
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
	if m.countErr != nil {
		return 0, m.countErr
	}
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

func (m *mockStoryRepo) UpdateStoryCurrentStage(_ context.Context, id uuid.UUID, currentStage *string) (*model.Story, error) {
	if s, ok := m.stories[id]; ok {
		s.CurrentStage = currentStage
		return s, nil
	}
	return &model.Story{ID: id, CurrentStage: currentStage}, nil
}

func (m *mockStoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.stories, id)
	return nil
}

func (m *mockStoryRepo) GetBySourceRef(_ context.Context, projectID uuid.UUID, source, externalID string) (*model.Story, error) {
	for _, s := range m.stories {
		if s.ProjectID == projectID && s.Source == source && s.ExternalID != nil && *s.ExternalID == externalID {
			return s, nil
		}
	}
	return nil, errors.NewNotFound("story", externalID)
}

func (m *mockStoryRepo) CreateFromImport(ctx context.Context, s *model.Story) (*model.Story, error) {
	return m.Create(ctx, s)
}

func (m *mockStoryRepo) UpdateFromImport(_ context.Context, s *model.Story) (*model.Story, error) {
	m.stories[s.ID] = s
	return s, nil
}

func (m *mockStoryRepo) UpdateProvenanceOnly(_ context.Context, s *model.Story) (*model.Story, error) {
	m.stories[s.ID] = s
	return s, nil
}

func setupStoryHandler() (*StoryHandler, *mockStoryRepo) {
	h, repo, _ := setupStoryHandlerWithRuns()
	return h, repo
}

func setupStoryHandlerWithRuns() (*StoryHandler, *mockStoryRepo, *storyHandlerRunRepo) {
	repo := newMockStoryRepo()
	runRepo := &storyHandlerRunRepo{latestByStory: make(map[uuid.UUID]*model.LatestRun)}
	svc := service.NewStoryService(repo)
	// The deprecated /stories/import shim routes through the markdown planning
	// connector; wire a real PlanningImportService over the same story repo.
	planningSvc := service.NewPlanningImportService(repo, newMockEpicRepo(), planningadapter.NewFactory(nil, nil, nil))
	h := NewStoryHandler(svc, runRepo, planningSvc)
	return h, repo, runRepo
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

func (m *mockStoryRepo) CountByEpicGroupedByStatus(_ context.Context, _ uuid.UUID) (model.StoryCounts, error) {
	return model.StoryCounts{}, nil
}

// storyHandlerRunRepo is a minimal mock of port.RunRepository for story handler
// tests, supporting only the latest-run lookups used to populate latest_run.
type storyHandlerRunRepo struct {
	latestByStory map[uuid.UUID]*model.LatestRun
}

var _ port.RunRepository = (*storyHandlerRunRepo)(nil)

func (m *storyHandlerRunRepo) GetLatestRunByStory(_ context.Context, storyID uuid.UUID) (*model.LatestRun, error) {
	return m.latestByStory[storyID], nil
}
func (m *storyHandlerRunRepo) GetLatestRunsByStories(_ context.Context, storyIDs []uuid.UUID) (map[uuid.UUID]*model.LatestRun, error) {
	out := make(map[uuid.UUID]*model.LatestRun)
	for _, id := range storyIDs {
		if lr, ok := m.latestByStory[id]; ok {
			out[id] = lr
		}
	}
	return out, nil
}

func (m *storyHandlerRunRepo) GetDAGNodeRunInfoByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]model.DAGNodeRunInfo, error) {
	return map[uuid.UUID]model.DAGNodeRunInfo{}, nil
}
func (m *storyHandlerRunRepo) CreateRun(_ context.Context, run *model.Run) (*model.Run, error) {
	return run, nil
}
func (m *storyHandlerRunRepo) GetRun(_ context.Context, id uuid.UUID) (*model.Run, error) {
	return nil, errors.NewNotFound("run", id)
}
func (m *storyHandlerRunRepo) GetActiveRunByStory(_ context.Context, _ uuid.UUID) (*model.Run, error) {
	return nil, nil
}
func (m *storyHandlerRunRepo) ListRunsByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *storyHandlerRunRepo) ListRunsByStory(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *storyHandlerRunRepo) ListRunsByStatus(_ context.Context, _ model.RunStatus) ([]*model.Run, error) {
	return nil, nil
}
func (m *storyHandlerRunRepo) MarkRunOrphanedIfRunning(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (bool, error) {
	return false, nil
}
func (m *storyHandlerRunRepo) UpdateRunStatus(_ context.Context, _ uuid.UUID, _ model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
	return nil, nil
}
func (m *storyHandlerRunRepo) CountRunsByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *storyHandlerRunRepo) CountRunsByStory(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *storyHandlerRunRepo) CreateRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}
func (m *storyHandlerRunRepo) GetRunStep(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
	return nil, errors.NewNotFound("run_step", id)
}
func (m *storyHandlerRunRepo) ListRunStepsByRun(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}
func (m *storyHandlerRunRepo) UpdateRunStepStatus(_ context.Context, _ uuid.UUID, _ model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
	return nil, nil
}
func (m *storyHandlerRunRepo) UpdateRunStepContainerInfo(_ context.Context, _ uuid.UUID, _ *string, _ *string) (*model.RunStep, error) {
	return nil, nil
}
func (m *storyHandlerRunRepo) CreateRetryRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}
func (m *storyHandlerRunRepo) ListRetryStepsByParent(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}

func (m *storyHandlerRunRepo) UpdateRunMetadata(_ context.Context, _ uuid.UUID, _ map[string]interface{}) error {
	return nil
}

func (m *storyHandlerRunRepo) AppendStepLogTail(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func TestGetStory_PopulatesLatestRun(t *testing.T) {
	h, repo, runRepo := setupStoryHandlerWithRuns()
	projectID := uuid.New()
	storyID := uuid.New()
	repo.stories[storyID] = &model.Story{
		ID:        storyID,
		ProjectID: projectID,
		Key:       "S-01",
		Title:     "running story",
		Status:    "running",
	}
	runID := uuid.New()
	stepID := uuid.New()
	containerID := "container-xyz789"
	runRepo.latestByStory[storyID] = &model.LatestRun{
		ID:     runID,
		Status: "running",
		CurrentStep: &model.LatestRunStep{
			ID:          stepID,
			Name:        "implement",
			ActionType:  "agent_run",
			Status:      "running",
			Index:       1,
			Total:       4,
			ContainerID: &containerID,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/stories/"+storyID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetStory(rec, req, projectID, storyID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp Story
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.LatestRun == nil {
		t.Fatalf("expected latest_run to be populated, got nil")
	}
	if resp.LatestRun.Id != runID {
		t.Errorf("expected latest_run.id %s, got %s", runID, resp.LatestRun.Id)
	}
	if resp.LatestRun.Status != "running" {
		t.Errorf("expected latest_run.status running, got %q", resp.LatestRun.Status)
	}
	if resp.LatestRun.CurrentStep == nil {
		t.Fatalf("expected current_step to be populated, got nil")
	}
	cs := resp.LatestRun.CurrentStep
	if cs.Id != stepID || cs.Name != "implement" || cs.ActionType != "agent_run" {
		t.Errorf("unexpected current_step: %+v", cs)
	}
	if cs.Index != 1 || cs.Total != 4 {
		t.Errorf("expected index 1/total 4, got %d/%d", cs.Index, cs.Total)
	}
	if cs.ContainerId == nil || *cs.ContainerId != containerID {
		t.Errorf("expected current_step.container_id %s, got %v", containerID, cs.ContainerId)
	}
}

func TestGetStory_NilLatestRun(t *testing.T) {
	h, repo, _ := setupStoryHandlerWithRuns()
	projectID := uuid.New()
	storyID := uuid.New()
	repo.stories[storyID] = &model.Story{
		ID:        storyID,
		ProjectID: projectID,
		Key:       "S-02",
		Title:     "never run",
		Status:    "backlog",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/stories/"+storyID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetStory(rec, req, projectID, storyID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp Story
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.LatestRun != nil {
		t.Errorf("expected latest_run nil for never-run story, got %+v", resp.LatestRun)
	}
}

func TestListStories_PopulatesLatestRunBatch(t *testing.T) {
	h, repo, runRepo := setupStoryHandlerWithRuns()
	projectID := uuid.New()

	withRun := uuid.New()
	repo.stories[withRun] = &model.Story{ID: withRun, ProjectID: projectID, Key: "S-01", Title: "a", Status: "running"}
	runRepo.latestByStory[withRun] = &model.LatestRun{ID: uuid.New(), Status: "running"}

	withoutRun := uuid.New()
	repo.stories[withoutRun] = &model.Story{ID: withoutRun, ProjectID: projectID, Key: "S-02", Title: "b", Status: "backlog"}

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
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 stories, got %d", len(resp.Data))
	}
	for _, s := range resp.Data {
		switch s.Key {
		case "S-01":
			if s.LatestRun == nil {
				t.Errorf("S-01 should have latest_run")
			}
		case "S-02":
			if s.LatestRun != nil {
				t.Errorf("S-02 should have nil latest_run, got %+v", s.LatestRun)
			}
		}
	}
}

// TestImportStories_ShimContract asserts the deprecated /stories/import endpoint
// keeps its byte-identical ImportStoriesResult contract after being rewired
// through the markdown planning connector (§16.14a): the JSON has exactly the
// {imported, updated, failed, errors} keys and the happy-path counts hold.
func TestImportStories_ShimContract(t *testing.T) {
	h, repo, _ := setupStoryHandlerWithRuns()
	projectID := uuid.New()

	content := "---\nkey: S-01\n---\n# First\nbody one\n---\nkey: S-02\n---\n# Second\nbody two"
	body, _ := json.Marshal(ImportStoriesRequest{Content: content})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/stories/import",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin))
	rec := httptest.NewRecorder()

	h.ImportStories(rec, req, projectID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Byte-identical contract: the response object has exactly these keys.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	wantKeys := map[string]bool{"imported": true, "updated": true, "failed": true, "errors": true}
	if len(raw) != len(wantKeys) {
		t.Errorf("ImportStoriesResult must keep exactly %v keys, got %v", keysOf(wantKeys), keysOf(rawKeys(raw)))
	}
	for k := range wantKeys {
		if _, ok := raw[k]; !ok {
			t.Errorf("missing expected key %q in shim response", k)
		}
	}

	var resp ImportStoriesResult
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if resp.Imported != 2 || resp.Updated != 0 || resp.Failed != 0 {
		t.Errorf("expected imported=2 updated=0 failed=0, got %+v", resp)
	}
	if len(repo.stories) != 2 {
		t.Errorf("expected 2 stories persisted via the shim, got %d", len(repo.stories))
	}

	// Re-import of unchanged content is an idempotent no-op: imported=0, updated=0.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/stories/import",
		bytes.NewReader(body))
	req2 = req2.WithContext(middleware.SetUserContext(req2.Context(), uuid.New(), model.RoleAdmin))
	h.ImportStories(rec2, req2, projectID)
	var resp2 ImportStoriesResult
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("decode result 2: %v", err)
	}
	if resp2.Imported != 0 || resp2.Updated != 0 {
		t.Errorf("unchanged re-import via shim should be a no-op, got %+v", resp2)
	}
	if len(repo.stories) != 2 {
		t.Errorf("re-import must not duplicate, got %d stories", len(repo.stories))
	}
}

func rawKeys(m map[string]json.RawMessage) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (m *mockStoryRepo) SetWritebackStatus(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
