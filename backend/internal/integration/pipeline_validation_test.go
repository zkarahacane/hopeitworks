// Package integration contains end-to-end integration tests that validate the
// full pipeline flow against a real Postgres database via testcontainers.
// These tests cover story import, run creation, pipeline execution, and state
// transitions using the reference test-project stories.
package integration

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zakari/hopeitworks/backend/internal/adapter/markdown"
	planningadapter "github.com/zakari/hopeitworks/backend/internal/adapter/planning"
	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/internal/testutil"
)

// newMarkdownImportService builds a PlanningImportService wired with the markdown
// adapter against a real DB — the replacement for the deleted StoryService.Import
// used to seed stories from the reference todo-stories.md fixture.
func newMarkdownImportService(storyRepo port.StoryRepository, queries *postgres.Queries) *service.PlanningImportService {
	epicRepo := postgres.NewEpicRepo(queries)
	projectRepo := postgres.NewProjectRepo(queries)
	factory := planningadapter.NewFactory(projectRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return service.NewPlanningImportService(storyRepo, epicRepo, factory)
}

// testProjectStoriesPath returns the path to testdata/todo-stories.md.
func testProjectStoriesPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", "todo-stories.md")
}

const (
	scopeBackend  = "backend"
	scopeFrontend = "frontend"
)

// createTestAgent inserts a minimal agent row in the DB and returns its UUID.
// The agent is global-scoped so it is visible to any project without additional
// project filtering.
func createTestAgent(t *testing.T, pool *pgxpool.Pool, name, agentType, agentModel string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO agents (id, name, model, image, template_content, type, scope)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, name, agentModel, "hopeitworks/agent:latest", "# template", agentType, "global",
	)
	if err != nil {
		t.Fatalf("failed to create test agent %q: %v", name, err)
	}
	return id
}

// noopAction implements model.Action for integration tests.
// It succeeds immediately without performing real work.
type noopAction struct {
	name string
}

func (a *noopAction) Name() string { return a.name }
func (a *noopAction) Execute(_ context.Context, _ *model.RunContext) error {
	return nil
}

// TestIntegration_PipelineValidation_StoryImport validates that stories from
// the test-project markdown can be imported into the system via the StoryService.
func TestIntegration_PipelineValidation_StoryImport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer db.Cleanup()

	ctx := context.Background()
	queries := postgres.New(db.Pool)
	storyRepo := postgres.NewStoryRepo(queries)
	importSvc := newMarkdownImportService(storyRepo, queries)

	projectID := testutil.CreateProject(t, db.Pool)

	// Read the reference stories markdown
	storiesPath := testProjectStoriesPath()
	content, err := os.ReadFile(storiesPath)
	if err != nil {
		t.Fatalf("failed to read test-project stories: %v", err)
	}

	// Sanity-check the fixture parses to >0 blocks before importing.
	parsed := markdown.ParseStoryMarkdown(string(content))
	if len(parsed) == 0 {
		t.Fatal("expected at least 1 parsed story, got 0")
	}

	importCfg := port.ImportConfig{
		Source:   port.SourceMarkdown,
		Markdown: &port.MarkdownConfig{Content: string(content)},
	}
	result, err := importSvc.Import(ctx, projectID, importCfg)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	t.Run("all 5 stories imported successfully", func(t *testing.T) {
		if result.StoriesCreated != 5 {
			t.Errorf("expected 5 created stories, got %d", result.StoriesCreated)
		}
		if result.Failed != 0 {
			t.Errorf("expected 0 failures, got %d: %v", result.Failed, result.Errors)
		}
	})

	t.Run("stories retrievable by key", func(t *testing.T) {
		expectedKeys := []string{"TODO-1", "TODO-2", "TODO-3", "TODO-4", "TODO-5"}
		for _, key := range expectedKeys {
			story, err := storyRepo.GetByKey(ctx, projectID, key)
			if err != nil {
				t.Errorf("GetByKey(%q) error = %v", key, err)
				continue
			}
			if story.Key != key {
				t.Errorf("expected key %q, got %q", key, story.Key)
			}
			if story.Status != model.StoryStatusBacklog {
				t.Errorf("story %s: expected status %q, got %q", key, model.StoryStatusBacklog, story.Status)
			}
		}
	})

	t.Run("story dependencies preserved", func(t *testing.T) {
		story3, err := storyRepo.GetByKey(ctx, projectID, "TODO-3")
		if err != nil {
			t.Fatalf("GetByKey(TODO-3) error = %v", err)
		}
		if len(story3.DependsOn) != 1 || story3.DependsOn[0] != "TODO-1" {
			t.Errorf("TODO-3 depends_on: expected [TODO-1], got %v", story3.DependsOn)
		}

		story5, err := storyRepo.GetByKey(ctx, projectID, "TODO-5")
		if err != nil {
			t.Fatalf("GetByKey(TODO-5) error = %v", err)
		}
		if len(story5.DependsOn) != 2 {
			t.Errorf("TODO-5 depends_on: expected 2 deps, got %v", story5.DependsOn)
		}
	})

	t.Run("story scopes preserved", func(t *testing.T) {
		story1, err := storyRepo.GetByKey(ctx, projectID, "TODO-1")
		if err != nil {
			t.Fatalf("GetByKey(TODO-1) error = %v", err)
		}
		if story1.Scope == nil || *story1.Scope != scopeBackend {
			t.Errorf("TODO-1 scope: expected %q, got %v", scopeBackend, story1.Scope)
		}

		story5, err := storyRepo.GetByKey(ctx, projectID, "TODO-5")
		if err != nil {
			t.Fatalf("GetByKey(TODO-5) error = %v", err)
		}
		if story5.Scope == nil || *story5.Scope != scopeFrontend {
			t.Errorf("TODO-5 scope: expected %q, got %v", scopeFrontend, story5.Scope)
		}
	})

	t.Run("re-import of unchanged content is an idempotent no-op", func(t *testing.T) {
		result2, err := importSvc.Import(ctx, projectID, importCfg)
		if err != nil {
			t.Fatalf("second Import() error = %v", err)
		}
		// New connector: an unchanged re-import is hash-gated to a true no-op
		// (Skipped), never a churny update, and never a duplicate.
		if result2.Skipped != 5 {
			t.Errorf("expected 5 skipped (hash no-op) stories, got %d", result2.Skipped)
		}
		if result2.StoriesCreated != 0 {
			t.Errorf("expected 0 new creates on re-import, got %d", result2.StoriesCreated)
		}
		if result2.StoriesUpdated != 0 {
			t.Errorf("expected 0 updates on unchanged re-import, got %d", result2.StoriesUpdated)
		}
	})
}

// TestIntegration_PipelineValidation_RunCreation validates that a run can
// be created from a pipeline config and story, with correct steps.
func TestIntegration_PipelineValidation_RunCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer db.Cleanup()

	ctx := context.Background()
	queries := postgres.New(db.Pool)

	projectID := testutil.CreateProject(t, db.Pool)

	// Create agents in DB so LaunchRun can resolve agent_id references.
	implementAgentID := createTestAgent(t, db.Pool, "implement-agent", "implement", "claude-opus-4-6")
	reviewAgentID := createTestAgent(t, db.Pool, "review-agent", "review", "claude-sonnet-4-6")
	mergeAgentID := createTestAgent(t, db.Pool, "merge-agent", "merge", "claude-sonnet-4-6")

	// Create pipeline config with agent_id on every agent_run step.
	pipelineYAML := "steps:\n" +
		"  - id: \"step-implement\"\n" +
		"    name: \"implement\"\n" +
		"    action_type: \"implement\"\n" +
		"    agent_id: \"" + implementAgentID.String() + "\"\n" +
		"    auto_approve: false\n" +
		"    retry_policy:\n" +
		"      max_retries: 0\n" +
		"      retry_type: \"none\"\n" +
		"  - id: \"step-review\"\n" +
		"    name: \"review\"\n" +
		"    action_type: \"review\"\n" +
		"    agent_id: \"" + reviewAgentID.String() + "\"\n" +
		"    auto_approve: true\n" +
		"    retry_policy:\n" +
		"      max_retries: 1\n" +
		"      retry_type: \"on-failure\"\n" +
		"  - id: \"step-merge\"\n" +
		"    name: \"merge\"\n" +
		"    action_type: \"merge\"\n" +
		"    agent_id: \"" + mergeAgentID.String() + "\"\n" +
		"    auto_approve: true\n" +
		"    retry_policy:\n" +
		"      max_retries: 0\n" +
		"      retry_type: \"none\"\n"

	testutil.UpsertPipelineConfig(t, db.Pool, projectID, pipelineYAML)

	// Create a story
	storyRepo := postgres.NewStoryRepo(queries)
	storySvc := service.NewStoryService(storyRepo)
	scope := scopeBackend
	story, err := storySvc.Create(ctx, service.CreateStoryParams{
		ProjectID: projectID,
		Key:       "TODO-1",
		Title:     "Add create todo endpoint",
		Scope:     &scope,
		Status:    model.StoryStatusBacklog,
	})
	if err != nil {
		t.Fatalf("Create story error = %v", err)
	}

	// Create run service dependencies
	runRepo := postgres.NewRunRepo(queries)
	projectRepo := postgres.NewProjectRepo(queries)
	pipelineConfigRepo := postgres.NewPipelineConfigRepo(queries)
	agentRepo := postgres.NewAgentRepo(queries)
	mockJobQueue := &noopJobQueue{}

	runSvc := service.NewRunService(runRepo, projectRepo, storyRepo, pipelineConfigRepo, mockJobQueue)
	runSvc.SetAgentRepo(agentRepo)

	// Launch run
	run, err := runSvc.LaunchRun(ctx, projectID, story.ID, uuid.Nil)
	if err != nil {
		t.Fatalf("LaunchRun() error = %v", err)
	}

	t.Run("run created with pending status", func(t *testing.T) {
		if run.Status != model.RunStatusPending {
			t.Errorf("expected run status %q, got %q", model.RunStatusPending, run.Status)
		}
		if run.ProjectID != projectID {
			t.Errorf("expected project_id %v, got %v", projectID, run.ProjectID)
		}
		if run.StoryID != story.ID {
			t.Errorf("expected story_id %v, got %v", story.ID, run.StoryID)
		}
	})

	t.Run("run has 3 steps in correct order", func(t *testing.T) {
		if len(run.Steps) != 3 {
			t.Fatalf("expected 3 steps, got %d", len(run.Steps))
		}

		expectedSteps := []struct {
			name   string
			action string
			order  int
		}{
			{"implement", "implement", 0},
			{"review", "review", 1},
			{"merge", "merge", 2},
		}

		for i, expected := range expectedSteps {
			step := run.Steps[i]
			if step.StepName != expected.name {
				t.Errorf("step %d: expected name %q, got %q", i, expected.name, step.StepName)
			}
			if step.Action != expected.action {
				t.Errorf("step %d: expected action %q, got %q", i, expected.action, step.Action)
			}
			if step.StepOrder != expected.order {
				t.Errorf("step %d: expected order %d, got %d", i, expected.order, step.StepOrder)
			}
			if step.Status != model.StepStatusPending {
				t.Errorf("step %d: expected status %q, got %q", i, model.StepStatusPending, step.Status)
			}
		}
	})

	t.Run("pipeline config snapshot persisted", func(t *testing.T) {
		if len(run.PipelineConfigSnapshot) == 0 {
			t.Fatal("expected pipeline config snapshot to be non-empty")
		}
		var snapshot model.PipelineConfigYAML
		if err := json.Unmarshal(run.PipelineConfigSnapshot, &snapshot); err != nil {
			t.Fatalf("failed to unmarshal config snapshot: %v", err)
		}
		flatSteps := snapshot.FlatSteps()
		if len(flatSteps) != 3 {
			t.Errorf("expected 3 steps in snapshot, got %d", len(flatSteps))
		}
	})

	t.Run("run retrievable by ID", func(t *testing.T) {
		retrieved, err := runSvc.GetRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetRun() error = %v", err)
		}
		if retrieved.ID != run.ID {
			t.Errorf("expected run ID %v, got %v", run.ID, retrieved.ID)
		}
		if len(retrieved.Steps) != 3 {
			t.Errorf("expected 3 steps, got %d", len(retrieved.Steps))
		}
	})

	t.Run("duplicate launch blocked for active run", func(t *testing.T) {
		_, err := runSvc.LaunchRun(ctx, projectID, story.ID, uuid.Nil)
		if err == nil {
			t.Fatal("expected error for duplicate launch, got nil")
		}
	})
}

// TestIntegration_PipelineValidation_Execution validates the full pipeline
// execution flow: run transitions, step transitions, and events.
func TestIntegration_PipelineValidation_Execution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer db.Cleanup()

	ctx := context.Background()
	queries := postgres.New(db.Pool)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	projectID := testutil.CreateProject(t, db.Pool)

	// Create agents in DB so LaunchRun can resolve agent_id references.
	implementAgentID := createTestAgent(t, db.Pool, "implement-agent", "implement", "claude-opus-4-6")
	reviewAgentID := createTestAgent(t, db.Pool, "review-agent", "review", "claude-sonnet-4-6")
	mergeAgentID := createTestAgent(t, db.Pool, "merge-agent", "merge", "claude-sonnet-4-6")

	// Create pipeline config with 3 steps, each referencing an agent_id.
	pipelineYAML := "steps:\n" +
		"  - id: \"step-implement\"\n" +
		"    name: \"implement\"\n" +
		"    action_type: \"implement\"\n" +
		"    agent_id: \"" + implementAgentID.String() + "\"\n" +
		"    auto_approve: false\n" +
		"    retry_policy:\n" +
		"      max_retries: 0\n" +
		"      retry_type: \"none\"\n" +
		"  - id: \"step-review\"\n" +
		"    name: \"review\"\n" +
		"    action_type: \"review\"\n" +
		"    agent_id: \"" + reviewAgentID.String() + "\"\n" +
		"    auto_approve: true\n" +
		"    retry_policy:\n" +
		"      max_retries: 0\n" +
		"      retry_type: \"none\"\n" +
		"  - id: \"step-merge\"\n" +
		"    name: \"merge\"\n" +
		"    action_type: \"merge\"\n" +
		"    agent_id: \"" + mergeAgentID.String() + "\"\n" +
		"    auto_approve: true\n" +
		"    retry_policy:\n" +
		"      max_retries: 0\n" +
		"      retry_type: \"none\"\n"

	testutil.UpsertPipelineConfig(t, db.Pool, projectID, pipelineYAML)

	// Create story
	storyRepo := postgres.NewStoryRepo(queries)
	storySvc := service.NewStoryService(storyRepo)
	scope := scopeBackend
	story, err := storySvc.Create(ctx, service.CreateStoryParams{
		ProjectID: projectID,
		Key:       "TODO-1",
		Title:     "Add create todo endpoint",
		Scope:     &scope,
		Status:    model.StoryStatusBacklog,
	})
	if err != nil {
		t.Fatalf("Create story error = %v", err)
	}

	// Create run
	runRepo := postgres.NewRunRepo(queries)
	projectRepo := postgres.NewProjectRepo(queries)
	pipelineConfigRepo := postgres.NewPipelineConfigRepo(queries)
	agentRepo := postgres.NewAgentRepo(queries)
	mockQueue := &noopJobQueue{}
	runSvc := service.NewRunService(runRepo, projectRepo, storyRepo, pipelineConfigRepo, mockQueue)
	runSvc.SetAgentRepo(agentRepo)

	run, err := runSvc.LaunchRun(ctx, projectID, story.ID, uuid.Nil)
	if err != nil {
		t.Fatalf("LaunchRun() error = %v", err)
	}

	// Setup action registry with noop actions
	actionReg := service.NewActionRegistry()
	actionReg.Register(&noopAction{name: "implement"})
	actionReg.Register(&noopAction{name: "review"})
	actionReg.Register(&noopAction{name: "merge"})

	// Setup event publisher
	eventRepo := postgres.NewEventRepo(queries)

	// Create pipeline executor
	executor := service.NewPipelineExecutor(runRepo, storyRepo, actionReg, eventRepo, logger)

	// Execute the pipeline
	err = executor.ExecuteRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("ExecuteRun() error = %v", err)
	}

	t.Run("run completed successfully", func(t *testing.T) {
		completedRun, err := runSvc.GetRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetRun() error = %v", err)
		}
		if completedRun.Status != model.RunStatusCompleted {
			t.Errorf("expected run status %q, got %q", model.RunStatusCompleted, completedRun.Status)
		}
		if completedRun.StartedAt == nil {
			t.Error("expected started_at to be set")
		}
		if completedRun.CompletedAt == nil {
			t.Error("expected completed_at to be set")
		}
	})

	t.Run("all steps completed", func(t *testing.T) {
		completedRun, err := runSvc.GetRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetRun() error = %v", err)
		}
		for _, step := range completedRun.Steps {
			if step.Status != model.StepStatusCompleted {
				t.Errorf("step %q: expected status %q, got %q", step.StepName, model.StepStatusCompleted, step.Status)
			}
			if step.StartedAt == nil {
				t.Errorf("step %q: expected started_at to be set", step.StepName)
			}
			if step.CompletedAt == nil {
				t.Errorf("step %q: expected completed_at to be set", step.StepName)
			}
		}
	})

	t.Run("events published for all steps", func(t *testing.T) {
		events, err := queries.ListEventsByProject(ctx, postgres.ListEventsByProjectParams{
			ProjectID: projectID,
			Limit:     100,
			Offset:    0,
		})
		if err != nil {
			t.Fatalf("ListEventsByProject() error = %v", err)
		}

		// Events are returned in created_at DESC order.
		// Expected events (10 total):
		// story.status_updated(running), run.started, 3x(step.started + step.completed), run.completed, story.status_updated(done)
		if len(events) < 8 {
			t.Fatalf("expected at least 8 events, got %d", len(events))
		}

		// Find run.completed and run.started events (story.status_updated may be before/after)
		var foundRunCompleted, foundRunStarted bool
		for _, e := range events {
			if e.EntityType == "run" && e.Action == "completed" {
				foundRunCompleted = true
			}
			if e.EntityType == "run" && e.Action == "started" {
				foundRunStarted = true
			}
		}
		if !foundRunCompleted {
			t.Error("expected run.completed event, not found")
		}
		if !foundRunStarted {
			t.Error("expected run.started event, not found")
		}

		// Count step events
		var stepStarted, stepCompleted int
		for _, e := range events {
			switch {
			case e.EntityType == "step" && e.Action == "started":
				stepStarted++
			case e.EntityType == "step" && e.Action == "completed":
				stepCompleted++
			}
		}
		if stepStarted != 3 {
			t.Errorf("expected 3 step.started events, got %d", stepStarted)
		}
		if stepCompleted != 3 {
			t.Errorf("expected 3 step.completed events, got %d", stepCompleted)
		}
	})
}

// TestIntegration_PipelineValidation_FullFlow validates the complete pipeline
// flow from story import through execution, mimicking the reference test project
// end-to-end validation scenario.
func TestIntegration_PipelineValidation_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer db.Cleanup()

	ctx := context.Background()
	queries := postgres.New(db.Pool)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// 1. Setup: create project with pipeline config
	projectID := testutil.CreateProject(t, db.Pool)

	// Create agents in DB so LaunchRun can resolve agent_id references.
	implementAgentID := createTestAgent(t, db.Pool, "implement-agent", "implement", "claude-opus-4-6")
	reviewAgentID := createTestAgent(t, db.Pool, "review-agent", "review", "claude-sonnet-4-6")

	pipelineYAML := "steps:\n" +
		"  - id: \"step-implement\"\n" +
		"    name: \"implement\"\n" +
		"    action_type: \"implement\"\n" +
		"    agent_id: \"" + implementAgentID.String() + "\"\n" +
		"    auto_approve: false\n" +
		"    retry_policy:\n" +
		"      max_retries: 0\n" +
		"      retry_type: \"none\"\n" +
		"  - id: \"step-review\"\n" +
		"    name: \"review\"\n" +
		"    action_type: \"review\"\n" +
		"    agent_id: \"" + reviewAgentID.String() + "\"\n" +
		"    auto_approve: true\n" +
		"    retry_policy:\n" +
		"      max_retries: 0\n" +
		"      retry_type: \"none\"\n"

	testutil.UpsertPipelineConfig(t, db.Pool, projectID, pipelineYAML)

	// 2. Import stories from test-project markdown
	storyRepo := postgres.NewStoryRepo(queries)
	importSvc := newMarkdownImportService(storyRepo, queries)

	storiesPath := testProjectStoriesPath()
	content, err := os.ReadFile(storiesPath)
	if err != nil {
		t.Fatalf("failed to read test-project stories: %v", err)
	}

	importResult, err := importSvc.Import(ctx, projectID, port.ImportConfig{
		Source:   port.SourceMarkdown,
		Markdown: &port.MarkdownConfig{Content: string(content)},
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if importResult.Failed > 0 {
		t.Fatalf("story import had %d failures: %v", importResult.Failed, importResult.Errors)
	}

	// 3. Verify stories exist
	stories, err := storyRepo.ListByProject(ctx, projectID, 100, 0)
	if err != nil {
		t.Fatalf("ListByProject() error = %v", err)
	}
	if len(stories) != 5 {
		t.Fatalf("expected 5 stories after import, got %d", len(stories))
	}

	// 4. Pick a story without dependencies (TODO-1) and launch a run
	story1, err := storyRepo.GetByKey(ctx, projectID, "TODO-1")
	if err != nil {
		t.Fatalf("GetByKey(TODO-1) error = %v", err)
	}

	runRepo := postgres.NewRunRepo(queries)
	projectRepo := postgres.NewProjectRepo(queries)
	pipelineConfigRepo := postgres.NewPipelineConfigRepo(queries)
	agentRepo := postgres.NewAgentRepo(queries)
	mockQueue := &noopJobQueue{}

	runSvc := service.NewRunService(runRepo, projectRepo, storyRepo, pipelineConfigRepo, mockQueue)
	runSvc.SetAgentRepo(agentRepo)

	run, err := runSvc.LaunchRun(ctx, projectID, story1.ID, uuid.Nil)
	if err != nil {
		t.Fatalf("LaunchRun() error = %v", err)
	}

	// 5. Setup action registry with noop actions and execute
	actionReg := service.NewActionRegistry()
	actionReg.Register(&noopAction{name: "implement"})
	actionReg.Register(&noopAction{name: "review"})

	eventRepo := postgres.NewEventRepo(queries)
	executor := service.NewPipelineExecutor(runRepo, storyRepo, actionReg, eventRepo, logger)

	err = executor.ExecuteRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("ExecuteRun() error = %v", err)
	}

	// 6. Verify final state
	completedRun, err := runSvc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}

	if completedRun.Status != model.RunStatusCompleted {
		t.Fatalf("expected run status %q, got %q", model.RunStatusCompleted, completedRun.Status)
	}

	for _, step := range completedRun.Steps {
		if step.Status != model.StepStatusCompleted {
			t.Errorf("step %q: expected %q, got %q", step.StepName, model.StepStatusCompleted, step.Status)
		}
	}

	// 7. Verify events were generated
	events, err := queries.ListEventsByProject(ctx, postgres.ListEventsByProjectParams{
		ProjectID: projectID,
		Limit:     100,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("ListEventsByProject() error = %v", err)
	}

	// run.started + 2*(step.started + step.completed) + run.completed = 6
	if len(events) < 6 {
		t.Errorf("expected at least 6 events, got %d", len(events))
	}

	t.Logf("Full pipeline validation passed: %d stories imported, run completed with %d steps, %d events generated",
		importResult.StoriesCreated, len(completedRun.Steps), len(events))
}

// noopJobQueue implements port.JobQueue for integration tests.
// It records enqueued runs without executing them (execution is done
// directly via PipelineExecutor in these tests).
type noopJobQueue struct {
	enqueuedRunIDs []uuid.UUID
}

func (q *noopJobQueue) EnqueueExecuteRun(_ context.Context, runID uuid.UUID) error {
	q.enqueuedRunIDs = append(q.enqueuedRunIDs, runID)
	return nil
}
