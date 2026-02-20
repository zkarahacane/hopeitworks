package action_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/action"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

const (
	retryTypeIncremental = "incremental"
	retryTypeFull        = "full"
)

// --- Mocks specific to incremental retry tests ---

// retryMockRunRepo is a minimal RunRepository mock for IncrementalRetryAction tests.
type retryMockRunRepo struct {
	mockRunRepo
	retryGetRunStepFn    func(ctx context.Context, id uuid.UUID) (*model.RunStep, error)
	createRetryRunStepFn func(ctx context.Context, step *model.RunStep) (*model.RunStep, error)
	listRetryByParentFn  func(ctx context.Context, parentStepID uuid.UUID) ([]*model.RunStep, error)
	createdRetrySteps    []*model.RunStep
}

func (m *retryMockRunRepo) GetRunStep(ctx context.Context, id uuid.UUID) (*model.RunStep, error) {
	if m.retryGetRunStepFn != nil {
		return m.retryGetRunStepFn(ctx, id)
	}
	return nil, errors.NewNotFound("run_step", id)
}

func (m *retryMockRunRepo) CreateRetryRunStep(ctx context.Context, step *model.RunStep) (*model.RunStep, error) {
	m.createdRetrySteps = append(m.createdRetrySteps, step)
	if m.createRetryRunStepFn != nil {
		return m.createRetryRunStepFn(ctx, step)
	}
	return step, nil
}

func (m *retryMockRunRepo) ListRetryStepsByParent(ctx context.Context, parentStepID uuid.UUID) ([]*model.RunStep, error) {
	if m.listRetryByParentFn != nil {
		return m.listRetryByParentFn(ctx, parentStepID)
	}
	return nil, nil
}

// mockAgentRunExecutor mocks AgentRunExecutor for test delegation.
type mockAgentRunExecutor struct {
	executeFn  func(ctx context.Context, runCtx *model.RunContext) error
	callCount  int
	lastRunCtx *model.RunContext
}

func (m *mockAgentRunExecutor) Execute(ctx context.Context, runCtx *model.RunContext) error {
	m.callCount++
	m.lastRunCtx = runCtx
	if m.executeFn != nil {
		return m.executeFn(ctx, runCtx)
	}
	return nil
}

// --- Test fixture ---

type retryFixture struct {
	projectID    uuid.UUID
	storyID      uuid.UUID
	runID        uuid.UUID
	parentStepID uuid.UUID
	parentStep   *model.RunStep

	runRepo   *retryMockRunRepo
	agentExec *mockAgentRunExecutor
	action    *action.IncrementalRetryAction
}

func newRetryFixture() *retryFixture {
	f := &retryFixture{
		projectID:    uuid.New(),
		storyID:      uuid.New(),
		runID:        uuid.New(),
		parentStepID: uuid.New(),
	}

	errMsg := "agent exited with code 1"
	logTail := "line 1\nline 2\nline 3\nerror: build failed"
	f.parentStep = &model.RunStep{
		ID:           f.parentStepID,
		RunID:        f.runID,
		StepName:     "dev-story",
		StepOrder:    1,
		Action:       "agent_run",
		Status:       model.StepStatusFailed,
		ErrorMessage: &errMsg,
		LogTail:      &logTail,
		RetryCount:   0,
	}

	f.runRepo = &retryMockRunRepo{
		retryGetRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			if id == f.parentStepID {
				return f.parentStep, nil
			}
			return nil, errors.NewNotFound("run_step", id)
		},
	}

	f.agentExec = &mockAgentRunExecutor{}

	f.action = action.NewIncrementalRetryAction(
		f.runRepo, f.agentExec, testLogger(),
	)

	return f
}

func (f *retryFixture) newRunContext() *model.RunContext {
	return &model.RunContext{
		Run: &model.Run{
			ID:        f.runID,
			ProjectID: f.projectID,
			StoryID:   f.storyID,
			Status:    model.RunStatusRunning,
		},
		RunStep: &model.RunStep{
			ID:     uuid.New(),
			RunID:  f.runID,
			Action: "incremental_retry",
		},
		ProjectID: f.projectID,
		StoryID:   f.storyID,
		Metadata: map[string]any{
			"parent_step_id": f.parentStepID.String(),
		},
	}
}

// --- Tests ---

func TestIncrementalRetryAction_Name(t *testing.T) {
	f := newRetryFixture()
	if f.action.Name() != "incremental_retry" {
		t.Errorf("expected action name %q, got %q", "incremental_retry", f.action.Name())
	}
}

func TestIncrementalRetryAction_FirstIncrementalRetry(t *testing.T) {
	f := newRetryFixture()
	f.parentStep.RetryCount = 0
	runCtx := f.newRunContext()

	err := f.action.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify retry step was created
	if len(f.runRepo.createdRetrySteps) != 1 {
		t.Fatalf("expected 1 retry step created, got %d", len(f.runRepo.createdRetrySteps))
	}

	step := f.runRepo.createdRetrySteps[0]
	if step.RetryCount != 1 {
		t.Errorf("expected retry_count=1, got %d", step.RetryCount)
	}
	if step.RetryType == nil || *step.RetryType != retryTypeIncremental {
		t.Errorf("expected retry_type=incremental, got %v", step.RetryType)
	}
	if step.ParentStepID == nil || *step.ParentStepID != f.parentStepID {
		t.Errorf("expected parent_step_id=%s, got %v", f.parentStepID, step.ParentStepID)
	}
	if step.Status != model.StepStatusPending {
		t.Errorf("expected status=pending, got %s", step.Status)
	}

	// Verify delegation to agent executor
	if f.agentExec.callCount != 1 {
		t.Fatalf("expected 1 agent execution, got %d", f.agentExec.callCount)
	}

	// Verify metadata was set with template name and error context
	delegatedCtx := f.agentExec.lastRunCtx
	if delegatedCtx.Metadata["template_name"] != "implement-retry" {
		t.Errorf("expected template_name=implement-retry, got %v", delegatedCtx.Metadata["template_name"])
	}
	if delegatedCtx.Metadata["error_context"] != "agent exited with code 1" {
		t.Errorf("expected error_context from parent, got %v", delegatedCtx.Metadata["error_context"])
	}
	if !strings.Contains(fmt.Sprint(delegatedCtx.Metadata["log_tail"]), "error: build failed") {
		t.Errorf("expected log_tail from parent, got %v", delegatedCtx.Metadata["log_tail"])
	}
}

func TestIncrementalRetryAction_SecondIncrementalRetry(t *testing.T) {
	f := newRetryFixture()
	f.parentStep.RetryCount = 1
	incType := retryTypeIncremental
	f.parentStep.RetryType = &incType
	runCtx := f.newRunContext()

	err := f.action.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	step := f.runRepo.createdRetrySteps[0]
	if step.RetryCount != 2 {
		t.Errorf("expected retry_count=2, got %d", step.RetryCount)
	}
	if step.RetryType == nil || *step.RetryType != retryTypeIncremental {
		t.Errorf("expected retry_type=incremental, got %v", step.RetryType)
	}

	delegatedCtx := f.agentExec.lastRunCtx
	if delegatedCtx.Metadata["template_name"] != "implement-retry" {
		t.Errorf("expected template_name=implement-retry, got %v", delegatedCtx.Metadata["template_name"])
	}
}

func TestIncrementalRetryAction_FallbackToFullRetry(t *testing.T) {
	f := newRetryFixture()
	// After 2 incremental retries, should fall back to full retry
	f.parentStep.RetryCount = 2
	incType := retryTypeIncremental
	f.parentStep.RetryType = &incType
	runCtx := f.newRunContext()
	runCtx.Metadata["retry_policy.max_incremental"] = 2

	err := f.action.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	step := f.runRepo.createdRetrySteps[0]
	if step.RetryCount != 3 {
		t.Errorf("expected retry_count=3, got %d", step.RetryCount)
	}
	if step.RetryType == nil || *step.RetryType != retryTypeFull {
		t.Errorf("expected retry_type=full, got %v", step.RetryType)
	}

	// Verify full retry uses the implement template (not implement-retry)
	delegatedCtx := f.agentExec.lastRunCtx
	if delegatedCtx.Metadata["template_name"] != "implement" {
		t.Errorf("expected template_name=implement for full retry, got %v", delegatedCtx.Metadata["template_name"])
	}
}

func TestIncrementalRetryAction_MaxRetriesExceeded(t *testing.T) {
	f := newRetryFixture()
	f.parentStep.RetryCount = 3
	runCtx := f.newRunContext()
	runCtx.Metadata["retry_policy.max_retries"] = 3

	err := f.action.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error for max retries exceeded, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "RETRY_MAX_EXCEEDED" {
		t.Errorf("expected code RETRY_MAX_EXCEEDED, got %s", domainErr.Code)
	}

	// Verify no retry step was created
	if len(f.runRepo.createdRetrySteps) != 0 {
		t.Errorf("expected 0 retry steps created, got %d", len(f.runRepo.createdRetrySteps))
	}

	// Verify agent was NOT called
	if f.agentExec.callCount != 0 {
		t.Errorf("expected 0 agent executions, got %d", f.agentExec.callCount)
	}
}

func TestIncrementalRetryAction_MissingParentStepID(t *testing.T) {
	f := newRetryFixture()
	runCtx := f.newRunContext()
	delete(runCtx.Metadata, "parent_step_id")

	err := f.action.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error for missing parent_step_id, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "RETRY_MISSING_PARENT" {
		t.Errorf("expected code RETRY_MISSING_PARENT, got %s", domainErr.Code)
	}
}

func TestIncrementalRetryAction_InvalidParentStepID(t *testing.T) {
	f := newRetryFixture()
	runCtx := f.newRunContext()
	runCtx.Metadata["parent_step_id"] = "not-a-uuid"

	err := f.action.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error for invalid parent_step_id, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "RETRY_MISSING_PARENT" {
		t.Errorf("expected code RETRY_MISSING_PARENT, got %s", domainErr.Code)
	}
}

func TestIncrementalRetryAction_ParentStepNotFound(t *testing.T) {
	f := newRetryFixture()
	runCtx := f.newRunContext()
	unknownID := uuid.New()
	runCtx.Metadata["parent_step_id"] = unknownID.String()

	err := f.action.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error for parent step not found, got nil")
	}
	if !strings.Contains(err.Error(), "fetch parent step") {
		t.Errorf("expected error to wrap 'fetch parent step', got: %v", err)
	}
}

func TestIncrementalRetryAction_CreateRetryStepFailure(t *testing.T) {
	f := newRetryFixture()
	f.runRepo.createRetryRunStepFn = func(_ context.Context, _ *model.RunStep) (*model.RunStep, error) {
		return nil, fmt.Errorf("db connection lost")
	}

	runCtx := f.newRunContext()
	err := f.action.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error for create retry step failure, got nil")
	}
	if !strings.Contains(err.Error(), "create retry step") {
		t.Errorf("expected error to wrap 'create retry step', got: %v", err)
	}

	// Verify agent was NOT called
	if f.agentExec.callCount != 0 {
		t.Errorf("expected 0 agent executions after create failure, got %d", f.agentExec.callCount)
	}
}

func TestIncrementalRetryAction_AgentExecutionFailure(t *testing.T) {
	f := newRetryFixture()
	f.agentExec.executeFn = func(_ context.Context, _ *model.RunContext) error {
		return fmt.Errorf("container creation failed")
	}

	runCtx := f.newRunContext()
	err := f.action.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error from agent execution, got nil")
	}
	if !strings.Contains(err.Error(), "container creation failed") {
		t.Errorf("expected agent error to propagate, got: %v", err)
	}

	// Retry step should still have been created
	if len(f.runRepo.createdRetrySteps) != 1 {
		t.Errorf("expected 1 retry step created before agent failure, got %d", len(f.runRepo.createdRetrySteps))
	}
}

func TestIncrementalRetryAction_NilErrorMessageAndLogTail(t *testing.T) {
	f := newRetryFixture()
	f.parentStep.ErrorMessage = nil
	f.parentStep.LogTail = nil
	runCtx := f.newRunContext()

	err := f.action.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	delegatedCtx := f.agentExec.lastRunCtx
	if delegatedCtx.Metadata["error_context"] != "" {
		t.Errorf("expected empty error_context, got %v", delegatedCtx.Metadata["error_context"])
	}
	if delegatedCtx.Metadata["log_tail"] != "" {
		t.Errorf("expected empty log_tail, got %v", delegatedCtx.Metadata["log_tail"])
	}
}

func TestIncrementalRetryAction_CustomRetryPolicy(t *testing.T) {
	f := newRetryFixture()
	f.parentStep.RetryCount = 4
	runCtx := f.newRunContext()
	runCtx.Metadata["retry_policy.max_retries"] = 5
	runCtx.Metadata["retry_policy.max_incremental"] = 3

	err := f.action.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	step := f.runRepo.createdRetrySteps[0]
	if step.RetryCount != 5 {
		t.Errorf("expected retry_count=5, got %d", step.RetryCount)
	}
	if step.RetryType == nil || *step.RetryType != retryTypeFull {
		t.Errorf("expected retry_type=full for count >= max_incremental, got %v", step.RetryType)
	}
}

func TestIncrementalRetryAction_PreservesExistingMetadata(t *testing.T) {
	f := newRetryFixture()
	runCtx := f.newRunContext()
	runCtx.Metadata["branch_name"] = "feat/test-branch"
	runCtx.Metadata["custom_key"] = "custom_value"

	err := f.action.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	delegatedCtx := f.agentExec.lastRunCtx
	if delegatedCtx.Metadata["branch_name"] != "feat/test-branch" {
		t.Errorf("expected branch_name preserved, got %v", delegatedCtx.Metadata["branch_name"])
	}
	if delegatedCtx.Metadata["custom_key"] != "custom_value" {
		t.Errorf("expected custom_key preserved, got %v", delegatedCtx.Metadata["custom_key"])
	}
}

func TestIncrementalRetryAction_StepFieldsPropagated(t *testing.T) {
	f := newRetryFixture()
	runCtx := f.newRunContext()

	err := f.action.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	step := f.runRepo.createdRetrySteps[0]
	if step.RunID != f.runID {
		t.Errorf("expected run_id=%s, got %s", f.runID, step.RunID)
	}
	if step.StepName != f.parentStep.StepName {
		t.Errorf("expected step_name=%s, got %s", f.parentStep.StepName, step.StepName)
	}
	if step.StepOrder != f.parentStep.StepOrder {
		t.Errorf("expected step_order=%d, got %d", f.parentStep.StepOrder, step.StepOrder)
	}
	if step.Action != f.parentStep.Action {
		t.Errorf("expected action=%s, got %s", f.parentStep.Action, step.Action)
	}
}
