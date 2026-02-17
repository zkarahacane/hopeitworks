package postgres_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

func createTestStory(t *testing.T, pool *pgxpool.Pool, projectID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	storyID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO stories (id, project_id, key, title, status, acceptance_criteria)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		storyID, projectID, "S-"+storyID.String()[:4], "Test Story", "backlog", "Test criteria",
	)
	if err != nil {
		t.Fatalf("failed to create test story: %v", err)
	}
	return storyID
}

func TestIntegration_RunRepo_GetRunWithProgress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	queries := postgres.New(db.pool)
	runRepo := postgres.NewRunRepo(queries)

	projectID := createTestProject(t, db.pool)
	storyID := createTestStory(t, db.pool, projectID)

	// Create a run
	run, err := runRepo.CreateRun(ctx, &model.Run{
		ProjectID:              projectID,
		StoryID:                storyID,
		Status:                 model.RunStatusRunning,
		PipelineConfigSnapshot: json.RawMessage(`{"steps":[{"name":"dev","action":"code"},{"name":"review","action":"review"}]}`),
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	// Create steps with mixed statuses
	step1, err := runRepo.CreateRunStep(ctx, &model.RunStep{
		RunID:     run.ID,
		StepName:  "dev",
		StepOrder: 0,
		Action:    "code",
		Status:    model.StepStatusPending,
	})
	if err != nil {
		t.Fatalf("CreateRunStep() error = %v", err)
	}

	// Transition step1 to completed
	_, err = runRepo.UpdateRunStepStatus(ctx, step1.ID, model.StepStatusCompleted, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateRunStepStatus() error = %v", err)
	}

	_, err = runRepo.CreateRunStep(ctx, &model.RunStep{
		RunID:     run.ID,
		StepName:  "review",
		StepOrder: 1,
		Action:    "review",
		Status:    model.StepStatusPending,
	})
	if err != nil {
		t.Fatalf("CreateRunStep() error = %v", err)
	}

	// Use the service to GetRun (which computes progress)
	svc := service.NewRunService(runRepo, nil, nil, nil, nil)
	result, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}

	if result.Progress != 50 {
		t.Errorf("expected progress 50, got %d", result.Progress)
	}
	if len(result.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(result.Steps))
	}
}

func TestIntegration_RunRepo_ListRunsByProjectWithProgress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	queries := postgres.New(db.pool)
	runRepo := postgres.NewRunRepo(queries)

	projectID := createTestProject(t, db.pool)
	storyID := createTestStory(t, db.pool, projectID)

	// Create 3 runs with different step statuses
	for i := 0; i < 3; i++ {
		run, err := runRepo.CreateRun(ctx, &model.Run{
			ProjectID:              projectID,
			StoryID:                storyID,
			Status:                 model.RunStatusRunning,
			PipelineConfigSnapshot: json.RawMessage(`{"steps":[{"name":"dev","action":"code"}]}`),
		})
		if err != nil {
			t.Fatalf("CreateRun(%d) error = %v", i, err)
		}

		step, err := runRepo.CreateRunStep(ctx, &model.RunStep{
			RunID:     run.ID,
			StepName:  "dev",
			StepOrder: 0,
			Action:    "code",
			Status:    model.StepStatusPending,
		})
		if err != nil {
			t.Fatalf("CreateRunStep(%d) error = %v", i, err)
		}

		// Complete the step for the last run only
		if i == 2 {
			_, err = runRepo.UpdateRunStepStatus(ctx, step.ID, model.StepStatusCompleted, nil, nil, nil)
			if err != nil {
				t.Fatalf("UpdateRunStepStatus(%d) error = %v", i, err)
			}
		}
	}

	svc := service.NewRunService(runRepo, nil, nil, nil, nil)
	result, err := svc.ListRunsByProject(ctx, projectID, 1, 20)
	if err != nil {
		t.Fatalf("ListRunsByProject() error = %v", err)
	}

	if len(result.Runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(result.Runs))
	}
	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}

	// Exactly 1 run should have progress 100 (the completed one) and 2 should have 0.
	completedCount := 0
	for _, r := range result.Runs {
		if r.Progress != 0 && r.Progress != 100 {
			t.Errorf("expected progress 0 or 100, got %d for run %s", r.Progress, r.ID)
		}
		if r.Progress == 100 {
			completedCount++
		}
	}
	if completedCount != 1 {
		t.Errorf("expected exactly 1 run with progress 100, got %d", completedCount)
	}
}

func TestIntegration_RunService_GetRunProgress_66Percent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	queries := postgres.New(db.pool)
	runRepo := postgres.NewRunRepo(queries)

	projectID := createTestProject(t, db.pool)
	storyID := createTestStory(t, db.pool, projectID)

	// Create run with 3 steps (2 completed, 1 pending) → expect 66%
	run, err := runRepo.CreateRun(ctx, &model.Run{
		ProjectID:              projectID,
		StoryID:                storyID,
		Status:                 model.RunStatusRunning,
		PipelineConfigSnapshot: json.RawMessage(`{"steps":[{"name":"a","action":"x"},{"name":"b","action":"y"},{"name":"c","action":"z"}]}`),
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	for i, name := range []string{"a", "b", "c"} {
		step, err := runRepo.CreateRunStep(ctx, &model.RunStep{
			RunID:     run.ID,
			StepName:  name,
			StepOrder: i,
			Action:    "action",
			Status:    model.StepStatusPending,
		})
		if err != nil {
			t.Fatalf("CreateRunStep() error = %v", err)
		}
		// Complete first 2 steps
		if i < 2 {
			_, err = runRepo.UpdateRunStepStatus(ctx, step.ID, model.StepStatusCompleted, nil, nil, nil)
			if err != nil {
				t.Fatalf("UpdateRunStepStatus() error = %v", err)
			}
		}
	}

	svc := service.NewRunService(runRepo, nil, nil, nil, nil)
	result, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}

	if result.Progress != 66 {
		t.Errorf("expected progress 66, got %d", result.Progress)
	}
	if len(result.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(result.Steps))
	}
}
