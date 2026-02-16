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

// Compile-time check that mockEpicRepo implements port.EpicRepository.
var _ port.EpicRepository = (*mockEpicRepo)(nil)

func newMockEpicRepo() *mockEpicRepo {
	return &mockEpicRepo{
		epics: make(map[uuid.UUID]*model.Epic),
	}
}

func (m *mockEpicRepo) Create(_ context.Context, epic *model.Epic) (*model.Epic, error) {
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

func setupEpicHandler() (*EpicHandler, *mockEpicRepo, uuid.UUID) {
	epicRepo := newMockEpicRepo()
	epicSvc := service.NewEpicService(epicRepo)

	projectRepo := newMockProjectRepo()
	puRepo := newMockProjectUserRepoForHandler()
	userRepo := newMockUserRepoForHandler()
	puSvc := service.NewProjectUserService(puRepo, projectRepo, userRepo)

	handler := NewEpicHandler(epicSvc, puSvc)

	projectID := uuid.New()
	return handler, epicRepo, projectID
}

func TestCreateEpic_Success(t *testing.T) {
	h, _, projectID := setupEpicHandler()

	body := `{"name":"Auth Epic","description":"All auth stories"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/epics", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.CreateEpic(rec, req, projectID)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Epic
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "Auth Epic" {
		t.Errorf("expected name 'Auth Epic', got %q", resp.Name)
	}
	if resp.Status != EpicStatusBacklog {
		t.Errorf("expected status 'backlog', got %q", resp.Status)
	}
}

func TestCreateEpic_WithStatus(t *testing.T) {
	h, _, projectID := setupEpicHandler()

	body := `{"name":"Active Epic","status":"in_progress"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/epics", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.CreateEpic(rec, req, projectID)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Epic
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != EpicStatusInProgress {
		t.Errorf("expected status 'in_progress', got %q", resp.Status)
	}
}

func TestCreateEpic_Validation(t *testing.T) {
	h, _, projectID := setupEpicHandler()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid",
			body:       `{"name":"test-epic"}`,
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/epics", bytes.NewBufferString(tt.body))
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

func TestCreateEpic_NonAdminNonMember(t *testing.T) {
	h, _, projectID := setupEpicHandler()

	body := `{"name":"test-epic"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/epics", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.CreateEpic(rec, req, projectID)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListEpics_Admin(t *testing.T) {
	h, epicRepo, projectID := setupEpicHandler()

	for i := 0; i < 3; i++ {
		id := uuid.New()
		epicRepo.epics[id] = &model.Epic{
			ID:        id,
			ProjectID: projectID,
			Name:      "epic-" + id.String()[:8],
			Status:    model.EpicStatusBacklog,
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListEpics(rec, req, projectID, ListEpicsParams{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
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

func TestGetEpic_Success(t *testing.T) {
	h, epicRepo, projectID := setupEpicHandler()

	epicID := uuid.New()
	epicRepo.epics[epicID] = &model.Epic{
		ID:        epicID,
		ProjectID: projectID,
		Name:      "test-epic",
		Status:    model.EpicStatusBacklog,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetEpic(rec, req, projectID, epicID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetEpic_NotFound(t *testing.T) {
	h, _, projectID := setupEpicHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics/"+uuid.New().String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetEpic(rec, req, projectID, uuid.New())

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetEpic_WrongProject(t *testing.T) {
	h, epicRepo, projectID := setupEpicHandler()

	otherProjectID := uuid.New()
	epicID := uuid.New()
	epicRepo.epics[epicID] = &model.Epic{
		ID:        epicID,
		ProjectID: otherProjectID,
		Name:      "other-project-epic",
		Status:    model.EpicStatusBacklog,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetEpic(rec, req, projectID, epicID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for epic in wrong project, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateEpic_Success(t *testing.T) {
	h, epicRepo, projectID := setupEpicHandler()

	epicID := uuid.New()
	epicRepo.epics[epicID] = &model.Epic{
		ID:        epicID,
		ProjectID: projectID,
		Name:      "original",
		Status:    model.EpicStatusBacklog,
	}

	body := `{"name":"updated","status":"in_progress"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String(),
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateEpic(rec, req, projectID, epicID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp Epic
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "updated" {
		t.Errorf("expected name 'updated', got %q", resp.Name)
	}
	if resp.Status != EpicStatusInProgress {
		t.Errorf("expected status 'in_progress', got %q", resp.Status)
	}
}

func TestDeleteEpic_Success(t *testing.T) {
	h, epicRepo, projectID := setupEpicHandler()

	epicID := uuid.New()
	epicRepo.epics[epicID] = &model.Epic{
		ID:        epicID,
		ProjectID: projectID,
		Name:      "to-delete",
		Status:    model.EpicStatusBacklog,
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.DeleteEpic(rec, req, projectID, epicID)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteEpic_NotFound(t *testing.T) {
	h, _, projectID := setupEpicHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/epics/"+uuid.New().String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.DeleteEpic(rec, req, projectID, uuid.New())

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteEpic_WrongProject(t *testing.T) {
	h, epicRepo, projectID := setupEpicHandler()

	otherProjectID := uuid.New()
	epicID := uuid.New()
	epicRepo.epics[epicID] = &model.Epic{
		ID:        epicID,
		ProjectID: otherProjectID,
		Name:      "other-project-epic",
		Status:    model.EpicStatusBacklog,
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/epics/"+epicID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.DeleteEpic(rec, req, projectID, epicID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for epic in wrong project, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
