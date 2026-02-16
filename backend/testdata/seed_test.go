package testdata_test

import (
	"os"
	"strings"
	"testing"
)

const seedFile = "seed.sql"

func TestSeedFileExists(t *testing.T) {
	info, err := os.Stat(seedFile)
	if err != nil {
		t.Fatalf("seed.sql not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("seed.sql is empty")
	}
}

func TestSeedFileContainsExpectedStatements(t *testing.T) {
	data, err := os.ReadFile(seedFile)
	if err != nil {
		t.Fatalf("failed to read seed.sql: %v", err)
	}
	content := string(data)

	expectedUsers := []string{
		"admin@hopeitworks.dev",
		"dev@hopeitworks.dev",
		"alice@hopeitworks.dev",
	}
	for _, email := range expectedUsers {
		if !strings.Contains(content, email) {
			t.Errorf("seed.sql missing user insert for %s", email)
		}
	}

	expectedProjects := []string{
		"Todo App",
		"E-commerce API",
		"Frontend Kit",
	}
	for _, name := range expectedProjects {
		if !strings.Contains(content, name) {
			t.Errorf("seed.sql missing project insert for %s", name)
		}
	}
}

func TestSeedFileIsIdempotent(t *testing.T) {
	data, err := os.ReadFile(seedFile)
	if err != nil {
		t.Fatalf("failed to read seed.sql: %v", err)
	}
	content := string(data)

	inserts := 0
	onConflicts := 0
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(strings.ToUpper(line))
		if strings.HasPrefix(trimmed, "INSERT INTO") {
			inserts++
		}
		if strings.Contains(trimmed, "ON CONFLICT") {
			onConflicts++
		}
	}

	if inserts == 0 {
		t.Fatal("seed.sql contains no INSERT statements")
	}
	if onConflicts == 0 {
		t.Fatal("seed.sql contains no ON CONFLICT clauses (not idempotent)")
	}
}

func TestSeedFileContainsTransaction(t *testing.T) {
	data, err := os.ReadFile(seedFile)
	if err != nil {
		t.Fatalf("failed to read seed.sql: %v", err)
	}
	content := strings.ToUpper(string(data))

	if !strings.Contains(content, "BEGIN") {
		t.Error("seed.sql missing BEGIN (should be wrapped in transaction)")
	}
	if !strings.Contains(content, "COMMIT") {
		t.Error("seed.sql missing COMMIT (should be wrapped in transaction)")
	}
}

func TestSeedFileUsesDeterministicUUIDs(t *testing.T) {
	data, err := os.ReadFile(seedFile)
	if err != nil {
		t.Fatalf("failed to read seed.sql: %v", err)
	}
	content := string(data)

	expectedUUIDs := []string{
		"00000000-0000-0000-0000-000000000001", // admin
		"00000000-0000-0000-0000-000000000002", // dev
		"00000000-0000-0000-0000-000000000003", // alice
		"00000000-0000-0000-0000-000000000101", // Todo App
		"00000000-0000-0000-0000-000000000102", // E-commerce API
		"00000000-0000-0000-0000-000000000103", // Frontend Kit
	}
	for _, uuid := range expectedUUIDs {
		if !strings.Contains(content, uuid) {
			t.Errorf("seed.sql missing deterministic UUID %s", uuid)
		}
	}
}
