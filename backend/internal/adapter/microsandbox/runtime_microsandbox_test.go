//go:build microsandbox

// Validation harness for the REAL microVM path. Compiled and run ONLY with the
// `microsandbox` build tag on a KVM/HVF host with libkrun present:
//
//	go test -tags microsandbox ./internal/adapter/microsandbox/ -run Integration -v
//
// It is NEVER compiled in the default build (no SDK import leaks into CI). On a
// host without /dev/kvm the SDK calls fail and the test reports a clear skip-ish
// failure; provision the VM via deploy/lima/microsandbox-vm.yaml first.

package microsandbox

import (
	"context"
	"testing"
	"time"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// integrationImage is a small public OCI image known to boot under microsandbox.
// Overridable by editing here; kept as a constant for goconst.
const integrationImage = "docker.io/library/alpine:3.20"

// TestRuntime_Disabled_RefusesLaunch is a tagged unit check that does NOT need a
// VM: a disabled runtime must refuse Launch before touching libkrun.
func TestRuntime_Disabled_RefusesLaunch(t *testing.T) {
	rt := NewRuntime(false, nil, nil)
	if _, err := rt.Launch(context.Background(), port.RunSpec{Image: integrationImage}); err == nil {
		t.Fatal("disabled runtime Launch() = nil error, want refusal")
	}
}

// TestIntegration_LaunchWaitStop exercises the full lifecycle against a real
// microVM. Requires -tags microsandbox AND a KVM/HVF host; skipped in -short.
func TestIntegration_LaunchWaitStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping microVM integration test in -short mode")
	}

	rt := NewRuntime(true, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Override the harness command for the test: run a trivial command that
	// exits 0, so we validate launch/wait/stop without needing agent-runtime in
	// the image. We reuse Shell via a spec that resolves to a stock image.
	spec := port.RunSpec{
		Image:  integrationImage,
		Labels: map[string]string{"run_id": "itest-" + time.Now().Format("150405")},
		Env:    []string{"HOPEITWORKS_TEST=1"},
		Capabilities: model.CapabilitySpec{
			ToolPolicy: &model.ToolPolicySpec{Allow: []string{"Bash"}},
		},
	}

	h, err := rt.Launch(ctx, spec)
	if err != nil {
		t.Fatalf("Launch: %v (is /dev/kvm present and libkrun installed?)", err)
	}
	t.Cleanup(func() {
		if err := rt.Stop(context.Background(), h); err != nil {
			t.Logf("cleanup Stop: %v", err)
		}
	})
	if h.ID == "" {
		t.Fatal("Launch returned empty handle id")
	}

	res, err := rt.Wait(ctx, h)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	t.Logf("microVM run finished: exit=%d err=%q", res.ExitCode, res.Error)

	if err := rt.Stop(ctx, h); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	// Stop is idempotent: a second Stop on a gone handle is a no-op.
	if err := rt.Stop(ctx, h); err != nil {
		t.Fatalf("second Stop should be a no-op, got %v", err)
	}
}

// TestIntegration_Wait_UnknownHandle confirms Wait on an unknown handle errors
// (registry miss), independent of an actual VM.
func TestIntegration_Wait_UnknownHandle(t *testing.T) {
	rt := NewRuntime(true, nil, nil)
	if _, err := rt.Wait(context.Background(), port.RunHandle{ID: "does-not-exist"}); err == nil {
		t.Fatal("Wait on unknown handle = nil error, want failure")
	}
	// Stop on unknown handle is a no-op, not an error.
	if err := rt.Stop(context.Background(), port.RunHandle{ID: "does-not-exist"}); err != nil {
		t.Fatalf("Stop on unknown handle = %v, want nil (idempotent)", err)
	}
}
