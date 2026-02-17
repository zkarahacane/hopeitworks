// Package testutil provides shared test infrastructure for integration tests.
// It includes helpers for spinning up testcontainers-based Postgres instances,
// applying migrations, and creating test data factories.
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDB holds a shared test database context for integration tests.
type TestDB struct {
	Pool       *pgxpool.Pool
	ConnString string
	Cleanup    func()
}

// SetupTestDB spins up a Postgres container, creates a connection pool,
// and applies all migrations. Call Cleanup() when done.
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	applyMigrations(t, pool)

	return &TestDB{
		Pool:       pool,
		ConnString: connStr,
		Cleanup: func() {
			pool.Close()
			if err := pgContainer.Terminate(ctx); err != nil {
				t.Logf("failed to terminate container: %v", err)
			}
		},
	}
}

// applyMigrations reads and applies all .up.sql migration files.
func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	migrationsDir := migrationsPath()
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir %s: %v", migrationsDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		if len(entry.Name()) > 7 && entry.Name()[len(entry.Name())-7:] != ".up.sql" {
			continue
		}

		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", entry.Name(), err)
		}

		_, err = pool.Exec(ctx, string(sqlBytes))
		if err != nil {
			t.Fatalf("failed to apply migration %s: %v", entry.Name(), err)
		}
	}
}

// migrationsPath returns the absolute path to the backend/migrations directory.
func migrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
}
