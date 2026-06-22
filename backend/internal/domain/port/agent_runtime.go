package port

import (
	"context"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// NOTE: This is the P0 target port — an agent-execution abstraction that
// supersedes the Docker-shaped ContainerManager as the domain's runtime port.
// It is scaffolding: no adapter implements it yet. The Docker substrate
// (microsandbox/K8s later) will implement AgentRuntime and keep ContainerManager
// as an internal detail. The live agent_run flow still drives ContainerManager
// directly; it migrates onto AgentRuntime in a later phase without touching the
// domain.

// RunSpec is everything an adapter needs to launch one agent execution. It is
// substrate-agnostic: the adapter realises it on Docker, a microVM or a Pod.
type RunSpec struct {
	RuntimeKind  string               // model.RuntimeKind* — selects the harness
	Model        string               // model id passed to the harness
	Provider     string               // model.Provider* — auth/provider selection
	Image        string               // stack image (free-form until the catalogue lands)
	Prompt       string               // rendered prompt
	Env          []string             // KEY=value pairs (no secrets once fetch-at-startup lands)
	Labels       map[string]string    // bookkeeping labels (run_id/step_id/story_key)
	Capabilities model.CapabilitySpec // composed skills/mcp/tool-policy for this run
}

// RunHandle identifies a launched execution (container/microVM/pod id).
type RunHandle struct {
	ID string
}

// RunResult is the terminal outcome of an execution.
type RunResult struct {
	ExitCode int
	Error    string
}

// AgentRuntime is the stable, agent-oriented runtime port. Adapters wrap an
// existing coding harness (claude / opencode CLI, CMA service) — they never
// reimplement the agentic loop as raw API calls.
type AgentRuntime interface {
	// Provision applies the agnostic capability spec to the adapter's native
	// mechanism. An unsupported capability is warn+skip, never a blocking error.
	Provision(ctx context.Context, spec model.CapabilitySpec) (model.ProvisionResult, error)

	// Launch starts an agent execution (clone + harness + capabilities) and
	// returns a handle to it.
	Launch(ctx context.Context, spec RunSpec) (RunHandle, error)

	// Wait blocks until the execution terminates and returns its result.
	Wait(ctx context.Context, h RunHandle) (RunResult, error)

	// Stop terminates a running execution.
	Stop(ctx context.Context, h RunHandle) error

	// SupportedCapabilities declares which capability kinds this adapter can
	// translate (skills / mcp-http / mcp-stdio / tool-policy …).
	SupportedCapabilities() model.CapabilitySet
}
