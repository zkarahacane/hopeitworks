package postgres_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// createTestRunStep inserts a run step row for FK constraint satisfaction.
func createTestRunStep(t *testing.T, runRepo *postgres.RunRepo, runID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	step, err := runRepo.CreateRunStep(ctx, &model.RunStep{
		RunID:     runID,
		StepName:  "test-step",
		StepOrder: 0,
		Action:    "agent_run",
		Status:    model.StepStatusPending,
	})
	if err != nil {
		t.Fatalf("failed to create test run step: %v", err)
	}
	return step.ID
}

// createTestRun inserts a run row for FK constraint satisfaction.
func createTestRun(t *testing.T, runRepo *postgres.RunRepo, projectID, storyID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	run, err := runRepo.CreateRun(ctx, &model.Run{
		ProjectID:              projectID,
		StoryID:                storyID,
		Status:                 model.RunStatusRunning,
		PipelineConfigSnapshot: json.RawMessage(`{"steps":[]}`),
	})
	if err != nil {
		t.Fatalf("failed to create test run: %v", err)
	}
	return run.ID
}

func TestIntegration_CostRepo_InsertAndGetByRunStep(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	queries := postgres.New(db.pool)
	costRepo := postgres.NewCostRepo(queries)
	runRepo := postgres.NewRunRepo(queries)

	projectID := createTestProject(t, db.pool)
	storyID := createTestStory(t, db.pool, projectID)
	runID := createTestRun(t, runRepo, projectID, storyID)
	stepID := createTestRunStep(t, runRepo, runID)

	record := &model.CostRecord{
		RunStepID:    stepID,
		ProjectID:    projectID,
		TokensInput:  1_000_000,
		TokensOutput: 500_000,
		CostUSD:      22.5,
		Model:        "claude-opus-4-6",
	}

	// Insert
	inserted, err := costRepo.InsertCostRecord(ctx, record)
	if err != nil {
		t.Fatalf("InsertCostRecord() error = %v", err)
	}
	if inserted.ID == uuid.Nil {
		t.Error("expected non-nil ID after insert")
	}
	if inserted.RunStepID != stepID {
		t.Errorf("expected run_step_id %v, got %v", stepID, inserted.RunStepID)
	}
	if inserted.ProjectID != projectID {
		t.Errorf("expected project_id %v, got %v", projectID, inserted.ProjectID)
	}
	if inserted.TokensInput != 1_000_000 {
		t.Errorf("expected tokens_input 1000000, got %d", inserted.TokensInput)
	}
	if inserted.TokensOutput != 500_000 {
		t.Errorf("expected tokens_output 500000, got %d", inserted.TokensOutput)
	}
	const epsilon = 0.000001
	if diff := inserted.CostUSD - 22.5; diff > epsilon || diff < -epsilon {
		t.Errorf("expected cost_usd 22.5, got %f", inserted.CostUSD)
	}
	if inserted.Model != "claude-opus-4-6" {
		t.Errorf("expected model claude-opus-4-6, got %s", inserted.Model)
	}

	// Fetch by run_step_id
	fetched, err := costRepo.GetCostByRunStep(ctx, stepID)
	if err != nil {
		t.Fatalf("GetCostByRunStep() error = %v", err)
	}
	if fetched.ID != inserted.ID {
		t.Errorf("expected id %v, got %v", inserted.ID, fetched.ID)
	}
	if diff := fetched.CostUSD - 22.5; diff > epsilon || diff < -epsilon {
		t.Errorf("expected cost_usd 22.5 on fetch, got %f", fetched.CostUSD)
	}
}

func TestIntegration_CostRepo_SumCostByProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	queries := postgres.New(db.pool)
	costRepo := postgres.NewCostRepo(queries)
	runRepo := postgres.NewRunRepo(queries)

	projectID := createTestProject(t, db.pool)
	storyID := createTestStory(t, db.pool, projectID)
	runID := createTestRun(t, runRepo, projectID, storyID)

	// Insert 2 cost records for steps in this run
	step1ID := createTestRunStep(t, runRepo, runID)
	step2ID := createTestRunStep(t, runRepo, runID)

	since := time.Now().Add(-1 * time.Hour)

	_, err := costRepo.InsertCostRecord(ctx, &model.CostRecord{
		RunStepID:    step1ID,
		ProjectID:    projectID,
		TokensInput:  500_000,
		TokensOutput: 250_000,
		CostUSD:      10.0,
		Model:        "claude-sonnet-4-6",
	})
	if err != nil {
		t.Fatalf("InsertCostRecord (step1) error = %v", err)
	}

	_, err = costRepo.InsertCostRecord(ctx, &model.CostRecord{
		RunStepID:    step2ID,
		ProjectID:    projectID,
		TokensInput:  500_000,
		TokensOutput: 250_000,
		CostUSD:      10.0,
		Model:        "claude-sonnet-4-6",
	})
	if err != nil {
		t.Fatalf("InsertCostRecord (step2) error = %v", err)
	}

	totalCost, totalInput, totalOutput, err := costRepo.SumCostByProject(ctx, projectID, since)
	if err != nil {
		t.Fatalf("SumCostByProject() error = %v", err)
	}

	const epsilon = 0.000001
	if diff := totalCost - 20.0; diff > epsilon || diff < -epsilon {
		t.Errorf("expected total_cost 20.0, got %f", totalCost)
	}
	if totalInput != 1_000_000 {
		t.Errorf("expected total_input 1000000, got %d", totalInput)
	}
	if totalOutput != 500_000 {
		t.Errorf("expected total_output 500000, got %d", totalOutput)
	}
}
