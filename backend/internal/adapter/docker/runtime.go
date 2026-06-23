package docker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Compile-time guarantee that Runtime satisfies the target runtime port. Docker
// is the back-compat substrate adapter (ADR Stage 1): it routes an agnostic
// port.RunSpec through the low-level ContainerManager, so the rest of the system
// can target port.AgentRuntime while keeping ContainerManager an internal detail.
var _ port.AgentRuntime = (*Runtime)(nil)

// stopTimeout bounds Stop's Stop+Remove cycle on a derived context so cleanup
// never hangs, mirroring the action-level cleanupContainer timeout.
const stopTimeout = 30 * time.Second

// Runtime is the Docker substrate adapter. It implements port.AgentRuntime by
// building model.ContainerOpts from an agnostic RunSpec and driving the
// low-level ContainerManager (Create/Start/Wait/Stop/Remove).
//
// The name is intentionally Runtime (not DockerRuntime) — the package qualifier
// already conveys the substrate, so callers write docker.Runtime without
// stutter.
type Runtime struct {
	// containerMgr is the internal low-level container CRUD dependency. Per the
	// ADR, ContainerManager is demoted to an adapter detail, not a domain port.
	containerMgr port.ContainerManager
	// networkName is the shared/primary agent network (a deployment config),
	// attached to every execution. It is NEVER a per-run identity — the per-run
	// isolated network arrives via RunSpec.Network and maps to ExtraNetworks.
	networkName string
	logger      *slog.Logger
}

// NewRuntime constructs the Docker substrate adapter. containerMgr is the
// low-level container CRUD dependency; networkName is the shared agent network
// every execution attaches to (deployment config, never per-run).
func NewRuntime(containerMgr port.ContainerManager, networkName string, logger *slog.Logger) *Runtime {
	if logger == nil {
		logger = slog.Default()
	}
	return &Runtime{containerMgr: containerMgr, networkName: networkName, logger: logger}
}

// Launch builds ContainerOpts from the agnostic spec, creates and starts the
// container, and returns its id as the run handle. For a no-Environment agent
// launch (zero RunSpec.Network, nil Entrypoint/Cmd) it emits ContainerOpts
// byte-identical to the legacy createContainer path — the regression oracle.
//
// On a Start failure AFTER a successful Create, the created-but-unstarted
// container is torn down inline (best-effort Stop+Remove, errors logged, never
// fatal) before the error is returned. This mirrors the legacy live path, whose
// deferred cleanupContainer reaps it immediately — without it the container
// would leak until the next API reboot (OrphanCleaner runs only at boot;
// TimeoutEnforcer reaps only running containers).
func (r *Runtime) Launch(ctx context.Context, spec port.RunSpec) (port.RunHandle, error) {
	opts := model.ContainerOpts{
		Image:       spec.Image,
		NetworkName: r.networkName, // shared network = adapter config, NOT spec
		Memory:      spec.Memory,
		CPUs:        spec.CPUs,
		Env:         spec.Env,
		Labels:      spec.Labels,
		Entrypoint:  spec.Entrypoint, // nil for an agent launch
		Cmd:         spec.Cmd,        // nil for an agent launch
	}
	// Per-run isolated network: dual-home the execution. Empty Name leaves the
	// container single-homed on the shared network, keeping ContainerOpts
	// byte-identical to the no-Environment case.
	if spec.Network.Name != "" {
		opts.ExtraNetworks = []string{spec.Network.Name}
		opts.Aliases = spec.Network.Aliases
	}
	// Note: spec.Workdir is not consumed by Docker in Stage 1 — model.ContainerOpts
	// has no workdir field and we must not change it. It is carried on RunSpec for
	// the one-shot env-command path (Stage 5) and other substrates.

	id, err := r.containerMgr.Create(ctx, opts)
	if err != nil {
		return port.RunHandle{}, fmt.Errorf("docker: create container (image %s): %w", spec.Image, err)
	}

	if err := r.containerMgr.Start(ctx, id); err != nil {
		// Start failed after Create succeeded: tear the created container down
		// inline (best-effort) so it does not leak, mirroring the legacy deferred
		// cleanupContainer, then surface the original Start error.
		r.logger.Warn("docker: start container failed; cleaning up created container",
			slog.String(containerIDKey, id),
			slog.String("error", err.Error()),
		)
		r.teardown(id)
		return port.RunHandle{}, fmt.Errorf("docker: start container %s: %w", id, err)
	}

	r.logger.Debug("docker: launched execution",
		slog.String(containerIDKey, id),
		slog.String("image", spec.Image),
	)
	return port.RunHandle{ID: id}, nil
}

// Wait blocks until the execution terminates and maps the container exit code
// into the agnostic RunResult.
func (r *Runtime) Wait(ctx context.Context, h port.RunHandle) (port.RunResult, error) {
	code, err := r.containerMgr.Wait(ctx, h.ID)
	if err != nil {
		return port.RunResult{}, fmt.Errorf("docker: wait container %s: %w", h.ID, err)
	}
	return port.RunResult{ExitCode: code}, nil
}

// Stop terminates and removes a running execution, mirroring the legacy
// cleanupContainer (Stop then Remove, best-effort).
//
// Stop deliberately IGNORES the caller's context deadline and runs on its own
// derived context bounded by stopTimeout, so teardown is never interrupted or
// cancelled mid-flight (e.g. when the parent context was cancelled to trigger
// the stop in the first place) — exactly like cleanupContainer. The unused
// context parameter is kept to satisfy the port.AgentRuntime signature.
//
// Stop errors are logged (best-effort); the Remove error is returned so a future
// caller can defer-and-log it.
func (r *Runtime) Stop(_ context.Context, h port.RunHandle) error {
	ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()

	if err := r.containerMgr.Stop(ctx, h.ID); err != nil {
		r.logger.Warn("docker: stop container failed during teardown",
			slog.String(containerIDKey, h.ID),
			slog.String("error", err.Error()),
		)
	}

	if err := r.containerMgr.Remove(ctx, h.ID); err != nil {
		return fmt.Errorf("docker: remove container %s: %w", h.ID, err)
	}

	r.logger.Debug("docker: execution stopped and removed", slog.String(containerIDKey, h.ID))
	return nil
}

// teardown stops then removes a container best-effort (errors logged, never
// returned). Used to clean up a created-but-unstarted container after a Start
// failure in Launch. It runs on its own context bounded by stopTimeout,
// independent of any caller deadline, so it never hangs.
func (r *Runtime) teardown(containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()

	if err := r.containerMgr.Stop(ctx, containerID); err != nil {
		r.logger.Warn("docker: stop container failed during launch cleanup",
			slog.String(containerIDKey, containerID),
			slog.String("error", err.Error()),
		)
	}
	if err := r.containerMgr.Remove(ctx, containerID); err != nil {
		r.logger.Warn("docker: remove container failed during launch cleanup",
			slog.String(containerIDKey, containerID),
			slog.String("error", err.Error()),
		)
	}
}

// SupportedCapabilities declares which agnostic capability kinds the Docker
// substrate can translate. Docker runs a full Pod-expressible workload, so every
// capability kind is supported (skills on disk, HTTP and stdio MCP servers, tool
// policy via harness flags).
func (r *Runtime) SupportedCapabilities() model.CapabilitySet {
	return model.CapabilitySet{
		Skills:          true,
		MCPServersHTTP:  true,
		MCPServersStdio: true,
		ToolPolicy:      true,
	}
}

// Provision triages the agnostic capability spec into Applied/Warnings. Docker
// supports every capability kind, so each capability is recorded as Applied with
// zero warnings. The actual materialisation happens in the harness at startup
// (fetch-at-startup); Provision is the testable triage. It NEVER returns a
// blocking error (warn+skip invariant).
func (r *Runtime) Provision(_ context.Context, spec model.CapabilitySpec) (model.ProvisionResult, error) {
	supported := r.SupportedCapabilities()
	var res model.ProvisionResult

	// Skills: on-disk files, supported by any Pod-expressible substrate.
	for _, skill := range spec.Skills {
		ref := capabilityRef(model.CapabilityKindSkill, skill.Name)
		if supported.Skills {
			res.Applied = append(res.Applied, ref)
		} else {
			res.Warnings = append(res.Warnings, model.ProvisionWarning{
				Capability: ref,
				Reason:     "skills not supported by this substrate",
			})
		}
	}

	// MCP servers: HTTP = network service, stdio = in-image binary. Docker
	// supports both.
	for _, srv := range spec.MCPServers {
		ref := capabilityRef(model.CapabilityKindMCPServer, srv.Name)
		switch srv.Transport {
		case transportHTTP:
			if supported.MCPServersHTTP {
				res.Applied = append(res.Applied, ref)
			} else {
				res.Warnings = append(res.Warnings, model.ProvisionWarning{
					Capability: ref,
					Reason:     "http MCP servers not supported by this substrate",
				})
			}
		case transportStdio:
			if supported.MCPServersStdio {
				res.Applied = append(res.Applied, ref)
			} else {
				res.Warnings = append(res.Warnings, model.ProvisionWarning{
					Capability: ref,
					Reason:     "stdio MCP servers not supported by this substrate",
				})
			}
		default:
			res.Warnings = append(res.Warnings, model.ProvisionWarning{
				Capability: ref,
				Reason:     fmt.Sprintf("%s: %q", reasonUnknownTransport, srv.Transport),
			})
		}
	}

	// Tool policy: harness allow/deny flags, supported.
	if spec.ToolPolicy != nil {
		ref := capabilityRef(model.CapabilityKindToolPolicy, "")
		if supported.ToolPolicy {
			res.Applied = append(res.Applied, ref)
		} else {
			res.Warnings = append(res.Warnings, model.ProvisionWarning{
				Capability: ref,
				Reason:     "tool policy not supported by this substrate",
			})
		}
	}

	r.logger.Debug("docker: provision triage",
		slog.Int("applied", len(res.Applied)),
		slog.Int("warnings", len(res.Warnings)),
	)
	return res, nil
}

// transportHTTP / transportStdio are the recognised MCP transports.
const (
	transportHTTP  = "http"
	transportStdio = "stdio"
)

// reasonUnknownTransport prefixes the warning for an unrecognised MCP transport.
const reasonUnknownTransport = "unknown MCP transport"

// containerIDKey is the structured-log key for a container id. Extracted as a
// constant so the literal is not repeated (goconst) across slog calls.
const containerIDKey = "container_id"

// capabilityRef formats a stable "kind/name" reference for Applied/Warnings
// entries. An empty name (e.g. tool policy is singular per run) yields just the
// kind.
func capabilityRef(kind, name string) string {
	if name == "" {
		return kind
	}
	return kind + "/" + name
}
