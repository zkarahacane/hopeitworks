package service

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockContainerManager implements port.ContainerManager for testing.
type mockContainerManager struct {
	mu            sync.Mutex
	listFn        func(ctx context.Context, labels map[string]string) ([]port.ContainerInfo, error)
	listRunningFn func(ctx context.Context, labels map[string]string) ([]port.ContainerInfo, error)
	stopCalls     []string
	stopFn        func(ctx context.Context, containerID string) error
	removeCalls   []string
	removeFn      func(ctx context.Context, containerID string) error
	createFn      func(ctx context.Context, opts model.ContainerOpts) (string, error)
	startFn       func(ctx context.Context, containerID string) error
	waitFn        func(ctx context.Context, containerID string) (int, error)
}

func (m *mockContainerManager) ListContainers(ctx context.Context, labels map[string]string) ([]port.ContainerInfo, error) {
	if m.listFn != nil {
		return m.listFn(ctx, labels)
	}
	return nil, nil
}

func (m *mockContainerManager) Stop(ctx context.Context, containerID string) error {
	m.mu.Lock()
	m.stopCalls = append(m.stopCalls, containerID)
	m.mu.Unlock()
	if m.stopFn != nil {
		return m.stopFn(ctx, containerID)
	}
	return nil
}

func (m *mockContainerManager) Remove(ctx context.Context, containerID string) error {
	m.mu.Lock()
	m.removeCalls = append(m.removeCalls, containerID)
	m.mu.Unlock()
	if m.removeFn != nil {
		return m.removeFn(ctx, containerID)
	}
	return nil
}

func (m *mockContainerManager) Create(ctx context.Context, opts model.ContainerOpts) (string, error) {
	if m.createFn != nil {
		return m.createFn(ctx, opts)
	}
	return "", nil
}

func (m *mockContainerManager) Start(ctx context.Context, containerID string) error {
	if m.startFn != nil {
		return m.startFn(ctx, containerID)
	}
	return nil
}

func (m *mockContainerManager) Wait(ctx context.Context, containerID string) (int, error) {
	if m.waitFn != nil {
		return m.waitFn(ctx, containerID)
	}
	return 0, nil
}

func (m *mockContainerManager) ListRunningContainers(ctx context.Context, labels map[string]string) ([]port.ContainerInfo, error) {
	if m.listRunningFn != nil {
		return m.listRunningFn(ctx, labels)
	}
	return nil, nil
}

func (m *mockContainerManager) CreateNetwork(_ context.Context, _ string, _ map[string]string) (string, error) {
	return "", nil
}

func (m *mockContainerManager) RemoveNetwork(_ context.Context, _ string) error {
	return nil
}

func (m *mockContainerManager) ConnectContainer(_ context.Context, _, _ string, _ []string) error {
	return nil
}

func (m *mockContainerManager) DisconnectContainer(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockContainerManager) ListNetworks(_ context.Context, _ map[string]string) ([]model.NetworkInfo, error) {
	return nil, nil
}

func (m *mockContainerManager) InspectHealth(_ context.Context, _ string) (string, error) {
	return model.HealthRunning, nil
}

func (m *mockContainerManager) getStopCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.stopCalls))
	copy(result, m.stopCalls)
	return result
}

func (m *mockContainerManager) getRemoveCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.removeCalls))
	copy(result, m.removeCalls)
	return result
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCheckTimeouts_ContainerExceedsTimeout(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()

	startedAt := time.Now().Add(-40 * time.Minute)

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-timeout",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID.String(), "step_id": stepID.String()},
				},
			}, nil
		},
	}

	var updatedStepStatus model.StepStatus
	var updatedStepErr *string
	var updatedRunStatus model.RunStatus
	var updatedRunErr *string

	runRepo := &mockRunRepo{
		getRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			if id == stepID {
				return &model.RunStep{
					ID:        stepID,
					RunID:     runID,
					Status:    model.StepStatusRunning,
					StartedAt: &startedAt,
				}, nil
			}
			return nil, errors.NewNotFound("run_step", id)
		},
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			if id == runID {
				return &model.Run{
					ID:        runID,
					ProjectID: projectID,
					Status:    model.RunStatusRunning,
				}, nil
			}
			return nil, errors.NewNotFound("run", id)
		},
		updateRunStepStatusFn: func(_ context.Context, id uuid.UUID, status model.StepStatus, _ *time.Time, _ *time.Time, errMsg *string) (*model.RunStep, error) {
			updatedStepStatus = status
			updatedStepErr = errMsg
			return &model.RunStep{ID: id, Status: status}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _ *time.Time, _ *time.Time, _ *time.Time, errMsg *string) (*model.Run, error) {
			updatedRunStatus = status
			updatedRunErr = errMsg
			return &model.Run{ID: id, Status: status}, nil
		},
	}

	projectRepo := newMockProjectRepoWithProject(&model.Project{
		ID:   projectID,
		Name: "test-project",
	})

	enforcer := NewTimeoutEnforcer(containerMgr, runRepo, projectRepo, discardLogger(), 30*time.Minute, 30*time.Second, nil)

	err := enforcer.CheckTimeouts(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stopCalls := containerMgr.getStopCalls()
	if len(stopCalls) != 1 {
		t.Fatalf("expected 1 stop call, got %d", len(stopCalls))
	}
	if stopCalls[0] != "container-timeout" {
		t.Errorf("expected stop on container-timeout, got %s", stopCalls[0])
	}

	if updatedStepStatus != model.StepStatusFailed {
		t.Errorf("expected step status failed, got %s", updatedStepStatus)
	}
	if updatedStepErr == nil || *updatedStepErr != containerTimeoutReason {
		t.Errorf("expected step error %q, got %v", containerTimeoutReason, updatedStepErr)
	}

	if updatedRunStatus != model.RunStatusFailed {
		t.Errorf("expected run status failed, got %s", updatedRunStatus)
	}
	if updatedRunErr == nil || *updatedRunErr != containerTimeoutReason {
		t.Errorf("expected run error %q, got %v", containerTimeoutReason, updatedRunErr)
	}
}

func TestCheckTimeouts_ContainerWithinTimeout(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()

	startedAt := time.Now().Add(-5 * time.Minute)

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-ok",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID.String(), "step_id": stepID.String()},
				},
			}, nil
		},
	}

	runRepo := &mockRunRepo{
		getRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			if id == stepID {
				return &model.RunStep{
					ID:        stepID,
					RunID:     runID,
					Status:    model.StepStatusRunning,
					StartedAt: &startedAt,
				}, nil
			}
			return nil, errors.NewNotFound("run_step", id)
		},
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			if id == runID {
				return &model.Run{
					ID:        runID,
					ProjectID: projectID,
					Status:    model.RunStatusRunning,
				}, nil
			}
			return nil, errors.NewNotFound("run", id)
		},
	}

	projectRepo := newMockProjectRepoWithProject(&model.Project{
		ID:   projectID,
		Name: "test-project",
	})

	enforcer := NewTimeoutEnforcer(containerMgr, runRepo, projectRepo, discardLogger(), 30*time.Minute, 30*time.Second, nil)

	err := enforcer.CheckTimeouts(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stopCalls := containerMgr.getStopCalls()
	if len(stopCalls) != 0 {
		t.Errorf("expected no stop calls, got %d", len(stopCalls))
	}
}

func TestCheckTimeouts_ProjectTimeoutOverridesDefault(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()

	// Container started 20 minutes ago
	startedAt := time.Now().Add(-20 * time.Minute)

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-project-timeout",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID.String(), "step_id": stepID.String()},
				},
			}, nil
		},
	}

	runRepo := &mockRunRepo{
		getRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			if id == stepID {
				return &model.RunStep{
					ID:        stepID,
					RunID:     runID,
					Status:    model.StepStatusRunning,
					StartedAt: &startedAt,
				}, nil
			}
			return nil, errors.NewNotFound("run_step", id)
		},
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			if id == runID {
				return &model.Run{
					ID:        runID,
					ProjectID: projectID,
					Status:    model.RunStatusRunning,
				}, nil
			}
			return nil, errors.NewNotFound("run", id)
		},
		updateRunStepStatusFn: func(_ context.Context, id uuid.UUID, status model.StepStatus, _ *time.Time, _ *time.Time, _ *string) (*model.RunStep, error) {
			return &model.RunStep{ID: id, Status: status}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, Status: status}, nil
		},
	}

	// Project has 15-minute timeout, container is running for 20 minutes → should timeout
	projectTimeout := 15 * time.Minute
	projectRepo := newMockProjectRepoWithProject(&model.Project{
		ID:                  projectID,
		Name:                "strict-project",
		MaxContainerTimeout: &projectTimeout,
	})

	enforcer := NewTimeoutEnforcer(containerMgr, runRepo, projectRepo, discardLogger(), 30*time.Minute, 30*time.Second, nil)

	err := enforcer.CheckTimeouts(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stopCalls := containerMgr.getStopCalls()
	if len(stopCalls) != 1 {
		t.Fatalf("expected 1 stop call (project timeout 15m < 20m elapsed), got %d", len(stopCalls))
	}
}

func TestCheckTimeouts_ProjectTimeoutNotExceeded(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()

	// Container started 20 minutes ago
	startedAt := time.Now().Add(-20 * time.Minute)

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-project-ok",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID.String(), "step_id": stepID.String()},
				},
			}, nil
		},
	}

	runRepo := &mockRunRepo{
		getRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			if id == stepID {
				return &model.RunStep{
					ID:        stepID,
					RunID:     runID,
					Status:    model.StepStatusRunning,
					StartedAt: &startedAt,
				}, nil
			}
			return nil, errors.NewNotFound("run_step", id)
		},
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			if id == runID {
				return &model.Run{
					ID:        runID,
					ProjectID: projectID,
					Status:    model.RunStatusRunning,
				}, nil
			}
			return nil, errors.NewNotFound("run", id)
		},
	}

	// Project has 60-minute timeout, container running 20m → should NOT timeout
	projectTimeout := 60 * time.Minute
	projectRepo := newMockProjectRepoWithProject(&model.Project{
		ID:                  projectID,
		Name:                "lenient-project",
		MaxContainerTimeout: &projectTimeout,
	})

	enforcer := NewTimeoutEnforcer(containerMgr, runRepo, projectRepo, discardLogger(), 30*time.Minute, 30*time.Second, nil)

	err := enforcer.CheckTimeouts(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stopCalls := containerMgr.getStopCalls()
	if len(stopCalls) != 0 {
		t.Errorf("expected no stop calls (project timeout 60m > 20m elapsed), got %d", len(stopCalls))
	}
}

func TestCheckTimeouts_MultipleContainers(t *testing.T) {
	runID1 := uuid.New()
	stepID1 := uuid.New()
	runID2 := uuid.New()
	stepID2 := uuid.New()
	projectID := uuid.New()

	startedRecent := time.Now().Add(-5 * time.Minute)
	startedOld := time.Now().Add(-40 * time.Minute)

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-ok",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID1.String(), "step_id": stepID1.String()},
				},
				{
					ID:     "container-timeout",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID2.String(), "step_id": stepID2.String()},
				},
			}, nil
		},
	}

	runRepo := &mockRunRepo{
		getRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			switch id {
			case stepID1:
				return &model.RunStep{ID: stepID1, RunID: runID1, Status: model.StepStatusRunning, StartedAt: &startedRecent}, nil
			case stepID2:
				return &model.RunStep{ID: stepID2, RunID: runID2, Status: model.StepStatusRunning, StartedAt: &startedOld}, nil
			default:
				return nil, errors.NewNotFound("run_step", id)
			}
		},
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			switch id {
			case runID1:
				return &model.Run{ID: runID1, ProjectID: projectID, Status: model.RunStatusRunning}, nil
			case runID2:
				return &model.Run{ID: runID2, ProjectID: projectID, Status: model.RunStatusRunning}, nil
			default:
				return nil, errors.NewNotFound("run", id)
			}
		},
		updateRunStepStatusFn: func(_ context.Context, id uuid.UUID, status model.StepStatus, _ *time.Time, _ *time.Time, _ *string) (*model.RunStep, error) {
			return &model.RunStep{ID: id, Status: status}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, Status: status}, nil
		},
	}

	projectRepo := newMockProjectRepoWithProject(&model.Project{
		ID:   projectID,
		Name: "test-project",
	})

	enforcer := NewTimeoutEnforcer(containerMgr, runRepo, projectRepo, discardLogger(), 30*time.Minute, 30*time.Second, nil)

	err := enforcer.CheckTimeouts(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stopCalls := containerMgr.getStopCalls()
	if len(stopCalls) != 1 {
		t.Fatalf("expected 1 stop call (only timed-out container), got %d", len(stopCalls))
	}
	if stopCalls[0] != "container-timeout" {
		t.Errorf("expected stop on container-timeout, got %s", stopCalls[0])
	}
}

func TestCheckTimeouts_StopFailureContinues(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()

	startedAt := time.Now().Add(-40 * time.Minute)

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-fail",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID.String(), "step_id": stepID.String()},
				},
			}, nil
		},
		stopFn: func(_ context.Context, _ string) error {
			return errors.NewInternal("stop failed", nil)
		},
	}

	runRepo := &mockRunRepo{
		getRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			if id == stepID {
				return &model.RunStep{ID: stepID, RunID: runID, Status: model.StepStatusRunning, StartedAt: &startedAt}, nil
			}
			return nil, errors.NewNotFound("run_step", id)
		},
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			if id == runID {
				return &model.Run{ID: runID, ProjectID: projectID, Status: model.RunStatusRunning}, nil
			}
			return nil, errors.NewNotFound("run", id)
		},
	}

	projectRepo := newMockProjectRepoWithProject(&model.Project{
		ID:   projectID,
		Name: "test-project",
	})

	enforcer := NewTimeoutEnforcer(containerMgr, runRepo, projectRepo, discardLogger(), 30*time.Minute, 30*time.Second, nil)

	// Should not panic or return error even though Stop fails
	err := enforcer.CheckTimeouts(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCheckTimeouts_SkipsMissingLabels(t *testing.T) {
	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-no-labels",
					Labels: map[string]string{"managed_by": "hopeitworks"},
				},
				{
					ID:     "container-no-step",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": uuid.New().String()},
				},
			}, nil
		},
	}

	runRepo := &mockRunRepo{}
	projectRepo := newMockProjectRepoForService()

	enforcer := NewTimeoutEnforcer(containerMgr, runRepo, projectRepo, discardLogger(), 30*time.Minute, 30*time.Second, nil)

	err := enforcer.CheckTimeouts(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stopCalls := containerMgr.getStopCalls()
	if len(stopCalls) != 0 {
		t.Errorf("expected no stop calls for containers without labels, got %d", len(stopCalls))
	}
}

func TestCheckTimeouts_NilStartedAt(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-not-started",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID.String(), "step_id": stepID.String()},
				},
			}, nil
		},
	}

	runRepo := &mockRunRepo{
		getRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			if id == stepID {
				return &model.RunStep{
					ID:        stepID,
					RunID:     runID,
					Status:    model.StepStatusPending,
					StartedAt: nil,
				}, nil
			}
			return nil, errors.NewNotFound("run_step", id)
		},
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			if id == runID {
				return &model.Run{ID: runID, ProjectID: projectID, Status: model.RunStatusRunning}, nil
			}
			return nil, errors.NewNotFound("run", id)
		},
	}

	projectRepo := newMockProjectRepoWithProject(&model.Project{
		ID:   projectID,
		Name: "test-project",
	})

	enforcer := NewTimeoutEnforcer(containerMgr, runRepo, projectRepo, discardLogger(), 30*time.Minute, 30*time.Second, nil)

	err := enforcer.CheckTimeouts(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stopCalls := containerMgr.getStopCalls()
	if len(stopCalls) != 0 {
		t.Errorf("expected no stop calls for step without started_at, got %d", len(stopCalls))
	}
}

func TestStart_StopsOnContextCancellation(t *testing.T) {
	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return nil, nil
		},
	}
	runRepo := &mockRunRepo{}
	projectRepo := newMockProjectRepoForService()

	enforcer := NewTimeoutEnforcer(containerMgr, runRepo, projectRepo, discardLogger(), 30*time.Minute, 50*time.Millisecond, nil)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- enforcer.Start(ctx)
	}()

	// Let it run at least one tick
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout enforcer did not stop after context cancellation")
	}
}
