package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectStructureExists(t *testing.T) {
	// Find the backend root (two levels up from cmd/api/)
	backendRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to resolve backend root: %v", err)
	}

	expectedDirs := []string{
		"cmd/api",
		"internal/domain/model",
		"internal/domain/port",
		"internal/domain/service",
		"internal/adapter/postgres",
		"internal/api/handler",
		"internal/api/middleware",
		"internal/eventbus",
		"internal/config",
		"pkg/log",
		"pkg/errors",
		"pkg/exec",
		"pkg/config",
		"migrations",
		"queries",
		"testdata",
	}

	for _, dir := range expectedDirs {
		fullPath := filepath.Join(backendRoot, dir)
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			t.Errorf("directory %s does not exist", dir)
			continue
		}
		if err != nil {
			t.Errorf("error checking directory %s: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s exists but is not a directory", dir)
		}
	}
}

func TestRequiredFilesExist(t *testing.T) {
	backendRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to resolve backend root: %v", err)
	}

	expectedFiles := []string{
		"cmd/api/main.go",
		"go.mod",
		".gitignore",
	}

	for _, file := range expectedFiles {
		fullPath := filepath.Join(backendRoot, file)
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			t.Errorf("file %s does not exist", file)
			continue
		}
		if err != nil {
			t.Errorf("error checking file %s: %v", file, err)
			continue
		}
		if info.IsDir() {
			t.Errorf("%s exists but is a directory, expected file", file)
		}
	}
}
