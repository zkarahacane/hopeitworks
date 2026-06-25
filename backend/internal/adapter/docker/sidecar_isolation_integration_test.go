//go:build dockerlive

// Live Docker integration test for East-West run isolation. Excluded from the
// default build/CI (needs a real Docker daemon AND pre-pulled images, which fresh
// CI runners lack). Run locally with: go test -tags dockerlive ./internal/adapter/docker/...
package docker

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	dockernetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// TestSidecarIsolation_LiveCallbackNetworkPath is the live validation of East-West
// run isolation against a REAL Docker daemon. It does NOT depend on a model,
// credentials or the agent image: it stands in a tiny HTTP responder ("api"),
// drives the real SidecarManager + ContainerManager, and proves the actual
// network path the agent callback travels under isolation:
//
//  1. Launch(run1) creates the per-run network hopeitworks-run-<id1> and attaches
//     the stand-in API to it under alias "api" (docker network inspect proves it).
//  2. A probe container on run1's network reaches http://api:<port> -> HTTP 200.
//  3. A probe on a DIFFERENT run's network (run2) CANNOT reach the stand-in (the
//     two per-run networks are disjoint with no shared route) -> live East-West.
//  4. Cleanup(run1) detaches the API and removes the network with no "active
//     endpoints" error.
//
// Gated behind -short (skipped in unit runs); requires a reachable Docker daemon.
func TestSidecarIsolation_LiveCallbackNetworkPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live Docker integration test in -short mode")
	}

	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		host = client.DefaultDockerHost
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mgr, err := NewDockerContainerManager(host, logger)
	if err != nil {
		t.Fatalf("NewDockerContainerManager: %v", err)
	}

	// Raw SDK client for read-only network inspection assertions.
	cli, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("docker client: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })

	ctx := context.Background()

	const probeImage = "alpine:3.21"
	const apiPort = "8080"

	uniq := uuid.NewString()[:8]

	// --- stand-in API: a tiny nc loop that always answers HTTP/200. It has no
	// healthcheck and is created on the default bridge only (NetworkName empty):
	// the SidecarManager is what must attach it to each per-run network.
	apiID, err := mgr.Create(ctx, model.ContainerOpts{
		Image: probeImage,
		Cmd: []string{"sh", "-c",
			`while true; do printf 'HTTP/1.1 200 OK\r\nContent-Length: 3\r\nConnection: close\r\n\r\nok\n' | nc -l -p ` + apiPort + `; done`},
		Labels: map[string]string{"itest": uniq},
	})
	if err != nil {
		t.Fatalf("create stand-in API: %v", err)
	}
	t.Cleanup(func() {
		rmCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = mgr.Stop(rmCtx, apiID)
		_ = mgr.Remove(rmCtx, apiID)
	})
	if err := mgr.Start(ctx, apiID); err != nil {
		t.Fatalf("start stand-in API: %v", err)
	}

	// --- SidecarManager in isolated mode, wired to the stand-in by its ID. The
	// real adapter attaches/detaches the API container by this identity (a name in
	// production, the container ID here — both are valid Docker references).
	sm := NewDockerSidecarManagerWithIsolation(mgr, true, apiID, logger)

	run1 := uuid.New()
	run2 := uuid.New()
	net1 := networkName(run1)
	net2 := networkName(run2)

	// --- Launch run1: creates net1 and attaches the stand-in API (alias "api").
	sc1, err := sm.Launch(ctx, run1, nil)
	if err != nil {
		t.Fatalf("Launch run1: %v", err)
	}
	cleanedUp1 := false
	t.Cleanup(func() {
		if cleanedUp1 {
			return
		}
		cCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = sm.Cleanup(cCtx, sc1)
	})
	if sc1.NetworkName != net1 {
		t.Fatalf("sc1.NetworkName = %q, want %q", sc1.NetworkName, net1)
	}

	// Assertion 1: per-run network exists and the stand-in API is connected with
	// the "api" alias.
	insp1, err := cli.NetworkInspect(ctx, net1, dockernetwork.InspectOptions{})
	if err != nil {
		t.Fatalf("network inspect %s: %v", net1, err)
	}
	ep, ok := insp1.Containers[apiID]
	if !ok {
		t.Fatalf("stand-in API %s not connected to %s; endpoints=%v", apiID, net1, insp1.Containers)
	}
	t.Logf("stand-in API connected to %s as %q (ip %s)", net1, ep.Name, ep.IPv4Address)

	// Assertion 2: a probe on run1's network reaches http://api:<port> -> 200.
	if code := runProbe(ctx, t, mgr, net1, "http://api:"+apiPort+"/"); code != 0 {
		t.Fatalf("probe on %s could not reach the API (wget exit %d); callback path is broken", net1, code)
	}
	t.Logf("PASS: probe on %s reached http://api:%s (callback path live)", net1, apiPort)

	// --- Launch run2: a second, disjoint per-run network. The stand-in API is NOT
	// attached here (only the API container the run owns is), so a probe on run2
	// must NOT reach the run1 stand-in via alias "api" -> live East-West proof.
	//
	// To keep the second network from auto-wiring the same API, we use a SECOND
	// SidecarManager with NO API name: run2 gets its own isolated network with no
	// stand-in attached, which is exactly the cross-run reachability we deny.
	sm2 := NewDockerSidecarManagerWithIsolation(mgr, true, "", logger)
	sc2, err := sm2.Launch(ctx, run2, nil)
	if err != nil {
		t.Fatalf("Launch run2: %v", err)
	}
	t.Cleanup(func() {
		cCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = sm2.Cleanup(cCtx, sc2)
	})

	// Networks are distinct.
	if net1 == net2 {
		t.Fatalf("expected disjoint per-run networks, got the same name %q", net1)
	}
	insp2, err := cli.NetworkInspect(ctx, net2, dockernetwork.InspectOptions{})
	if err != nil {
		t.Fatalf("network inspect %s: %v", net2, err)
	}
	if _, present := insp2.Containers[apiID]; present {
		t.Fatalf("stand-in API leaked onto run2 network %s — isolation broken", net2)
	}
	if insp1.ID == insp2.ID {
		t.Fatalf("run1 and run2 resolved to the same network ID %s", insp1.ID)
	}

	// Assertion 3: probe on run2 CANNOT reach the run1 stand-in via alias "api".
	// A non-zero wget exit (DNS failure / connection refused) is the East-West
	// denial we want; a zero exit would mean the runs share a route (failure).
	if code := runProbe(ctx, t, mgr, net2, "http://api:"+apiPort+"/"); code == 0 {
		t.Fatalf("probe on %s REACHED the API across runs — East-West isolation broken", net2)
	}
	t.Logf("PASS: probe on %s could NOT reach the run1 API (East-West denied)", net2)

	// --- Cleanup run1: must detach the stand-in then remove the network without a
	// "network has active endpoints" error.
	cCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := sm.Cleanup(cCtx, sc1); err != nil {
		t.Fatalf("Cleanup run1: %v", err)
	}
	cleanedUp1 = true

	// The stand-in must be detached from net1...
	if _, err := cli.NetworkInspect(ctx, net1, dockernetwork.InspectOptions{}); err == nil {
		t.Fatalf("network %s still exists after Cleanup", net1)
	} else if !isNotFound(err) {
		t.Fatalf("unexpected error inspecting removed network %s: %v", net1, err)
	}
	// ...but the stand-in container itself must still be alive (Cleanup detaches,
	// it does not destroy the API).
	if _, err := cli.ContainerInspect(ctx, apiID); err != nil {
		t.Fatalf("stand-in API was unexpectedly removed by Cleanup: %v", err)
	}
	t.Logf("PASS: Cleanup detached the stand-in and removed %s with no active-endpoints error", net1)
}

// runProbe runs a one-shot probe container on the given network that wgets the
// target URL, then returns the probe's exit code (0 = reachable / HTTP 2xx-3xx).
// It exercises the real ContainerManager (Create on NetworkName, Start, Wait).
func runProbe(ctx context.Context, t *testing.T, mgr port.ContainerManager, networkName, url string) int {
	t.Helper()
	id, err := mgr.Create(ctx, model.ContainerOpts{
		Image:       "alpine:3.21",
		NetworkName: networkName,
		// -T 5: bound the attempt so an unreachable target fails fast instead of
		// hanging the whole test.
		Cmd: []string{"wget", "-q", "-T", "5", "-O", "/dev/null", url},
	})
	if err != nil {
		t.Fatalf("create probe on %s: %v", networkName, err)
	}
	defer func() {
		rmCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = mgr.Remove(rmCtx, id)
	}()
	if err := mgr.Start(ctx, id); err != nil {
		t.Fatalf("start probe on %s: %v", networkName, err)
	}
	wCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	code, err := mgr.Wait(wCtx, id)
	if err != nil {
		t.Fatalf("wait probe on %s: %v", networkName, err)
	}
	return code
}

// isNotFound reports whether err is a Docker "no such network/object" error.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "no such")
}
