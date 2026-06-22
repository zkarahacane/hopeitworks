package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Clear any environment variables that could interfere with the test
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_SSLMODE", "")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("LOG_LEVEL", "")

	yaml := `
server:
  port: 8080
  read_timeout: 15s
  write_timeout: 15s

database:
  host: localhost
  port: 5432
  name: testdb
  user: testuser
  password: testpass
  sslmode: disable
  max_conns: 10
  min_conns: 2
  max_conn_lifetime: 30m

logging:
  level: info
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "localhost")
	}
	if cfg.Database.Name != "testdb" {
		t.Errorf("Database.Name = %q, want %q", cfg.Database.Name, "testdb")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	yaml := `
server:
  port: 8080

database:
  host: localhost
  port: 5432
  name: testdb
  user: testuser
  password: testpass
  sslmode: disable

logging:
  level: info
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SERVER_PORT", "9000")
	t.Setenv("DB_HOST", "remotehost")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != 9000 {
		t.Errorf("Server.Port = %d, want 9000", cfg.Server.Port)
	}
	if cfg.Database.Host != "remotehost" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "remotehost")
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "debug")
	}
}

func TestLoad_ValidationFails(t *testing.T) {
	// Clear any environment variables that could provide values and prevent validation failure
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_SSLMODE", "")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("LOG_LEVEL", "")

	yaml := `
server:
  port: 8080

database:
  host: ""
  port: 5432
  name: ""
  user: ""
  password: ""

logging:
  level: info
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() should have returned an error for missing required fields")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("Load() should have returned an error for missing file")
	}
}

func TestLoad_StacksCatalogue(t *testing.T) {
	yaml := `
database:
  host: localhost
  port: 5432
  name: testdb
  user: testuser
  password: testpass
  sslmode: disable

logging:
  level: info

stacks:
  - key: go
    image_ref: ghcr.io/x/agent-go:latest
    toolchain:
      go: "1.23"
      cli: ["claude", "opencode"]
  - key: node
    image_ref: ghcr.io/x/agent-node:latest
    toolchain:
      node: "22"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.Stacks) != 2 {
		t.Fatalf("Stacks len = %d, want 2", len(cfg.Stacks))
	}
	if cfg.Stacks[0].Key != "go" || cfg.Stacks[0].ImageRef != "ghcr.io/x/agent-go:latest" {
		t.Errorf("Stacks[0] = %+v, unexpected key/image_ref", cfg.Stacks[0])
	}
	if cfg.Stacks[0].Toolchain["go"] != "1.23" {
		t.Errorf("Stacks[0].Toolchain[go] = %v, want 1.23", cfg.Stacks[0].Toolchain["go"])
	}
	cli, ok := cfg.Stacks[0].Toolchain["cli"].([]any)
	if !ok || len(cli) != 2 {
		t.Errorf("Stacks[0].Toolchain[cli] = %v, want 2-element list", cfg.Stacks[0].Toolchain["cli"])
	}
}

func TestLoad_NoStacksSection(t *testing.T) {
	yaml := `
database:
  host: localhost
  port: 5432
  name: testdb
  user: testuser
  password: testpass
  sslmode: disable

logging:
  level: info
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Stacks) != 0 {
		t.Errorf("Stacks len = %d, want 0 when section absent", len(cfg.Stacks))
	}
}
