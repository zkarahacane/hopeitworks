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
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockEpicRepo is a mock implementation of port.EpicRepository for handler tests.
type mockEpicRepo struct {
	epics map[uuid.UUID]*model.Epic
}

var _ port.EpicRepository = (*mockEpicRepo)(nil)

func newMockEpicRepo() *mockEpicRepo {
	return &mockEpicRepo{
		epics: make(map[uuid.UUID]*model.Epic),
	}
}

func (m *mockEpicRepo) Create(_ context.Context, epic *model.Epic) (*model.Epic, error) {
	for _, e := range m.epics {
		if e.ProjectID == epic.ProjectID && e.Name == epic.Name {
			return nil, errors.NewConflict("epic", epic.Name)
		}
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

// errorStoryRepo is a mock that always returns an error on ListByEpic.
type errorStoryRepo struct {
	mockStoryRepo
	listByEpicErr error
}

func (m *errorStoryRepo) ListByEpic(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, m.listByEpicErr
}

func setupEpicHandler() (*EpicHandler, *mockEpicRepo) {
	repo := newMockEpicRepo()
	storyRepo := newMockStoryRepo()
	svc := service.NewEpicService(repo)
	scheduler := service.NewSchedulerService()
	h := NewEpicHandler(svc, scheduler, storyRepo, newEpicRunRepo())
	return h, repo
}

func setupEpicHandlerWithStoryRepo(storyRepo port.StoryRepository) (*EpicHandler, *mockEpicRepo) {
	h, repo, _ := setupEpicHandlerWithStoryAndRunRepo(storyRepo, newEpicRunRepo())
	return h, repo
}

func setupEpicHandlerWithStoryAndRunRepo(storyRepo port.StoryRepository, runRepo *epicRunRepo) (*EpicHandler, *mockEpicRepo, *epicRunRepo) {
	epicRepo := newMockEpicRepo()
	svc := service.NewEpicService(epicRepo)
	scheduler := service.NewSchedulerService()
	h := NewEpicHandler(svc, scheduler, storyRepo, runRepo)
	return h, epicRepo, runRepo
}

// epicRunRepo is a configurable port.RunRepository mock for epic DAG tests. Only
// GetDAGNodeRunInfoByStories carries behaviour; everything else is a no-op stub.
type epicRunRepo struct {
	dagNodeInfo map[uuid.UUID]model.DAGNodeRunInfo
	dagErr      error
	run         *model.Run
}

var _ port.RunRepository = (*epicRunRepo)(nil)

func newEpicRunRepo() *epicRunRepo {
	return &epicRunRepo{dagNodeInfo: map[uuid.UUID]model.DAGNodeRunInfo{}}
}

func (m *epicRunRepo) GetDAGNodeRunInfoByStories(_ context.Context, storyIDs []uuid.UUID) (map[uuid.UUID]model.DAGNodeRunInfo, error) {
	if m.dagErr != nil {
		return nil, m.dagErr
	}
	out := make(map[uuid.UUID]model.DAGNodeRunInfo)
	for _, id := range storyIDs {
		if info, ok := m.dagNodeInfo[id]; ok {
			out[id] = info
		}
	}
	return out, nil
}

func (m *epicRunRepo) CreateRun(_ context.Context, run *model.Run) (*model.Run, error) {
	return run, nil
}
func (m *epicRunRepo) GetRun(_ context.Context, id uuid.UUID) (*model.Run, error) {
	if m.run != nil && m.run.ID == id {
		return m.run, nil
	}
	return nil, errors.NewNotFound("run", id)
}
func (m *epicRunRepo) GetActiveRunByStory(_ context.Context, _ uuid.UUID) (*model.Run, error) {
	return nil, nil
}
func (m *epicRunRepo) GetLatestRunByStory(_ context.Context, _ uuid.UUID) (*model.LatestRun, error) {
	return nil, nil
}
func (m *epicRunRepo) GetLatestRunsByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*model.LatestRun, error) {
	return map[uuid.UUID]*model.LatestRun{}, nil
}
func (m *epicRunRepo) ListRunsByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *epicRunRepo) ListRunsByStory(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *epicRunRepo) UpdateRunStatus(_ context.Context, _ uuid.UUID, _ model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
	return nil, nil
}
func (m *epicRunRepo) UpdateRunMetadata(_ context.Context, _ uuid.UUID, _ map[string]interface{}) error {
	return nil
}
func (m *epicRunRepo) AppendStepLogTail(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *epicRunRepo) CountRunsByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *epicRunRepo) CountRunsByStory(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *epicRunRepo) CreateRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}
func (m *epicRunRepo) GetRunStep(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
	return nil, errors.NewNotFound("run_step", id)
}
func (m *epicRunRepo) ListRunStepsByRun(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}
func (m *epicRunRepo) UpdateRunStepStatus(_ context.Context, _ uuid.UUID, _ model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
	return nil, nil
}
func (m *epicRunRepo) UpdateRunStepContainerInfo(_ context.Context, _ uuid.UUID, _ *string, _ *string) (*model.RunStep, error) {
	return nil, nil
}
func (m *epicRunRepo) CreateRetryRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}
func (m *epicRunRepo) ListRetryStepsByParent(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}

func TestCreateEpic_AdminOnly(t *testing.T) {
	h, _ := setupEpicHandler()
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
			body:       `{"name":"Epic 1"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			body:       `{"name":"Epic 1"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no role gets 403",
			role:       "",
			body:       `{"name":"Epic 1"}`,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/epics",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.role != "" {
				ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.CreateEpic(rec, req, projectID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateEpic_Validation(t *testing.T) {
	h, _ := setupEpicHandler()
	projectID := uuid.New()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid",
			body:       `{"name":"Epic 1"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "with description",
			body:       `{"name":"Epic 2","description":"A test epic"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "with status",
			body:       `{"name":"Epic 3","status":"in_progress"}`,
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/epics",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			h.CreateEpic(rec, req, projectID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateEpic_Conflict(t *testing.T) {
	h, repo := setupEpicHandler()
	projectID := uuid.New()

	// Pre-seed an epic
	id := uuid.New()
	repo.epics[id] = &model.Epic{
		ID:        id,
		ProjectID: projectID,
		Name:      "Existing Epic",
		Status:    "backlog",
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/epics",
		bytes.NewBufferString(`{"name":"Existing Epic"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.CreateEpic(rec, req, projectID)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListEpics(t *testing.T) {
	h, repo := setupEpicHandler()
	projectID := uuid.New()

	// Seed data
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.epics[id] = &model.Epic{
			ID:        id,
			ProjectID: projectID,
			Name:      "epic-" + id.String()[:8],
			Status:    "backlog",
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListEpics(rec, req, projectID, ListEpicsParams{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp EpicList
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Errorf("expected 3 epics, got %d", len(resp.Data))
	}
	if resp.Pagination.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Pagination.Total)
	}
}

func TestListEpics_NonAdmin(t *testing.T) {
	h, repo := setupEpicHandler()
	projectID := uuid.New()

	id := uuid.New()
	repo.epics[id] = &model.Epic{
		ID:        id,
		ProjectID: projectID,
		Name:      "epic-1",
		Status:    "backlog",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListEpics(rec, req, projectID, ListEpicsParams{})

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for non-admin list, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetEpic_Found(t *testing.T) {
	h, repo := setupEpicHandler()
	projectID := uuid.New()
	epicID := uuid.New()
	repo.epics[epicID] = &model.Epic{
		ID:        epicID,
		ProjectID: projectID,
		Name:      "test-epic",
		Status:    "backlog",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetEpic(rec, req, projectID, epicID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Epic
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "test-epic" {
		t.Errorf("expected name 'test-epic', got %q", resp.Name)
	}
}

func TestGetEpic_NotFound(t *testing.T) {
	h, _ := setupEpicHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics/"+uuid.New().String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetEpic(rec, req, projectID, uuid.New())

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateEpic_AdminOnly(t *testing.T) {
	h, repo := setupEpicHandler()
	projectID := uuid.New()
	epicID := uuid.New()
	repo.epics[epicID] = &model.Epic{
		ID:        epicID,
		ProjectID: projectID,
		Name:      "original",
		Status:    "backlog",
	}

	// Non-admin should get 403
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String(),
		bytes.NewBufferString(`{"name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.UpdateEpic(rec, req, projectID, epicID)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rec.Code)
	}

	// Admin should succeed
	req = httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String(),
		bytes.NewBufferString(`{"name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx = middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.UpdateEpic(rec, req, projectID, epicID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateEpic_StatusChange(t *testing.T) {
	h, repo := setupEpicHandler()
	projectID := uuid.New()
	epicID := uuid.New()
	repo.epics[epicID] = &model.Epic{
		ID:        epicID,
		ProjectID: projectID,
		Name:      "epic-1",
		Status:    "backlog",
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String(),
		bytes.NewBufferString(`{"status":"in_progress"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateEpic(rec, req, projectID, epicID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Epic
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "in_progress" {
		t.Errorf("expected status 'in_progress', got %q", resp.Status)
	}
}

func TestDeleteEpic_AdminOnly(t *testing.T) {
	h, repo := setupEpicHandler()
	projectID := uuid.New()
	epicID := uuid.New()
	repo.epics[epicID] = &model.Epic{
		ID:        epicID,
		ProjectID: projectID,
		Name:      "to-delete",
		Status:    "backlog",
	}

	// Non-admin should get 403
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.DeleteEpic(rec, req, projectID, epicID)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rec.Code)
	}

	// Admin should succeed
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String(), nil)
	ctx = middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.DeleteEpic(rec, req, projectID, epicID)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for admin, got %d", rec.Code)
	}
}

func TestDeleteEpic_NotFound(t *testing.T) {
	h, _ := setupEpicHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/epics/"+uuid.New().String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.DeleteEpic(rec, req, projectID, uuid.New())

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetEpicDAG_NoDeps(t *testing.T) {
	projectID := uuid.New()
	epicID := uuid.New()

	storyRepo := newMockStoryRepo()
	storyRepo.stories[uuid.New()] = &model.Story{ID: uuid.New(), ProjectID: projectID, EpicID: &epicID, Key: "S-01", Title: "Story 1", Status: "backlog"}
	storyRepo.stories[uuid.New()] = &model.Story{ID: uuid.New(), ProjectID: projectID, EpicID: &epicID, Key: "S-02", Title: "Story 2", Status: "backlog"}
	storyRepo.stories[uuid.New()] = &model.Story{ID: uuid.New(), ProjectID: projectID, EpicID: &epicID, Key: "S-03", Title: "Story 3", Status: "backlog"}

	h, _ := setupEpicHandlerWithStoryRepo(storyRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String()+"/dag", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetEpicDAG(rec, req, projectID, epicID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp EpicDAGResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(resp.Nodes))
	}
	for _, node := range resp.Nodes {
		if node.Layer != 0 {
			t.Errorf("expected all nodes in layer 0, got node %s in layer %d", node.Key, node.Layer)
		}
	}
	if len(resp.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(resp.Edges))
	}
}

func TestGetEpicDAG_WithDeps(t *testing.T) {
	projectID := uuid.New()
	epicID := uuid.New()

	storyRepo := newMockStoryRepo()
	storyRepo.stories[uuid.New()] = &model.Story{ID: uuid.New(), ProjectID: projectID, EpicID: &epicID, Key: "S-01", Title: "Story 1", Status: "backlog"}
	storyRepo.stories[uuid.New()] = &model.Story{ID: uuid.New(), ProjectID: projectID, EpicID: &epicID, Key: "S-02", Title: "Story 2", Status: "backlog", DependsOn: []string{"S-01"}}
	storyRepo.stories[uuid.New()] = &model.Story{ID: uuid.New(), ProjectID: projectID, EpicID: &epicID, Key: "S-03", Title: "Story 3", Status: "backlog", DependsOn: []string{"S-01"}}

	h, _ := setupEpicHandlerWithStoryRepo(storyRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String()+"/dag", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetEpicDAG(rec, req, projectID, epicID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp EpicDAGResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(resp.Nodes))
	}

	// S-01 should be layer 0, S-02 and S-03 should be layer 1
	layerMap := make(map[string]int)
	for _, node := range resp.Nodes {
		layerMap[node.Key] = node.Layer
	}
	if layerMap["S-01"] != 0 {
		t.Errorf("expected S-01 in layer 0, got %d", layerMap["S-01"])
	}
	if layerMap["S-02"] != 1 {
		t.Errorf("expected S-02 in layer 1, got %d", layerMap["S-02"])
	}
	if layerMap["S-03"] != 1 {
		t.Errorf("expected S-03 in layer 1, got %d", layerMap["S-03"])
	}

	if len(resp.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(resp.Edges))
	}
}

func TestGetEpicDAG_CycleReturns422(t *testing.T) {
	projectID := uuid.New()
	epicID := uuid.New()

	storyRepo := newMockStoryRepo()
	storyRepo.stories[uuid.New()] = &model.Story{ID: uuid.New(), ProjectID: projectID, EpicID: &epicID, Key: "S-01", Title: "Story 1", Status: "backlog", DependsOn: []string{"S-02"}}
	storyRepo.stories[uuid.New()] = &model.Story{ID: uuid.New(), ProjectID: projectID, EpicID: &epicID, Key: "S-02", Title: "Story 2", Status: "backlog", DependsOn: []string{"S-01"}}

	h, _ := setupEpicHandlerWithStoryRepo(storyRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String()+"/dag", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetEpicDAG(rec, req, projectID, epicID)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Error
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if resp.Error.Code != "DAG_CYCLE_DETECTED" {
		t.Errorf("expected error code DAG_CYCLE_DETECTED, got %s", resp.Error.Code)
	}
}

func TestGetEpicDAG_StoryRepoError(t *testing.T) {
	projectID := uuid.New()
	epicID := uuid.New()

	errRepo := &errorStoryRepo{
		mockStoryRepo: *newMockStoryRepo(),
		listByEpicErr: errors.NewNotFound("epic", epicID),
	}
	h, _ := setupEpicHandlerWithStoryRepo(errRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String()+"/dag", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetEpicDAG(rec, req, projectID, epicID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetEpicDAG_EnrichesNodesWithRunInfo(t *testing.T) {
	projectID := uuid.New()
	epicID := uuid.New()

	const runningStatus = "running"
	s1ID := uuid.New()
	s2ID := uuid.New()
	storyRepo := newMockStoryRepo()
	storyRepo.stories[s1ID] = &model.Story{ID: s1ID, ProjectID: projectID, EpicID: &epicID, Key: "S-01", Title: "Story 1", Status: runningStatus}
	storyRepo.stories[s2ID] = &model.Story{ID: s2ID, ProjectID: projectID, EpicID: &epicID, Key: "S-02", Title: "Story 2", Status: "backlog"}

	runRepo := newEpicRunRepo()
	runID := uuid.New()
	containerID := "container-abc123"
	// S-01 has a run with container + cost; S-02 has no run (absent from map).
	runRepo.dagNodeInfo[s1ID] = model.DAGNodeRunInfo{
		RunID:       runID,
		RunStatus:   runningStatus,
		ContainerID: &containerID,
		CostUSD:     1.23,
	}

	h, _, _ := setupEpicHandlerWithStoryAndRunRepo(storyRepo, runRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String()+"/dag", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetEpicDAG(rec, req, projectID, epicID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp EpicDAGResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	byKey := make(map[string]EpicDAGNode)
	for _, n := range resp.Nodes {
		byKey[n.Key] = n
	}

	// S-01 enriched with run id/status/container/cost.
	n1 := byKey["S-01"]
	if n1.RunId == nil || *n1.RunId != runID {
		t.Errorf("expected S-01 run_id %s, got %v", runID, n1.RunId)
	}
	if n1.RunStatus == nil || *n1.RunStatus != runningStatus {
		t.Errorf("expected S-01 run_status running, got %v", n1.RunStatus)
	}
	if n1.ContainerId == nil || *n1.ContainerId != containerID {
		t.Errorf("expected S-01 container_id %s, got %v", containerID, n1.ContainerId)
	}
	if n1.CostUsd == nil || *n1.CostUsd != 1.23 {
		t.Errorf("expected S-01 cost_usd 1.23, got %v", n1.CostUsd)
	}

	// S-02 has no run: enrichment fields stay nil.
	n2 := byKey["S-02"]
	if n2.RunId != nil || n2.RunStatus != nil || n2.ContainerId != nil || n2.CostUsd != nil {
		t.Errorf("expected S-02 to have nil run fields, got run_id=%v status=%v container=%v cost=%v",
			n2.RunId, n2.RunStatus, n2.ContainerId, n2.CostUsd)
	}
}

func TestGetEpicDAG_RunInfoErrorIsNonFatal(t *testing.T) {
	projectID := uuid.New()
	epicID := uuid.New()

	s1ID := uuid.New()
	storyRepo := newMockStoryRepo()
	storyRepo.stories[s1ID] = &model.Story{ID: s1ID, ProjectID: projectID, EpicID: &epicID, Key: "S-01", Title: "Story 1", Status: "backlog"}

	runRepo := newEpicRunRepo()
	runRepo.dagErr = errors.NewInternal("db down", nil)

	h, _, _ := setupEpicHandlerWithStoryAndRunRepo(storyRepo, runRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String()+"/dag", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetEpicDAG(rec, req, projectID, epicID)

	// DAG still renders (enrichment failure is best-effort) with bare nodes.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp EpicDAGResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(resp.Nodes))
	}
	if resp.Nodes[0].RunId != nil || resp.Nodes[0].CostUsd != nil {
		t.Errorf("expected bare node (nil run fields) when enrichment fails, got %+v", resp.Nodes[0])
	}
}

// countsStoryRepo is a story repo mock returning configurable per-epic counts.
type countsStoryRepo struct {
	mockStoryRepo
	counts model.StoryCounts
}

func (m *countsStoryRepo) CountByEpicGroupedByStatus(_ context.Context, _ uuid.UUID) (model.StoryCounts, error) {
	return m.counts, nil
}

func TestGetEpic_PopulatesStoryCounts(t *testing.T) {
	storyRepo := &countsStoryRepo{
		mockStoryRepo: *newMockStoryRepo(),
		counts:        model.StoryCounts{Backlog: 3, Running: 1, Done: 5, Failed: 2},
	}
	h, epicRepo := setupEpicHandlerWithStoryRepo(storyRepo)

	projectID := uuid.New()
	epicID := uuid.New()
	epicRepo.epics[epicID] = &model.Epic{
		ID:        epicID,
		ProjectID: projectID,
		Name:      "Auth",
		Status:    model.EpicStatusBacklog,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetEpic(rec, req, projectID, epicID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var resp Epic
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.StoryCounts.Backlog != 3 || resp.StoryCounts.Running != 1 ||
		resp.StoryCounts.Done != 5 || resp.StoryCounts.Failed != 2 {
		t.Errorf("unexpected story_counts: %+v", resp.StoryCounts)
	}
}

func TestListEpics_PopulatesStoryCounts(t *testing.T) {
	storyRepo := &countsStoryRepo{
		mockStoryRepo: *newMockStoryRepo(),
		counts:        model.StoryCounts{Backlog: 1, Running: 2, Done: 0, Failed: 0},
	}
	h, epicRepo := setupEpicHandlerWithStoryRepo(storyRepo)

	projectID := uuid.New()
	for i := 0; i < 2; i++ {
		id := uuid.New()
		epicRepo.epics[id] = &model.Epic{ID: id, ProjectID: projectID, Name: "E", Status: model.EpicStatusBacklog}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListEpics(rec, req, projectID, ListEpicsParams{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp EpicList
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 epics, got %d", len(resp.Data))
	}
	for _, e := range resp.Data {
		if e.StoryCounts.Running != 2 || e.StoryCounts.Backlog != 1 {
			t.Errorf("unexpected story_counts: %+v", e.StoryCounts)
		}
	}
}
