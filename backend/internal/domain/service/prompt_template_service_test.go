package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockPromptTemplateRepo is a mock implementation of port.PromptTemplateRepository for testing.
type mockPromptTemplateRepo struct {
	templates map[uuid.UUID]*model.PromptTemplate
	createFn  func(ctx context.Context, t *model.PromptTemplate) (*model.PromptTemplate, error)
}

func newMockPromptTemplateRepo() *mockPromptTemplateRepo {
	return &mockPromptTemplateRepo{
		templates: make(map[uuid.UUID]*model.PromptTemplate),
	}
}

func (m *mockPromptTemplateRepo) Create(ctx context.Context, tmpl *model.PromptTemplate) (*model.PromptTemplate, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tmpl)
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

func TestPromptTemplateService_Create(t *testing.T) {
	projectID := uuid.New()

	tests := []struct {
		name    string
		params  CreatePromptTemplateParams
		wantErr bool
		errCode string
	}{
		{
			name: "valid template",
			params: CreatePromptTemplateParams{
				ProjectID:       projectID,
				Name:            "Test Template",
				TemplateContent: "You are an agent...",
				Type:            "implement",
			},
			wantErr: false,
		},
		{
			name: "all valid types - retry",
			params: CreatePromptTemplateParams{
				ProjectID:       projectID,
				Name:            "Retry Template",
				TemplateContent: "Retry logic...",
				Type:            "retry",
			},
			wantErr: false,
		},
		{
			name: "all valid types - review",
			params: CreatePromptTemplateParams{
				ProjectID:       projectID,
				Name:            "Review Template",
				TemplateContent: "Review code...",
				Type:            "review",
			},
			wantErr: false,
		},
		{
			name: "all valid types - merge",
			params: CreatePromptTemplateParams{
				ProjectID:       projectID,
				Name:            "Merge Template",
				TemplateContent: "Merge logic...",
				Type:            "merge",
			},
			wantErr: false,
		},
		{
			name: "all valid types - custom",
			params: CreatePromptTemplateParams{
				ProjectID:       projectID,
				Name:            "Custom Template",
				TemplateContent: "Custom logic...",
				Type:            "custom",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			params: CreatePromptTemplateParams{
				ProjectID:       projectID,
				Name:            "",
				TemplateContent: "content",
				Type:            "implement",
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "name too long",
			params: CreatePromptTemplateParams{
				ProjectID:       projectID,
				Name:            string(make([]byte, 256)),
				TemplateContent: "content",
				Type:            "implement",
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "empty template_content",
			params: CreatePromptTemplateParams{
				ProjectID:       projectID,
				Name:            "Test",
				TemplateContent: "",
				Type:            "implement",
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "missing project_id",
			params: CreatePromptTemplateParams{
				Name:            "Test",
				TemplateContent: "content",
				Type:            "implement",
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "invalid type",
			params: CreatePromptTemplateParams{
				ProjectID:       projectID,
				Name:            "Test",
				TemplateContent: "content",
				Type:            "invalid",
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name: "empty type",
			params: CreatePromptTemplateParams{
				ProjectID:       projectID,
				Name:            "Test",
				TemplateContent: "content",
				Type:            "",
			},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockPromptTemplateRepo()
			svc := NewPromptTemplateService(repo)

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
			if string(result.Type) != tt.params.Type {
				t.Errorf("expected type %q, got %q", tt.params.Type, result.Type)
			}
		})
	}
}

func TestPromptTemplateService_GetByID(t *testing.T) {
	repo := newMockPromptTemplateRepo()
	svc := NewPromptTemplateService(repo)

	id := uuid.New()
	repo.templates[id] = &model.PromptTemplate{
		ID:              id,
		Name:            "test-template",
		ProjectID:       uuid.New(),
		TemplateContent: "content",
		Type:            model.TemplateTypeImplement,
	}

	result, err := svc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "test-template" {
		t.Errorf("expected name 'test-template', got %q", result.Name)
	}

	// Get non-existent template
	_, err = svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent template, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %q", domainErr.Category)
	}
}

func TestPromptTemplateService_ListByProject(t *testing.T) {
	repo := newMockPromptTemplateRepo()
	svc := NewPromptTemplateService(repo)

	projectID := uuid.New()
	otherProjectID := uuid.New()

	for i := 0; i < 5; i++ {
		id := uuid.New()
		repo.templates[id] = &model.PromptTemplate{
			ID:              id,
			ProjectID:       projectID,
			Name:            "tmpl-" + id.String()[:8],
			TemplateContent: "content",
			Type:            model.TemplateTypeImplement,
		}
	}
	otherID := uuid.New()
	repo.templates[otherID] = &model.PromptTemplate{
		ID:              otherID,
		ProjectID:       otherProjectID,
		Name:            "other-template",
		TemplateContent: "content",
		Type:            model.TemplateTypeCustom,
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

func TestPromptTemplateService_Update(t *testing.T) {
	repo := newMockPromptTemplateRepo()
	svc := NewPromptTemplateService(repo)

	id := uuid.New()
	repo.templates[id] = &model.PromptTemplate{
		ID:              id,
		ProjectID:       uuid.New(),
		Name:            "original",
		TemplateContent: "original content",
		Type:            model.TemplateTypeImplement,
	}

	tests := []struct {
		name    string
		params  UpdatePromptTemplateParams
		wantErr bool
		errCode string
	}{
		{
			name:    "valid name update",
			params:  UpdatePromptTemplateParams{ID: id, Name: strPtr("updated")},
			wantErr: false,
		},
		{
			name:    "valid content update",
			params:  UpdatePromptTemplateParams{ID: id, TemplateContent: strPtr("new content")},
			wantErr: false,
		},
		{
			name:    "valid type update",
			params:  UpdatePromptTemplateParams{ID: id, Type: strPtr("review")},
			wantErr: false,
		},
		{
			name:    "not found",
			params:  UpdatePromptTemplateParams{ID: uuid.New(), Name: strPtr("test")},
			wantErr: true,
			errCode: "PROMPT_TEMPLATE_NOT_FOUND",
		},
		{
			name:    "empty name",
			params:  UpdatePromptTemplateParams{ID: id, Name: strPtr("")},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "name too long",
			params:  UpdatePromptTemplateParams{ID: id, Name: strPtr(string(make([]byte, 256)))},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "empty template_content",
			params:  UpdatePromptTemplateParams{ID: id, TemplateContent: strPtr("")},
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "invalid type",
			params:  UpdatePromptTemplateParams{ID: id, Type: strPtr("invalid")},
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

func TestPromptTemplateService_Delete(t *testing.T) {
	repo := newMockPromptTemplateRepo()
	svc := NewPromptTemplateService(repo)

	id := uuid.New()
	repo.templates[id] = &model.PromptTemplate{
		ID:              id,
		Name:            "to-delete",
		ProjectID:       uuid.New(),
		TemplateContent: "content",
		Type:            model.TemplateTypeImplement,
	}

	// Delete existing template
	err := svc.Delete(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's gone
	_, err = svc.GetByID(context.Background(), id)
	if err == nil {
		t.Fatal("expected not found error after delete")
	}

	// Delete non-existent template
	err = svc.Delete(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent template, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %q", domainErr.Category)
	}
}
