package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// logStreamClient defines the subset of the Docker SDK client used by LogStreamer.
type logStreamClient interface {
	ContainerLogs(ctx context.Context, container string, options dockercontainer.LogsOptions) (io.ReadCloser, error)
	ContainerWait(ctx context.Context, containerID string, condition dockercontainer.WaitCondition) (<-chan dockercontainer.WaitResponse, <-chan error)
}

// DefaultIdleTimeout is how long to wait without log output before emitting a warning.
const DefaultIdleTimeout = 60 * time.Second

// Ensure LogStreamer implements port.LogStreamer at compile time.
var _ port.LogStreamer = (*LogStreamer)(nil)

// LogStreamer implements port.LogStreamer using the Docker SDK.
type LogStreamer struct {
	client      logStreamClient
	logger      *slog.Logger
	idleTimeout time.Duration
}

// NewDockerLogStreamer creates a LogStreamer with an existing Docker client.
func NewDockerLogStreamer(client logStreamClient, logger *slog.Logger) *LogStreamer {
	return &LogStreamer{client: client, logger: logger, idleTimeout: DefaultIdleTimeout}
}

// NewDockerLogStreamerFromHost creates a LogStreamer by connecting to Docker
// via the specified host URL (e.g., "tcp://socket-proxy:2375").
func NewDockerLogStreamerFromHost(dockerHost string, logger *slog.Logger) (*LogStreamer, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating docker client for log streamer: %w", err)
	}
	return &LogStreamer{client: cli, logger: logger, idleTimeout: DefaultIdleTimeout}, nil
}

// StreamLogs streams log events from a container.
// The returned log channel receives LogEvent structs as they are parsed.
// The returned done channel receives the container exit code when streaming ends.
// Both channels are closed when the container exits or context is cancelled.
func (s *LogStreamer) StreamLogs(ctx context.Context, containerID string, runID string, stepID string) (<-chan model.LogEvent, <-chan int, error) {
	logReader, err := s.client.ContainerLogs(ctx, containerID, dockercontainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
	})
	if err != nil {
		return nil, nil, apperrors.NewDomainError(
			apperrors.ErrCodeLogStreamFailed,
			fmt.Sprintf("failed to attach to container logs: %v", err),
			map[string]any{"container_id": containerID},
		)
	}

	logCh := make(chan model.LogEvent, 100)
	doneCh := make(chan int, 1)

	go s.streamLoop(ctx, logReader, containerID, runID, stepID, logCh, doneCh)

	return logCh, doneCh, nil
}

// scanResult holds the result of a single scanner.Scan() call.
type scanResult struct {
	line string
	ok   bool
}

func (s *LogStreamer) streamLoop(ctx context.Context, reader io.ReadCloser, containerID string, runID string, stepID string, logCh chan model.LogEvent, doneCh chan int) {
	defer close(logCh)
	defer close(doneCh)
	defer reader.Close()

	s.logger.Debug("log streaming started",
		slog.String("container_id", containerID),
		slog.String("run_id", runID),
		slog.String("step_id", stepID),
	)

	// Docker ContainerLogs returns a multiplexed stream (stdout/stderr frames with
	// 8-byte headers). We must demultiplex it via stdcopy.StdCopy before scanning.
	// We pipe both stdout and stderr into a single reader for line-by-line scanning.
	pr, pw := io.Pipe()
	go func() {
		_, copyErr := stdcopy.StdCopy(pw, pw, reader)
		pw.CloseWithError(copyErr)
	}()

	scanner := bufio.NewScanner(pr)
	lineCh := make(chan scanResult)

	// Goroutine that reads lines from the scanner and sends them on lineCh.
	// When the scanner reaches EOF or errors, it sends ok=false and returns.
	go func() {
		defer close(lineCh)
		for scanner.Scan() {
			lineCh <- scanResult{line: scanner.Text(), ok: true}
		}
		// EOF or error
		lineCh <- scanResult{ok: false}
	}()

	idleTimer := time.NewTimer(s.idleTimeout)
	defer idleTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("log streaming cancelled",
				slog.String("container_id", containerID),
			)
			return

		case <-idleTimer.C:
			logCh <- model.LogEvent{
				RunID:     runID,
				StepID:    stepID,
				Timestamp: time.Now(),
				Level:     "warn",
				Message:   fmt.Sprintf("No log output for %s", s.idleTimeout),
				IsJSON:    false,
			}
			idleTimer.Reset(s.idleTimeout)

		case result, chanOpen := <-lineCh:
			if !chanOpen {
				// lineCh closed unexpectedly; treat as EOF.
				s.handleContainerExit(ctx, containerID, runID, stepID, scanner, logCh, doneCh)
				return
			}
			if !result.ok {
				// Scanner done (EOF or error).
				s.handleContainerExit(ctx, containerID, runID, stepID, scanner, logCh, doneCh)
				return
			}

			if event := parseNDJSONLine(result.line, runID, stepID); event != nil {
				logCh <- *event
				if !idleTimer.Stop() {
					select {
					case <-idleTimer.C:
					default:
					}
				}
				idleTimer.Reset(s.idleTimeout)
			}
		}
	}
}

func (s *LogStreamer) handleContainerExit(_ context.Context, containerID string, runID string, stepID string, scanner *bufio.Scanner, logCh chan model.LogEvent, doneCh chan int) {
	if err := scanner.Err(); err != nil {
		logCh <- model.LogEvent{
			RunID:     runID,
			StepID:    stepID,
			Timestamp: time.Now(),
			Level:     "error",
			Message:   fmt.Sprintf("log stream error: %v", err),
			IsJSON:    false,
		}
	}

	// Use a background context for ContainerWait to avoid missing the exit code
	// if the streaming context was cancelled.
	waitCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	statusCh, errCh := s.client.ContainerWait(waitCtx, containerID, dockercontainer.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			s.logger.Debug("failed to get container exit code",
				slog.String("container_id", containerID),
				slog.String("error", err.Error()),
			)
		}
	case status := <-statusCh:
		doneCh <- int(status.StatusCode)
		s.logger.Debug("container exited",
			slog.String("container_id", containerID),
			slog.Int("exit_code", int(status.StatusCode)),
		)
	case <-waitCtx.Done():
		s.logger.Debug("timed out waiting for container exit code",
			slog.String("container_id", containerID),
		)
	}
}
