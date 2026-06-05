package model

import (
	"time"

	"github.com/google/uuid"
)

// Story status constants.
const (
	StoryStatusBacklog = "backlog"
	StoryStatusRunning = "running"
	StoryStatusDone    = "done"
	StoryStatusFailed  = "failed"
)

// Story scope constants.
const (
	StoryScopeBackend  = "backend"
	StoryScopeFrontend = "frontend"
	StoryScopeShared   = "shared"
)

// StoryCounts holds the number of stories per lifecycle status. Used to populate
// an epic's aggregate progress on the board without N per-status queries.
type StoryCounts struct {
	Backlog int
	Running int
	Done    int
	Failed  int
}

// Story represents a user story within a project.
type Story struct {
	ID                 uuid.UUID
	ProjectID          uuid.UUID
	EpicID             *uuid.UUID
	Key                string
	Title              string
	Objective          *string
	TargetFiles        []string
	DependsOn          []string
	Scope              *string
	Status             string
	AcceptanceCriteria *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
