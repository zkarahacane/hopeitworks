package model

import (
	"time"

	"github.com/google/uuid"
)

// P2c1 persists the Environment (table `environments`, migration 000036) behind a
// repository: data model + storage only, still additive. Sidecar orchestration on a
// per-run isolated network, conn-string injection and golden images land in P2c2.
// The invariant is portability: an Environment must stay K8s-Pod-expressible
// (services are sidecars-in-Pod, never DinD).

// Environment sources — where the run composition is derived from. The repo's
// own files win when present; otherwise the stack + services are declared in UI.
const (
	EnvironmentSourceDevcontainer = "devcontainer"
	EnvironmentSourceCompose      = "compose"
	EnvironmentSourceMakefile     = "makefile"
	EnvironmentSourceDeclared     = "declared"
)

// Environment is a project's execution composition: stack(s) + sidecar services
// + the commands to run (test/migrate/seed). It is distinct from a Stack image:
// a Stack is a base image, an Environment is how a project actually runs.
type Environment struct {
	ID        uuid.UUID            `json:"id"`
	ProjectID uuid.UUID            `json:"project_id"`
	Stacks    []string             `json:"stacks"`   // one or more StackKey values
	Services  []EnvironmentService `json:"services"` // sidecars (postgres/redis/mailhog…)
	Source    string               `json:"source"`   // EnvironmentSource*
	Commands  map[string]string    `json:"commands"` // {test:"make test", migrate:"…", seed:"…"}
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
}

// EnvironmentService is a sidecar brought up alongside the agent on a per-run
// isolated network, with its connection string injected into the run env.
type EnvironmentService struct {
	Name  string            `json:"name"`
	Image string            `json:"image"`
	Env   map[string]string `json:"env"`
}
