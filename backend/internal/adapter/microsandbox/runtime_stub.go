//go:build !microsandbox

// This file is the DEFAULT build of the microVM live-execution path. It carries
// no microsandbox-SDK import (and therefore no libkrun dependency), so the
// standard `go build ./...` compiles on any host — including CI and the
// devcontainer, which have no KVM.
//
// The real microVM implementation lives in runtime_microsandbox.go behind
// //go:build microsandbox; build with `-tags microsandbox` on a KVM/HVF host
// (see deploy/lima/microsandbox-vm.yaml) to get a functional Launch/Wait/Stop.

package microsandbox

import (
	"context"
	"errors"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// ErrNotBuilt is returned by Launch/Wait/Stop in the default build. Selecting
// SUBSTRATE=microsandbox on a binary compiled without the `microsandbox` build
// tag fails clearly here instead of silently doing nothing.
var ErrNotBuilt = errors.New("microsandbox: live execution not available (binary not built with the 'microsandbox' build tag; rebuild with `-tags microsandbox` on a KVM/HVF host)")

// Launch reports that the binary lacks the microsandbox live path. See ErrNotBuilt.
func (r *Runtime) Launch(_ context.Context, _ port.RunSpec) (port.RunHandle, error) {
	return port.RunHandle{}, ErrNotBuilt
}

// Wait reports that the binary lacks the microsandbox live path. See ErrNotBuilt.
func (r *Runtime) Wait(_ context.Context, _ port.RunHandle) (port.RunResult, error) {
	return port.RunResult{}, ErrNotBuilt
}

// Stop reports that the binary lacks the microsandbox live path. See ErrNotBuilt.
func (r *Runtime) Stop(_ context.Context, _ port.RunHandle) error {
	return ErrNotBuilt
}
