//go:build microsandbox

// Command microsandbox-smoke is a standalone validation harness for the microVM
// substrate adapter. It is compiled ONLY with the `microsandbox` build tag (so
// it never pulls the libkrun SDK into the default build / CI) and is meant to be
// run by hand inside the KVM/HVF VM provisioned by deploy/lima/microsandbox-vm.yaml:
//
//	go run -tags microsandbox ./cmd/microsandbox-smoke -image docker.io/library/alpine:3.20
//
// It drives the public port.AgentRuntime surface end to end — Launch → Wait →
// Stop — and prints the result, exiting non-zero on any failure. This is the
// "runnable in the VM" check the P3b design calls for; CI cannot run it (no KVM).
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/zakari/hopeitworks/backend/internal/adapter/microsandbox"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

func main() {
	image := flag.String("image", "docker.io/library/alpine:3.20", "OCI image to boot in the microVM")
	timeout := flag.Duration("timeout", 2*time.Minute, "overall timeout for launch+wait")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	if err := run(*image, *timeout, logger); err != nil {
		logger.Error("microsandbox smoke test failed", "err", err)
		os.Exit(1)
	}
	logger.Info("microsandbox smoke test passed")
}

func run(image string, timeout time.Duration, logger *slog.Logger) error {
	rt := microsandbox.NewRuntime(true, nil, logger)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	spec := port.RunSpec{
		Image:  image,
		Labels: map[string]string{"run_id": "smoke-" + time.Now().Format("150405")},
		Env:    []string{"HOPEITWORKS_SMOKE=1"},
	}

	h, err := rt.Launch(ctx, spec)
	if err != nil {
		return fmt.Errorf("launch: %w", err)
	}
	defer func() {
		if serr := rt.Stop(context.Background(), h); serr != nil {
			logger.Warn("stop failed", "err", serr)
		}
	}()

	res, err := rt.Wait(ctx, h)
	if err != nil {
		return fmt.Errorf("wait: %w", err)
	}
	logger.Info("run finished", "handle", h.ID, "exit_code", res.ExitCode, "error", res.Error)
	return rt.Stop(ctx, h)
}
