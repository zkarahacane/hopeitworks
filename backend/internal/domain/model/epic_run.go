package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// EpicRunStatus represents the status of an epic run.
type EpicRunStatus string

const (
	// EpicRunStatusPending indicates the epic run has been created but not started.
	EpicRunStatusPending EpicRunStatus = "pending"
	// EpicRunStatusRunning indicates the epic run is actively executing.
	EpicRunStatusRunning EpicRunStatus = "running"
	// EpicRunStatusCompleted indicates all stories completed successfully.
	EpicRunStatusCompleted EpicRunStatus = "completed"
	// EpicRunStatusFailed indicates the epic run failed due to a story failure.
	EpicRunStatusFailed EpicRunStatus = "failed"
	// EpicRunStatusPaused indicates the epic run is paused.
	EpicRunStatusPaused EpicRunStatus = "paused"
)

// EpicRun represents a batch execution of all stories in an epic.
type EpicRun struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	EpicID      uuid.UUID
	Status      EpicRunStatus
	CreatedAt   time.Time
	CompletedAt *time.Time
	Stories     []EpicRunStory
}

// EpicRunStory represents a single story within an epic run.
type EpicRunStory struct {
	EpicRunID  uuid.UUID
	StoryID    uuid.UUID
	RunID      *uuid.UUID
	GroupIndex int
	Status     string
}

var validEpicRunTransitions = map[EpicRunStatus][]EpicRunStatus{
	EpicRunStatusPending: {EpicRunStatusRunning},
	EpicRunStatusRunning: {EpicRunStatusCompleted, EpicRunStatusFailed, EpicRunStatusPaused},
}

// ValidateEpicRunTransition checks if an epic run status transition is valid.
func ValidateEpicRunTransition(from, to EpicRunStatus) error {
	allowed, ok := validEpicRunTransitions[from]
	if !ok {
		return errors.NewInvalidState("INVALID_STATE_TRANSITION",
			fmt.Sprintf("no transitions allowed from epic run status: %s", from))
	}
	for _, valid := range allowed {
		if valid == to {
			return nil
		}
	}
	return errors.NewInvalidState("INVALID_STATE_TRANSITION",
		fmt.Sprintf("cannot transition epic run from %s to %s", from, to))
}
