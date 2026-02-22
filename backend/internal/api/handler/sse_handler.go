package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	authmw "github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

const keepaliveInterval = 30 * time.Second

// SSEHandler serves Server-Sent Events for real-time project event streaming.
// This is a long-lived HTTP connection — it must NOT go through the oapi-codegen
// generated mux. Register manually on the chi router.
//
// The handler uses http.ResponseController.SetWriteDeadline to disable the
// server's WriteTimeout for SSE connections, allowing them to stay open
// indefinitely. Connection lifetime is managed by client disconnect detection
// and keepalive heartbeats.
type SSEHandler struct {
	eventSub        port.EventSubscriber
	eventRepo       port.EventRepository
	projectUserRepo port.ProjectUserRepository
	logger          *slog.Logger
}

// NewSSEHandler creates a new SSEHandler with all required dependencies.
func NewSSEHandler(
	eventSub port.EventSubscriber,
	eventRepo port.EventRepository,
	projectUserRepo port.ProjectUserRepository,
	logger *slog.Logger,
) *SSEHandler {
	return &SSEHandler{
		eventSub:        eventSub,
		eventRepo:       eventRepo,
		projectUserRepo: projectUserRepo,
		logger:          logger,
	}
}

// ServeHTTP handles SSE streaming requests.
// Route: GET /api/v1/events/stream?project_id={uuid}
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse project_id query param
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		writeErrorResponse(w, errors.NewValidation("project_id", "missing required query parameter"))
		return
	}
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		writeErrorResponse(w, errors.NewValidation("project_id", "must be a valid UUID"))
		return
	}

	// Extract authenticated user from context
	userID, ok := authmw.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, errors.NewUnauthorized("authentication required"))
		return
	}

	// Check project membership (admins bypass via middleware, but SSE is registered
	// without RequireProjectAccess, so we check inline)
	if !authmw.IsAdmin(r.Context()) {
		isMember, memberErr := h.projectUserRepo.IsUserInProject(r.Context(), projectID, userID)
		if memberErr != nil {
			h.logger.Error("failed to check project membership", "error", memberErr, "project_id", projectID, "user_id", userID)
			writeErrorResponse(w, errors.NewInternal("check project membership", memberErr))
			return
		}
		if !isMember {
			writeErrorResponse(w, errors.NewForbidden("you are not a member of this project"))
			return
		}
	}

	// Assert http.Flusher support
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Error("response writer does not support http.Flusher")
		writeErrorResponse(w, errors.NewInternal("SSE streaming not supported", fmt.Errorf("http.Flusher not available")))
		return
	}

	// Disable the server's WriteTimeout for this long-lived SSE connection.
	// Without this, Go's http.Server.WriteTimeout (default 15s) would kill
	// the SSE stream before the first keepalive (30s) fires.
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		h.logger.Warn("failed to clear write deadline for SSE connection", "error", err)
	}

	// Set SSE response headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Flush headers immediately so the browser's EventSource transitions
	// from CONNECTING to OPEN without waiting for the first keepalive.
	flusher.Flush()

	// Replay missed events if Last-Event-ID is present
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		afterID, parseErr := uuid.Parse(lastEventID)
		if parseErr == nil {
			missedEvents, replayErr := h.eventRepo.GetEventsSince(r.Context(), projectID, afterID)
			if replayErr != nil {
				h.logger.Error("failed to replay events", "error", replayErr, "last_event_id", lastEventID)
			} else {
				for _, event := range missedEvents {
					if writeErr := writeSSEEvent(w, flusher, *event); writeErr != nil {
						h.logger.Error("failed to write replay event", "error", writeErr)
						return
					}
				}
			}
		}
	}

	// Subscribe to live events
	eventCh, cleanup, subErr := h.eventSub.Subscribe(r.Context(), projectID)
	if subErr != nil {
		h.logger.Error("failed to subscribe to events", "error", subErr, "project_id", projectID)
		writeErrorResponse(w, errors.NewInternal("subscribe to events", subErr))
		return
	}
	defer cleanup()

	h.logger.Info("SSE client connected", "project_id", projectID, "user_id", userID)

	// Streaming loop
	ticker := time.NewTicker(keepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				// Channel closed, end stream
				return
			}
			if writeErr := writeSSEEvent(w, flusher, event); writeErr != nil {
				h.logger.Error("failed to write SSE event", "error", writeErr)
				return
			}

		case <-ticker.C:
			// Send keepalive comment to prevent proxy timeout
			if _, writeErr := fmt.Fprint(w, ": keepalive\n\n"); writeErr != nil {
				h.logger.Debug("keepalive write failed, client likely disconnected", "error", writeErr)
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			h.logger.Info("SSE client disconnected", "project_id", projectID, "user_id", userID)
			return
		}
	}
}

// writeSSEEvent writes a single SSE frame and flushes the response.
func writeSSEEvent(w io.Writer, f http.Flusher, event model.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\nid: %s\n\n",
		event.EventName(), payload, event.ID)
	if err != nil {
		return fmt.Errorf("writing SSE frame: %w", err)
	}
	f.Flush()
	return nil
}
