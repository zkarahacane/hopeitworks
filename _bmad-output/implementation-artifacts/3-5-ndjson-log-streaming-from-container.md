# Story 3.5: [BACK] NDJSON log streaming from container

Status: ready-for-dev

## Story

As a backend developer, I want to stream NDJSON logs from running agent containers, so that the system can capture and forward real-time agent output.

## Acceptance Criteria (BDD)

**AC1: LogEvent domain model captures structured and raw log data**
- **Given** a LogEvent struct in `backend/internal/domain/model/log_event.go`
- **When** the struct is reviewed
- **Then** it includes RunID string field
- **And** it includes StepID string field
- **And** it includes Timestamp time.Time field
- **And** it includes Level string field (info, warn, error, debug)
- **And** it includes Message string field
- **And** it includes RawLine string field
- **And** it includes IsJSON bool field (true if line was valid NDJSON)
- **And** it includes Data map[string]any field for parsed JSON fields (optional)

**AC2: LogStreamer port interface defines log streaming contract**
- **Given** a LogStreamer port interface in `backend/internal/domain/port/log_streamer.go`
- **When** the interface is reviewed
- **Then** it declares StreamLogs(ctx, containerID, runID, stepID) (<-chan LogEvent, <-chan int, error)
- **And** the returned log channel receives LogEvent structs as they are parsed
- **And** the returned done channel receives the container exit code when streaming ends
- **And** the log channel is closed when the container exits or context is cancelled
- **And** all methods return domain errors with contextual information

**AC3: NDJSON parser validates and parses JSON lines**
- **Given** a parseNDJSONLine function in `backend/internal/adapter/docker/ndjson_parser.go`
- **When** a valid JSON line is passed (e.g., `{"level":"info","message":"test"}`)
- **Then** it returns a LogEvent with IsJSON=true
- **And** it extracts level, message, timestamp from JSON fields
- **And** it populates Data map with all JSON fields
- **When** an invalid JSON line is passed (e.g., plain text)
- **Then** it returns a LogEvent with IsJSON=false
- **And** it sets RawLine to the original line
- **And** it sets Level="info" as default
- **When** an empty line is passed
- **Then** it returns nil (skip empty lines)

**AC4: Docker log streamer reads container stdout/stderr line by line**
- **Given** a Docker log streamer in `backend/internal/adapter/docker/log_streamer.go`
- **When** StreamLogs is called with a running container ID
- **Then** it calls Docker SDK ContainerLogs with Follow=true, ShowStdout=true, ShowStderr=true
- **And** it returns an io.ReadCloser for the log stream
- **And** it wraps the stream with bufio.Scanner for line-by-line reading
- **And** it wraps errors in DomainError with container ID context

**AC5: Docker log streamer parses and forwards NDJSON events**
- **Given** a running Docker container streaming logs
- **When** the log stream produces a valid NDJSON line
- **Then** the log streamer parses the line via parseNDJSONLine
- **And** it sends the LogEvent on the log channel
- **And** it continues reading the next line
- **When** the log stream produces an invalid JSON line
- **Then** the log streamer wraps it as a LogEvent with IsJSON=false
- **And** it sends the LogEvent on the log channel
- **And** it continues reading (no stream interruption)

**AC6: Docker log streamer detects and warns on idle timeout**
- **Given** a Docker container streaming logs
- **When** no log output is received for 60 seconds
- **Then** the log streamer emits a warning LogEvent with Level="warn", Message="No log output for 60s"
- **And** it continues streaming (does not close the stream)
- **And** it resets the idle timer on the next log line

**AC7: Docker log streamer handles container exit cleanly**
- **Given** a Docker container that exits during log streaming
- **When** the log stream reaches EOF
- **Then** the log streamer stops reading logs
- **And** it calls ContainerWait to capture the exit code
- **And** it sends the exit code on the done channel
- **And** it closes both the log channel and done channel
- **And** it returns without error

**AC8: Docker log streamer handles context cancellation**
- **Given** a Docker container streaming logs
- **When** the context is cancelled (ctx.Done())
- **Then** the log streamer stops reading logs
- **And** it closes the log stream reader
- **And** it closes both the log channel and done channel
- **And** it does not send an exit code on the done channel

**AC9: Unit tests verify NDJSON parser behavior**
- **Given** unit tests in `backend/internal/adapter/docker/ndjson_parser_test.go`
- **When** tests are executed
- **Then** parseNDJSONLine tests verify valid JSON parsing (level, message, timestamp, data)
- **And** parseNDJSONLine tests verify malformed JSON wrapping (IsJSON=false, RawLine set)
- **And** parseNDJSONLine tests verify empty line handling (nil return)
- **And** parseNDJSONLine tests verify missing fields fallback (default level="info")
- **And** all tests use table-driven test format for readability

**AC10: Unit tests verify log streamer behavior with mock Docker client**
- **Given** unit tests in `backend/internal/adapter/docker/log_streamer_test.go`
- **When** tests are executed
- **Then** StreamLogs tests verify correct Docker SDK ContainerLogs call
- **And** StreamLogs tests verify line-by-line parsing (mock log stream)
- **And** StreamLogs tests verify LogEvent forwarding to channel
- **And** StreamLogs tests verify idle timeout warning (mock 60s delay)
- **And** StreamLogs tests verify container exit handling (EOF → exit code on done channel)
- **And** StreamLogs tests verify context cancellation (channels closed, no exit code)
- **And** error handling tests verify DomainError wrapping
- **And** all tests use mock Docker client to avoid real Docker operations

## Tasks / Subtasks

- [ ] [BACK] Task 1: Create LogEvent domain model (AC: #1)
  - [ ] Create `backend/internal/domain/model/log_event.go`
  - [ ] Define LogEvent struct with RunID, StepID, Timestamp, Level, Message, RawLine, IsJSON, Data fields
  - [ ] Document struct fields with godoc comments
  - [ ] Add JSON tags for serialization

- [ ] [BACK] Task 2: Define LogStreamer port interface (AC: #2)
  - [ ] Create `backend/internal/domain/port/log_streamer.go`
  - [ ] Define LogStreamer interface with StreamLogs method
  - [ ] Document interface method with godoc comments (describe channel semantics)
  - [ ] Add context.Context as first parameter

- [ ] [BACK] Task 3: Create NDJSON parser with validation (AC: #3)
  - [ ] Create `backend/internal/adapter/docker/ndjson_parser.go`
  - [ ] Implement parseNDJSONLine(line string, runID, stepID string) (*model.LogEvent, error)
  - [ ] Valid JSON: unmarshal, extract level/message/timestamp, set IsJSON=true, populate Data
  - [ ] Invalid JSON: create LogEvent with IsJSON=false, RawLine=line, Level="info"
  - [ ] Empty lines: return nil (skip)
  - [ ] Handle missing JSON fields gracefully (default level="info", timestamp=time.Now())

- [ ] [BACK] Task 4: Implement Docker log streamer (AC: #4, #5)
  - [ ] Create `backend/internal/adapter/docker/log_streamer.go`
  - [ ] Add DockerLogStreamer struct with Docker SDK client dependency
  - [ ] Implement StreamLogs method: call ContainerLogs with Follow=true
  - [ ] Wrap stream with bufio.Scanner for line-by-line reading
  - [ ] Parse each line via parseNDJSONLine
  - [ ] Send LogEvent on log channel
  - [ ] Wrap errors in DomainError with container ID context

- [ ] [BACK] Task 5: Add idle timeout detection (AC: #6)
  - [ ] Use time.AfterFunc(60s) to detect idle timeout
  - [ ] Reset timer on each log line received
  - [ ] On timeout: emit warning LogEvent with Level="warn", Message="No log output for 60s"
  - [ ] Continue streaming after warning (do not close stream)

- [ ] [BACK] Task 6: Handle container exit and context cancellation (AC: #7, #8)
  - [ ] On EOF: call ContainerWait to get exit code
  - [ ] Send exit code on done channel
  - [ ] Close log channel and done channel
  - [ ] On context cancellation: close reader, close channels, skip exit code
  - [ ] Ensure goroutine cleanup (no leaks)

- [ ] [BACK] Task 7: Write unit tests for NDJSON parser (AC: #9)
  - [ ] Test parseNDJSONLine with valid JSON (all fields present)
  - [ ] Test parseNDJSONLine with valid JSON (missing level → default "info")
  - [ ] Test parseNDJSONLine with valid JSON (missing timestamp → time.Now())
  - [ ] Test parseNDJSONLine with malformed JSON (IsJSON=false, RawLine set)
  - [ ] Test parseNDJSONLine with empty line (returns nil)
  - [ ] Use table-driven test format for readability

- [ ] [BACK] Task 8: Write unit tests for Docker log streamer (AC: #10)
  - [ ] Create mock Docker client with ContainerLogs returning io.ReadCloser
  - [ ] Test StreamLogs with valid NDJSON log stream (verify LogEvent forwarding)
  - [ ] Test StreamLogs with malformed JSON lines (verify IsJSON=false wrapping)
  - [ ] Test StreamLogs idle timeout (mock 60s delay, verify warning LogEvent)
  - [ ] Test StreamLogs container exit (EOF → verify exit code on done channel)
  - [ ] Test StreamLogs context cancellation (verify channels closed, no exit code)
  - [ ] Test error handling and DomainError wrapping
  - [ ] No actual Docker daemon required in unit tests

- [ ] [BACK] Task 9: Write integration test with real Docker container (optional, bonus)
  - [ ] Create integration test file with `//go:build integration` tag
  - [ ] Use real Docker SDK client
  - [ ] Create container with alpine:latest that echoes NDJSON lines
  - [ ] Stream logs via StreamLogs
  - [ ] Verify LogEvent parsing (IsJSON=true for valid NDJSON)
  - [ ] Verify exit code on done channel
  - [ ] Clean up container in test teardown

## Dev Notes

### Dependencies
- Story 3-4: Docker container lifecycle manager (provides ContainerManager, tested separately)
- Docker SDK: `github.com/docker/docker/client`
- Docker API types: `github.com/docker/docker/api/types/container`
- bufio package for line-by-line log reading
- time package for idle timeout detection

### Architecture Requirements
- **Hexagonal architecture:** LogStreamer is a port in domain/port, Docker adapter in adapter/docker
- **Testability:** Docker SDK client is injected as dependency, allowing mock client in unit tests
- **Error handling:** All adapter errors wrapped in DomainError via pkg/errors
- **Structured logging:** Use slog to log streaming lifecycle events (debug level)
- **Channel safety:** All channels properly closed to avoid goroutine leaks

### File Paths (exact)
```
backend/internal/domain/model/log_event.go          # LogEvent struct
backend/internal/domain/port/log_streamer.go         # LogStreamer port interface
backend/internal/adapter/docker/log_streamer.go      # Docker SDK implementation
backend/internal/adapter/docker/log_streamer_test.go # Unit tests with mock Docker client
backend/internal/adapter/docker/ndjson_parser.go     # NDJSON line parser
backend/internal/adapter/docker/ndjson_parser_test.go # Parser unit tests
```

### Technical Specifications

**LogEvent domain model:**
```go
package model

import "time"

// LogEvent represents a single log event from an agent container.
type LogEvent struct {
    // RunID is the ID of the run this log belongs to
    RunID string `json:"run_id"`

    // StepID is the ID of the step this log belongs to
    StepID string `json:"step_id"`

    // Timestamp is when the log event was generated
    Timestamp time.Time `json:"timestamp"`

    // Level is the log level (info, warn, error, debug)
    Level string `json:"level"`

    // Message is the log message
    Message string `json:"message"`

    // RawLine is the raw log line before parsing (for debugging)
    RawLine string `json:"raw_line"`

    // IsJSON indicates whether the line was valid NDJSON
    IsJSON bool `json:"is_json"`

    // Data contains parsed JSON fields (only if IsJSON=true)
    Data map[string]any `json:"data,omitempty"`
}
```

**LogStreamer port interface:**
```go
package port

import (
    "context"
    "hopeitworks/backend/internal/domain/model"
)

// LogStreamer abstracts streaming logs from running containers.
type LogStreamer interface {
    // StreamLogs streams log events from a container.
    // The returned log channel receives LogEvent structs as they are parsed.
    // The returned done channel receives the container exit code when streaming ends.
    // Both channels are closed when the container exits or context is cancelled.
    StreamLogs(ctx context.Context, containerID string, runID string, stepID string) (<-chan model.LogEvent, <-chan int, error)
}
```

**NDJSON parser implementation:**
```go
package docker

import (
    "encoding/json"
    "strings"
    "time"

    "hopeitworks/backend/internal/domain/model"
)

// parseNDJSONLine parses a single log line as NDJSON.
// Returns nil if the line is empty (skip).
// Returns LogEvent with IsJSON=false if JSON parsing fails.
func parseNDJSONLine(line string, runID string, stepID string) *model.LogEvent {
    // Skip empty lines
    line = strings.TrimSpace(line)
    if line == "" {
        return nil
    }

    event := &model.LogEvent{
        RunID:     runID,
        StepID:    stepID,
        RawLine:   line,
        Timestamp: time.Now(),
    }

    // Try to parse as JSON
    var data map[string]any
    if err := json.Unmarshal([]byte(line), &data); err != nil {
        // Not valid JSON, wrap as raw text
        event.IsJSON = false
        event.Level = "info"
        event.Message = line
        return event
    }

    // Valid JSON, extract fields
    event.IsJSON = true
    event.Data = data

    // Extract level (default to "info")
    if level, ok := data["level"].(string); ok {
        event.Level = level
    } else {
        event.Level = "info"
    }

    // Extract message (default to empty string)
    if message, ok := data["message"].(string); ok {
        event.Message = message
    }

    // Extract timestamp (default to time.Now())
    if ts, ok := data["timestamp"].(string); ok {
        if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
            event.Timestamp = parsed
        }
    }

    return event
}
```

**Docker log streamer implementation:**
```go
package docker

import (
    "bufio"
    "context"
    "fmt"
    "io"
    "time"

    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/client"

    "hopeitworks/backend/internal/domain/model"
    "hopeitworks/backend/internal/domain/port"
    "hopeitworks/backend/pkg/errors"
)

type DockerLogStreamer struct {
    client *client.Client
}

func NewDockerLogStreamer(dockerHost string) (*DockerLogStreamer, error) {
    cli, err := client.NewClientWithOpts(
        client.WithHost(dockerHost),
        client.WithAPIVersionNegotiation(),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create Docker client: %w", err)
    }
    return &DockerLogStreamer{client: cli}, nil
}

func (s *DockerLogStreamer) StreamLogs(ctx context.Context, containerID string, runID string, stepID string) (<-chan model.LogEvent, <-chan int, error) {
    // Attach to container logs
    logReader, err := s.client.ContainerLogs(ctx, containerID, container.LogsOptions{
        ShowStdout: true,
        ShowStderr: true,
        Follow:     true,
        Timestamps: false,
    })
    if err != nil {
        return nil, nil, errors.NewDomainError(
            errors.ErrCodeContainerNotFound,
            fmt.Sprintf("failed to attach to container logs: %v", err),
            map[string]any{"container_id": containerID},
        )
    }

    logCh := make(chan model.LogEvent, 100)
    doneCh := make(chan int, 1)

    // Start streaming goroutine
    go s.streamLoop(ctx, logReader, containerID, runID, stepID, logCh, doneCh)

    return logCh, doneCh, nil
}

func (s *DockerLogStreamer) streamLoop(ctx context.Context, reader io.ReadCloser, containerID string, runID string, stepID string, logCh chan model.LogEvent, doneCh chan int) {
    defer close(logCh)
    defer close(doneCh)
    defer reader.Close()

    scanner := bufio.NewScanner(reader)
    idleTimer := time.NewTimer(60 * time.Second)
    defer idleTimer.Stop()

    for {
        select {
        case <-ctx.Done():
            // Context cancelled, stop streaming
            return

        case <-idleTimer.C:
            // Idle timeout, emit warning and continue
            logCh <- model.LogEvent{
                RunID:     runID,
                StepID:    stepID,
                Timestamp: time.Now(),
                Level:     "warn",
                Message:   "No log output for 60s",
                IsJSON:    false,
            }
            idleTimer.Reset(60 * time.Second)

        default:
            if !scanner.Scan() {
                // EOF or error
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

                // Container exited, get exit code
                statusCh, errCh := s.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
                select {
                case err := <-errCh:
                    if err != nil {
                        logCh <- model.LogEvent{
                            RunID:     runID,
                            StepID:    stepID,
                            Timestamp: time.Now(),
                            Level:     "error",
                            Message:   fmt.Sprintf("failed to get exit code: %v", err),
                            IsJSON:    false,
                        }
                    }
                case status := <-statusCh:
                    doneCh <- int(status.StatusCode)
                }

                return
            }

            // Parse line and forward to channel
            line := scanner.Text()
            if event := parseNDJSONLine(line, runID, stepID); event != nil {
                logCh <- *event
                idleTimer.Reset(60 * time.Second)
            }
        }
    }
}
```

**Error codes to add to pkg/errors:**
```go
const (
    ErrCodeContainerNotFound = "CONTAINER_NOT_FOUND"
    ErrCodeLogStreamFailed   = "LOG_STREAM_FAILED"
)
```

### Testing Requirements

**Unit tests (ndjson_parser_test.go):**
- Table-driven tests for parseNDJSONLine
- Test valid JSON with all fields (level, message, timestamp)
- Test valid JSON with missing level (defaults to "info")
- Test valid JSON with missing timestamp (defaults to time.Now())
- Test valid JSON with missing message (defaults to empty string)
- Test malformed JSON (IsJSON=false, RawLine set, Level="info")
- Test empty line (returns nil)
- Verify Data map populated for valid JSON
- No actual Docker client needed

**Unit tests (log_streamer_test.go):**
- Mock Docker client with ContainerLogs returning io.ReadCloser (pipe)
- Mock ContainerWait returning exit code
- Test StreamLogs with valid NDJSON log stream (verify LogEvent forwarding)
- Test StreamLogs with mixed valid/invalid JSON lines (verify IsJSON flag)
- Test StreamLogs idle timeout (mock scanner.Scan() delay, verify warning LogEvent after 60s)
- Test StreamLogs container exit (close pipe → verify exit code on done channel)
- Test StreamLogs context cancellation (verify channels closed, no exit code sent)
- Test error handling and DomainError wrapping (ContainerLogs error)
- Verify channels are closed properly (no goroutine leaks)
- No actual Docker daemon required

**Integration tests (optional, bonus):**
- Tag with `//go:build integration`
- Use real Docker SDK client
- Create container with alpine:latest: `echo '{"level":"info","message":"test"}' && sleep 1 && exit 0`
- StreamLogs and verify LogEvent received with IsJSON=true
- Verify exit code = 0 on done channel
- Test container with invalid JSON output (verify IsJSON=false)
- Clean up container in test teardown

### References
- Story 3-4: Docker container lifecycle manager (ContainerManager port)
- Story 3-6: Events table + pgxlisten (Wave 6, logs will be persisted via events)
- Architecture doc: `_bmad-output/planning-artifacts/architecture.md`
- Docker SDK docs: https://pkg.go.dev/github.com/docker/docker/client
- NDJSON format: http://ndjson.org/

## Dev Agent Record

(To be filled during implementation)

## Change Log

- 2026-02-17: Story created for Wave 5 backend infrastructure
