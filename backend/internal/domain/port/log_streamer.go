package port

import (
	"context"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// LogStreamer abstracts streaming logs from running containers.
type LogStreamer interface {
	// StreamLogs streams log events from a container.
	// The returned log channel receives LogEvent structs as they are parsed.
	// The returned done channel receives the container exit code when streaming ends.
	// Both channels are closed when the container exits or context is cancelled.
	StreamLogs(ctx context.Context, containerID string, runID string, stepID string) (<-chan model.LogEvent, <-chan int, error)
}
