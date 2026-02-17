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

// mockPromptTemplateRepo is a mock implementation of port.PromptTemplateRepository for handler tests.
type mockPromptTemplateRepo struct {
	templates map[uuid.UUID]*model.PromptTemplate
}

var _ port.PromptTemplateRepository = (*mockPromptTemplateRepo)(nil)

func newMockPromptTemplateRepo() *mockPromptTemplateRepo {
	return &mockPromptTemplateRepo{
		templates: make(map[uuid.UUID]*model.PromptTemplate),
	}
}

func (m *mockPromptTemplateRepo) Create(_ context.Context, tmpl *model.PromptTemplate) (*model.PromptTemplate, error) {
	for _, t := range m.templates {
		if t.ProjectID == tmpl.ProjectID && t.Name == tmpl.Name {
			return nil, errors.NewConflict("prompt_template", tmpl.Name)
		}
	}
	tmpl.ID = uuid.New()
	m.templates[tmpl.ID] = tmpl
	return tmpl, nil
}

func (m *mockPromptTemplateRepo) GetByID(_ context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	t, ok := m.templates[id]
	if !ok {
		return nil, errors.NewNotFound("prompt_template", id)
	}
	return t, nil
}

func (m *mockPromptTemplateRepo) GetByProjectAndName(_ context.Context, projectID uuid.UUID, name string) (*model.PromptTemplate, error) {
	for _, t := range m.templates {
		if t.ProjectID == projectID && t.Name == name {
			return t, nil
		}
	}
	return nil, errors.NewNotFound("prompt_template", name)
}

func (m *mockPromptTemplateRepo) ListByProject(_ context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.PromptTemplate, error) {
	result := make([]*model.PromptTemplate, 0)
	i := int32(0)
	for _, t := range m.templates {
		if t.ProjectID == projectID {
			if i >= offset && i < offset+limit {
				result = append(result, t)
			}
			i++
		}
	}
	return result, nil
}

func (m *mockPromptTemplateRepo) CountByProject(_ context.Context, projectID uuid.UUID) (int64, error) {
	count := int64(0)
	for _, t := range m.templates {
		if t.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}

func (m *mockPromptTemplateRepo) Update(_ context.Context, tmpl *model.PromptTemplate) (*model.PromptTemplate, error) {
	m.templates[tmpl.ID] = tmpl
	return tmpl, nil
}

func (m *mockPromptTemplateRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.templates, id)
	return nil
}

func setupPromptTemplateHandler() (*PromptTemplateHandler, *mockPromptTemplateRepo) {
	repo := newMockPromptTemplateRepo()
	svc := service.NewPromptTemplateService(repo)
	h := NewPromptTemplateHandler(svc)
	return h, repo
}

func TestCreatePromptTemplate_AdminOnly(t *testing.T) {
	h, _ := setupPromptTemplateHandler()
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
			body:       `{"name":"Template 1","template_content":"content","type":"implement"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "non-admin gets 403",
			role:       model.RoleUser,
			body:       `{"name":"Template 1","template_content":"content","type":"implement"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no role gets 403",
			role:       "",
			body:       `{"name":"Template 1","template_content":"content","type":"implement"}`,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/templates",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			if tt.role != "" {
				ctx := middleware.SetUserContext(req.Context(), uuid.New(), tt.role)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.CreatePromptTemplate(rec, req, projectID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreatePromptTemplate_Validation(t *testing.T) {
	h, _ := setupPromptTemplateHandler()
	projectID := uuid.New()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid",
			body:       `{"name":"Template 1","template_content":"content","type":"implement"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "valid review type",
			body:       `{"name":"Template 2","template_content":"content","type":"review"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "valid custom type",
			body:       `{"name":"Template 3","template_content":"content","type":"custom"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty name",
			body:       `{"name":"","template_content":"content","type":"implement"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty template_content",
			body:       `{"name":"Test","template_content":"","type":"implement"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid type",
			body:       `{"name":"Test","template_content":"content","type":"invalid"}`,
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/templates",
				bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			h.CreatePromptTemplate(rec, req, projectID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreatePromptTemplate_Conflict(t *testing.T) {
	h, repo := setupPromptTemplateHandler()
	projectID := uuid.New()

	id := uuid.New()
	repo.templates[id] = &model.PromptTemplate{
		ID:              id,
		ProjectID:       projectID,
		Name:            "Existing Template",
		TemplateContent: "content",
		Type:            model.TemplateTypeImplement,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/templates",
		bytes.NewBufferString(`{"name":"Existing Template","template_content":"content","type":"implement"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.CreatePromptTemplate(rec, req, projectID)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListPromptTemplates(t *testing.T) {
	h, repo := setupPromptTemplateHandler()
	projectID := uuid.New()

	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.templates[id] = &model.PromptTemplate{
			ID:              id,
			ProjectID:       projectID,
			Name:            "tmpl-" + id.String()[:8],
			TemplateContent: "content",
			Type:            model.TemplateTypeImplement,
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/templates", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListPromptTemplates(rec, req, projectID, ListPromptTemplatesParams{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp PromptTemplateList
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Errorf("expected 3 templates, got %d", len(resp.Data))
	}
	if resp.Pagination.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Pagination.Total)
	}
}

func TestListPromptTemplates_NonAdmin(t *testing.T) {
	h, repo := setupPromptTemplateHandler()
	projectID := uuid.New()

	id := uuid.New()
	repo.templates[id] = &model.PromptTemplate{
		ID:              id,
		ProjectID:       projectID,
		Name:            "tmpl-1",
		TemplateContent: "content",
		Type:            model.TemplateTypeImplement,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/templates", nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListPromptTemplates(rec, req, projectID, ListPromptTemplatesParams{})

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for non-admin list, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetPromptTemplate_Found(t *testing.T) {
	h, repo := setupPromptTemplateHandler()
	projectID := uuid.New()
	templateID := uuid.New()
	repo.templates[templateID] = &model.PromptTemplate{
		ID:              templateID,
		ProjectID:       projectID,
		Name:            "test-template",
		TemplateContent: "test content",
		Type:            model.TemplateTypeReview,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/templates/"+templateID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetPromptTemplate(rec, req, projectID, templateID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp PromptTemplate
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "test-template" {
		t.Errorf("expected name 'test-template', got %q", resp.Name)
	}
	if resp.Type != PromptTemplateTypeReview {
		t.Errorf("expected type 'review', got %q", resp.Type)
	}
}

func TestGetPromptTemplate_NotFound(t *testing.T) {
	h, _ := setupPromptTemplateHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/templates/"+uuid.New().String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetPromptTemplate(rec, req, projectID, uuid.New())

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestUpdatePromptTemplate_AdminOnly(t *testing.T) {
	h, repo := setupPromptTemplateHandler()
	projectID := uuid.New()
	templateID := uuid.New()
	repo.templates[templateID] = &model.PromptTemplate{
		ID:              templateID,
		ProjectID:       projectID,
		Name:            "original",
		TemplateContent: "original content",
		Type:            model.TemplateTypeImplement,
	}

	// Non-admin should get 403
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/templates/"+templateID.String(),
		bytes.NewBufferString(`{"name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.UpdatePromptTemplate(rec, req, projectID, templateID)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rec.Code)
	}

	// Admin should succeed
	req = httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/templates/"+templateID.String(),
		bytes.NewBufferString(`{"name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx = middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.UpdatePromptTemplate(rec, req, projectID, templateID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdatePromptTemplate_TypeChange(t *testing.T) {
	h, repo := setupPromptTemplateHandler()
	projectID := uuid.New()
	templateID := uuid.New()
	repo.templates[templateID] = &model.PromptTemplate{
		ID:              templateID,
		ProjectID:       projectID,
		Name:            "tmpl-1",
		TemplateContent: "content",
		Type:            model.TemplateTypeImplement,
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID.String()+"/templates/"+templateID.String(),
		bytes.NewBufferString(`{"type":"review"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdatePromptTemplate(rec, req, projectID, templateID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp PromptTemplate
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Type != PromptTemplateTypeReview {
		t.Errorf("expected type 'review', got %q", resp.Type)
	}
}

func TestDeletePromptTemplate_AdminOnly(t *testing.T) {
	h, repo := setupPromptTemplateHandler()
	projectID := uuid.New()
	templateID := uuid.New()
	repo.templates[templateID] = &model.PromptTemplate{
		ID:              templateID,
		ProjectID:       projectID,
		Name:            "to-delete",
		TemplateContent: "content",
		Type:            model.TemplateTypeImplement,
	}

	// Non-admin should get 403
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/templates/"+templateID.String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.DeletePromptTemplate(rec, req, projectID, templateID)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rec.Code)
	}

	// Admin should succeed
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/templates/"+templateID.String(), nil)
	ctx = middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	h.DeletePromptTemplate(rec, req, projectID, templateID)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for admin, got %d", rec.Code)
	}
}

func TestDeletePromptTemplate_NotFound(t *testing.T) {
	h, _ := setupPromptTemplateHandler()
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID.String()+"/templates/"+uuid.New().String(), nil)
	ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleAdmin)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.DeletePromptTemplate(rec, req, projectID, uuid.New())

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
