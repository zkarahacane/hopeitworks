package service

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockTemplateRenderer implements port.TemplateRenderer for testing.
type mockTemplateRenderer struct {
	renderFn func(templateContent string, ctx *model.TemplateContext) (string, error)
}

func (m *mockTemplateRenderer) Render(templateContent string, ctx *model.TemplateContext) (string, error) {
	if m.renderFn != nil {
		return m.renderFn(templateContent, ctx)
	}
	// Default: just return the template content as-is (no substitution)
	return templateContent, nil
}

// mockTemplateRepo extends mockPromptTemplateRepo with GetByProjectAndName.
type mockTemplateRepo struct {
	templates          map[string]*model.PromptTemplate // keyed by "projectID:name"
	getByProjAndNameFn func(ctx context.Context, projectID uuid.UUID, name string) (*model.PromptTemplate, error)
}

func newMockTemplateRepo() *mockTemplateRepo {
	return &mockTemplateRepo{
		templates: make(map[string]*model.PromptTemplate),
	}
}

func (m *mockTemplateRepo) Create(_ context.Context, tmpl *model.PromptTemplate) (*model.PromptTemplate, error) {
	tmpl.ID = uuid.New()
	key := tmpl.ProjectID.String() + ":" + tmpl.Name
	m.templates[key] = tmpl
	return tmpl, nil
}

func (m *mockTemplateRepo) GetByID(_ context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	for _, t := range m.templates {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, errors.NewNotFound("prompt_template", id)
}

func (m *mockTemplateRepo) GetByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*model.PromptTemplate, error) {
	if m.getByProjAndNameFn != nil {
		return m.getByProjAndNameFn(ctx, projectID, name)
	}
	key := projectID.String() + ":" + name
	t, ok := m.templates[key]
	if !ok {
		return nil, errors.NewNotFound("prompt_template", name)
	}
	return t, nil
}

func (m *mockTemplateRepo) ListByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.PromptTemplate, error) {
	return nil, nil
}

func (m *mockTemplateRepo) CountByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockTemplateRepo) Update(_ context.Context, tmpl *model.PromptTemplate) (*model.PromptTemplate, error) {
	return tmpl, nil
}

func (m *mockTemplateRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func templateTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestTemplateService_RenderForStory_DBTemplate(t *testing.T) {
	repo := newMockTemplateRepo()
	renderer := &mockTemplateRenderer{
		renderFn: func(templateContent string, _ *model.TemplateContext) (string, error) {
			return "rendered:" + templateContent, nil
		},
	}
	svc := NewTemplateService(repo, renderer, templateTestLogger())

	projectID := uuid.New()
	tmplID := uuid.New()
	repo.templates[projectID.String()+":implement"] = &model.PromptTemplate{
		ID:              tmplID,
		ProjectID:       projectID,
		Name:            "implement",
		TemplateContent: "DB template content for {{story_key}}",
		Type:            model.TemplateTypeImplement,
	}

	tmplCtx := &model.TemplateContext{StoryKey: "S-42"}
	result, err := svc.RenderForStory(context.Background(), projectID, "implement", tmplCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "rendered:DB template content for {{story_key}}" {
		t.Errorf("expected rendered DB template, got %q", result)
	}
}

func TestTemplateService_RenderForStory_FallbackToDefault(t *testing.T) {
	repo := newMockTemplateRepo() // no templates in DB
	renderer := &mockTemplateRenderer{
		renderFn: func(templateContent string, _ *model.TemplateContext) (string, error) {
			return "rendered:" + templateContent, nil
		},
	}
	svc := NewTemplateService(repo, renderer, templateTestLogger())

	projectID := uuid.New()
	tmplCtx := &model.TemplateContext{StoryKey: "S-42"}

	result, err := svc.RenderForStory(context.Background(), projectID, "implement", tmplCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(result, "rendered:") {
		t.Errorf("expected rendered default template, got %q", result)
	}
	if !strings.Contains(result, "Implement story") {
		t.Errorf("expected default implement template content, got %q", result)
	}
}

func TestTemplateService_RenderForStory_UnknownTemplate(t *testing.T) {
	repo := newMockTemplateRepo()
	renderer := &mockTemplateRenderer{}
	svc := NewTemplateService(repo, renderer, templateTestLogger())

	projectID := uuid.New()
	tmplCtx := &model.TemplateContext{StoryKey: "S-42"}

	_, err := svc.RenderForStory(context.Background(), projectID, "nonexistent-template", tmplCtx)
	if err == nil {
		t.Fatal("expected error for unknown template, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "TEMPLATE_NOT_FOUND" {
		t.Errorf("expected error code TEMPLATE_NOT_FOUND, got %q", domainErr.Code)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %q", domainErr.Category)
	}
}

func TestTemplateService_RenderForStory_RendererError(t *testing.T) {
	repo := newMockTemplateRepo()
	projectID := uuid.New()
	tmplID := uuid.New()
	repo.templates[projectID.String()+":implement"] = &model.PromptTemplate{
		ID:              tmplID,
		ProjectID:       projectID,
		Name:            "implement",
		TemplateContent: "{{#if broken",
		Type:            model.TemplateTypeImplement,
	}

	renderer := &mockTemplateRenderer{
		renderFn: func(_ string, _ *model.TemplateContext) (string, error) {
			return "", &errors.DomainError{
				Category: errors.CategoryValidation,
				Code:     "TEMPLATE_RENDER_FAILED",
				Message:  "failed to render template: parse error",
			}
		},
	}
	svc := NewTemplateService(repo, renderer, templateTestLogger())

	tmplCtx := &model.TemplateContext{StoryKey: "S-42"}
	_, err := svc.RenderForStory(context.Background(), projectID, "implement", tmplCtx)
	if err == nil {
		t.Fatal("expected error from renderer, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "TEMPLATE_RENDER_FAILED" {
		t.Errorf("expected error code TEMPLATE_RENDER_FAILED, got %q", domainErr.Code)
	}
}

func TestTemplateService_RenderForStory_AllDefaultTemplates(t *testing.T) {
	repo := newMockTemplateRepo()
	renderer := &mockTemplateRenderer{
		renderFn: func(templateContent string, _ *model.TemplateContext) (string, error) {
			return templateContent, nil
		},
	}
	svc := NewTemplateService(repo, renderer, templateTestLogger())

	projectID := uuid.New()
	tmplCtx := &model.TemplateContext{StoryKey: "S-42"}

	defaultNames := []struct {
		name     string
		contains string
	}{
		{TemplateNameImplement, "Implement story"},
		{TemplateNameImplementRetry, "Retry implementation"},
		{TemplateNameReview, "Review changes"},
		{TemplateNameMerge, "Merge changes"},
		{TemplateNameMergeConflict, "Resolve merge conflict"},
	}

	for _, tc := range defaultNames {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.RenderForStory(context.Background(), projectID, tc.name, tmplCtx)
			if err != nil {
				t.Fatalf("unexpected error for default template %q: %v", tc.name, err)
			}
			if !strings.Contains(result, tc.contains) {
				t.Errorf("expected default template %q to contain %q, got:\n%s", tc.name, tc.contains, result)
			}
		})
	}
}

func TestTemplateService_RenderForStory_RepoInternalError(t *testing.T) {
	repo := newMockTemplateRepo()
	repo.getByProjAndNameFn = func(_ context.Context, _ uuid.UUID, _ string) (*model.PromptTemplate, error) {
		return nil, errors.NewInternal("database connection lost", nil)
	}
	renderer := &mockTemplateRenderer{}
	svc := NewTemplateService(repo, renderer, templateTestLogger())

	projectID := uuid.New()
	tmplCtx := &model.TemplateContext{StoryKey: "S-42"}

	_, err := svc.RenderForStory(context.Background(), projectID, "implement", tmplCtx)
	if err == nil {
		t.Fatal("expected error from repo internal error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryInternal {
		t.Errorf("expected internal category, got %q", domainErr.Category)
	}
}
