// Package microsandbox is the microVM substrate adapter. It implements the
// target domain port port.AgentRuntime.
//
// Build tags split this package into a default (pure) part and a
// libkrun-dependent part:
//
//   - This file (runtime.go, untagged) holds the pure, always-compiled logic:
//     SupportedCapabilities, Provision (agnostic→native triage, warn+skip) and
//     the ResolveImage helper. It carries NO microsandbox-SDK import, so the
//     default build (and CI) compiles it on any host without libkrun.
//   - runtime_microsandbox.go (//go:build microsandbox) holds the REAL live
//     execution (Launch/Wait/Stop) via the microsandbox Go SDK. Building it
//     requires the `microsandbox` tag AND a KVM/HVF host with libkrun present
//     (see deploy/lima/microsandbox-vm.yaml). This is P3b.
//   - runtime_stub.go (//go:build !microsandbox) provides the default fallback:
//     Launch/Wait/Stop return ErrNotBuilt ("not built with microsandbox tag"),
//     so a binary compiled without the tag selecting SUBSTRATE=microsandbox
//     fails clearly instead of silently doing nothing.
//
// The adapter is NOT wired into the live agent_run flow (which still drives the
// Docker ContainerManager directly). The factory in main.go only logs the
// selected substrate and constructs this adapter; selecting "microsandbox" does
// not intercept the live Docker execution path.
package microsandbox

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Compile-time guarantee that Runtime satisfies the target runtime port. The
// Launch/Wait/Stop half lives in the build-tagged files; this assertion holds
// for both the tagged and the fallback build.
var _ port.AgentRuntime = (*Runtime)(nil)

// Reasons recorded in ProvisionWarning for capabilities a microVM substrate
// cannot (yet) translate. Kept as constants for goconst and reuse in tests.
const (
	reasonMCPStdioUnsupported = "stdio MCP servers require an in-image binary; not yet translated by the microsandbox substrate"
	reasonUnknownTransport    = "unknown MCP transport"

	transportHTTP  = "http"
	transportStdio = "stdio"
)

// Runtime is the microsandbox (microVM) substrate adapter. It satisfies
// port.AgentRuntime so the rest of the system can target the stable port. The
// pure half (Provision/SupportedCapabilities/ResolveImage) lives here; the live
// half (Launch/Wait/Stop) lives in the build-tagged files.
//
// The name is intentionally Runtime (not MicrosandboxRuntime) — the package
// qualifier already conveys the substrate, so callers write microsandbox.Runtime
// without stutter.
type Runtime struct {
	// enabled gates the live-execution path in the tagged build. When false the
	// tagged Launch refuses to start a microVM (e.g. KVM not provisioned); the
	// fallback build ignores it and always returns ErrNotBuilt.
	enabled bool
	logger  *slog.Logger
	// stacks resolves a stack key to its catalogued image. Optional: when nil,
	// ResolveImage falls back to the free-form RunSpec.Image, preserving the
	// image-only invariant.
	stacks port.StackRepository
}

// NewRuntime constructs the scaffold microVM adapter. stacks may be nil (image
// resolution then falls back to the free-form RunSpec.Image).
func NewRuntime(enabled bool, stacks port.StackRepository, logger *slog.Logger) *Runtime {
	if logger == nil {
		logger = slog.Default()
	}
	return &Runtime{enabled: enabled, logger: logger, stacks: stacks}
}

// SupportedCapabilities declares, as pure data, which agnostic capability kinds a
// microVM substrate can translate to its native mechanism.
//
// A microVM runs a normal Pod-expressible workload (the harness + materialised
// capabilities on disk), so it supports the same on-disk capabilities as the
// container substrate: skills (files), HTTP MCP servers (network) and tool
// policy (CLI flags). Stdio MCP servers need an in-image binary and stronger
// in-VM process plumbing, which the scaffold does not translate yet — hence
// warn+skip in Provision.
func (r *Runtime) SupportedCapabilities() model.CapabilitySet {
	return model.CapabilitySet{
		Skills:          true,
		MCPServersHTTP:  true,
		MCPServersStdio: false,
		ToolPolicy:      true,
	}
}

// Provision applies the agnostic capability spec to the (eventual) microVM-native
// mechanism. P3a performs only the triage: every supported capability is recorded
// as Applied; every unsupported one is recorded as a ProvisionWarning. It NEVER
// returns a blocking error for an unsupported capability (warn+skip invariant).
//
// The actual native materialisation (skills onto the VM disk, .mcp.json, harness
// tool flags) is P3b — marked TODO below. The triage itself is fully testable.
func (r *Runtime) Provision(_ context.Context, spec model.CapabilitySpec) (model.ProvisionResult, error) {
	supported := r.SupportedCapabilities()
	var res model.ProvisionResult

	// Skills: on-disk files, supported by any Pod-expressible substrate.
	for _, skill := range spec.Skills {
		ref := capabilityRef(model.CapabilityKindSkill, skill.Name)
		if supported.Skills {
			// TODO(P3b): materialise skill files onto the microVM workdir.
			res.Applied = append(res.Applied, ref)
		} else {
			res.Warnings = append(res.Warnings, model.ProvisionWarning{
				Capability: ref,
				Reason:     "skills not supported by this substrate",
			})
		}
	}

	// MCP servers: HTTP = network service (supported); stdio = in-image binary
	// (not yet translated by the microVM scaffold).
	for _, srv := range spec.MCPServers {
		ref := capabilityRef(model.CapabilityKindMCPServer, srv.Name)
		switch srv.Transport {
		case transportHTTP:
			if supported.MCPServersHTTP {
				// TODO(P3b): write .mcp.json into the microVM workdir.
				res.Applied = append(res.Applied, ref)
			} else {
				res.Warnings = append(res.Warnings, model.ProvisionWarning{
					Capability: ref,
					Reason:     "http MCP servers not supported by this substrate",
				})
			}
		case transportStdio:
			// Always warn+skip in P3a (MCPServersStdio is false).
			res.Warnings = append(res.Warnings, model.ProvisionWarning{
				Capability: ref,
				Reason:     reasonMCPStdioUnsupported,
			})
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
			// TODO(P3b): pass allow/deny lists to the in-VM harness.
			res.Applied = append(res.Applied, ref)
		} else {
			res.Warnings = append(res.Warnings, model.ProvisionWarning{
				Capability: ref,
				Reason:     "tool policy not supported by this substrate",
			})
		}
	}

	r.logger.Debug("microsandbox: provision triage (scaffold)",
		"applied", len(res.Applied), "warnings", len(res.Warnings))
	return res, nil
}

// ResolveImage composes the effective launch image for a run, mirroring the
// RunService invariant: a catalogued stack (resolved by key via the stack
// catalogue) wins; otherwise the free-form spec.Image is used as-is. This is a
// pure helper (no microVM I/O) so it is unit-testable in isolation.
//
// spec.Image may carry either a stack key (e.g. "go") or a free-form image
// reference. When the stack catalogue is configured and the value matches a
// catalogued key, the catalogued (digest-pinned) ImageRef is returned.
func (r *Runtime) ResolveImage(ctx context.Context, spec port.RunSpec) (string, error) {
	if r.stacks != nil && spec.Image != "" {
		if stack, err := r.stacks.GetByKey(ctx, spec.Image); err == nil && stack != nil {
			return stack.ImageRef, nil
		}
	}
	if spec.Image == "" {
		return "", fmt.Errorf("microsandbox: no image resolvable from run spec (empty image, no matching stack)")
	}
	return spec.Image, nil
}

// capabilityRef formats a stable "kind/name" reference for Applied/Warnings
// entries. An empty name (e.g. tool policy is singular per run) yields just the
// kind.
func capabilityRef(kind, name string) string {
	if name == "" {
		return kind
	}
	return kind + "/" + name
}
