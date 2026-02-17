package markdown

import (
	"testing"
)

const testKeyS03 = "S-03"

func TestParseStoryMarkdown_SingleStory(t *testing.T) {
	content := `---
key: S-03
epic: E-01
depends_on:
  - S-01
  - S-02
scope: backend
status: backlog
---
# Story Title Here

Acceptance criteria and other body text here.
`
	stories := ParseStoryMarkdown(content)

	if len(stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(stories))
	}

	s := stories[0]
	if s.ParseError != nil {
		t.Fatalf("unexpected parse error: %v", s.ParseError)
	}
	if s.Key != testKeyS03 {
		t.Errorf("expected key %q, got %q", testKeyS03, s.Key)
	}
	if s.Title != "Story Title Here" {
		t.Errorf("expected title 'Story Title Here', got %q", s.Title)
	}
	if s.Epic != "E-01" {
		t.Errorf("expected epic 'E-01', got %q", s.Epic)
	}
	if len(s.DependsOn) != 2 || s.DependsOn[0] != "S-01" || s.DependsOn[1] != "S-02" {
		t.Errorf("expected depends_on [S-01, S-02], got %v", s.DependsOn)
	}
	if s.Scope != "backend" {
		t.Errorf("expected scope 'backend', got %q", s.Scope)
	}
	if s.Status != "backlog" {
		t.Errorf("expected status 'backlog', got %q", s.Status)
	}
	if s.AcceptanceCriteria != "Acceptance criteria and other body text here." {
		t.Errorf("expected acceptance criteria 'Acceptance criteria and other body text here.', got %q", s.AcceptanceCriteria)
	}
}

func TestParseStoryMarkdown_MultiStory(t *testing.T) {
	content := `---
key: S-03
scope: backend
---
# First Story

Body of first story.

---
key: S-04
scope: frontend
depends_on:
  - S-03
---
# Second Story

Body of second story.
`
	stories := ParseStoryMarkdown(content)

	if len(stories) != 2 {
		t.Fatalf("expected 2 stories, got %d", len(stories))
	}

	if stories[0].Key != testKeyS03 {
		t.Errorf("first story: expected key %q, got %q", testKeyS03, stories[0].Key)
	}
	if stories[0].Title != "First Story" {
		t.Errorf("first story: expected title 'First Story', got %q", stories[0].Title)
	}
	if stories[0].AcceptanceCriteria != "Body of first story." {
		t.Errorf("first story: expected acceptance criteria 'Body of first story.', got %q", stories[0].AcceptanceCriteria)
	}

	if stories[1].Key != "S-04" {
		t.Errorf("second story: expected key 'S-04', got %q", stories[1].Key)
	}
	if stories[1].Title != "Second Story" {
		t.Errorf("second story: expected title 'Second Story', got %q", stories[1].Title)
	}
	if len(stories[1].DependsOn) != 1 || stories[1].DependsOn[0] != testKeyS03 {
		t.Errorf("second story: expected depends_on [%s], got %v", testKeyS03, stories[1].DependsOn)
	}
}

func TestParseStoryMarkdown_InvalidYAML(t *testing.T) {
	content := `---
key: S-05
scope: [invalid yaml
---
# Invalid Story

Body text.
`
	stories := ParseStoryMarkdown(content)

	if len(stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(stories))
	}

	if stories[0].ParseError == nil {
		t.Fatal("expected parse error for invalid YAML, got nil")
	}
}

func TestParseStoryMarkdown_MissingTitle(t *testing.T) {
	content := `---
key: S-06
scope: backend
---

No heading here, just body text.
`
	stories := ParseStoryMarkdown(content)

	if len(stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(stories))
	}

	if stories[0].Title != "" {
		t.Errorf("expected empty title, got %q", stories[0].Title)
	}
	if stories[0].Key != "S-06" {
		t.Errorf("expected key 'S-06', got %q", stories[0].Key)
	}
}

func TestParseStoryMarkdown_OnlyKeyInFrontmatter(t *testing.T) {
	content := `---
key: S-07
---
# Minimal Story
`
	stories := ParseStoryMarkdown(content)

	if len(stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(stories))
	}

	s := stories[0]
	if s.ParseError != nil {
		t.Fatalf("unexpected parse error: %v", s.ParseError)
	}
	if s.Key != "S-07" {
		t.Errorf("expected key 'S-07', got %q", s.Key)
	}
	if s.Title != "Minimal Story" {
		t.Errorf("expected title 'Minimal Story', got %q", s.Title)
	}
	if s.Epic != "" {
		t.Errorf("expected empty epic, got %q", s.Epic)
	}
	if len(s.DependsOn) != 0 {
		t.Errorf("expected empty depends_on, got %v", s.DependsOn)
	}
	if s.Scope != "" {
		t.Errorf("expected empty scope, got %q", s.Scope)
	}
	if s.Status != "" {
		t.Errorf("expected empty status, got %q", s.Status)
	}
}

func TestParseStoryMarkdown_NoFrontmatterDelimiters(t *testing.T) {
	content := `Just some plain text without any frontmatter.

# Not a story

This is not a story block.
`
	stories := ParseStoryMarkdown(content)

	if len(stories) != 0 {
		t.Fatalf("expected 0 stories for content without frontmatter, got %d", len(stories))
	}
}

func TestParseStoryMarkdown_DependsOnAsList(t *testing.T) {
	content := `---
key: S-08
depends_on:
  - S-01
  - S-02
  - S-03
---
# Dependencies Story

Has multiple dependencies.
`
	stories := ParseStoryMarkdown(content)

	if len(stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(stories))
	}

	if len(stories[0].DependsOn) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(stories[0].DependsOn))
	}
	expected := []string{"S-01", "S-02", "S-03"}
	for i, dep := range stories[0].DependsOn {
		if dep != expected[i] {
			t.Errorf("expected dependency[%d] = %q, got %q", i, expected[i], dep)
		}
	}
}

func TestParseStoryMarkdown_EmptyContent(t *testing.T) {
	stories := ParseStoryMarkdown("")

	if len(stories) != 0 {
		t.Fatalf("expected 0 stories for empty content, got %d", len(stories))
	}
}

func TestParseStoryMarkdown_MixedValidAndInvalid(t *testing.T) {
	content := `---
key: S-10
scope: backend
---
# Valid Story

This is valid.

---
key: S-11
scope: [broken
---
# Invalid Story

This has bad YAML.

---
key: S-12
scope: frontend
---
# Another Valid Story

This is also valid.
`
	stories := ParseStoryMarkdown(content)

	if len(stories) != 3 {
		t.Fatalf("expected 3 stories, got %d", len(stories))
	}

	if stories[0].ParseError != nil {
		t.Errorf("first story should be valid, got error: %v", stories[0].ParseError)
	}
	if stories[0].Key != "S-10" {
		t.Errorf("first story key should be 'S-10', got %q", stories[0].Key)
	}

	if stories[1].ParseError == nil {
		t.Error("second story should have parse error")
	}

	if stories[2].ParseError != nil {
		t.Errorf("third story should be valid, got error: %v", stories[2].ParseError)
	}
	if stories[2].Key != "S-12" {
		t.Errorf("third story key should be 'S-12', got %q", stories[2].Key)
	}
}
