package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

func TestCleanupOrphans_NoOrphans(t *testing.T) {
	runID1 := uuid.New()
	runID2 := uuid.New()

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-active-1",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID1.String()},
				},
				{
					ID:     "container-active-2",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID2.String()},
				},
			}, nil
		},
	}

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			switch id {
			case runID1:
				return &model.Run{ID: runID1, Status: model.RunStatusRunning}, nil
			case runID2:
				return &model.Run{ID: runID2, Status: model.RunStatusPending}, nil
			default:
				return nil, errors.NewNotFound("run", id)
			}
		},
	}

	cleaner := NewOrphanCleaner(containerMgr, runRepo, discardLogger())

	err := cleaner.CleanupOrphans(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	removeCalls := containerMgr.getRemoveCalls()
	if len(removeCalls) != 0 {
		t.Errorf("expected no remove calls, got %d", len(removeCalls))
	}
}

func TestCleanupOrphans_MultipleOrphans(t *testing.T) {
	activeRunID := uuid.New()
	completedRunID := uuid.New()
	missingRunID := uuid.New()

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-active",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": activeRunID.String()},
				},
				{
					ID:     "container-no-label",
					Labels: map[string]string{"managed_by": "hopeitworks"},
				},
				{
					ID:     "container-completed",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": completedRunID.String()},
				},
				{
					ID:     "container-missing-run",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": missingRunID.String()},
				},
			}, nil
		},
	}

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			switch id {
			case activeRunID:
				return &model.Run{ID: activeRunID, Status: model.RunStatusRunning}, nil
			case completedRunID:
				return &model.Run{ID: completedRunID, Status: model.RunStatusCompleted}, nil
			default:
				return nil, errors.NewNotFound("run", id)
			}
		},
	}

	cleaner := NewOrphanCleaner(containerMgr, runRepo, discardLogger())

	err := cleaner.CleanupOrphans(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	removeCalls := containerMgr.getRemoveCalls()
	if len(removeCalls) != 3 {
		t.Fatalf("expected 3 remove calls (no-label, completed, missing-run), got %d", len(removeCalls))
	}

	expected := map[string]bool{
		"container-no-label":    false,
		"container-completed":   false,
		"container-missing-run": false,
	}
	for _, id := range removeCalls {
		if _, ok := expected[id]; !ok {
			t.Errorf("unexpected remove call for %s", id)
		}
		expected[id] = true
	}
	for id, called := range expected {
		if !called {
			t.Errorf("expected remove call for %s, but it was not called", id)
		}
	}
}

func TestCleanupOrphans_OrphanReasons(t *testing.T) {
	failedRunID := uuid.New()
	cancelledRunID := uuid.New()

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-no-run-id",
					Labels: map[string]string{"managed_by": "hopeitworks"},
				},
				{
					ID:     "container-invalid-uuid",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": "not-a-uuid"},
				},
				{
					ID:     "container-failed-run",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": failedRunID.String()},
				},
				{
					ID:     "container-cancelled-run",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": cancelledRunID.String()},
				},
			}, nil
		},
	}

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			switch id {
			case failedRunID:
				return &model.Run{ID: failedRunID, Status: model.RunStatusFailed}, nil
			case cancelledRunID:
				return &model.Run{ID: cancelledRunID, Status: model.RunStatusCancelled}, nil
			default:
				return nil, errors.NewNotFound("run", id)
			}
		},
	}

	cleaner := NewOrphanCleaner(containerMgr, runRepo, discardLogger())

	err := cleaner.CleanupOrphans(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	removeCalls := containerMgr.getRemoveCalls()
	if len(removeCalls) != 4 {
		t.Fatalf("expected 4 remove calls, got %d", len(removeCalls))
	}
}

func TestCleanupOrphans_RemoveFailureContinues(t *testing.T) {
	runID1 := uuid.New()
	runID2 := uuid.New()

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-fail-remove",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID1.String()},
				},
				{
					ID:     "container-ok-remove",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID2.String()},
				},
			}, nil
		},
		removeFn: func(_ context.Context, containerID string) error {
			if containerID == "container-fail-remove" {
				return errors.NewInternal("remove failed", nil)
			}
			return nil
		},
	}

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			// Both runs not found → both containers are orphans
			return nil, errors.NewNotFound("run", id)
		},
	}

	cleaner := NewOrphanCleaner(containerMgr, runRepo, discardLogger())

	err := cleaner.CleanupOrphans(context.Background())
	if err != nil {
		t.Fatalf("expected no error (cleanup continues on failure), got %v", err)
	}

	// Both containers should have Remove called, even though first one fails
	removeCalls := containerMgr.getRemoveCalls()
	if len(removeCalls) != 2 {
		t.Fatalf("expected 2 remove calls (cleanup continues), got %d", len(removeCalls))
	}
}

func TestCleanupOrphans_EmptyContainerList(t *testing.T) {
	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{}, nil
		},
	}

	runRepo := &mockRunRepo{}
	cleaner := NewOrphanCleaner(containerMgr, runRepo, discardLogger())

	err := cleaner.CleanupOrphans(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	removeCalls := containerMgr.getRemoveCalls()
	if len(removeCalls) != 0 {
		t.Errorf("expected no remove calls, got %d", len(removeCalls))
	}
}

func TestCleanupOrphans_ListContainersError(t *testing.T) {
	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return nil, errors.NewInternal("docker unreachable", nil)
		},
	}

	runRepo := &mockRunRepo{}
	cleaner := NewOrphanCleaner(containerMgr, runRepo, discardLogger())

	err := cleaner.CleanupOrphans(context.Background())
	if err == nil {
		t.Fatal("expected error when ListContainers fails, got nil")
	}
}
