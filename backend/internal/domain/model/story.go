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
