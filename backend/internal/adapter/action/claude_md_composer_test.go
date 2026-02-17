package action_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zakari/hopeitworks/backend/internal/adapter/action"
)

func setupTestClaudeMDDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFile(t, dir, "base.md", "# Base Instructions\nCommon guidelines.")
	writeFile(t, dir, "backend.md", "# Backend Instructions\nGo patterns.")
	writeFile(t, dir, "frontend.md", "# Frontend Instructions\nVue patterns.")
	writeFile(t, dir, "project.md", "# Project Context\nCurrent status.")

	return dir
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file %s: %v", name, err)
	}
}

func TestCLAUDEMDComposer_BackendScope(t *testing.T) {
	dir := setupTestClaudeMDDir(t)
	composer := action.NewCLAUDEMDComposer(dir)

	result, err := composer.Compose("backend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain all three files: base + backend + project
	if !strings.Contains(result, "# Base Instructions") {
		t.Error("expected base.md content in result")
	}
	if !strings.Contains(result, "# Backend Instructions") {
		t.Error("expected backend.md content in result")
	}
	if !strings.Contains(result, "# Project Context") {
		t.Error("expected project.md content in result")
	}
	// Should NOT contain frontend content
	if strings.Contains(result, "# Frontend Instructions") {
		t.Error("unexpected frontend.md content in result")
	}

	// Verify parts are separated by double newline
	parts := strings.Split(result, "\n\n")
	if len(parts) < 3 {
		t.Errorf("expected at least 3 parts separated by double newline, got %d", len(parts))
	}
}

func TestCLAUDEMDComposer_FrontendScope(t *testing.T) {
	dir := setupTestClaudeMDDir(t)
	composer := action.NewCLAUDEMDComposer(dir)

	result, err := composer.Compose("frontend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "# Base Instructions") {
		t.Error("expected base.md content in result")
	}
	if !strings.Contains(result, "# Frontend Instructions") {
		t.Error("expected frontend.md content in result")
	}
	if !strings.Contains(result, "# Project Context") {
		t.Error("expected project.md content in result")
	}
	if strings.Contains(result, "# Backend Instructions") {
		t.Error("unexpected backend.md content in result")
	}
}

func TestCLAUDEMDComposer_SharedScope(t *testing.T) {
	dir := setupTestClaudeMDDir(t)
	composer := action.NewCLAUDEMDComposer(dir)

	result, err := composer.Compose("shared")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "# Base Instructions") {
		t.Error("expected base.md content in result")
	}
	if !strings.Contains(result, "# Project Context") {
		t.Error("expected project.md content in result")
	}
	// No scope-specific content
	if strings.Contains(result, "# Backend Instructions") {
		t.Error("unexpected backend.md content in result")
	}
	if strings.Contains(result, "# Frontend Instructions") {
		t.Error("unexpected frontend.md content in result")
	}
}

func TestCLAUDEMDComposer_EmptyScope(t *testing.T) {
	dir := setupTestClaudeMDDir(t)
	composer := action.NewCLAUDEMDComposer(dir)

	result, err := composer.Compose("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "# Base Instructions") {
		t.Error("expected base.md content in result")
	}
	if !strings.Contains(result, "# Project Context") {
		t.Error("expected project.md content in result")
	}
	if strings.Contains(result, "# Backend Instructions") {
		t.Error("unexpected backend.md content in result")
	}
	if strings.Contains(result, "# Frontend Instructions") {
		t.Error("unexpected frontend.md content in result")
	}
}

func TestCLAUDEMDComposer_CaseInsensitive(t *testing.T) {
	dir := setupTestClaudeMDDir(t)
	composer := action.NewCLAUDEMDComposer(dir)

	result, err := composer.Compose("BACKEND")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "# Backend Instructions") {
		t.Error("expected backend.md content for uppercase scope")
	}
}

func TestCLAUDEMDComposer_MissingBaseFile(t *testing.T) {
	dir := t.TempDir()
	// Only write project.md, not base.md
	writeFile(t, dir, "project.md", "# Project")

	composer := action.NewCLAUDEMDComposer(dir)

	_, err := composer.Compose("")
	if err == nil {
		t.Fatal("expected error for missing base.md, got nil")
	}
	if !strings.Contains(err.Error(), "base.md") {
		t.Errorf("error should mention base.md, got: %v", err)
	}
}

func TestCLAUDEMDComposer_MissingProjectFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "base.md", "# Base")
	// No project.md

	composer := action.NewCLAUDEMDComposer(dir)

	_, err := composer.Compose("")
	if err == nil {
		t.Fatal("expected error for missing project.md, got nil")
	}
	if !strings.Contains(err.Error(), "project.md") {
		t.Errorf("error should mention project.md, got: %v", err)
	}
}

func TestCLAUDEMDComposer_TrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "base.md", "\n  # Base  \n\n")
	writeFile(t, dir, "project.md", "\n  # Project  \n\n")

	composer := action.NewCLAUDEMDComposer(dir)

	result, err := composer.Compose("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Each part should be trimmed
	if strings.HasPrefix(result, "\n") {
		t.Error("result should not start with newline")
	}
	if strings.HasSuffix(result, "\n\n") {
		t.Error("result should not end with double newline")
	}
}
