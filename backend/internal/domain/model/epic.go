package model

import (
	"time"

	"github.com/google/uuid"
)

// Epic status constants.
const (
	EpicStatusBacklog    = "backlog"
	EpicStatusInProgress = "in_progress"
	EpicStatusDone       = "done"
)

// Epic represents an epic within a project.
type Epic struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Description *string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
