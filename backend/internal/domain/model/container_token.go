package model

import (
	"time"

	"github.com/google/uuid"
)

// ContainerToken is a short-lived bearer token issued to an agent container.
// It authenticates callback HTTP requests from the container back to the API.
type ContainerToken struct {
	Token     string
	RunID     uuid.UUID
	StepID    uuid.UUID
	ExpiresAt time.Time
}
