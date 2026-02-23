package model

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// RunStatus represents the status of a pipeline run.
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusPaused    RunStatus = "paused"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

// StepStatus represents the status of a run step.
type StepStatus string

const (
	StepStatusPending         StepStatus = "pending"
	StepStatusRunning         StepStatus = "running"
	StepStatusCompleted       StepStatus = "completed"
	StepStatusFailed          StepStatus = "failed"
	StepStatusCancelled       StepStatus = "cancelled"
	StepStatusWaitingApproval StepStatus = "waiting_approval"
)

// Run represents a pipeline execution run.
type Run struct {
	ID                     uuid.UUID
	ProjectID              uuid.UUID
	StoryID                uuid.UUID
	StoryKey               string // optional: populated via JOIN with stories table, empty if unavailable
	Status                 RunStatus
	PipelineConfigSnapshot json.RawMessage
	Metadata               map[string]interface{}
	StartedAt              *time.Time
	CompletedAt            *time.Time
	PausedAt               *time.Time
	ErrorMessage           *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	Steps                  []RunStep
	Progress               int // computed, not persisted
}

// ComputeProgress computes the run progress as a percentage (0–100)
// based on the number of completed steps. Returns 0 if there are no steps.
func (r *Run) ComputeProgress(steps []RunStep) int {
	if len(steps) == 0 {
		return 0
	}
	completed := 0
	for _, s := range steps {
		if s.Status == StepStatusCompleted {
			completed++
		}
	}
	return int(float64(completed) / float64(len(steps)) * 100)
}

// RunStep represents an individual step within a pipeline run.
type RunStep struct {
	ID           uuid.UUID
	RunID        uuid.UUID
	StepName     string
	StepOrder    int
	Action       string
	Status       StepStatus
	StartedAt    *time.Time
	CompletedAt  *time.Time
	ErrorMessage *string
	ContainerID  *string
	LogTail      *string
	// RetryCount is the number of retries attempted from this step.
	RetryCount int
	// RetryType is "incremental" or "full"; nil for original (non-retry) steps.
	RetryType *string
	// ParentStepID is the ID of the step this was retried from; nil for original steps.
	ParentStepID *uuid.UUID
	CreatedAt    time.Time
	// Config holds step-specific configuration from the pipeline YAML.
	// This is a transient field populated by the executor at runtime, not persisted in the database.
	Config map[string]string
}

var validRunTransitions = map[RunStatus][]RunStatus{
	RunStatusPending: {RunStatusRunning, RunStatusCancelled},
	RunStatusRunning: {RunStatusPaused, RunStatusCompleted, RunStatusFailed, RunStatusCancelled},
	RunStatusPaused:  {RunStatusRunning, RunStatusCancelled},
	RunStatusFailed:  {RunStatusRunning}, // retry transitions failed run back to running
}

var validStepTransitions = map[StepStatus][]StepStatus{
	StepStatusPending:         {StepStatusRunning, StepStatusCancelled},
	StepStatusRunning:         {StepStatusCompleted, StepStatusFailed, StepStatusCancelled, StepStatusWaitingApproval},
	StepStatusWaitingApproval: {StepStatusRunning, StepStatusCompleted, StepStatusFailed, StepStatusCancelled},
}

// ValidateRunTransition checks if a run status transition is valid.
func ValidateRunTransition(from, to RunStatus) error {
	allowed, ok := validRunTransitions[from]
	if !ok {
		return errors.NewInvalidState(errors.ErrCodeInvalidStateTransition,
			fmt.Sprintf("no transitions allowed from run status: %s", from))
	}
	for _, valid := range allowed {
		if valid == to {
			return nil
		}
	}
	return errors.NewInvalidState(errors.ErrCodeInvalidStateTransition,
		fmt.Sprintf("cannot transition run from %s to %s", from, to))
}

// ValidateStepTransition checks if a step status transition is valid.
func ValidateStepTransition(from, to StepStatus) error {
	allowed, ok := validStepTransitions[from]
	if !ok {
		return errors.NewInvalidState(errors.ErrCodeInvalidStateTransition,
			fmt.Sprintf("no transitions allowed from step status: %s", from))
	}
	for _, valid := range allowed {
		if valid == to {
			return nil
		}
	}
	return errors.NewInvalidState(errors.ErrCodeInvalidStateTransition,
		fmt.Sprintf("cannot transition step from %s to %s", from, to))
}
