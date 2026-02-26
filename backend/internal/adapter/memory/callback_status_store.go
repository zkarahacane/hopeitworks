package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// statusResult holds the final exit status for an agent container step.
type statusResult struct {
	ExitCode int
	ErrMsg   string
}

// CallbackStatusStore is an in-memory implementation of port.CallbackStatusStore.
// It uses buffered channels per step to coordinate between the HTTP callback handler
// (which calls SetStatus) and the pipeline executor (which calls WaitForStatus).
type CallbackStatusStore struct {
	mu       sync.Mutex
	channels map[uuid.UUID]chan statusResult
}

// NewCallbackStatusStore creates a new in-memory callback status store.
func NewCallbackStatusStore() *CallbackStatusStore {
	return &CallbackStatusStore{
		channels: make(map[uuid.UUID]chan statusResult),
	}
}

// WaitForStatus blocks until a status is set for the given step, the context is
// cancelled, or the timeout elapses. Returns the exit code and an optional error message.
func (s *CallbackStatusStore) WaitForStatus(ctx context.Context, stepID uuid.UUID, timeout time.Duration) (int, string, error) {
	ch := s.getOrCreateChannel(stepID)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result := <-ch:
		s.removeChannel(stepID)
		return result.ExitCode, result.ErrMsg, nil
	case <-ctx.Done():
		s.removeChannel(stepID)
		return -1, "", ctx.Err()
	case <-timer.C:
		s.removeChannel(stepID)
		return -1, "", fmt.Errorf("timeout waiting for status callback for step %s", stepID)
	}
}

// SetStatus sets the final status for a step, unblocking any WaitForStatus call.
// If nobody is waiting yet, the result is buffered in the channel (size 1).
func (s *CallbackStatusStore) SetStatus(_ context.Context, stepID uuid.UUID, exitCode int, errMsg string) error {
	ch := s.getOrCreateChannel(stepID)

	result := statusResult{ExitCode: exitCode, ErrMsg: errMsg}
	select {
	case ch <- result:
		// Sent successfully
	default:
		// Channel already has a value (duplicate callback); ignore
	}

	return nil
}

// getOrCreateChannel returns the existing channel for a step or creates a new buffered one.
func (s *CallbackStatusStore) getOrCreateChannel(stepID uuid.UUID) chan statusResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, ok := s.channels[stepID]
	if !ok {
		ch = make(chan statusResult, 1)
		s.channels[stepID] = ch
	}
	return ch
}

// removeChannel cleans up the channel for a step after it has been consumed.
func (s *CallbackStatusStore) removeChannel(stepID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.channels, stepID)
}
