package testutil

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateProject inserts a minimal project row and returns its ID.
func CreateProject(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	return CreateProjectWithName(t, pool, "test-project-"+uuid.New().String()[:8])
}

// CreateProjectWithName inserts a project with a specific name and returns its ID.
func CreateProjectWithName(t *testing.T, pool *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	projectID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO projects (id, name, git_provider, agent_runtime, repo_url)
		 VALUES ($1, $2, $3, $4, $5)`,
		projectID, name, "github", "docker", "https://github.com/test/test-project",
	)
	if err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}
	return projectID
}

// CreateEpic inserts a minimal epic row and returns its ID.
func CreateEpic(t *testing.T, pool *pgxpool.Pool, projectID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	epicID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO epics (id, project_id, name, status)
		 VALUES ($1, $2, $3, $4)`,
		epicID, projectID, name, "backlog",
	)
	if err != nil {
		t.Fatalf("failed to create test epic: %v", err)
	}
	return epicID
}

// UpsertPipelineConfig inserts or updates a pipeline config for a project.
func UpsertPipelineConfig(t *testing.T, pool *pgxpool.Pool, projectID uuid.UUID, configYAML string) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	configID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO pipeline_configs (id, project_id, config_yaml, version)
		 VALUES ($1, $2, $3, 1)
		 ON CONFLICT (project_id) DO UPDATE SET config_yaml = EXCLUDED.config_yaml, version = pipeline_configs.version + 1`,
		configID, projectID, configYAML,
	)
	if err != nil {
		t.Fatalf("failed to upsert pipeline config: %v", err)
	}
	return configID
}
