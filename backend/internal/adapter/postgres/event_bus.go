package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Ensure EventBus implements port.EventSubscriber at compile time.
var _ port.EventSubscriber = (*EventBus)(nil)

const (
	eventsChannel       = "events"
	notifyBufferSize    = 100
	maxReconnectRetries = 5
	baseReconnectDelay  = 1 * time.Second
	waitTimeout         = 5 * time.Second
)

// EventBus implements port.EventSubscriber using pgx LISTEN/NOTIFY.
// It maintains a dedicated Postgres connection (separate from the pool)
// for receiving notifications.
type EventBus struct {
	connString string
	conn       *pgx.Conn
	logger     *slog.Logger

	mu          sync.Mutex
	subscribers map[uuid.UUID][]chan<- model.Event
	listening   bool
	stopCh      chan struct{}
	doneCh      chan struct{} // Closed when listenLoop exits
	closed      bool
}

// NewEventBus creates a new EventBus with a dedicated pgx connection for LISTEN.
func NewEventBus(ctx context.Context, connString string, logger *slog.Logger) (*EventBus, error) {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("creating event bus connection: %w", err)
	}

	bus := &EventBus{
		connString:  connString,
		conn:        conn,
		logger:      logger,
		subscribers: make(map[uuid.UUID][]chan<- model.Event),
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}

	return bus, nil
}

// Subscribe returns a channel of events for the given project and a cleanup function.
// The subscriber will only receive events matching the specified projectID.
func (b *EventBus) Subscribe(ctx context.Context, projectID uuid.UUID) (<-chan model.Event, func(), error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, nil, fmt.Errorf("event bus is closed")
	}

	// Start the LISTEN loop if not already running
	if !b.listening {
		if err := b.startListening(ctx); err != nil {
			return nil, nil, fmt.Errorf("starting event listener: %w", err)
		}
	}

	eventChan := make(chan model.Event, notifyBufferSize)
	b.subscribers[projectID] = append(b.subscribers[projectID], eventChan)

	cleanup := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.removeSubscriber(projectID, eventChan)
		close(eventChan)
	}

	return eventChan, cleanup, nil
}

// Close gracefully shuts down all subscriptions and the Postgres connection.
func (b *EventBus) Close() error {
	b.mu.Lock()

	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true

	// Signal the listener goroutine to stop
	close(b.stopCh)

	// Wait for listenLoop to finish if it was started
	wasListening := b.listening
	b.mu.Unlock()

	if wasListening {
		// Wait for listenLoop goroutine to exit before closing the connection
		<-b.doneCh
	}

	// Now safe to close the connection (no goroutine is using it)
	b.mu.Lock()
	conn := b.conn
	b.mu.Unlock()

	// Close all subscriber channels
	b.mu.Lock()
	for projectID, chans := range b.subscribers {
		for _, ch := range chans {
			close(ch)
		}
		delete(b.subscribers, projectID)
	}
	b.mu.Unlock()

	// Close the Postgres connection
	if conn != nil {
		return conn.Close(context.Background())
	}
	return nil
}

// startListening begins listening on the events channel and starts the
// notification dispatch goroutine. Must be called with b.mu held.
func (b *EventBus) startListening(ctx context.Context) error {
	_, err := b.conn.Exec(ctx, "LISTEN "+eventsChannel)
	if err != nil {
		return fmt.Errorf("executing LISTEN: %w", err)
	}

	b.listening = true
	go b.listenLoop()
	return nil
}

// listenLoop waits for notifications and dispatches them to subscribers.
func (b *EventBus) listenLoop() {
	defer close(b.doneCh) // Signal that we've exited

	for {
		select {
		case <-b.stopCh:
			return
		default:
		}

		// Get connection safely
		b.mu.Lock()
		conn := b.conn
		b.mu.Unlock()

		if conn == nil {
			return
		}

		// Use a timeout context so we periodically check for stop signal
		waitCtx, cancel := context.WithTimeout(context.Background(), waitTimeout)
		notification, err := conn.WaitForNotification(waitCtx)
		cancel()

		if err != nil {
			// Check if we're shutting down
			select {
			case <-b.stopCh:
				return
			default:
			}

			// Context deadline exceeded is normal (timeout), just retry
			if waitCtx.Err() != nil {
				continue
			}

			// Connection error: attempt reconnection
			b.logger.Error("notification error, attempting reconnect", "error", err)
			if reconnErr := b.reconnect(); reconnErr != nil {
				b.logger.Error("reconnection failed, stopping listener", "error", reconnErr)
				return
			}
			continue
		}

		b.handleNotification(notification)
	}
}

// handleNotification parses a notification payload and dispatches to matching subscribers.
func (b *EventBus) handleNotification(notification *pgconn.Notification) {
	var notif notificationPayload
	if err := json.Unmarshal([]byte(notification.Payload), &notif); err != nil {
		b.logger.Error("failed to parse notification payload", "error", err, "payload", notification.Payload)
		return
	}

	event := model.Event{
		ID:         notif.ID,
		ProjectID:  notif.ProjectID,
		EntityType: notif.EntityType,
		EntityID:   notif.EntityID,
		Action:     notif.Action,
	}

	b.mu.Lock()
	chans := b.subscribers[notif.ProjectID]
	// Copy the slice to avoid holding the lock during sends
	chansCopy := make([]chan<- model.Event, len(chans))
	copy(chansCopy, chans)
	b.mu.Unlock()

	for _, ch := range chansCopy {
		select {
		case ch <- event:
		default:
			b.logger.Warn("subscriber channel full, dropping event",
				"event_id", event.ID,
				"project_id", event.ProjectID,
			)
		}
	}
}

// reconnect attempts to re-establish the Postgres connection with exponential backoff.
func (b *EventBus) reconnect() error {
	delay := baseReconnectDelay

	for attempt := 1; attempt <= maxReconnectRetries; attempt++ {
		select {
		case <-b.stopCh:
			return fmt.Errorf("event bus closing during reconnect")
		default:
		}

		b.logger.Info("attempting reconnection", "attempt", attempt, "delay", delay)
		time.Sleep(delay)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		conn, err := pgx.Connect(ctx, b.connString)
		cancel()

		if err != nil {
			b.logger.Error("reconnection attempt failed", "attempt", attempt, "error", err)
			delay *= 2 // exponential backoff
			continue
		}

		// Re-issue LISTEN on the new connection
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		_, err = conn.Exec(ctx, "LISTEN "+eventsChannel)
		cancel()

		if err != nil {
			b.logger.Error("failed to LISTEN after reconnect", "attempt", attempt, "error", err)
			_ = conn.Close(context.Background())
			delay *= 2
			continue
		}

		// Swap connection
		b.mu.Lock()
		oldConn := b.conn
		b.conn = conn
		b.mu.Unlock()

		if oldConn != nil {
			_ = oldConn.Close(context.Background())
		}

		b.logger.Info("reconnected successfully", "attempt", attempt)
		return nil
	}

	return fmt.Errorf("exhausted %d reconnection attempts", maxReconnectRetries)
}

// removeSubscriber removes a specific channel from the subscribers map.
func (b *EventBus) removeSubscriber(projectID uuid.UUID, ch chan<- model.Event) {
	chans := b.subscribers[projectID]
	for i, c := range chans {
		if c == ch {
			b.subscribers[projectID] = append(chans[:i], chans[i+1:]...)
			break
		}
	}
	if len(b.subscribers[projectID]) == 0 {
		delete(b.subscribers, projectID)
	}
}

// notificationPayload represents the JSON payload from the Postgres NOTIFY trigger.
type notificationPayload struct {
	ID         uuid.UUID `json:"id"`
	ProjectID  uuid.UUID `json:"project_id"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Action     string    `json:"action"`
}
