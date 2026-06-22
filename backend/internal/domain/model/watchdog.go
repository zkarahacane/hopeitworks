package model

import (
	"time"

	"github.com/google/uuid"
)

// RunningStep is the watchdog read model for a step that is currently running.
// It carries just enough context to evaluate the board-side guards (INC 4a):
// log_silence (now - LastLogAt), wallclock (now - StartedAt) and cost_batch
// (cumulative run cost looked up separately by RunID).
type RunningStep struct {
	StepID    uuid.UUID
	RunID     uuid.UUID
	StepName  string
	StageID   string
	StageName string
	ProjectID uuid.UUID
	StoryID   uuid.UUID
	// StartedAt is when the step transitioned to running; nil if not yet stamped.
	StartedAt *time.Time
	// LastLogAt is the created_at of the most recent log.emitted event for this
	// step; nil when the step has produced no log yet (fall back to StartedAt for
	// the log_silence baseline).
	LastLogAt *time.Time
}
