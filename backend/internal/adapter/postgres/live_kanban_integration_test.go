package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

func createTestEpic(t *testing.T, pool *pgxpool.Pool, projectID uuid.UUID) uuid.UUID {
	t.Helper()
	epicID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO epics (id, project_id, name, status) VALUES ($1, $2, $3, $4)`,
		epicID, projectID, "Epic-"+epicID.String()[:8], "backlog",
	)
	if err != nil {
		t.Fatalf("failed to create test epic: %v", err)
	}
	return epicID
}

func createTestStoryWithEpic(t *testing.T, pool *pgxpool.Pool, projectID, epicID uuid.UUID, status string) uuid.UUID {
	t.Helper()
	storyID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO stories (id, project_id, epic_id, key, title, status)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		storyID, projectID, epicID, "S-"+storyID.String()[:6], "Story", status,
	)
	if err != nil {
		t.Fatalf("failed to create test story: %v", err)
	}
	return storyID
}

// TestIntegration_GetLatestRunByStory exercises the latest-run projection against
// a real Postgres, including the NULL current_step case (no step in progress).
func TestIntegration_GetLatestRunByStory(t *testing.T) {
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

	// No run yet -> nil.
	latest, err := runRepo.GetLatestRunByStory(ctx, storyID)
	if err != nil {
		t.Fatalf("GetLatestRunByStory() error = %v", err)
	}
	if latest != nil {
		t.Fatalf("expected nil latest run, got %+v", latest)
	}

	// Create a run with 3 steps: completed, running, pending.
	run, err := runRepo.CreateRun(ctx, &model.Run{
		ProjectID: projectID,
		StoryID:   storyID,
		Status:    model.RunStatusRunning,
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	s0, _ := runRepo.CreateRunStep(ctx, &model.RunStep{RunID: run.ID, StepName: "branch", StepOrder: 0, Action: "git_branch", Status: model.StepStatusPending})
	s1, _ := runRepo.CreateRunStep(ctx, &model.RunStep{RunID: run.ID, StepName: "implement", StepOrder: 1, Action: "agent_run", Status: model.StepStatusPending})
	_, _ = runRepo.CreateRunStep(ctx, &model.RunStep{RunID: run.ID, StepName: "review", StepOrder: 2, Action: "agent_run", Status: model.StepStatusPending})

	if _, err := runRepo.UpdateRunStepStatus(ctx, s0.ID, model.StepStatusCompleted, nil, nil, nil); err != nil {
		t.Fatalf("UpdateRunStepStatus(s0) error = %v", err)
	}
	if _, err := runRepo.UpdateRunStepStatus(ctx, s1.ID, model.StepStatusRunning, nil, nil, nil); err != nil {
		t.Fatalf("UpdateRunStepStatus(s1) error = %v", err)
	}

	latest, err = runRepo.GetLatestRunByStory(ctx, storyID)
	if err != nil {
		t.Fatalf("GetLatestRunByStory() error = %v", err)
	}
	if latest == nil {
		t.Fatal("expected latest run, got nil")
	}
	if latest.ID != run.ID {
		t.Errorf("expected run %s, got %s", run.ID, latest.ID)
	}
	if latest.CurrentStep == nil {
		t.Fatal("expected current step (running), got nil")
	}
	if latest.CurrentStep.ID != s1.ID {
		t.Errorf("expected current step %s, got %s", s1.ID, latest.CurrentStep.ID)
	}
	if latest.CurrentStep.Name != "implement" || latest.CurrentStep.ActionType != "agent_run" {
		t.Errorf("unexpected current step fields: %+v", latest.CurrentStep)
	}
	if latest.CurrentStep.Index != 1 {
		t.Errorf("expected current step index 1, got %d", latest.CurrentStep.Index)
	}
	if latest.CurrentStep.Total != 3 {
		t.Errorf("expected total 3, got %d", latest.CurrentStep.Total)
	}

	// Complete the running step -> no step in progress -> current_step NULL.
	if _, err := runRepo.UpdateRunStepStatus(ctx, s1.ID, model.StepStatusCompleted, nil, nil, nil); err != nil {
		t.Fatalf("UpdateRunStepStatus(s1 complete) error = %v", err)
	}
	latest, err = runRepo.GetLatestRunByStory(ctx, storyID)
	if err != nil {
		t.Fatalf("GetLatestRunByStory() error = %v", err)
	}
	if latest == nil {
		t.Fatal("expected latest run, got nil")
	}
	if latest.CurrentStep != nil {
		t.Errorf("expected nil current step when none in progress, got %+v", latest.CurrentStep)
	}
}

// TestIntegration_GetLatestRunsByStories verifies the batch latest-run query
// returns the most recent run per story and skips stories without runs.
func TestIntegration_GetLatestRunsByStories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	queries := postgres.New(db.pool)
	runRepo := postgres.NewRunRepo(queries)

	projectID := createTestProject(t, db.pool)
	storyA := createTestStory(t, db.pool, projectID)
	storyB := createTestStory(t, db.pool, projectID)
	storyNoRun := createTestStory(t, db.pool, projectID)

	// storyA: two runs; the latest should win.
	_, _ = runRepo.CreateRun(ctx, &model.Run{ProjectID: projectID, StoryID: storyA, Status: model.RunStatusFailed})
	runA2, _ := runRepo.CreateRun(ctx, &model.Run{ProjectID: projectID, StoryID: storyA, Status: model.RunStatusRunning})
	sA, _ := runRepo.CreateRunStep(ctx, &model.RunStep{RunID: runA2.ID, StepName: "impl", StepOrder: 0, Action: "agent_run", Status: model.StepStatusPending})
	_, _ = runRepo.UpdateRunStepStatus(ctx, sA.ID, model.StepStatusWaitingApproval, nil, nil, nil)

	// storyB: one run, no step in progress.
	runB, _ := runRepo.CreateRun(ctx, &model.Run{ProjectID: projectID, StoryID: storyB, Status: model.RunStatusCompleted})

	got, err := runRepo.GetLatestRunsByStories(ctx, []uuid.UUID{storyA, storyB, storyNoRun})
	if err != nil {
		t.Fatalf("GetLatestRunsByStories() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries (storyNoRun absent), got %d", len(got))
	}
	if got[storyA] == nil || got[storyA].ID != runA2.ID {
		t.Errorf("storyA: expected latest run %s, got %+v", runA2.ID, got[storyA])
	}
	if got[storyA].CurrentStep == nil || got[storyA].CurrentStep.Status != string(model.StepStatusWaitingApproval) {
		t.Errorf("storyA: expected waiting_approval current step, got %+v", got[storyA].CurrentStep)
	}
	if got[storyB] == nil || got[storyB].ID != runB.ID {
		t.Errorf("storyB: expected latest run %s, got %+v", runB.ID, got[storyB])
	}
	if got[storyB].CurrentStep != nil {
		t.Errorf("storyB: expected nil current step, got %+v", got[storyB].CurrentStep)
	}
	if _, ok := got[storyNoRun]; ok {
		t.Errorf("storyNoRun should be absent from result map")
	}
}

// TestIntegration_CountStoriesByEpicGroupedByStatus verifies the grouped count
// query aggregates story statuses for an epic in a single query.
func TestIntegration_CountStoriesByEpicGroupedByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	queries := postgres.New(db.pool)
	storyRepo := postgres.NewStoryRepo(queries)

	projectID := createTestProject(t, db.pool)
	epicID := createTestEpic(t, db.pool, projectID)

	createTestStoryWithEpic(t, db.pool, projectID, epicID, model.StoryStatusBacklog)
	createTestStoryWithEpic(t, db.pool, projectID, epicID, model.StoryStatusBacklog)
	createTestStoryWithEpic(t, db.pool, projectID, epicID, model.StoryStatusRunning)
	createTestStoryWithEpic(t, db.pool, projectID, epicID, model.StoryStatusDone)
	createTestStoryWithEpic(t, db.pool, projectID, epicID, model.StoryStatusDone)
	createTestStoryWithEpic(t, db.pool, projectID, epicID, model.StoryStatusFailed)

	counts, err := storyRepo.CountByEpicGroupedByStatus(ctx, epicID)
	if err != nil {
		t.Fatalf("CountByEpicGroupedByStatus() error = %v", err)
	}
	if counts.Backlog != 2 || counts.Running != 1 || counts.Done != 2 || counts.Failed != 1 {
		t.Errorf("unexpected counts: %+v", counts)
	}

	// Empty epic -> all zeros.
	emptyEpic := createTestEpic(t, db.pool, projectID)
	empty, err := storyRepo.CountByEpicGroupedByStatus(ctx, emptyEpic)
	if err != nil {
		t.Fatalf("CountByEpicGroupedByStatus(empty) error = %v", err)
	}
	if empty.Backlog != 0 || empty.Running != 0 || empty.Done != 0 || empty.Failed != 0 {
		t.Errorf("expected zero counts for empty epic, got %+v", empty)
	}
}
