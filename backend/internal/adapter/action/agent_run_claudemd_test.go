package action

// Internal white-box tests for buildClaudeMD / buildStorySection.
// These live in package action (not action_test) to access the unexported helpers.

import (
	"strings"
	"testing"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

func strPtr2(s string) *string { return &s }

// --- buildStorySection tests ---

func TestBuildStorySection_ManualSource(t *testing.T) {
	story := &model.Story{Key: "S-01", Title: "My story", Source: string(port.SourceManual)}
	got := buildStorySection(story)

	assertContains(t, got, "## Story")
	assertContains(t, got, "S-01 — My story")
	assertContains(t, got, "Source: manual:S-01")
	assertNotContains(t, got, "Objective:")
	assertNotContains(t, got, "Scope:")
	assertNotContains(t, got, "Target files:")
	assertNotContains(t, got, "Acceptance criteria:")
}

func TestBuildStorySection_DefaultSource(t *testing.T) {
	// Empty/legacy source falls through to manual variant.
	story := &model.Story{Key: "S-02", Source: ""}
	got := buildStorySection(story)

	assertContains(t, got, "Source: manual:S-02")
}

func TestBuildStorySection_MarkdownSource(t *testing.T) {
	story := &model.Story{Key: "US-10", Title: "Auth", Source: string(port.SourceMarkdown)}
	got := buildStorySection(story)

	assertContains(t, got, "Source: markdown:US-10")
	assertNotContains(t, got, "Source: manual:")
	assertNotContains(t, got, "Source: github")
}

func TestBuildStorySection_GitHubSourceWithURL(t *testing.T) {
	url := "https://github.com/orgs/acme/projects/3/views/1?pane=issue&itemId=42"
	story := &model.Story{
		Key:       "S-05",
		Title:     "GitHub story",
		Source:    string(port.SourceGitHub),
		SourceURL: strPtr2(url),
	}
	got := buildStorySection(story)

	assertContains(t, got, "Source: github — "+url)
	assertNotContains(t, got, "Source: manual:")
	assertNotContains(t, got, "Source: markdown:")
}

func TestBuildStorySection_GitHubSourceWithoutURL(t *testing.T) {
	// SourceGitHub but no URL falls back to manual variant (nil SourceURL).
	story := &model.Story{Key: "S-06", Source: string(port.SourceGitHub), SourceURL: nil}
	got := buildStorySection(story)

	assertContains(t, got, "Source: manual:S-06")
	assertNotContains(t, got, "Source: github")
}

func TestBuildStorySection_OptionalFieldsPresent(t *testing.T) {
	story := &model.Story{
		Key:                "S-07",
		Title:              "Full story",
		Source:             string(port.SourceMarkdown),
		Objective:          strPtr2("Deliver OAuth login"),
		Scope:              strPtr2("backend"),
		TargetFiles:        []string{"auth/handler.go", "auth/service.go"},
		AcceptanceCriteria: strPtr2("- User can log in\n- Token is valid"),
	}
	got := buildStorySection(story)

	assertContains(t, got, "Objective: Deliver OAuth login")
	assertContains(t, got, "Scope: backend")
	assertContains(t, got, "Target files: auth/handler.go, auth/service.go")
	assertContains(t, got, "Acceptance criteria:")
	assertContains(t, got, "- User can log in")
}

func TestBuildStorySection_OptionalFieldsNil(t *testing.T) {
	story := &model.Story{Key: "S-08", Source: string(port.SourceManual)}
	got := buildStorySection(story)

	assertNotContains(t, got, "Objective:")
	assertNotContains(t, got, "Scope:")
	assertNotContains(t, got, "Target files:")
	assertNotContains(t, got, "Acceptance criteria:")
}

func TestBuildStorySection_EmptyStringFieldsOmitted(t *testing.T) {
	// Explicitly set to empty string — should be omitted (same as nil).
	story := &model.Story{
		Key:                "S-09",
		Source:             string(port.SourceManual),
		Objective:          strPtr2(""),
		Scope:              strPtr2(""),
		AcceptanceCriteria: strPtr2(""),
	}
	got := buildStorySection(story)

	assertNotContains(t, got, "Objective:")
	assertNotContains(t, got, "Scope:")
	assertNotContains(t, got, "Acceptance criteria:")
}

// --- buildClaudeMD integration tests (verify story section is embedded) ---

func TestBuildClaudeMD_DefaultRole_ContainsStorySection(t *testing.T) {
	project := &model.Project{Name: "myproject"}
	url := "https://github.com/orgs/acme/projects/1"
	story := &model.Story{
		Key:                "S-01",
		Title:              "Implement login",
		Source:             string(port.SourceGitHub),
		SourceURL:          strPtr2(url),
		Objective:          strPtr2("Enable user auth"),
		Scope:              strPtr2("backend"),
		TargetFiles:        []string{"auth.go"},
		AcceptanceCriteria: strPtr2("- Login works"),
	}

	got := buildClaudeMD(project, "dev", story)

	assertContains(t, got, "## Story")
	assertContains(t, got, "S-01 — Implement login")
	assertContains(t, got, "Source: github — "+url)
	assertContains(t, got, "Objective: Enable user auth")
	assertContains(t, got, "Scope: backend")
	assertContains(t, got, "Target files: auth.go")
	assertContains(t, got, "Acceptance criteria:")
	assertContains(t, got, "## Conventions")
}

func TestBuildClaudeMD_ReviewRole_ContainsStorySection(t *testing.T) {
	project := &model.Project{Name: "myproject"}
	story := &model.Story{
		Key:                "S-02",
		Title:              "Fix bug",
		Source:             string(port.SourceMarkdown),
		AcceptanceCriteria: strPtr2("- Bug is fixed"),
	}

	got := buildClaudeMD(project, "review", story)

	assertContains(t, got, "## Story")
	assertContains(t, got, "S-02 — Fix bug")
	assertContains(t, got, "Source: markdown:S-02")
	assertContains(t, got, "Acceptance criteria:")
	assertContains(t, got, "## Review checklist")
}

func TestBuildClaudeMD_ManualSource_CleanOutput(t *testing.T) {
	project := &model.Project{Name: "proj"}
	story := &model.Story{Key: "S-03", Source: string(port.SourceManual)}

	got := buildClaudeMD(project, "", story)

	assertContains(t, got, "Source: manual:S-03")
	assertNotContains(t, got, "Source: github")
	assertNotContains(t, got, "Source: markdown")
	// No optional fields present.
	assertNotContains(t, got, "Objective:")
}

// --- helpers ---

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q\ngot:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected output NOT to contain %q\ngot:\n%s", substr, s)
	}
}
