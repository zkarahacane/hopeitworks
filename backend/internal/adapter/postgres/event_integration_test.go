package postgres_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// testDB holds a shared test database context for integration tests.
type testDB struct {
	pool       *pgxpool.Pool
	connString string
	cleanup    func()
}

func setupTestDB(t *testing.T) *testDB {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	// Apply migrations
	applyMigrations(t, pool)

	return &testDB{
		pool:       pool,
		connString: connStr,
		cleanup: func() {
			pool.Close()
			if err := pgContainer.Terminate(ctx); err != nil {
				t.Logf("failed to terminate container: %v", err)
			}
		},
	}
}

func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	// Find migration files relative to the test file location
	migrationsDir := filepath.Join("..", "..", "..", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir %s: %v", migrationsDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		// Only apply up migrations
		if len(entry.Name()) > 4 && entry.Name()[len(entry.Name())-7:] != ".up.sql" {
			continue
		}

		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", entry.Name(), err)
		}

		_, err = pool.Exec(ctx, string(sqlBytes))
		if err != nil {
			t.Fatalf("failed to apply migration %s: %v", entry.Name(), err)
		}
	}
}

// createTestProject inserts a minimal project row needed for FK constraints.
func createTestProject(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	projectID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO projects (id, name, git_provider, agent_runtime)
		 VALUES ($1, $2, $3, $4)`,
		projectID, "test-project-"+projectID.String()[:8], "github", "docker",
	)
	if err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}
	return projectID
}

func TestIntegration_EventPublisher_Publish(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	queries := postgres.New(db.pool)
	publisher := postgres.NewEventRepo(queries)

	projectID := createTestProject(t, db.pool)

	t.Run("success inserts event with correct fields", func(t *testing.T) {
		entityID := uuid.New()
		payload := json.RawMessage(`{"step_count": 5}`)

		event := model.Event{
			ProjectID:  projectID,
			EntityType: "run",
			EntityID:   entityID,
			Action:     "started",
			Payload:    payload,
		}

		err := publisher.Publish(ctx, event)
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}

		// Verify the row exists in the database
		var count int
		err = db.pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM events WHERE entity_type = $1 AND entity_id = $2 AND action = $3",
			"run", entityID, "started",
		).Scan(&count)
		if err != nil {
			t.Fatalf("failed to query events: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 event row, got %d", count)
		}
	})

	t.Run("returns error for missing project FK", func(t *testing.T) {
		event := model.Event{
			ProjectID:  uuid.New(), // non-existent project
			EntityType: "run",
			EntityID:   uuid.New(),
			Action:     "started",
		}

		err := publisher.Publish(ctx, event)
		if err == nil {
			t.Fatal("expected error for missing project FK, got nil")
		}
	})

	t.Run("event format follows dot-notation convention", func(t *testing.T) {
		events := []struct {
			entityType string
			action     string
			wantName   string
		}{
			{"run", "started", "run.started"},
			{"step", "completed", "step.completed"},
			{"hitl", "pending", "hitl.pending"},
		}

		for _, tt := range events {
			event := model.Event{
				ProjectID:  projectID,
				EntityType: tt.entityType,
				EntityID:   uuid.New(),
				Action:     tt.action,
			}

			if event.EventName() != tt.wantName {
				t.Errorf("EventName() = %q, want %q", event.EventName(), tt.wantName)
			}

			err := publisher.Publish(ctx, event)
			if err != nil {
				t.Fatalf("Publish() error = %v", err)
			}
		}
	})
}

func TestIntegration_EventBus_SubscribeAndPublish(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	queries := postgres.New(db.pool)
	publisher := postgres.NewEventRepo(queries)

	bus, err := postgres.NewEventBus(ctx, db.connString, logger)
	if err != nil {
		t.Fatalf("failed to create event bus: %v", err)
	}
	defer func() { _ = bus.Close() }()

	projectID := createTestProject(t, db.pool)

	t.Run("subscriber receives event via LISTEN/NOTIFY", func(t *testing.T) {
		eventChan, cleanup, err := bus.Subscribe(ctx, projectID)
		if err != nil {
			t.Fatalf("Subscribe() error = %v", err)
		}
		defer cleanup()

		// Give the listener goroutine time to start
		time.Sleep(100 * time.Millisecond)

		entityID := uuid.New()
		event := model.Event{
			ProjectID:  projectID,
			EntityType: "run",
			EntityID:   entityID,
			Action:     "started",
			Payload:    json.RawMessage(`{"step_count": 3}`),
		}

		err = publisher.Publish(ctx, event)
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}

		select {
		case received := <-eventChan:
			if received.EntityType != "run" {
				t.Errorf("expected entity_type 'run', got %q", received.EntityType)
			}
			if received.Action != "started" {
				t.Errorf("expected action 'started', got %q", received.Action)
			}
			if received.EntityID != entityID {
				t.Errorf("expected entity_id %v, got %v", entityID, received.EntityID)
			}
			if received.ProjectID != projectID {
				t.Errorf("expected project_id %v, got %v", projectID, received.ProjectID)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for event")
		}
	})

	t.Run("project filtering: only receives events for subscribed project", func(t *testing.T) {
		otherProjectID := createTestProject(t, db.pool)

		bus2, err := postgres.NewEventBus(ctx, db.connString, logger)
		if err != nil {
			t.Fatalf("failed to create second event bus: %v", err)
		}
		defer func() { _ = bus2.Close() }()

		eventChan, cleanup, err := bus2.Subscribe(ctx, projectID)
		if err != nil {
			t.Fatalf("Subscribe() error = %v", err)
		}
		defer cleanup()

		time.Sleep(100 * time.Millisecond)

		// Publish event to other project
		err = publisher.Publish(ctx, model.Event{
			ProjectID:  otherProjectID,
			EntityType: "run",
			EntityID:   uuid.New(),
			Action:     "started",
		})
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}

		// Publish event to subscribed project
		targetEntityID := uuid.New()
		err = publisher.Publish(ctx, model.Event{
			ProjectID:  projectID,
			EntityType: "step",
			EntityID:   targetEntityID,
			Action:     "completed",
		})
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}

		select {
		case received := <-eventChan:
			if received.ProjectID != projectID {
				t.Errorf("expected project_id %v, got %v", projectID, received.ProjectID)
			}
			if received.EntityID != targetEntityID {
				t.Errorf("expected entity_id %v, got %v", targetEntityID, received.EntityID)
			}
			if received.EntityType != "step" {
				t.Errorf("expected entity_type 'step', got %q", received.EntityType)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for event")
		}
	})

	t.Run("cleanup function stops event delivery", func(t *testing.T) {
		bus3, err := postgres.NewEventBus(ctx, db.connString, logger)
		if err != nil {
			t.Fatalf("failed to create event bus: %v", err)
		}
		defer func() { _ = bus3.Close() }()

		eventChan, cleanup, err := bus3.Subscribe(ctx, projectID)
		if err != nil {
			t.Fatalf("Subscribe() error = %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		// Call cleanup to unsubscribe
		cleanup()

		// After cleanup, the channel should be closed
		_, ok := <-eventChan
		if ok {
			t.Error("expected channel to be closed after cleanup")
		}
	})
}

func TestIntegration_EventBus_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	bus, err := postgres.NewEventBus(ctx, db.connString, logger)
	if err != nil {
		t.Fatalf("failed to create event bus: %v", err)
	}

	projectID := createTestProject(t, db.pool)

	eventChan, _, err := bus.Subscribe(ctx, projectID)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	// Close should shut down all subscriptions
	err = bus.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Channel should be closed
	_, ok := <-eventChan
	if ok {
		t.Error("expected channel to be closed after bus.Close()")
	}

	// Second close should be no-op
	err = bus.Close()
	if err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestIntegration_EventsTable_AppendOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	projectID := createTestProject(t, db.pool)

	// Insert an event
	eventID := uuid.New()
	_, err := db.pool.Exec(ctx,
		`INSERT INTO events (id, project_id, entity_type, entity_id, action)
		 VALUES ($1, $2, $3, $4, $5)`,
		eventID, projectID, "run", uuid.New(), "started",
	)
	if err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	t.Run("UPDATE is prevented by trigger", func(t *testing.T) {
		_, err := db.pool.Exec(ctx,
			"UPDATE events SET action = 'completed' WHERE id = $1",
			eventID,
		)
		if err == nil {
			t.Fatal("expected error on UPDATE, got nil")
		}
	})

	t.Run("DELETE is prevented by trigger", func(t *testing.T) {
		_, err := db.pool.Exec(ctx,
			"DELETE FROM events WHERE id = $1",
			eventID,
		)
		if err == nil {
			t.Fatal("expected error on DELETE, got nil")
		}
	})
}

func TestIntegration_NotifyTrigger(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	projectID := createTestProject(t, db.pool)

	// Create a separate connection to LISTEN
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("failed to acquire connection: %v", err)
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, "LISTEN events")
	if err != nil {
		t.Fatalf("failed to LISTEN: %v", err)
	}

	// Insert event using a different connection
	entityID := uuid.New()
	eventID := uuid.New()
	_, err = db.pool.Exec(ctx,
		`INSERT INTO events (id, project_id, entity_type, entity_id, action)
		 VALUES ($1, $2, $3, $4, $5)`,
		eventID, projectID, "run", entityID, "started",
	)
	if err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	// Wait for notification
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	notification, err := conn.Conn().WaitForNotification(waitCtx)
	if err != nil {
		t.Fatalf("failed to receive notification: %v", err)
	}

	if notification.Channel != "events" {
		t.Errorf("expected channel 'events', got %q", notification.Channel)
	}

	// Verify notification payload contains expected fields
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
		t.Fatalf("failed to parse notification payload: %v", err)
	}

	if payload["entity_type"] != "run" {
		t.Errorf("expected entity_type 'run', got %v", payload["entity_type"])
	}
	if payload["action"] != "started" {
		t.Errorf("expected action 'started', got %v", payload["action"])
	}
	if fmt.Sprint(payload["id"]) != eventID.String() {
		t.Errorf("expected id %v, got %v", eventID, payload["id"])
	}
}

func TestIntegration_ListEventsByProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.cleanup()

	ctx := context.Background()
	queries := postgres.New(db.pool)
	publisher := postgres.NewEventRepo(queries)

	projectID := createTestProject(t, db.pool)
	otherProjectID := createTestProject(t, db.pool)

	// Publish several events to different projects
	for i := 0; i < 5; i++ {
		err := publisher.Publish(ctx, model.Event{
			ProjectID:  projectID,
			EntityType: "run",
			EntityID:   uuid.New(),
			Action:     fmt.Sprintf("action_%d", i),
		})
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	}
	err := publisher.Publish(ctx, model.Event{
		ProjectID:  otherProjectID,
		EntityType: "run",
		EntityID:   uuid.New(),
		Action:     "other",
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	// Query events for the target project
	events, err := queries.ListEventsByProject(ctx, postgres.ListEventsByProjectParams{
		ProjectID: projectID,
		Limit:     20,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("ListEventsByProject() error = %v", err)
	}

	if len(events) != 5 {
		t.Errorf("expected 5 events, got %d", len(events))
	}

	// Verify all events belong to the target project
	for _, e := range events {
		if e.ProjectID != projectID {
			t.Errorf("expected project_id %v, got %v", projectID, e.ProjectID)
		}
	}
}
