package model

import (
	"time"

	"github.com/google/uuid"
)

// EpicStatus represents the status of an epic.
type EpicStatus string

const (
	EpicStatusBacklog    EpicStatus = "backlog"
	EpicStatusInProgress EpicStatus = "in_progress"
	EpicStatusDone       EpicStatus = "done"
)

// IsValid checks if the epic status is a known value.
func (s EpicStatus) IsValid() bool {
	return s == EpicStatusBacklog || s == EpicStatusInProgress || s == EpicStatusDone
}

// Epic represents an epic within a project.
type Epic struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Description *string
	Status      EpicStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
