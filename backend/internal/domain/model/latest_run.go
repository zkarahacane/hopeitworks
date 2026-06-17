package model

import "github.com/google/uuid"

// LatestRunStep is the currently active step (running or waiting_approval) of a
// run, with its position within the pipeline. Used by the live kanban to show
// the current pipeline stage on a story card.
type LatestRunStep struct {
	ID         uuid.UUID
	Name       string
	ActionType string
	Status     string
	// Index is the zero-based step_order of the current step.
	Index int
	// Total is the number of steps in the run.
	Total int
	// ContainerID is the Docker container id of the current step, nil when no
	// container is attached yet.
	ContainerID *string
}

// LatestRun is a lightweight projection of a story's most recent run, carrying
// the run status and the current in-progress step (nil when no step is active).
type LatestRun struct {
	ID          uuid.UUID
	Status      string
	CurrentStep *LatestRunStep
}
