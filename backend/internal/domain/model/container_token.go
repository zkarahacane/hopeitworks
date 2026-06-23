package model

import (
	"time"

	"github.com/google/uuid"
)

// ContainerToken is a short-lived bearer token issued to an agent container.
// It authenticates callback HTTP requests from the container back to the API,
// including the fetch-at-startup capability bundle. AgentID identifies which agent
// the container runs, so the bundle endpoint can resolve its composed capabilities
// server-side (the container never names the agent itself — it cannot fetch another
// agent's bundle/secrets). AgentID is uuid.Nil when no agent is bound, which yields
// an empty bundle (back-compat).
//
// Role is the step role resolved from the pipeline config (e.g. "dev", "review", "merge").
// An empty role means the step is not role-scoped; in that case only universal capabilities
// (Roles == nil/empty) are included in the bundle.
type ContainerToken struct {
	Token     string
	RunID     uuid.UUID
	StepID    uuid.UUID
	AgentID   uuid.UUID
	Role      string // role du step, ex. "dev"/"review"/"merge" ; "" = non scopé
	ExpiresAt time.Time
}
