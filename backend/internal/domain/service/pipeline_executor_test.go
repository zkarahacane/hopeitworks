package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// testLogger creates a logger for tests using slog directly to avoid stdout noise.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// mockAction implements model.Action for testing.
type mockAction struct {
	name      string
	executeFn func(ctx context.Context, runCtx *model.RunContext) error
}

func (a *mockAction) Name() string { return a.name }
func (a *mockAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
	if a.executeFn != nil {
		return a.executeFn(ctx, runCtx)
	}
	return nil
}

// mockActionRegistry implements port.ActionRegistry for testing.
type mockActionRegistry struct {
	actions map[string]model.Action
}

func newMockActionRegistry() *mockActionRegistry {
	return &mockActionRegistry{actions: make(map[string]model.Action)}
}

func (r *mockActionRegistry) Register(action model.Action) {
	r.actions[action.Name()] = action
}

func (r *mockActionRegistry) Get(name string) (model.Action, error) {
	action, ok := r.actions[name]
	if !ok {
		return nil, errors.NewNotFound("action", name)
	}
	return action, nil
}

// publishedEvent records a call to EventPublisher.Publish.
type publishedEvent struct {
	Event model.Event
}

// mockEventPublisher implements port.EventPublisher for testing.
type mockEventPublisher struct {
	mu     sync.Mutex
	events []publishedEvent
}

func newMockEventPublisher() *mockEventPublisher {
	return &mockEventPublisher{}
}

func (p *mockEventPublisher) Publish(_ context.Context, event model.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, publishedEvent{Event: event})
	return nil
}

func (p *mockEventPublisher) getEvents() []publishedEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]publishedEvent, len(p.events))
	copy(result, p.events)
	return result
}

// runStatusCall records a call to UpdateRunStatus.
type runStatusCall struct {
	ID          uuid.UUID
	Status      model.RunStatus
	StartedAt   *time.Time
	CompletedAt *time.Time
	ErrorMsg    *string
}

// stepStatusCall records a call to UpdateRunStepStatus.
type stepStatusCall struct {
	ID          uuid.UUID
	Status      model.StepStatus
	StartedAt   *time.Time
	CompletedAt *time.Time
	ErrorMsg    *string
}

// executorTestFixture provides shared test setup for PipelineExecutor tests.
type executorTestFixture struct {
	runID     uuid.UUID
	projectID uuid.UUID
	storyID   uuid.UUID
	steps     []*model.RunStep
	run       *model.Run

	runRepo   *mockRunRepo
	actionReg *mockActionRegistry
	eventPub  *mockEventPublisher
	executor  *PipelineExecutor

	runStatusCalls  []runStatusCall
	stepStatusCalls []stepStatusCall
	mu              sync.Mutex
}

func newExecutorTestFixture(stepCount int) *executorTestFixture {
	f := &executorTestFixture{
		runID:     uuid.New(),
		projectID: uuid.New(),
		storyID:   uuid.New(),
		actionReg: newMockActionRegistry(),
		eventPub:  newMockEventPublisher(),
	}

	f.run = &model.Run{
		ID:        f.runID,
		ProjectID: f.projectID,
		StoryID:   f.storyID,
		Status:    model.RunStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	f.steps = make([]*model.RunStep, stepCount)
	for i := 0; i < stepCount; i++ {
		f.steps[i] = &model.RunStep{
			ID:        uuid.New(),
			RunID:     f.runID,
			StepName:  fmt.Sprintf("step-%d", i),
			StepOrder: i,
			Action:    fmt.Sprintf("action_%d", i),
			Status:    model.StepStatusPending,
			CreatedAt: time.Now(),
		}
	}

	f.runRepo = &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			if id == f.runID {
				return f.run, nil
			}
			return nil, errors.NewNotFound("run", id)
		},
		listRunStepsByRunFn: func(_ context.Context, runID uuid.UUID) ([]*model.RunStep, error) {
			if runID == f.runID {
				result := make([]*model.RunStep, len(f.steps))
				for i, s := range f.steps {
					cp := *s
					result[i] = &cp
				}
				return result, nil
			}
			return nil, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.Run, error) {
			f.mu.Lock()
			f.runStatusCalls = append(f.runStatusCalls, runStatusCall{
				ID: id, Status: status, StartedAt: startedAt, CompletedAt: completedAt, ErrorMsg: errorMsg,
			})
			f.mu.Unlock()

			run := *f.run
			run.Status = status
			if startedAt != nil {
				run.StartedAt = startedAt
			}
			if completedAt != nil {
				run.CompletedAt = completedAt
			}
			if errorMsg != nil {
				run.ErrorMessage = errorMsg
			}
			return &run, nil
		},
		updateRunStepStatusFn: func(_ context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.RunStep, error) {
			f.mu.Lock()
			f.stepStatusCalls = append(f.stepStatusCalls, stepStatusCall{
				ID: id, Status: status, StartedAt: startedAt, CompletedAt: completedAt, ErrorMsg: errorMsg,
			})
			f.mu.Unlock()

			for _, s := range f.steps {
				if s.ID == id {
					cp := *s
					cp.Status = status
					if startedAt != nil {
						cp.StartedAt = startedAt
					}
					if completedAt != nil {
						cp.CompletedAt = completedAt
					}
					if errorMsg != nil {
						cp.ErrorMessage = errorMsg
					}
					return &cp, nil
				}
			}
			return nil, errors.NewNotFound("run_step", id)
		},
	}

	f.executor = NewPipelineExecutor(f.runRepo, f.actionReg, f.eventPub, testLogger())

	return f
}

// registerSuccessActions registers actions that succeed for all steps.
func (f *executorTestFixture) registerSuccessActions() {
	for _, step := range f.steps {
		f.actionReg.Register(&mockAction{name: step.Action})
	}
}

func TestExecuteRun_HappyPath(t *testing.T) {
	f := newExecutorTestFixture(3)
	f.registerSuccessActions()

	var executionOrder []string
	for _, step := range f.steps {
		stepAction := step.Action
		f.actionReg.Register(&mockAction{
			name: stepAction,
			executeFn: func(_ context.Context, runCtx *model.RunContext) error {
				executionOrder = append(executionOrder, runCtx.RunStep.StepName)
				return nil
			},
		})
	}

	err := f.executor.ExecuteRun(context.Background(), f.runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify steps executed in order
	if len(executionOrder) != 3 {
		t.Fatalf("expected 3 steps executed, got %d", len(executionOrder))
	}
	for i, name := range executionOrder {
		expected := fmt.Sprintf("step-%d", i)
		if name != expected {
			t.Errorf("step %d: expected %q, got %q", i, expected, name)
		}
	}

	// Verify run status transitions: pending → running → completed
	if len(f.runStatusCalls) != 2 {
		t.Fatalf("expected 2 run status updates, got %d", len(f.runStatusCalls))
	}
	if f.runStatusCalls[0].Status != model.RunStatusRunning {
		t.Errorf("first run status update: expected running, got %s", f.runStatusCalls[0].Status)
	}
	if f.runStatusCalls[0].StartedAt == nil {
		t.Error("first run status update: expected started_at to be set")
	}
	if f.runStatusCalls[1].Status != model.RunStatusCompleted {
		t.Errorf("second run status update: expected completed, got %s", f.runStatusCalls[1].Status)
	}
	if f.runStatusCalls[1].CompletedAt == nil {
		t.Error("second run status update: expected completed_at to be set")
	}

	// Verify step status transitions: each step goes running → completed
	if len(f.stepStatusCalls) != 6 {
		t.Fatalf("expected 6 step status updates (2 per step), got %d", len(f.stepStatusCalls))
	}
	for i := 0; i < 3; i++ {
		runningCall := f.stepStatusCalls[i*2]
		completedCall := f.stepStatusCalls[i*2+1]

		if runningCall.Status != model.StepStatusRunning {
			t.Errorf("step %d: expected running status, got %s", i, runningCall.Status)
		}
		if runningCall.StartedAt == nil {
			t.Errorf("step %d: expected started_at to be set on running transition", i)
		}
		if completedCall.Status != model.StepStatusCompleted {
			t.Errorf("step %d: expected completed status, got %s", i, completedCall.Status)
		}
		if completedCall.CompletedAt == nil {
			t.Errorf("step %d: expected completed_at to be set on completed transition", i)
		}
	}
}

func TestExecuteRun_EventsPublishedInOrder(t *testing.T) {
	f := newExecutorTestFixture(2)
	f.registerSuccessActions()

	err := f.executor.ExecuteRun(context.Background(), f.runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	events := f.eventPub.getEvents()

	// Expected event order:
	// run.started, step.started(0), step.completed(0), step.started(1), step.completed(1), run.completed
	expectedEvents := []string{
		"run.started",
		"step.started",
		"step.completed",
		"step.started",
		"step.completed",
		"run.completed",
	}

	if len(events) != len(expectedEvents) {
		t.Fatalf("expected %d events, got %d", len(expectedEvents), len(events))
	}

	for i, expected := range expectedEvents {
		actual := events[i].Event.EventName()
		if actual != expected {
			t.Errorf("event %d: expected %q, got %q", i, expected, actual)
		}
	}

	// Verify run.started event payload
	var startedPayload map[string]any
	if err := json.Unmarshal(events[0].Event.Payload, &startedPayload); err != nil {
		t.Fatalf("failed to unmarshal run.started payload: %v", err)
	}
	if startedPayload["run_id"] != f.runID.String() {
		t.Errorf("run.started payload: expected run_id %s, got %v", f.runID, startedPayload["run_id"])
	}
	if startedPayload["status"] != string(model.RunStatusRunning) {
		t.Errorf("run.started payload: expected status running, got %v", startedPayload["status"])
	}

	// Verify step events include step_id and step_name
	var stepPayload map[string]any
	if err := json.Unmarshal(events[1].Event.Payload, &stepPayload); err != nil {
		t.Fatalf("failed to unmarshal step.started payload: %v", err)
	}
	if _, ok := stepPayload["step_id"]; !ok {
		t.Error("step.started payload: missing step_id")
	}
	if _, ok := stepPayload["step_name"]; !ok {
		t.Error("step.started payload: missing step_name")
	}

	// Verify run.completed event payload
	var completedPayload map[string]any
	if err := json.Unmarshal(events[len(events)-1].Event.Payload, &completedPayload); err != nil {
		t.Fatalf("failed to unmarshal run.completed payload: %v", err)
	}
	if completedPayload["run_id"] != f.runID.String() {
		t.Errorf("run.completed payload: expected run_id %s, got %v", f.runID, completedPayload["run_id"])
	}
}

func TestExecuteRun_StepFailure(t *testing.T) {
	f := newExecutorTestFixture(3)

	stepErr := fmt.Errorf("step 1 execution failed: compilation error")

	// step 0 succeeds, step 1 fails, step 2 should not run
	f.actionReg.Register(&mockAction{name: f.steps[0].Action})
	f.actionReg.Register(&mockAction{
		name: f.steps[1].Action,
		executeFn: func(_ context.Context, _ *model.RunContext) error {
			return stepErr
		},
	})

	var step2Executed bool
	f.actionReg.Register(&mockAction{
		name: f.steps[2].Action,
		executeFn: func(_ context.Context, _ *model.RunContext) error {
			step2Executed = true
			return nil
		},
	})

	err := f.executor.ExecuteRun(context.Background(), f.runID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != stepErr.Error() {
		t.Errorf("expected error %q, got %q", stepErr.Error(), err.Error())
	}

	// Step 2 should NOT have been executed
	if step2Executed {
		t.Error("step 2 should not have been executed after step 1 failure")
	}

	// Verify run status transitions: running → failed
	if len(f.runStatusCalls) < 2 {
		t.Fatalf("expected at least 2 run status updates, got %d", len(f.runStatusCalls))
	}
	if f.runStatusCalls[0].Status != model.RunStatusRunning {
		t.Errorf("first run update: expected running, got %s", f.runStatusCalls[0].Status)
	}
	lastRunCall := f.runStatusCalls[len(f.runStatusCalls)-1]
	if lastRunCall.Status != model.RunStatusFailed {
		t.Errorf("last run update: expected failed, got %s", lastRunCall.Status)
	}
	if lastRunCall.ErrorMsg == nil {
		t.Error("last run update: expected error message to be set")
	} else if *lastRunCall.ErrorMsg != stepErr.Error() {
		t.Errorf("last run update: expected error message %q, got %q", stepErr.Error(), *lastRunCall.ErrorMsg)
	}

	// Verify step 1 marked as failed with error message
	var foundStepFailed bool
	for _, call := range f.stepStatusCalls {
		if call.ID == f.steps[1].ID && call.Status == model.StepStatusFailed {
			foundStepFailed = true
			if call.ErrorMsg == nil {
				t.Error("step 1 failure: expected error message to be set")
			} else if *call.ErrorMsg != stepErr.Error() {
				t.Errorf("step 1 failure: expected error message %q, got %q", stepErr.Error(), *call.ErrorMsg)
			}
		}
	}
	if !foundStepFailed {
		t.Error("expected step 1 to be marked as failed")
	}

	// Verify events include step.failed and run.failed
	events := f.eventPub.getEvents()
	var foundStepFailedEvent, foundRunFailedEvent bool
	for _, e := range events {
		eventName := e.Event.EventName()
		if eventName == "step.failed" {
			foundStepFailedEvent = true
			var payload map[string]any
			if err := json.Unmarshal(e.Event.Payload, &payload); err == nil {
				if _, ok := payload["error_message"]; !ok {
					t.Error("step.failed event: missing error_message in payload")
				}
			}
		}
		if eventName == "run.failed" {
			foundRunFailedEvent = true
			var payload map[string]any
			if err := json.Unmarshal(e.Event.Payload, &payload); err == nil {
				if _, ok := payload["error_message"]; !ok {
					t.Error("run.failed event: missing error_message in payload")
				}
			}
		}
	}
	if !foundStepFailedEvent {
		t.Error("expected step.failed event to be published")
	}
	if !foundRunFailedEvent {
		t.Error("expected run.failed event to be published")
	}
}

func TestExecuteRun_Cancellation(t *testing.T) {
	f := newExecutorTestFixture(3)

	ctx, cancel := context.WithCancel(context.Background())

	// step 0 succeeds and then cancels the context
	f.actionReg.Register(&mockAction{
		name: f.steps[0].Action,
		executeFn: func(_ context.Context, _ *model.RunContext) error {
			cancel()
			return nil
		},
	})
	f.actionReg.Register(&mockAction{name: f.steps[1].Action})
	f.actionReg.Register(&mockAction{name: f.steps[2].Action})

	err := f.executor.ExecuteRun(ctx, f.runID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Verify run ended up as cancelled
	lastRunCall := f.runStatusCalls[len(f.runStatusCalls)-1]
	if lastRunCall.Status != model.RunStatusCancelled {
		t.Errorf("last run update: expected cancelled, got %s", lastRunCall.Status)
	}

	// Verify a step was marked as cancelled
	var foundStepCancelled bool
	for _, call := range f.stepStatusCalls {
		if call.Status == model.StepStatusCancelled {
			foundStepCancelled = true
		}
	}
	if !foundStepCancelled {
		t.Error("expected at least one step to be marked as cancelled")
	}

	// Verify events include step.cancelled and run.cancelled
	events := f.eventPub.getEvents()
	var foundStepCancelledEvent, foundRunCancelledEvent bool
	for _, e := range events {
		eventName := e.Event.EventName()
		if eventName == "step.cancelled" {
			foundStepCancelledEvent = true
		}
		if eventName == "run.cancelled" {
			foundRunCancelledEvent = true
		}
	}
	if !foundStepCancelledEvent {
		t.Error("expected step.cancelled event to be published")
	}
	if !foundRunCancelledEvent {
		t.Error("expected run.cancelled event to be published")
	}
}

func TestExecuteRun_StepTimestamps(t *testing.T) {
	f := newExecutorTestFixture(1)
	f.registerSuccessActions()

	before := time.Now()
	err := f.executor.ExecuteRun(context.Background(), f.runID)
	after := time.Now()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify step status calls have timestamps
	if len(f.stepStatusCalls) != 2 {
		t.Fatalf("expected 2 step status calls, got %d", len(f.stepStatusCalls))
	}

	// Running transition should have started_at
	runningCall := f.stepStatusCalls[0]
	if runningCall.Status != model.StepStatusRunning {
		t.Errorf("expected running status, got %s", runningCall.Status)
	}
	if runningCall.StartedAt == nil {
		t.Fatal("expected started_at to be set for running transition")
	}
	if runningCall.StartedAt.Before(before) || runningCall.StartedAt.After(after) {
		t.Error("started_at timestamp not within test execution window")
	}

	// Completed transition should have completed_at
	completedCall := f.stepStatusCalls[1]
	if completedCall.Status != model.StepStatusCompleted {
		t.Errorf("expected completed status, got %s", completedCall.Status)
	}
	if completedCall.CompletedAt == nil {
		t.Fatal("expected completed_at to be set for completed transition")
	}
	if completedCall.CompletedAt.Before(before) || completedCall.CompletedAt.After(after) {
		t.Error("completed_at timestamp not within test execution window")
	}

	// started_at should be before or equal to completed_at
	if runningCall.StartedAt.After(*completedCall.CompletedAt) {
		t.Error("started_at should not be after completed_at")
	}
}

func TestExecuteRun_MetadataSharedBetweenSteps(t *testing.T) {
	f := newExecutorTestFixture(2)

	// step 0 writes to metadata
	f.actionReg.Register(&mockAction{
		name: f.steps[0].Action,
		executeFn: func(_ context.Context, runCtx *model.RunContext) error {
			runCtx.Metadata["branch_name"] = "feat/test-branch"
			return nil
		},
	})

	// step 1 reads from metadata
	var branchName string
	f.actionReg.Register(&mockAction{
		name: f.steps[1].Action,
		executeFn: func(_ context.Context, runCtx *model.RunContext) error {
			if val, ok := runCtx.Metadata["branch_name"]; ok {
				branchName = val.(string)
			}
			return nil
		},
	})

	err := f.executor.ExecuteRun(context.Background(), f.runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if branchName != "feat/test-branch" {
		t.Errorf("expected branch_name %q from metadata, got %q", "feat/test-branch", branchName)
	}
}

func TestExecuteRun_RunNotFound(t *testing.T) {
	f := newExecutorTestFixture(0)
	unknownID := uuid.New()

	err := f.executor.ExecuteRun(context.Background(), unknownID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %s", domainErr.Category)
	}
}

func TestExecuteRun_ActionNotFound(t *testing.T) {
	f := newExecutorTestFixture(1)
	// Do NOT register any actions — action lookup should fail

	err := f.executor.ExecuteRun(context.Background(), f.runID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify run ended up failed
	lastRunCall := f.runStatusCalls[len(f.runStatusCalls)-1]
	if lastRunCall.Status != model.RunStatusFailed {
		t.Errorf("expected run to be failed, got %s", lastRunCall.Status)
	}
}

func TestExecuteRun_StepOrderRespected(t *testing.T) {
	f := newExecutorTestFixture(3)

	// Return steps in reverse order from the repository
	f.runRepo.listRunStepsByRunFn = func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
		reversed := make([]*model.RunStep, len(f.steps))
		for i, s := range f.steps {
			cp := *s
			reversed[len(f.steps)-1-i] = &cp
		}
		return reversed, nil
	}

	var executionOrder []int
	for _, step := range f.steps {
		order := step.StepOrder
		f.actionReg.Register(&mockAction{
			name: step.Action,
			executeFn: func(_ context.Context, _ *model.RunContext) error {
				executionOrder = append(executionOrder, order)
				return nil
			},
		})
	}

	err := f.executor.ExecuteRun(context.Background(), f.runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(executionOrder) != 3 {
		t.Fatalf("expected 3 steps executed, got %d", len(executionOrder))
	}
	for i, order := range executionOrder {
		if order != i {
			t.Errorf("execution position %d: expected step_order %d, got %d", i, i, order)
		}
	}
}

func TestExecuteRun_FailureEventOrder(t *testing.T) {
	f := newExecutorTestFixture(3)

	// step 0 succeeds, step 1 fails
	f.actionReg.Register(&mockAction{name: f.steps[0].Action})
	f.actionReg.Register(&mockAction{
		name: f.steps[1].Action,
		executeFn: func(_ context.Context, _ *model.RunContext) error {
			return fmt.Errorf("build failed")
		},
	})
	f.actionReg.Register(&mockAction{name: f.steps[2].Action})

	_ = f.executor.ExecuteRun(context.Background(), f.runID)

	events := f.eventPub.getEvents()

	// Expected event order:
	// run.started, step.started(0), step.completed(0), step.started(1), step.failed(1), run.failed
	expectedEvents := []string{
		"run.started",
		"step.started",
		"step.completed",
		"step.started",
		"step.failed",
		"run.failed",
	}

	if len(events) != len(expectedEvents) {
		t.Fatalf("expected %d events, got %d", len(expectedEvents), len(events))
	}

	for i, expected := range expectedEvents {
		actual := events[i].Event.EventName()
		if actual != expected {
			t.Errorf("event %d: expected %q, got %q", i, expected, actual)
		}
	}
}

// TestActionRegistry tests the mock ActionRegistry behavior (AC #3, #9).
func TestExecuteRun_StepSuspendedForApproval(t *testing.T) {
	f := newExecutorTestFixture(3)

	// step 0 succeeds normally
	f.actionReg.Register(&mockAction{name: f.steps[0].Action})

	// step 1 simulates hitl_gate: action returns nil, but the step is
	// transitioned to waiting_approval during execution
	f.actionReg.Register(&mockAction{
		name: f.steps[1].Action,
		executeFn: func(_ context.Context, _ *model.RunContext) error {
			// Simulate what HITLGateAction does: update step status to waiting_approval
			// The executor re-fetches step after Execute() returns nil
			return nil
		},
	})

	// Override GetRunStep to return waiting_approval for step 1
	f.runRepo.getRunStepFn = func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
		if id == f.steps[1].ID {
			return &model.RunStep{
				ID:     id,
				RunID:  f.runID,
				Status: model.StepStatusWaitingApproval,
			}, nil
		}
		// For other steps, return normal status
		for _, s := range f.steps {
			if s.ID == id {
				cp := *s
				return &cp, nil
			}
		}
		return nil, errors.NewNotFound("run_step", id)
	}

	var step2Executed bool
	f.actionReg.Register(&mockAction{
		name: f.steps[2].Action,
		executeFn: func(_ context.Context, _ *model.RunContext) error {
			step2Executed = true
			return nil
		},
	})

	err := f.executor.ExecuteRun(context.Background(), f.runID)
	// ExecuteRun should return nil (suspension is not an error)
	if err != nil {
		t.Fatalf("expected nil error (suspension), got %v", err)
	}

	// Step 2 should NOT have been executed
	if step2Executed {
		t.Error("step 2 should not have been executed after step 1 was suspended")
	}

	// Run should NOT be marked as completed (it's still running, waiting for approval)
	var runCompleted bool
	for _, call := range f.runStatusCalls {
		if call.Status == model.RunStatusCompleted {
			runCompleted = true
		}
	}
	if runCompleted {
		t.Error("run should not be marked as completed when a step is suspended")
	}

	// Run should NOT be marked as failed
	var runFailed bool
	for _, call := range f.runStatusCalls {
		if call.Status == model.RunStatusFailed {
			runFailed = true
		}
	}
	if runFailed {
		t.Error("run should not be marked as failed when a step is suspended")
	}
}

func TestActionRegistry_RegisterAndGet(t *testing.T) {
	reg := newMockActionRegistry()

	action := &mockAction{name: "test_action"}
	reg.Register(action)

	result, err := reg.Get("test_action")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name() != "test_action" {
		t.Errorf("expected action name %q, got %q", "test_action", result.Name())
	}
}

func TestActionRegistry_GetUnknown(t *testing.T) {
	reg := newMockActionRegistry()

	_, err := reg.Get("unknown_action")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %s", domainErr.Category)
	}
}

func TestActionRegistry_RegisterOverwrites(t *testing.T) {
	reg := newMockActionRegistry()

	action1 := &mockAction{
		name: "test_action",
		executeFn: func(_ context.Context, _ *model.RunContext) error {
			return fmt.Errorf("action1")
		},
	}
	action2 := &mockAction{
		name: "test_action",
		executeFn: func(_ context.Context, _ *model.RunContext) error {
			return fmt.Errorf("action2")
		},
	}

	reg.Register(action1)
	reg.Register(action2)

	result, err := reg.Get("test_action")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Execute to verify it's the second action
	execErr := result.Execute(context.Background(), nil)
	if execErr == nil || execErr.Error() != "action2" {
		t.Errorf("expected action2 error, got %v", execErr)
	}
}
