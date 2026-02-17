package model

import "github.com/google/uuid"

// RunContext provides context for action execution.
// It carries the current run, step, and shared metadata across the pipeline.
type RunContext struct {
	// Run is the current pipeline run.
	Run *Run

	// RunStep is the current step being executed.
	RunStep *RunStep

	// ProjectID is the ID of the project owning this run.
	ProjectID uuid.UUID

	// StoryID is the ID of the story being processed.
	StoryID uuid.UUID

	// Metadata holds inter-step data (e.g., branch name, PR URL).
	// Previous steps can write to this map, later steps can read from it.
	Metadata map[string]any
}
