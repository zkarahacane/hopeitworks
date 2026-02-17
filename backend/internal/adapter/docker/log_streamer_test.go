//nolint:goconst // Test file with many repeated test IDs
package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"

	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

const (
	testLogContainerID = "log-container-123"
	testLogRunID       = "run-abc"
	testLogStepID      = "step-xyz"
)

// mockLogStreamClient is a test double for the logStreamClient interface.
type mockLogStreamClient struct {
	logsReader io.ReadCloser
	logsErr    error
	waitStatus dockercontainer.WaitResponse
	waitErr    error
}

func (m *mockLogStreamClient) ContainerLogs(_ context.Context, _ string, _ dockercontainer.LogsOptions) (io.ReadCloser, error) {
	return m.logsReader, m.logsErr
}

func (m *mockLogStreamClient) ContainerWait(_ context.Context, _ string, _ dockercontainer.WaitCondition) (<-chan dockercontainer.WaitResponse, <-chan error) {
	statusCh := make(chan dockercontainer.WaitResponse, 1)
	errCh := make(chan error, 1)

	if m.waitErr != nil {
		errCh <- m.waitErr
	} else {
		statusCh <- m.waitStatus
	}

	return statusCh, errCh
}

// stdcopyReader wraps lines as a multiplexed Docker log stream (stdout frames).
func stdcopyReader(lines string) io.ReadCloser {
	var buf bytes.Buffer
	w := stdcopy.NewStdWriter(&buf, stdcopy.Stdout)
	_, _ = io.WriteString(w, lines)
	return io.NopCloser(&buf)
}

func newTestLogStreamer(mock logStreamClient) *LogStreamer {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return &LogStreamer{client: mock, logger: logger, idleTimeout: DefaultIdleTimeout}
}

func newTestLogStreamerWithTimeout(mock logStreamClient, timeout time.Duration) *LogStreamer {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return &LogStreamer{client: mock, logger: logger, idleTimeout: timeout}
}

func TestStreamLogs_ValidNDJSON(t *testing.T) {
	lines := `{"level":"info","message":"starting"}
{"level":"warn","message":"something"}
{"level":"error","message":"failed"}
`
	mock := &mockLogStreamClient{
		logsReader: stdcopyReader(lines),
		waitStatus: dockercontainer.WaitResponse{StatusCode: 0},
	}
	streamer := newTestLogStreamer(mock)

	logCh, doneCh, err := streamer.StreamLogs(context.Background(), testLogContainerID, testLogRunID, testLogStepID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var events []string
	for event := range logCh {
		events = append(events, event.Level+":"+event.Message)
		if !event.IsJSON {
			t.Errorf("expected IsJSON=true for line %q", event.RawLine)
		}
		if event.RunID != testLogRunID {
			t.Errorf("expected RunID=%s, got %s", testLogRunID, event.RunID)
		}
		if event.StepID != testLogStepID {
			t.Errorf("expected StepID=%s, got %s", testLogStepID, event.StepID)
		}
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d: %v", len(events), events)
	}

	expected := []string{"info:starting", "warn:something", "error:failed"}
	for i, want := range expected {
		if events[i] != want {
			t.Errorf("event[%d]: expected %s, got %s", i, want, events[i])
		}
	}

	exitCode, ok := <-doneCh
	if !ok {
		t.Fatal("expected exit code on done channel, channel was closed")
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestStreamLogs_MixedValidInvalidJSON(t *testing.T) {
	lines := `{"level":"info","message":"valid"}
plain text log line
{"level":"debug","message":"also valid"}
`
	mock := &mockLogStreamClient{
		logsReader: stdcopyReader(lines),
		waitStatus: dockercontainer.WaitResponse{StatusCode: 0},
	}
	streamer := newTestLogStreamer(mock)

	logCh, _, err := streamer.StreamLogs(context.Background(), testLogContainerID, testLogRunID, testLogStepID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var events []struct {
		isJSON  bool
		message string
	}
	for event := range logCh {
		events = append(events, struct {
			isJSON  bool
			message string
		}{isJSON: event.IsJSON, message: event.Message})
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	if !events[0].isJSON {
		t.Error("event[0]: expected IsJSON=true")
	}
	if events[1].isJSON {
		t.Error("event[1]: expected IsJSON=false for plain text")
	}
	if events[1].message != "plain text log line" {
		t.Errorf("event[1]: expected message=%q, got %q", "plain text log line", events[1].message)
	}
	if !events[2].isJSON {
		t.Error("event[2]: expected IsJSON=true")
	}
}

func TestStreamLogs_ContainerExit(t *testing.T) {
	lines := `{"level":"info","message":"done"}
`
	mock := &mockLogStreamClient{
		logsReader: stdcopyReader(lines),
		waitStatus: dockercontainer.WaitResponse{StatusCode: 42},
	}
	streamer := newTestLogStreamer(mock)

	logCh, doneCh, err := streamer.StreamLogs(context.Background(), testLogContainerID, testLogRunID, testLogStepID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Drain log channel.
	for e := range logCh {
		_ = e
	}

	exitCode, ok := <-doneCh
	if !ok {
		t.Fatal("expected exit code on done channel")
	}
	if exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", exitCode)
	}
}

func TestStreamLogs_ContextCancellation(t *testing.T) {
	// Use a pipe so we can control when data arrives.
	pr, pw := io.Pipe()

	// Wrap the pipe in a stdcopy writer so the streamer can demultiplex it.
	stdPR, stdPW := io.Pipe()
	go func() {
		w := stdcopy.NewStdWriter(stdPW, stdcopy.Stdout)
		buf := make([]byte, 4096)
		for {
			n, readErr := pr.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					stdPW.CloseWithError(writeErr)
					return
				}
			}
			if readErr != nil {
				stdPW.CloseWithError(readErr)
				return
			}
		}
	}()

	mock := &mockLogStreamClient{
		logsReader: stdPR,
		waitStatus: dockercontainer.WaitResponse{StatusCode: 0},
	}
	streamer := newTestLogStreamer(mock)

	ctx, cancel := context.WithCancel(context.Background())

	logCh, doneCh, err := streamer.StreamLogs(ctx, testLogContainerID, testLogRunID, testLogStepID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Write one line, then cancel.
	line := `{"level":"info","message":"before cancel"}` + "\n"
	_, _ = pw.Write([]byte(line))

	// Read the first event.
	event, ok := <-logCh
	if !ok {
		t.Fatal("expected event before cancel")
	}
	if event.Message != "before cancel" {
		t.Errorf("expected message=%q, got %q", "before cancel", event.Message)
	}

	// Cancel context.
	cancel()
	// Close the writer to unblock internal goroutines.
	pw.Close()

	// Wait for channels to close.
	for e := range logCh {
		_ = e
	}

	// Done channel should be closed (with or without exit code — context cancel wins).
	<-doneCh
}

func TestStreamLogs_ContainerLogsError(t *testing.T) {
	mock := &mockLogStreamClient{
		logsErr: errors.New("container not found"),
	}
	streamer := newTestLogStreamer(mock)

	_, _, err := streamer.StreamLogs(context.Background(), testLogContainerID, testLogRunID, testLogStepID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *apperrors.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != apperrors.ErrCodeLogStreamFailed {
		t.Errorf("expected error code %s, got %s", apperrors.ErrCodeLogStreamFailed, domainErr.Code)
	}
	if domainErr.Details["container_id"] != testLogContainerID {
		t.Errorf("expected container_id in details, got %v", domainErr.Details)
	}
}

func TestStreamLogs_EmptyLinesSkipped(t *testing.T) {
	lines := `{"level":"info","message":"first"}

{"level":"info","message":"second"}

`
	mock := &mockLogStreamClient{
		logsReader: stdcopyReader(lines),
		waitStatus: dockercontainer.WaitResponse{StatusCode: 0},
	}
	streamer := newTestLogStreamer(mock)

	logCh, _, err := streamer.StreamLogs(context.Background(), testLogContainerID, testLogRunID, testLogStepID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	count := 0
	for range logCh {
		count++
	}

	if count != 2 {
		t.Errorf("expected 2 events (empty lines skipped), got %d", count)
	}
}

func TestStreamLogs_ChannelsClosed(t *testing.T) {
	mock := &mockLogStreamClient{
		logsReader: stdcopyReader(""),
		waitStatus: dockercontainer.WaitResponse{StatusCode: 0},
	}
	streamer := newTestLogStreamer(mock)

	logCh, doneCh, err := streamer.StreamLogs(context.Background(), testLogContainerID, testLogRunID, testLogStepID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	timeout := time.After(5 * time.Second)

	// Drain logCh until closed.
	logDrained := false
	for !logDrained {
		select {
		case _, ok := <-logCh:
			if !ok {
				logDrained = true
			}
		case <-timeout:
			t.Fatal("timed out waiting for logCh to close")
		}
	}

	// doneCh should close after logCh (exit code may or may not be present).
	select {
	case _, ok := <-doneCh:
		if ok {
			// Got exit code — wait for channel to close.
			select {
			case _, ok = <-doneCh:
				if ok {
					t.Error("expected doneCh to be closed after exit code")
				}
			case <-timeout:
				t.Fatal("timed out waiting for doneCh to close after exit code")
			}
		}
	case <-timeout:
		t.Fatal("timed out waiting for doneCh to close")
	}
}

func TestStreamLogs_IdleTimeout(t *testing.T) {
	// Use a pipe so we can control when data arrives.
	pr, pw := io.Pipe()

	// Wrap the pipe in a stdcopy writer.
	stdPR, stdPW := io.Pipe()
	go func() {
		w := stdcopy.NewStdWriter(stdPW, stdcopy.Stdout)
		buf := make([]byte, 4096)
		for {
			n, readErr := pr.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					stdPW.CloseWithError(writeErr)
					return
				}
			}
			if readErr != nil {
				stdPW.CloseWithError(readErr)
				return
			}
		}
	}()

	mock := &mockLogStreamClient{
		logsReader: stdPR,
		waitStatus: dockercontainer.WaitResponse{StatusCode: 0},
	}
	// Use a short idle timeout for testing (100ms).
	idleTimeout := 100 * time.Millisecond
	streamer := newTestLogStreamerWithTimeout(mock, idleTimeout)

	logCh, _, err := streamer.StreamLogs(context.Background(), testLogContainerID, testLogRunID, testLogStepID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedMsg := fmt.Sprintf("No log output for %s", idleTimeout)

	// Wait for idle timeout warning (should arrive after ~100ms).
	timeout := time.After(5 * time.Second)
	select {
	case event := <-logCh:
		if event.Level != "warn" {
			t.Errorf("expected Level=warn, got %s", event.Level)
		}
		if event.Message != expectedMsg {
			t.Errorf("expected idle timeout message %q, got %q", expectedMsg, event.Message)
		}
		if event.IsJSON {
			t.Error("expected IsJSON=false for idle timeout warning")
		}
	case <-timeout:
		t.Fatal("timed out waiting for idle timeout warning")
	}

	// After warning, streamer should continue. Write a line.
	line := `{"level":"info","message":"resumed"}` + "\n"
	_, _ = pw.Write([]byte(line))

	select {
	case event := <-logCh:
		if event.Message != "resumed" {
			t.Errorf("expected message=resumed, got %q", event.Message)
		}
	case <-timeout:
		t.Fatal("timed out waiting for log event after idle timeout")
	}

	// Close pipe to end streaming.
	pw.Close()
	for e := range logCh {
		_ = e
	}
}

func TestStreamLogs_IdleTimeoutResets(t *testing.T) {
	// Use a pipe to control timing.
	pr, pw := io.Pipe()

	// Wrap the pipe in a stdcopy writer.
	stdPR, stdPW := io.Pipe()
	go func() {
		w := stdcopy.NewStdWriter(stdPW, stdcopy.Stdout)
		buf := make([]byte, 4096)
		for {
			n, readErr := pr.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					stdPW.CloseWithError(writeErr)
					return
				}
			}
			if readErr != nil {
				stdPW.CloseWithError(readErr)
				return
			}
		}
	}()

	mock := &mockLogStreamClient{
		logsReader: stdPR,
		waitStatus: dockercontainer.WaitResponse{StatusCode: 0},
	}
	// Use 200ms idle timeout.
	streamer := newTestLogStreamerWithTimeout(mock, 200*time.Millisecond)

	logCh, _, err := streamer.StreamLogs(context.Background(), testLogContainerID, testLogRunID, testLogStepID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Write a line before the idle timeout fires (at 100ms, well within 200ms).
	time.Sleep(50 * time.Millisecond)
	_, _ = pw.Write([]byte(`{"level":"info","message":"keep alive"}` + "\n"))

	// Read that event.
	timeout := time.After(5 * time.Second)
	select {
	case event := <-logCh:
		if event.Message != "keep alive" {
			t.Errorf("expected message=%q, got %q", "keep alive", event.Message)
		}
	case <-timeout:
		t.Fatal("timed out waiting for keep alive event")
	}

	// Now wait for idle timeout to fire (should be 200ms from last line).
	select {
	case event := <-logCh:
		if event.Level != "warn" {
			t.Errorf("expected idle timeout warning, got level=%s msg=%q", event.Level, event.Message)
		}
	case <-timeout:
		t.Fatal("timed out waiting for idle timeout after reset")
	}

	pw.Close()
	for e := range logCh {
		_ = e
	}
}

func TestStreamLogs_WaitError(t *testing.T) {
	mock := &mockLogStreamClient{
		logsReader: stdcopyReader(`{"level":"info","message":"test"}` + "\n"),
		waitErr:    errors.New("wait failed"),
	}
	streamer := newTestLogStreamer(mock)

	logCh, doneCh, err := streamer.StreamLogs(context.Background(), testLogContainerID, testLogRunID, testLogStepID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Drain log channel.
	for e := range logCh {
		_ = e
	}

	// Done channel should be closed without exit code since wait failed.
	_, ok := <-doneCh
	if ok {
		t.Error("expected doneCh to be closed without exit code when wait fails")
	}
}
