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

// createTestAgent inserts an agent of the given type for cost attribution tests.
func createTestAgent(t *testing.T, db *testDB, projectID uuid.UUID, agentType string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	agentID := uuid.New()
	_, err := db.pool.Exec(ctx,
		`INSERT INTO agents (id, project_id, name, template_content, type)
		 VALUES ($1, $2, $3, $4, $5)`,
		agentID, projectID, agentType+"-"+agentID.String()[:8], "prompt", agentType,
	)
	if err != nil {
		t.Fatalf("failed to create test agent: %v", err)
	}
	return agentID
}

// TestIntegration_CostRepo_ListByProjectByRole exercises the project-level
// by-role aggregation: attributed cost is grouped by agent type, unattributed
// cost lands in the "unknown" bucket (RG4), and the summed by-role total equals
// the by-agent total over the attributed records (RG1).
func TestIntegration_CostRepo_ListByProjectByRole(t *testing.T) {
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

	implAgent := createTestAgent(t, db, projectID, "implement")
	reviewAgent := createTestAgent(t, db, projectID, "review")

	implStep := createTestRunStep(t, runRepo, runID)
	reviewStep := createTestRunStep(t, runRepo, runID)
	orphanStep := createTestRunStep(t, runRepo, runID)

	mustInsert := func(stepID uuid.UUID, agentID *uuid.UUID, in, out int64, cost float64) {
		_, err := costRepo.InsertCostRecord(ctx, &model.CostRecord{
			RunStepID:    stepID,
			ProjectID:    projectID,
			AgentID:      agentID,
			TokensInput:  in,
			TokensOutput: out,
			CostUSD:      cost,
			Model:        "claude-opus-4-6",
		})
		if err != nil {
			t.Fatalf("InsertCostRecord error = %v", err)
		}
	}

	mustInsert(implStep, &implAgent, 100000, 20000, 4.00)
	mustInsert(reviewStep, &reviewAgent, 60000, 10000, 1.50)
	mustInsert(orphanStep, nil, 30000, 5000, 0.75) // unattributed → "unknown"

	roles, err := costRepo.ListByProjectByRole(ctx, projectID)
	if err != nil {
		t.Fatalf("ListByProjectByRole error = %v", err)
	}

	const epsilon = 0.000001
	byRole := make(map[string]model.ProjectRoleCostBreakdown, len(roles))
	var roleTotal float64
	for _, r := range roles {
		byRole[r.Role] = r
		roleTotal += r.CostUSD
	}

	if _, ok := byRole["implement"]; !ok {
		t.Errorf("expected an 'implement' role bucket, got roles: %+v", roles)
	}
	if _, ok := byRole["review"]; !ok {
		t.Errorf("expected a 'review' role bucket, got roles: %+v", roles)
	}
	// RG4: unattributed cost is bucketed under "unknown" and present.
	unknown, ok := byRole["unknown"]
	if !ok {
		t.Fatalf("expected an 'unknown' role bucket for unattributed cost, got roles: %+v", roles)
	}
	if diff := unknown.CostUSD - 0.75; diff > epsilon || diff < -epsilon {
		t.Errorf("expected unknown cost 0.75, got %f", unknown.CostUSD)
	}

	// RG4: the unknown cost is counted in the rolled-up by-role total.
	if diff := roleTotal - 6.25; diff > epsilon || diff < -epsilon {
		t.Errorf("expected by-role total 6.25 (incl. unknown), got %f", roleTotal)
	}

	// RG1: over the ATTRIBUTED records, by-role total == by-agent total.
	agents, err := costRepo.ListByProjectByAgent(ctx, projectID)
	if err != nil {
		t.Fatalf("ListByProjectByAgent error = %v", err)
	}
	var agentTotal float64
	for _, a := range agents {
		agentTotal += a.CostUSD
	}
	attributedRoleTotal := byRole["implement"].CostUSD + byRole["review"].CostUSD
	if diff := attributedRoleTotal - agentTotal; diff > epsilon || diff < -epsilon {
		t.Errorf("attributed by-role total (%f) must equal by-agent total (%f)", attributedRoleTotal, agentTotal)
	}
}

// TestIntegration_CostRepo_ListByProjectByRunPaginated_Tokens proves the
// paginated run rows carry the per-run token sums (RG2: Recent Runs shows real
// tokens, not 0).
func TestIntegration_CostRepo_ListByProjectByRunPaginated_Tokens(t *testing.T) {
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

	step1 := createTestRunStep(t, runRepo, runID)
	step2 := createTestRunStep(t, runRepo, runID)

	for _, s := range []struct {
		id  uuid.UUID
		in  int64
		out int64
	}{{step1, 100000, 20000}, {step2, 50000, 5000}} {
		if _, err := costRepo.InsertCostRecord(ctx, &model.CostRecord{
			RunStepID:    s.id,
			ProjectID:    projectID,
			TokensInput:  s.in,
			TokensOutput: s.out,
			CostUSD:      1.0,
			Model:        "claude-opus-4-6",
		}); err != nil {
			t.Fatalf("InsertCostRecord error = %v", err)
		}
	}

	rows, err := costRepo.ListCostsByProjectByRunPaginated(ctx, projectID, time.Now().Add(-1*time.Hour), 20, 0)
	if err != nil {
		t.Fatalf("ListCostsByProjectByRunPaginated error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 run row, got %d", len(rows))
	}
	if rows[0].TokensInput != 150000 {
		t.Errorf("expected tokens_input 150000, got %d", rows[0].TokensInput)
	}
	if rows[0].TokensOutput != 25000 {
		t.Errorf("expected tokens_output 25000, got %d", rows[0].TokensOutput)
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
