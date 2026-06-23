package port

import (
	"context"

	"github.com/google/uuid"

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

	// NEW (Stage 1) — resources. The Action copies AgentConfig.DefaultMemory/
	// DefaultCPUs here so the adapter, not the Action, owns ContainerOpts.
	Memory int64
	CPUs   float64

	// NEW (Stage 1) — agnostic per-run connectivity. Replaces the leaking
	// sidecarCtx.NetworkName. The adapter realises it: Docker attaches the
	// execution to Network.Name as an ExtraNetworks entry (+Aliases); a microVM
	// maps it to host routing or degrades to the conn-strings already in Env.
	// Zero value = no extra attachment (no-Environment case → ContainerOpts
	// stays byte-identical).
	Network RunNetwork

	// NEW (Stage 1) — one-shot overrides so build/migrate/seed/test commands run
	// on the SAME port on every substrate later. Empty for an agent launch (the
	// image entrypoint is kept untouched).
	Entrypoint []string
	Cmd        []string
	Workdir    string

	// NEW (Stage 3a) — typed callback contract for callback-mode runs. The Action
	// mints the token and fills this once (substrate-agnostic) so the lifecycle —
	// mint at launch, revoke after the callback resolves — runs identically on
	// every substrate. nil for legacy (non-callback) runs. The auth values ALSO
	// live in Env (AUTH_TOKEN/CALLBACK_URL/RUN_ID/STEP_ID) for the harness; this is
	// the typed mirror the adapter MAY use directly.
	Callback *CallbackSpec
}

// CallbackSpec is the typed callback contract carried on a RunSpec. The Action
// mints the AuthToken (per agent/role) and fills this; the adapter MAY use the
// typed fields, but the actual values also live in RunSpec.Env for the harness.
type CallbackSpec struct {
	URL       string
	AuthToken string
	RunID     uuid.UUID
	StepID    uuid.UUID
}

// RunNetwork is the agnostic description of the per-run isolated network the
// execution joins, plus the service endpoints reachable on it.
type RunNetwork struct {
	// Name is the run-scoped isolated network identity (e.g. hopeitworks-run-<id>).
	// It is NOT the substrate's shared/primary network — that is an adapter-level
	// deployment concern, never carried per-run. Empty = no extra attachment.
	Name string
	// Aliases maps a network name to a DNS alias for this execution on it. Nil for
	// an agent launch (the agent reaches sidecars by THEIR alias, registers none of
	// its own), matching today's ContainerOpts.
	Aliases map[string]string
	// Endpoints are the service endpoints reachable on Name (host+port), derived
	// from the sidecar context. Informational for substrates that route by host;
	// Docker reaches them by DNS on Name. Optional.
	Endpoints []ServiceEndpoint
}

// ServiceEndpoint is one reachable service on the run network.
type ServiceEndpoint struct {
	Name string
	Host string
	Port int
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
