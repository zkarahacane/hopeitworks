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
	h := NewEpicHandler(svc, scheduler, storyRepo)
	return h, repo
}

func setupEpicHandlerWithStoryRepo(storyRepo port.StoryRepository) (*EpicHandler, *mockEpicRepo) {
	epicRepo := newMockEpicRepo()
	svc := service.NewEpicService(epicRepo)
	scheduler := service.NewSchedulerService()
	h := NewEpicHandler(svc, scheduler, storyRepo)
	return h, epicRepo
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
