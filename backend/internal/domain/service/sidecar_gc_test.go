package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

const (
	errWantCanceled = "expected context.Canceled, got %v"
	errGCNotStopped = "sidecar gc did not stop after context cancellation"
)

// mockSidecarManagerForGC is a minimal SidecarManager that records GC calls. Only
// GC and the windows it is invoked with are exercised here; the other methods are
// no-ops to satisfy the interface.
type mockSidecarManagerForGC struct {
	mu        sync.Mutex
	gcCalls   int
	gcWindows []time.Duration
	gcErr     error
}

func (m *mockSidecarManagerForGC) Launch(_ context.Context, _ uuid.UUID, _ *model.Environment) (*port.SidecarContext, error) {
	return &port.SidecarContext{}, nil
}

func (m *mockSidecarManagerForGC) Stop(_ context.Context, _ *port.SidecarContext) error { return nil }

func (m *mockSidecarManagerForGC) Cleanup(_ context.Context, _ *port.SidecarContext) error {
	return nil
}

func (m *mockSidecarManagerForGC) ListOrphanNetworks(_ context.Context) ([]model.NetworkInfo, error) {
	return nil, nil
}

func (m *mockSidecarManagerForGC) GC(_ context.Context, olderThan time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcCalls++
	m.gcWindows = append(m.gcWindows, olderThan)
	return m.gcErr
}

func (m *mockSidecarManagerForGC) calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.gcCalls
}

func (m *mockSidecarManagerForGC) windows() []time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]time.Duration, len(m.gcWindows))
	copy(out, m.gcWindows)
	return out
}

func TestNewSidecarGC_DefaultsOnNonPositive(t *testing.T) {
	gc := NewSidecarGC(&mockSidecarManagerForGC{}, discardLogger(), 0, -1)
	if gc.interval != DefaultSidecarGCInterval {
		t.Errorf("expected default interval %s, got %s", DefaultSidecarGCInterval, gc.interval)
	}
	if gc.window != DefaultSidecarGCWindow {
		t.Errorf("expected default window %s, got %s", DefaultSidecarGCWindow, gc.window)
	}
}

func TestSidecarGC_DefaultWindowIsSafe(t *testing.T) {
	// The window must stay far larger than any plausible run startup latency so a
	// network created moments before its container starts is never reaped.
	if DefaultSidecarGCWindow < 30*time.Minute {
		t.Fatalf("default GC window %s is too short to be race-safe", DefaultSidecarGCWindow)
	}
}

func TestSidecarGC_CallsGCOnTick(t *testing.T) {
	mgr := &mockSidecarManagerForGC{}
	// Small interval so the test runs fast; window is the safe one we ship.
	gc := NewSidecarGC(mgr, discardLogger(), 20*time.Millisecond, DefaultSidecarGCWindow)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- gc.Start(ctx)
	}()

	// Let it run several ticks.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf(errWantCanceled, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal(errGCNotStopped)
	}

	if mgr.calls() == 0 {
		t.Fatal("expected GC to be called at least once on tick")
	}
	for _, w := range mgr.windows() {
		if w != DefaultSidecarGCWindow {
			t.Errorf("expected GC called with window %s, got %s", DefaultSidecarGCWindow, w)
		}
	}
}

func TestSidecarGC_StopsOnContextCancellation(t *testing.T) {
	mgr := &mockSidecarManagerForGC{}
	// Long interval so the loop is parked on the ticker, proving cancellation
	// returns promptly regardless of the tick schedule.
	gc := NewSidecarGC(mgr, discardLogger(), 1*time.Hour, DefaultSidecarGCWindow)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- gc.Start(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf(errWantCanceled, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal(errGCNotStopped)
	}
}

func TestSidecarGC_ContinuesOnGCError(t *testing.T) {
	mgr := &mockSidecarManagerForGC{gcErr: assertGCError{}}
	gc := NewSidecarGC(mgr, discardLogger(), 20*time.Millisecond, DefaultSidecarGCWindow)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- gc.Start(ctx)
	}()

	// A failing GC must not stop the loop: it should keep ticking.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf(errWantCanceled, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal(errGCNotStopped)
	}

	if mgr.calls() < 2 {
		t.Errorf("expected GC to keep being called despite errors, got %d calls", mgr.calls())
	}
}

// assertGCError is a trivial error used to make GC fail in tests.
type assertGCError struct{}

func (assertGCError) Error() string { return "gc failed" }
