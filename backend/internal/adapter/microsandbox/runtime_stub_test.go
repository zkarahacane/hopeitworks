//go:build !microsandbox

package microsandbox

import (
	"context"
	"errors"
	"testing"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// TestRuntime_LiveStubs_ReturnNotBuilt exercises the DEFAULT build's fallback
// (runtime_stub.go): without the `microsandbox` tag, the live operations must
// report ErrNotBuilt so a misconfigured SUBSTRATE=microsandbox fails clearly.
// Under `-tags microsandbox` this file is excluded by the build constraint and
// the real SDK path is exercised by the tagged validation harness instead.
func TestRuntime_LiveStubs_ReturnNotBuilt(t *testing.T) {
	rt := NewRuntime(true, nil, nil) // enabled is irrelevant in the fallback build
	ctx := context.Background()

	if _, err := rt.Launch(ctx, port.RunSpec{}); !errors.Is(err, ErrNotBuilt) {
		t.Errorf("Launch err = %v, want ErrNotBuilt", err)
	}
	if _, err := rt.Wait(ctx, port.RunHandle{}); !errors.Is(err, ErrNotBuilt) {
		t.Errorf("Wait err = %v, want ErrNotBuilt", err)
	}
	if err := rt.Stop(ctx, port.RunHandle{}); !errors.Is(err, ErrNotBuilt) {
		t.Errorf("Stop err = %v, want ErrNotBuilt", err)
	}
}
