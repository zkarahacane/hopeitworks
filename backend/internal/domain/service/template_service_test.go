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

// mockAgentRepoForTemplate implements port.AgentRepository for TemplateService tests.
type mockAgentRepoForTemplate struct {
	agents       []*model.Agent
	listMergedFn func(ctx context.Context, projectID uuid.UUID) ([]*model.Agent, error)
}

func newMockAgentRepoForTemplate() *mockAgentRepoForTemplate {
	return &mockAgentRepoForTemplate{
		agents: make([]*model.Agent, 0),
	}
}

func (m *mockAgentRepoForTemplate) CreateAgent(_ context.Context, a *model.Agent) (*model.Agent, error) {
	a.ID = uuid.New()
	m.agents = append(m.agents, a)
	return a, nil
}

func (m *mockAgentRepoForTemplate) GetAgent(_ context.Context, id uuid.UUID) (*model.Agent, error) {
	for _, a := range m.agents {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, errors.NewNotFound("agent", id)
}

func (m *mockAgentRepoForTemplate) ListAgentsByProject(_ context.Context, projectID uuid.UUID) ([]*model.Agent, error) {
	var result []*model.Agent
	for _, a := range m.agents {
		if a.ProjectID != nil && *a.ProjectID == projectID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAgentRepoForTemplate) ListGlobalAgents(_ context.Context) ([]*model.Agent, error) {
	var result []*model.Agent
	for _, a := range m.agents {
		if a.Scope == "global" {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAgentRepoForTemplate) ListAgentsByProjectMerged(ctx context.Context, projectID uuid.UUID) ([]*model.Agent, error) {
	if m.listMergedFn != nil {
		return m.listMergedFn(ctx, projectID)
	}
	var result []*model.Agent
	for _, a := range m.agents {
		if a.Scope == "global" || (a.ProjectID != nil && *a.ProjectID == projectID) {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAgentRepoForTemplate) UpdateAgent(_ context.Context, a *model.Agent) (*model.Agent, error) {
	return a, nil
}

func (m *mockAgentRepoForTemplate) DeleteAgent(_ context.Context, _ uuid.UUID) error {
	return nil
}

func templateTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestTemplateService_RenderForStory_DBTemplate(t *testing.T) {
	repo := newMockAgentRepoForTemplate()
	renderer := &mockTemplateRenderer{
		renderFn: func(templateContent string, _ *model.TemplateContext) (string, error) {
			return "rendered:" + templateContent, nil
		},
	}
	svc := NewTemplateService(repo, renderer, templateTestLogger())

	projectID := uuid.New()
	repo.agents = append(repo.agents, &model.Agent{
		ID:              uuid.New(),
		ProjectID:       &projectID,
		Name:            "implement",
		TemplateContent: "DB template content for {{story_key}}",
		Scope:           "project",
	})

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
	repo := newMockAgentRepoForTemplate() // no agents in DB
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
	repo := newMockAgentRepoForTemplate()
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
	repo := newMockAgentRepoForTemplate()
	projectID := uuid.New()
	repo.agents = append(repo.agents, &model.Agent{
		ID:              uuid.New(),
		ProjectID:       &projectID,
		Name:            "implement",
		TemplateContent: "{{#if broken",
		Scope:           "project",
	})

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
	repo := newMockAgentRepoForTemplate()
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
	repo := newMockAgentRepoForTemplate()
	repo.listMergedFn = func(_ context.Context, _ uuid.UUID) ([]*model.Agent, error) {
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
