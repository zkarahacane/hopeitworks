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

// TestIntegration_RunRepo_ListRunsByProject_Cost covers the cost aggregation added
// to ListRunsByProject for #290:
//   - RG1: a completed run's listed cost equals SumCostByRun (same source of truth).
//   - RG2: a run with no cost record reports nil (distinct from a real $0.00).
//   - RG3: a multi-step run reports the sum across all its steps.
//   - RG5: a running run reports its cumulative cost so far (no status special-case).
//   - RG4: every run's cost arrives from the single ListRunsByProject query — the
//     test issues exactly one repo list call and reads cost off each row, with no
//     per-run cost fetch.
func TestIntegration_RunRepo_ListRunsByProject_Cost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	queries := postgres.New(db.pool)
	runRepo := postgres.NewRunRepo(queries)
	costRepo := postgres.NewCostRepo(queries)

	projectID := createTestProject(t, db.pool)

	// Run A — completed, two steps with cost records → RG1 + RG3.
	storyA := createTestStory(t, db.pool, projectID)
	runA := createTestRun(t, runRepo, projectID, storyA)
	stepA1 := createTestRunStep(t, runRepo, runA)
	stepA2 := createTestRunStep(t, runRepo, runA)
	insertCost(ctx, t, costRepo, stepA1, projectID, 0.5000)
	insertCost(ctx, t, costRepo, stepA2, projectID, 0.3145)

	// Run B — no cost record at all → RG2.
	storyB := createTestStory(t, db.pool, projectID)
	runB := createTestRun(t, runRepo, projectID, storyB)
	_ = createTestRunStep(t, runRepo, runB) // step without any cost record

	// Run C — still running, one cost record so far → RG5.
	storyC := createTestStory(t, db.pool, projectID)
	runC := createTestRun(t, runRepo, projectID, storyC) // createTestRun creates with status running
	stepC1 := createTestRunStep(t, runRepo, runC)
	insertCost(ctx, t, costRepo, stepC1, projectID, 0.2000)

	// RG4: a single list call returns every run carrying its own cost — no N+1.
	runs, err := runRepo.ListRunsByProject(ctx, projectID, 50, 0)
	if err != nil {
		t.Fatalf("ListRunsByProject() error = %v", err)
	}
	costByRun := map[uuid.UUID]*float64{}
	for _, r := range runs {
		costByRun[r.ID] = r.CostUSD
	}

	const epsilon = 0.000001

	// RG3: run A is the sum of both steps (0.5000 + 0.3145).
	gotA := costByRun[runA]
	if gotA == nil {
		t.Fatalf("RG3: expected run A to report a cost, got nil")
	}
	if diff := *gotA - 0.8145; diff > epsilon || diff < -epsilon {
		t.Errorf("RG3: expected run A cost 0.8145, got %f", *gotA)
	}

	// RG1: the listed cost matches SumCostByRun, the source used by the run detail.
	sumA, err := costRepo.SumCostByRun(ctx, runA)
	if err != nil {
		t.Fatalf("SumCostByRun() error = %v", err)
	}
	if diff := *gotA - sumA; diff > epsilon || diff < -epsilon {
		t.Errorf("RG1: listed cost %f != detail SumCostByRun %f", *gotA, sumA)
	}

	// RG2: run B has no cost record → nil (not a 0).
	gotB, ok := costByRun[runB]
	if !ok {
		t.Fatalf("RG2: expected run B in the list")
	}
	if gotB != nil {
		t.Errorf("RG2: expected run B cost nil, got %f", *gotB)
	}

	// RG5: the running run C reports its cumulative cost so far.
	gotC := costByRun[runC]
	if gotC == nil {
		t.Fatalf("RG5: expected running run C to report a cost, got nil")
	}
	if diff := *gotC - 0.2000; diff > epsilon || diff < -epsilon {
		t.Errorf("RG5: expected run C cost 0.2000, got %f", *gotC)
	}
}

// insertCost is a test helper that records a single cost for a run step.
func insertCost(ctx context.Context, t *testing.T, costRepo *postgres.CostRepo, stepID, projectID uuid.UUID, costUSD float64) {
	t.Helper()
	_, err := costRepo.InsertCostRecord(ctx, &model.CostRecord{
		RunStepID:    stepID,
		ProjectID:    projectID,
		TokensInput:  1000,
		TokensOutput: 500,
		CostUSD:      costUSD,
		Model:        "claude-sonnet-4-6",
	})
	if err != nil {
		t.Fatalf("InsertCostRecord() error = %v", err)
	}
}
