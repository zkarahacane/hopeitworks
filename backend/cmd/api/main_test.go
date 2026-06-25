package main

import (
	"io"
	"log/slog"
	"testing"

	dockeradapter "github.com/zakari/hopeitworks/backend/internal/adapter/docker"
	microsandboxadapter "github.com/zakari/hopeitworks/backend/internal/adapter/microsandbox"
	pkgconfig "github.com/zakari/hopeitworks/backend/pkg/config"
)

// quietLogger discards output so factory tests don't spam the test log.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSelectSubstrate(t *testing.T) {
	// The default substrate now returns the Docker adapter directly (docker.Runtime)
	// instead of nil-then-inject, so the live agent_run flow ALWAYS dispatches
	// through port.AgentRuntime — Docker is an adapter behind the port, not a
	// special path. containerMgr is stored, not dereferenced at construction, so a
	// nil one is safe for this routing test.
	t.Run("docker default returns the Docker adapter (docker.Runtime)", func(t *testing.T) {
		got := selectSubstrate(pkgconfig.SubstrateDocker, nil, nil, "agent-net", quietLogger())
		if got == nil {
			t.Fatal("selectSubstrate(docker) = nil, want *docker.Runtime")
		}
		if _, ok := got.(*dockeradapter.Runtime); !ok {
			t.Fatalf("selectSubstrate(docker) = %T, want *docker.Runtime", got)
		}
	})

	t.Run("unknown kind falls through to the docker default (docker.Runtime)", func(t *testing.T) {
		// validate() rejects unknown kinds at load time; the factory itself
		// defaults defensively so it never panics on an unexpected value.
		got := selectSubstrate("k8s", nil, nil, "agent-net", quietLogger())
		if _, ok := got.(*dockeradapter.Runtime); !ok {
			t.Fatalf("selectSubstrate(unknown) = %T, want *docker.Runtime", got)
		}
	})

	t.Run("microsandbox constructs the scaffold AgentRuntime", func(t *testing.T) {
		got := selectSubstrate(pkgconfig.SubstrateMicrosandbox, nil, nil, "", quietLogger())
		if got == nil {
			t.Fatal("selectSubstrate(microsandbox) = nil, want a scaffold adapter")
		}
		if _, ok := got.(*microsandboxadapter.Runtime); !ok {
			t.Fatalf("selectSubstrate(microsandbox) = %T, want *microsandbox.Runtime", got)
		}
	})
}
