package handlebars

import (
	"strings"
	"testing"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

func TestRenderer_Render_AllVariables(t *testing.T) {
	r := NewRenderer()

	ctx := &model.TemplateContext{
		StoryKey:           "S-42",
		StoryTitle:         "Add user auth",
		StoryObjective:     "Implement JWT-based authentication",
		TargetFiles:        []string{"auth.go", "middleware.go"},
		AcceptanceCriteria: "Users can log in with valid credentials",
		ErrorContext:       "Previous attempt failed with timeout",
		DiffContent:        "+func Login() {}",
		BranchName:         "feat/auth",
		RepoURL:            "https://github.com/example/repo",
	}

	tmpl := `Story: {{story_key}} - {{story_title}}
Objective: {{story_objective}}
Branch: {{branch_name}}
Repo: {{repo_url}}
Criteria: {{acceptance_criteria}}
Error: {{error_context}}
Diff: {{diff_content}}`

	result, err := r.Render(tmpl, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectations := []string{
		"Story: S-42 - Add user auth",
		"Objective: Implement JWT-based authentication",
		"Branch: feat/auth",
		"Repo: https://github.com/example/repo",
		"Criteria: Users can log in with valid credentials",
		"Error: Previous attempt failed with timeout",
		"Diff: +func Login() {}",
	}

	for _, expected := range expectations {
		if !strings.Contains(result, expected) {
			t.Errorf("expected result to contain %q, got:\n%s", expected, result)
		}
	}
}

func TestRenderer_Render_EachLoop(t *testing.T) {
	r := NewRenderer()

	ctx := &model.TemplateContext{
		StoryKey:    "S-10",
		TargetFiles: []string{"file1.go", "file2.go", "file3.go"},
	}

	tmpl := `Files:
{{#each target_files}}
- {{this}}
{{/each}}`

	result, err := r.Render(tmpl, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, file := range ctx.TargetFiles {
		if !strings.Contains(result, "- "+file) {
			t.Errorf("expected result to contain '- %s', got:\n%s", file, result)
		}
	}
}

func TestRenderer_Render_EmptyTargetFiles(t *testing.T) {
	r := NewRenderer()

	ctx := &model.TemplateContext{
		StoryKey:    "S-10",
		TargetFiles: nil,
	}

	tmpl := `Key: {{story_key}}
{{#each target_files}}
- {{this}}
{{/each}}
Done`

	result, err := r.Render(tmpl, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Key: S-10") {
		t.Errorf("expected result to contain 'Key: S-10', got:\n%s", result)
	}
	if !strings.Contains(result, "Done") {
		t.Errorf("expected result to contain 'Done', got:\n%s", result)
	}
}

func TestRenderer_Render_MissingVariables(t *testing.T) {
	r := NewRenderer()

	ctx := &model.TemplateContext{
		StoryKey: "S-42",
	}

	tmpl := `Key: {{story_key}}, Title: {{story_title}}, Objective: {{story_objective}}`

	result, err := r.Render(tmpl, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Key: S-42") {
		t.Errorf("expected result to contain 'Key: S-42', got:\n%s", result)
	}
	// Missing variables should render as empty strings
	if !strings.Contains(result, "Title: , Objective: ") {
		t.Errorf("expected missing variables to render as empty strings, got:\n%s", result)
	}
}

func TestRenderer_Render_InvalidSyntax(t *testing.T) {
	r := NewRenderer()

	ctx := &model.TemplateContext{
		StoryKey: "S-42",
	}

	tmpl := `{{#if story_key}}open but not closed`

	_, err := r.Render(tmpl, ctx)
	if err == nil {
		t.Fatal("expected error for invalid Handlebars syntax, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "TEMPLATE_RENDER_FAILED" {
		t.Errorf("expected error code TEMPLATE_RENDER_FAILED, got %q", domainErr.Code)
	}
}

func TestRenderer_Render_SpecialCharacters(t *testing.T) {
	r := NewRenderer()

	ctx := &model.TemplateContext{
		StoryKey:       "S-42",
		StoryTitle:     `Title with "quotes" & <angle brackets>`,
		StoryObjective: "Line1\nLine2\tTabbed",
	}

	// Use triple-stache to avoid HTML escaping
	tmpl := `Key: {{story_key}}, Title: {{{story_title}}}, Objective: {{{story_objective}}}`

	result, err := r.Render(tmpl, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, `Title with "quotes" & <angle brackets>`) {
		t.Errorf("expected unescaped special characters in result, got:\n%s", result)
	}
	if !strings.Contains(result, "Line1\nLine2\tTabbed") {
		t.Errorf("expected newlines and tabs preserved, got:\n%s", result)
	}
}

func TestRenderer_Render_HTMLEscaping(t *testing.T) {
	r := NewRenderer()

	ctx := &model.TemplateContext{
		StoryTitle: `<script>alert("xss")</script>`,
	}

	// Double-stache should HTML-escape by default
	tmpl := `Title: {{story_title}}`

	result, err := r.Render(tmpl, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "<script>") {
		t.Errorf("expected HTML-escaped output, got:\n%s", result)
	}
}

func TestRenderer_Render_EmptyTemplate(t *testing.T) {
	r := NewRenderer()

	ctx := &model.TemplateContext{StoryKey: "S-42"}

	result, err := r.Render("", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result for empty template, got %q", result)
	}
}
