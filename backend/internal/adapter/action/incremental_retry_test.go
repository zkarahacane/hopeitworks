package action_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/action"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// --- Partial mocks for IncrementalRetryAction ---

// retryMockRunRepo implements port.RunRepository with configurable fns for retry tests.
type retryMockRunRepo struct {
	getRunStepFn         func(ctx context.Context, id uuid.UUID) (*model.RunStep, error)
	createRetryRunStepFn func(ctx context.Context, step *model.RunStep) (*model.RunStep, error)
	createdStep          *model.RunStep
}

func (m *retryMockRunRepo) GetRunStep(ctx context.Context, id uuid.UUID) (*model.RunStep, error) {
	if m.getRunStepFn != nil {
		return m.getRunStepFn(ctx, id)
	}
	return nil, apperrors.NewNotFound("run step", id)
}

func (m *retryMockRunRepo) CreateRetryRunStep(ctx context.Context, step *model.RunStep) (*model.RunStep, error) {
	if m.createRetryRunStepFn != nil {
		return m.createRetryRunStepFn(ctx, step)
	}
	m.createdStep = step
	return step, nil
}

// Stub implementations for the full port.RunRepository interface.
func (m *retryMockRunRepo) CreateRun(_ context.Context, run *model.Run) (*model.Run, error) {
	return run, nil
}
func (m *retryMockRunRepo) GetRun(_ context.Context, id uuid.UUID) (*model.Run, error) {
	return nil, apperrors.NewNotFound("run", id)
}
func (m *retryMockRunRepo) GetActiveRunByStory(_ context.Context, _ uuid.UUID) (*model.Run, error) {
	return nil, nil
}
func (m *retryMockRunRepo) ListRunsByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *retryMockRunRepo) ListRunsByStory(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *retryMockRunRepo) ListRunsByStatus(_ context.Context, _ model.RunStatus) ([]*model.Run, error) {
	return nil, nil
}
func (m *retryMockRunRepo) MarkRunOrphanedIfRunning(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (bool, error) {
	return false, nil
}
func (m *retryMockRunRepo) UpdateRunStatus(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
	return &model.Run{ID: id, Status: status}, nil
}
func (m *retryMockRunRepo) CountRunsByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *retryMockRunRepo) CountRunsByStory(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *retryMockRunRepo) CreateRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}
func (m *retryMockRunRepo) ListRunStepsByRun(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}
func (m *retryMockRunRepo) UpdateRunStepStatus(_ context.Context, id uuid.UUID, status model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
	return &model.RunStep{ID: id, Status: status}, nil
}
func (m *retryMockRunRepo) UpdateRunStepContainerInfo(_ context.Context, id uuid.UUID, _ *string, _ *string) (*model.RunStep, error) {
	return &model.RunStep{ID: id}, nil
}
func (m *retryMockRunRepo) ListRetryStepsByParent(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}

// retryMockAgentRun is a mock AgentRunExecutor.
type retryMockAgentRun struct {
	executeFn  func(ctx context.Context, runCtx *model.RunContext) error
	lastRunCtx *model.RunContext
}

func (m *retryMockAgentRun) Execute(ctx context.Context, runCtx *model.RunContext) error {
	m.lastRunCtx = runCtx
	if m.executeFn != nil {
		return m.executeFn(ctx, runCtx)
	}
	return nil
}

// buildRetryRunCtx creates a minimal RunContext for retry tests.
func buildRetryRunCtx(parentStepID uuid.UUID, extraMeta map[string]any) *model.RunContext {
	meta := map[string]any{
		"parent_step_id":   parentStepID.String(),
		"agent_image":      "test/agent:latest",
		"template_content": "Implement {{story_key}}",
	}
	for k, v := range extraMeta {
		meta[k] = v
	}
	return &model.RunContext{
		Run:       &model.Run{ID: uuid.New()},
		RunStep:   &model.RunStep{ID: uuid.New()},
		ProjectID: uuid.New(),
		StoryID:   uuid.New(),
		Metadata:  meta,
	}
}

// makeParentStep creates a RunStep suitable as a parent for retry tests.
func makeParentStep(retryCount int, errorMsg, logTail string) *model.RunStep {
	step := &model.RunStep{
		ID:         uuid.New(),
		RunID:      uuid.New(),
		StepName:   "implement",
		StepOrder:  1,
		Action:     "agent_run",
		Status:     model.StepStatusFailed,
		RetryCount: retryCount,
	}
	if errorMsg != "" {
		step.ErrorMessage = &errorMsg
	}
	if logTail != "" {
		step.LogTail = &logTail
	}
	return step
}

// --- Tests ---

// TestIncrementalRetryAction_Name verifies the action identifier.
func TestIncrementalRetryAction_Name(t *testing.T) {
	t.Parallel()

	a := action.NewIncrementalRetryAction(
		&retryMockRunRepo{}, &retryMockAgentRun{}, testLogger(),
	)
	if got := a.Name(); got != "incremental_retry" {
		t.Errorf("Name() = %q; want %q", got, "incremental_retry")
	}
}

// TestIncrementalRetryAction_MissingParentStepID verifies RETRY_MISSING_PARENT error.
func TestIncrementalRetryAction_MissingParentStepID(t *testing.T) {
	t.Parallel()

	repo := &retryMockRunRepo{}
	agentRun := &retryMockAgentRun{}
	a := action.NewIncrementalRetryAction(repo, agentRun, testLogger())

	runCtx := &model.RunContext{
		Run:      &model.Run{ID: uuid.New()},
		RunStep:  &model.RunStep{ID: uuid.New()},
		Metadata: map[string]any{}, // no parent_step_id
	}

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error for missing parent_step_id, got nil")
	}
	if !strings.Contains(err.Error(), "RETRY_MISSING_PARENT") {
		t.Errorf("expected RETRY_MISSING_PARENT in error, got: %v", err)
	}
	if agentRun.lastRunCtx != nil {
		t.Error("expected AgentRun not to be called, but it was")
	}
}

// TestIncrementalRetryAction_FirstIncrementalRetry verifies the happy path:
// parent.retry_count=0 → new step with retry_type="incremental", error_context injected.
func TestIncrementalRetryAction_FirstIncrementalRetry(t *testing.T) {
	t.Parallel()

	parent := makeParentStep(0, "test error", "last log line")
	agentRun := &retryMockAgentRun{}
	repo := &retryMockRunRepo{
		getRunStepFn: func(_ context.Context, _ uuid.UUID) (*model.RunStep, error) {
			return parent, nil
		},
	}

	a := action.NewIncrementalRetryAction(repo, agentRun, testLogger())

	runCtx := buildRetryRunCtx(parent.ID, nil)
	if err := a.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	createdStep := repo.createdStep
	if createdStep == nil {
		t.Fatal("expected a retry step to be created")
	}
	if createdStep.RetryCount != 1 {
		t.Errorf("RetryCount = %d; want 1", createdStep.RetryCount)
	}
	if createdStep.RetryType == nil || *createdStep.RetryType != "incremental" {
		t.Errorf("RetryType = %v; want %q", createdStep.RetryType, "incremental")
	}
	if createdStep.ParentStepID == nil || *createdStep.ParentStepID != parent.ID {
		t.Errorf("ParentStepID = %v; want %v", createdStep.ParentStepID, parent.ID)
	}

	if agentRun.lastRunCtx == nil {
		t.Fatal("expected AgentRun.Execute to be called")
	}
	if agentRun.lastRunCtx.RunStep.ID != createdStep.ID {
		t.Errorf("delegated RunStep.ID = %v; want %v", agentRun.lastRunCtx.RunStep.ID, createdStep.ID)
	}
	if agentRun.lastRunCtx.Metadata["error_context"] != "test error" {
		t.Errorf("error_context = %v; want %q", agentRun.lastRunCtx.Metadata["error_context"], "test error")
	}
	if agentRun.lastRunCtx.Metadata["log_tail"] != "last log line" {
		t.Errorf("log_tail = %v; want %q", agentRun.lastRunCtx.Metadata["log_tail"], "last log line")
	}
}

// TestIncrementalRetryAction_SecondFailureFallsToFullRetry verifies that when
// parent.retry_count >= max_incremental (default 2), retry_type switches to "full"
// and error_context/log_tail are cleared.
func TestIncrementalRetryAction_SecondFailureFallsToFullRetry(t *testing.T) {
	t.Parallel()

	parent := makeParentStep(2, "persistent error", "some logs")
	retryType := "incremental"
	parent.RetryType = &retryType

	agentRun := &retryMockAgentRun{}
	repo := &retryMockRunRepo{
		getRunStepFn: func(_ context.Context, _ uuid.UUID) (*model.RunStep, error) {
			return parent, nil
		},
	}

	a := action.NewIncrementalRetryAction(repo, agentRun, testLogger())

	runCtx := buildRetryRunCtx(parent.ID, nil)
	if err := a.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	createdStep := repo.createdStep
	if createdStep == nil {
		t.Fatal("expected a retry step to be created")
	}
	if createdStep.RetryCount != 3 {
		t.Errorf("RetryCount = %d; want 3", createdStep.RetryCount)
	}
	if createdStep.RetryType == nil || *createdStep.RetryType != "full" {
		t.Errorf("RetryType = %v; want %q", createdStep.RetryType, "full")
	}
	if agentRun.lastRunCtx == nil {
		t.Fatal("expected AgentRun.Execute to be called")
	}
	// Full retry should NOT have error_context
	if _, ok := agentRun.lastRunCtx.Metadata["error_context"]; ok {
		t.Error("expected error_context to be cleared for full retry")
	}
	if _, ok := agentRun.lastRunCtx.Metadata["log_tail"]; ok {
		t.Error("expected log_tail to be cleared for full retry")
	}
}

// TestIncrementalRetryAction_MaxRetriesExceeded verifies RETRY_MAX_EXCEEDED error.
func TestIncrementalRetryAction_MaxRetriesExceeded(t *testing.T) {
	t.Parallel()

	parent := makeParentStep(3, "error", "") // retry_count == default max_retries (3)
	agentRun := &retryMockAgentRun{}
	repo := &retryMockRunRepo{
		getRunStepFn: func(_ context.Context, _ uuid.UUID) (*model.RunStep, error) {
			return parent, nil
		},
	}

	a := action.NewIncrementalRetryAction(repo, agentRun, testLogger())

	runCtx := buildRetryRunCtx(parent.ID, nil)
	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected RETRY_MAX_EXCEEDED error, got nil")
	}
	if !strings.Contains(err.Error(), "RETRY_MAX_EXCEEDED") {
		t.Errorf("expected RETRY_MAX_EXCEEDED in error, got: %v", err)
	}
	if repo.createdStep != nil {
		t.Error("expected no retry step to be created on max-retries-exceeded")
	}
	if agentRun.lastRunCtx != nil {
		t.Error("expected AgentRun not to be called on max-retries-exceeded")
	}
}

// TestIncrementalRetryAction_CreateRetryStepFailure verifies that when the repo
// fails to create a retry step, Execute returns an error and AgentRun is not called.
func TestIncrementalRetryAction_CreateRetryStepFailure(t *testing.T) {
	t.Parallel()

	parent := makeParentStep(0, "error", "")
	agentRun := &retryMockAgentRun{}
	createErr := fmt.Errorf("db write failure")
	repo := &retryMockRunRepo{
		getRunStepFn: func(_ context.Context, _ uuid.UUID) (*model.RunStep, error) {
			return parent, nil
		},
		createRetryRunStepFn: func(_ context.Context, _ *model.RunStep) (*model.RunStep, error) {
			return nil, createErr
		},
	}

	a := action.NewIncrementalRetryAction(repo, agentRun, testLogger())

	runCtx := buildRetryRunCtx(parent.ID, nil)
	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error from CreateRetryRunStep, got nil")
	}
	if agentRun.lastRunCtx != nil {
		t.Error("expected AgentRun not to be called when step creation fails")
	}
}

// TestIncrementalRetryAction_CustomRetryPolicy verifies that metadata-specified
// retry policy values override the defaults.
func TestIncrementalRetryAction_CustomRetryPolicy(t *testing.T) {
	t.Parallel()

	// max_retries=5, max_incremental=1 → with retry_count=1, should switch to full
	parent := makeParentStep(1, "error", "")
	agentRun := &retryMockAgentRun{}
	repo := &retryMockRunRepo{
		getRunStepFn: func(_ context.Context, _ uuid.UUID) (*model.RunStep, error) {
			return parent, nil
		},
	}

	a := action.NewIncrementalRetryAction(repo, agentRun, testLogger())

	extraMeta := map[string]any{
		"retry_policy.max_retries":     5,
		"retry_policy.max_incremental": 1,
	}
	runCtx := buildRetryRunCtx(parent.ID, extraMeta)
	if err := a.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	createdStep := repo.createdStep
	if createdStep == nil {
		t.Fatal("expected a retry step to be created")
	}
	if createdStep.RetryType == nil || *createdStep.RetryType != "full" {
		t.Errorf("RetryType = %v; want %q", createdStep.RetryType, "full")
	}
	// Full retry should clear error_context
	if _, ok := agentRun.lastRunCtx.Metadata["error_context"]; ok {
		t.Error("expected error_context to be cleared for full retry")
	}
}

// TestIncrementalRetryAction_StageStampedFromParent verifies that a retry step
// carries the StageID and StageName of the parent step (Dette D1 fix).
func TestIncrementalRetryAction_StageStampedFromParent(t *testing.T) {
	t.Parallel()

	const wantStageID = "stage-dev"
	const wantStageName = "Development"

	parent := makeParentStep(0, "error", "")
	parent.StageID = wantStageID
	parent.StageName = wantStageName

	agentRun := &retryMockAgentRun{}
	repo := &retryMockRunRepo{
		getRunStepFn: func(_ context.Context, _ uuid.UUID) (*model.RunStep, error) {
			return parent, nil
		},
	}

	a := action.NewIncrementalRetryAction(repo, agentRun, testLogger())
	if err := a.Execute(context.Background(), buildRetryRunCtx(parent.ID, nil)); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	createdStep := repo.createdStep
	if createdStep == nil {
		t.Fatal("expected a retry step to be created")
	}
	if createdStep.StageID != wantStageID {
		t.Errorf("retry step StageID = %q; want %q", createdStep.StageID, wantStageID)
	}
	if createdStep.StageName != wantStageName {
		t.Errorf("retry step StageName = %q; want %q", createdStep.StageName, wantStageName)
	}
}

func (m *retryMockRunRepo) GetLatestRunByStory(_ context.Context, _ uuid.UUID) (*model.LatestRun, error) {
	return nil, nil
}

func (m *retryMockRunRepo) GetLatestRunsByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*model.LatestRun, error) {
	return map[uuid.UUID]*model.LatestRun{}, nil
}

func (m *retryMockRunRepo) GetDAGNodeRunInfoByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]model.DAGNodeRunInfo, error) {
	return map[uuid.UUID]model.DAGNodeRunInfo{}, nil
}

func (m *retryMockRunRepo) UpdateRunMetadata(_ context.Context, _ uuid.UUID, _ map[string]interface{}) error {
	return nil
}

func (m *retryMockRunRepo) AppendStepLogTail(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
