//go:build microsandbox

// This file is the REAL microVM live-execution path. It imports the microsandbox
// Go SDK (libkrun under the hood) and is therefore compiled ONLY with
// `-tags microsandbox` on a KVM/HVF host (see deploy/lima/microsandbox-vm.yaml).
// The default build uses runtime_stub.go instead, which has no SDK import.
//
// Design: one agent run == one ephemeral microsandbox sandbox.
//
//	Launch  → CreateSandbox(image, cpus, mem, env) boots the VM → ShellStream(harness cmd)
//	Wait    → ExecHandle.Wait + Collect → RunResult{ExitCode, stderr-on-failure}
//	Stop    → Kill the exec + Stop + RemoveSandbox (idempotent teardown)
//
// P3c follow-ups (tracked): port.RunSpec is too thin to carry callback-mode env
// (CALLBACK_URL / AUTH_TOKEN / API_KEY / prompt are assembled in agent_run.go above
// the port), so an end-to-end app run on this substrate needs that env-assembly fed
// into RunSpec.Env first. Also pending: sb.Close() on teardown + an IdleTimeout/
// MaxDuration safety net, and re-attach-by-name so Wait/Stop survive an API restart.
//
// A process-local registry maps the opaque port.RunHandle.ID (the sandbox name)
// to the live SDK objects, because port.RunHandle only carries a string id.
//
// CAVEATS (tracked in docs/agent-runtime-p3-microsandbox-plan.md):
//   - port.RunSpec carries no CPU/mem/network fields yet; sensible defaults are
//     used here and the isolated-network + sidecar topology is deferred (P3c).
//   - Capabilities are provisioned via the pure Provision triage (warn+skip);
//     native on-disk materialisation inside the VM is a follow-up.
//   - The microsandbox SDK is BETA (pinned in go.mod). Treat signatures here as
//     the integration contract; re-verify against the pinned version on upgrade.

package microsandbox

import (
	"context"
	"fmt"
	"strings"
	"sync"

	msb "github.com/superradcompany/microsandbox/sdk/go"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Default microVM sizing, used until port.RunSpec grows CPU/memory fields (P3c).
const (
	defaultCPUs      = uint8(2)
	defaultMemoryMiB = uint32(2048)

	// harnessShellCmd is the shell entrypoint launched inside the microVM. The
	// agent-runtime binary is baked into the stack image (substrate-agnostic);
	// the microVM just runs it like any other Pod-expressible workload.
	harnessShellCmd = "agent-runtime"
)

// liveSandbox couples an SDK sandbox to the in-flight exec handle of its harness
// process, so Wait/Stop can act on the same run by opaque handle id.
type liveSandbox struct {
	sb   *msb.Sandbox
	exec *msb.ExecHandle
}

// registry is the process-local handle→live-sandbox map. The microsandbox SDK
// objects are not serialisable into a port.RunHandle, so we keep them here for
// the lifetime of the process that launched them.
var registry = struct {
	sync.Mutex
	m map[string]*liveSandbox
}{m: make(map[string]*liveSandbox)}

func registryPut(id string, ls *liveSandbox) {
	registry.Lock()
	defer registry.Unlock()
	registry.m[id] = ls
}

func registryGet(id string) (*liveSandbox, bool) {
	registry.Lock()
	defer registry.Unlock()
	ls, ok := registry.m[id]
	return ls, ok
}

func registryDelete(id string) {
	registry.Lock()
	defer registry.Unlock()
	delete(registry.m, id)
}

// sandboxName derives a stable, microsandbox-legal sandbox name for a run from
// its labels (run_id is the natural key); it falls back to a generic prefix.
func sandboxName(spec port.RunSpec) string {
	if id := spec.Labels["run_id"]; id != "" {
		return "hopeitworks-" + id
	}
	if id := spec.Labels["step_id"]; id != "" {
		return "hopeitworks-step-" + id
	}
	return "hopeitworks-run"
}

// envMap converts the KEY=value slice of port.RunSpec into the map the SDK's
// WithEnv option expects. Entries without '=' are skipped.
func envMap(env []string) map[string]string {
	out := make(map[string]string, len(env))
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok && k != "" {
			out[k] = v
		}
	}
	return out
}

// Launch creates and starts an ephemeral microVM for one agent run, then starts
// the harness process inside it via a streaming shell exec. The returned handle
// id is the sandbox name; Wait/Stop look the live objects up by it.
//
// enabled gates the path: when false (e.g. the operator selected microsandbox
// but the host has no KVM) Launch refuses rather than crashing in libkrun.
func (r *Runtime) Launch(ctx context.Context, spec port.RunSpec) (port.RunHandle, error) {
	if !r.enabled {
		return port.RunHandle{}, fmt.Errorf("microsandbox: runtime disabled (no KVM host or substrate not enabled)")
	}

	image, err := r.ResolveImage(ctx, spec)
	if err != nil {
		return port.RunHandle{}, fmt.Errorf("microsandbox: resolve image: %w", err)
	}

	name := sandboxName(spec)
	opts := []msb.SandboxOption{
		msb.WithImage(image),
		msb.WithCPUs(defaultCPUs),
		msb.WithMemory(defaultMemoryMiB),
	}
	if env := envMap(spec.Env); len(env) > 0 {
		opts = append(opts, msb.WithEnv(env))
	}

	sb, err := msb.CreateSandbox(ctx, name, opts...)
	if err != nil {
		return port.RunHandle{}, fmt.Errorf("microsandbox: create sandbox %q: %w", name, err)
	}

	// Best-effort capability triage (warn+skip). Native in-VM materialisation is
	// a follow-up; failing it must never block the launch.
	if _, perr := r.Provision(ctx, spec.Capabilities); perr != nil {
		r.logger.Warn("microsandbox: provision triage returned error (ignored, warn+skip)", "err", perr)
	}

	exec, err := sb.ShellStream(ctx, harnessShellCmd)
	if err != nil {
		// Tear the just-created sandbox down so a failed launch leaks nothing.
		_ = sb.Kill(ctx)
		_ = msb.RemoveSandbox(ctx, name)
		return port.RunHandle{}, fmt.Errorf("microsandbox: start harness in %q: %w", name, err)
	}

	registryPut(name, &liveSandbox{sb: sb, exec: exec})
	r.logger.Info("microsandbox: launched microVM run", "sandbox", name, "image", image)
	return port.RunHandle{ID: name}, nil
}

// Wait blocks until the harness process in the microVM exits and returns its
// terminal result. A non-zero exit code is NOT a Go error (mirrors the SDK):
// it is reported in RunResult.ExitCode, with collected stderr in Error.
func (r *Runtime) Wait(ctx context.Context, h port.RunHandle) (port.RunResult, error) {
	ls, ok := registryGet(h.ID)
	if !ok {
		return port.RunResult{}, fmt.Errorf("microsandbox: unknown run handle %q", h.ID)
	}

	exitCode, err := ls.exec.Wait(ctx)
	if err != nil {
		return port.RunResult{}, fmt.Errorf("microsandbox: wait on %q: %w", h.ID, err)
	}

	res := port.RunResult{ExitCode: exitCode}
	if exitCode != 0 {
		if out, cerr := ls.exec.Collect(ctx); cerr == nil && out != nil {
			res.Error = strings.TrimSpace(out.Stderr())
		}
	}
	r.logger.Info("microsandbox: microVM run finished", "sandbox", h.ID, "exit_code", exitCode)
	return res, nil
}

// Stop terminates the run's microVM and removes it. It is idempotent and
// best-effort: each teardown step is attempted even if an earlier one fails, and
// an unknown handle is treated as already stopped.
func (r *Runtime) Stop(ctx context.Context, h port.RunHandle) error {
	ls, ok := registryGet(h.ID)
	if !ok {
		return nil // already gone
	}

	var errs []string
	if ls.exec != nil {
		if err := ls.exec.Kill(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("kill exec: %v", err))
		}
		_ = ls.exec.Close()
	}
	if ls.sb != nil {
		if err := ls.sb.Stop(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("stop sandbox: %v", err))
		}
	}
	if err := msb.RemoveSandbox(ctx, h.ID); err != nil {
		errs = append(errs, fmt.Sprintf("remove sandbox: %v", err))
	}
	registryDelete(h.ID)

	if len(errs) > 0 {
		return fmt.Errorf("microsandbox: stop %q: %s", h.ID, strings.Join(errs, "; "))
	}
	r.logger.Info("microsandbox: microVM run stopped", "sandbox", h.ID)
	return nil
}
