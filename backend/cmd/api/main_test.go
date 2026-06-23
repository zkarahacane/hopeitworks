package main

import (
	"io"
	"log/slog"
	"testing"

	microsandboxadapter "github.com/zakari/hopeitworks/backend/internal/adapter/microsandbox"
	pkgconfig "github.com/zakari/hopeitworks/backend/pkg/config"
)

// quietLogger discards output so factory tests don't spam the test log.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSelectSubstrate(t *testing.T) {
	t.Run("docker default returns nil (live path needs no extra adapter)", func(t *testing.T) {
		got := selectSubstrate(pkgconfig.SubstrateDocker, nil, quietLogger())
		if got != nil {
			t.Fatalf("selectSubstrate(docker) = %T, want nil", got)
		}
	})

	t.Run("unknown kind falls through to docker default (returns nil)", func(t *testing.T) {
		// validate() rejects unknown kinds at load time; the factory itself
		// defaults defensively so it never panics on an unexpected value.
		got := selectSubstrate("k8s", nil, quietLogger())
		if got != nil {
			t.Fatalf("selectSubstrate(unknown) = %T, want nil", got)
		}
	})

	t.Run("microsandbox constructs the scaffold AgentRuntime", func(t *testing.T) {
		got := selectSubstrate(pkgconfig.SubstrateMicrosandbox, nil, quietLogger())
		if got == nil {
			t.Fatal("selectSubstrate(microsandbox) = nil, want a scaffold adapter")
		}
		if _, ok := got.(*microsandboxadapter.Runtime); !ok {
			t.Fatalf("selectSubstrate(microsandbox) = %T, want *microsandbox.Runtime", got)
		}
	})
}
