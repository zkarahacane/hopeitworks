package model

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a project in the domain.
type Project struct {
	ID                   uuid.UUID
	Name                 string
	Description          *string
	OwnerID              *uuid.UUID
	RepoURL              *string
	GitProvider          string
	GitTokenEnv          *string
	AgentRuntime         string
	DefaultModel         *string
	MaxBudget            *float64
	MaxContainerTimeout  *time.Duration
	CircuitBreakerCount  int
	CircuitBreakerActive bool
	CircuitBreakerMax    int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
