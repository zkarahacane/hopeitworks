package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainBinaryBuilds(t *testing.T) {
	// Build the binary
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "api")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = filepath.Dir(must(filepath.Abs(".")))
	// Run from the package directory
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\nOutput: %s", err, out)
	}

	// Verify binary exists
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Fatal("binary was not produced")
	}
}

func TestMainBinaryOutput(t *testing.T) {
	// Build the binary
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "api")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\nOutput: %s", err, out)
	}

	// Run the binary and check output
	runCmd := exec.Command(binPath)
	runOut, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("binary execution failed: %v\nOutput: %s", err, runOut)
	}

	expected := "Starting API server..."
	if got := strings.TrimSpace(string(runOut)); got != expected {
		t.Errorf("unexpected output: got %q, want %q", got, expected)
	}
}

func must(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}
