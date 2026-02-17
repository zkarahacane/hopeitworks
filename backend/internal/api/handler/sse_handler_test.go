package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// --- Mock EventSubscriber ---

type mockEventSubscriber struct {
	mu       sync.Mutex
	channels map[uuid.UUID]chan model.Event
	subErr   error
}

func newMockEventSubscriber() *mockEventSubscriber {
	return &mockEventSubscriber{
		channels: make(map[uuid.UUID]chan model.Event),
	}
}

func (m *mockEventSubscriber) Subscribe(_ context.Context, projectID uuid.UUID) (<-chan model.Event, func(), error) {
	if m.subErr != nil {
		return nil, nil, m.subErr
	}
	m.mu.Lock()
	ch := make(chan model.Event, 100)
	m.channels[projectID] = ch
	m.mu.Unlock()

	cleanup := func() {
		m.mu.Lock()
		delete(m.channels, projectID)
		m.mu.Unlock()
	}
	return ch, cleanup, nil
}

func (m *mockEventSubscriber) Close() error {
	return nil
}

func (m *mockEventSubscriber) send(projectID uuid.UUID, event model.Event) {
	m.mu.Lock()
	ch, ok := m.channels[projectID]
	m.mu.Unlock()
	if ok {
		ch <- event
	}
}

func (m *mockEventSubscriber) closeChan(projectID uuid.UUID) {
	m.mu.Lock()
	ch, ok := m.channels[projectID]
	m.mu.Unlock()
	if ok {
		close(ch)
	}
}

// --- Mock EventRepository ---

type mockEventRepository struct {
	events []*model.Event
	err    error
}

func (m *mockEventRepository) GetEventsSince(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]*model.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

// --- Mock ProjectUserRepository for SSE ---

type mockProjectUserRepoForSSE struct {
	members map[string]bool // key: "projectID:userID"
	err     error
}

func newMockProjectUserRepoForSSE() *mockProjectUserRepoForSSE {
	return &mockProjectUserRepoForSSE{
		members: make(map[string]bool),
	}
}

func (m *mockProjectUserRepoForSSE) key(projectID, userID uuid.UUID) string {
	return projectID.String() + ":" + userID.String()
}

func (m *mockProjectUserRepoForSSE) AddUser(_ context.Context, projectID, userID uuid.UUID, role model.ProjectRole) (*model.ProjectUser, error) {
	m.members[m.key(projectID, userID)] = true
	return &model.ProjectUser{ProjectID: projectID, UserID: userID, Role: role}, nil
}

func (m *mockProjectUserRepoForSSE) RemoveUser(_ context.Context, projectID, userID uuid.UUID) error {
	delete(m.members, m.key(projectID, userID))
	return nil
}

func (m *mockProjectUserRepoForSSE) ListMembers(_ context.Context, _ uuid.UUID) ([]*model.ProjectMember, error) {
	return nil, nil
}

func (m *mockProjectUserRepoForSSE) IsUserInProject(_ context.Context, projectID, userID uuid.UUID) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.members[m.key(projectID, userID)], nil
}

func (m *mockProjectUserRepoForSSE) ListProjectsByUser(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Project, error) {
	return nil, nil
}

func (m *mockProjectUserRepoForSSE) CountProjectsByUser(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}

// --- Test helpers ---

func newTestSSEHandler(sub *mockEventSubscriber, repo *mockEventRepository, puRepo *mockProjectUserRepoForSSE) *SSEHandler {
	logger := slog.Default()
	return NewSSEHandler(sub, repo, puRepo, logger)
}

func makeAuthenticatedRequest(method, url string, userID uuid.UUID, role model.Role) *http.Request {
	req := httptest.NewRequest(method, url, nil)
	ctx := middleware.SetUserContext(req.Context(), userID, role)
	return req.WithContext(ctx)
}

// --- Tests ---

func TestSSEHandler_MissingProjectID(t *testing.T) {
	sub := newMockEventSubscriber()
	repo := &mockEventRepository{}
	puRepo := newMockProjectUserRepoForSSE()
	h := newTestSSEHandler(sub, repo, puRepo)

	userID := uuid.New()
	req := makeAuthenticatedRequest(http.MethodGet, "/api/v1/events/stream", userID, model.RoleUser)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var errResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	errObj, ok := errResp["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["code"] != "VALIDATION_ERROR" {
		t.Errorf("expected error code VALIDATION_ERROR, got %v", errObj["code"])
	}
}

func TestSSEHandler_InvalidProjectID(t *testing.T) {
	sub := newMockEventSubscriber()
	repo := &mockEventRepository{}
	puRepo := newMockProjectUserRepoForSSE()
	h := newTestSSEHandler(sub, repo, puRepo)

	userID := uuid.New()
	req := makeAuthenticatedRequest(http.MethodGet, "/api/v1/events/stream?project_id=not-a-uuid", userID, model.RoleUser)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSSEHandler_Unauthenticated(t *testing.T) {
	sub := newMockEventSubscriber()
	repo := &mockEventRepository{}
	puRepo := newMockProjectUserRepoForSSE()
	h := newTestSSEHandler(sub, repo, puRepo)

	projectID := uuid.New()
	// No user in context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream?project_id="+projectID.String(), nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestSSEHandler_NonMember(t *testing.T) {
	sub := newMockEventSubscriber()
	repo := &mockEventRepository{}
	puRepo := newMockProjectUserRepoForSSE()
	h := newTestSSEHandler(sub, repo, puRepo)

	projectID := uuid.New()
	userID := uuid.New()
	// User is NOT a member of the project

	req := makeAuthenticatedRequest(http.MethodGet, "/api/v1/events/stream?project_id="+projectID.String(), userID, model.RoleUser)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d. Body: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

func TestSSEHandler_ReceivesLiveEvent(t *testing.T) {
	sub := newMockEventSubscriber()
	repo := &mockEventRepository{}
	puRepo := newMockProjectUserRepoForSSE()
	h := newTestSSEHandler(sub, repo, puRepo)

	projectID := uuid.New()
	userID := uuid.New()
	puRepo.members[puRepo.key(projectID, userID)] = true

	req := makeAuthenticatedRequest(http.MethodGet, "/api/v1/events/stream?project_id="+projectID.String(), userID, model.RoleUser)

	// Use a cancellable context so we can stop the handler
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	eventID := uuid.New()
	entityID := uuid.New()
	testEvent := model.Event{
		ID:         eventID,
		ProjectID:  projectID,
		EntityType: "run",
		EntityID:   entityID,
		Action:     "started",
		Payload:    json.RawMessage(`{"key":"value"}`),
		CreatedAt:  time.Now().UTC(),
	}

	// Run handler in a goroutine
	done := make(chan struct{})
	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	// Wait for subscription to be established
	waitForSubscription(t, sub, projectID)

	// Send an event then close the channel so the handler processes it and exits naturally.
	sub.send(projectID, testEvent)
	sub.closeChan(projectID)

	// Wait for handler to finish (channel closed causes it to return)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not exit after channel closed")
	}
	cancel() // release context resources

	body := rec.Body.String()

	// Verify SSE headers
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %s", cc)
	}
	if xab := rec.Header().Get("X-Accel-Buffering"); xab != "no" {
		t.Errorf("expected X-Accel-Buffering no, got %s", xab)
	}

	// Verify SSE frame contains correct event
	if !strings.Contains(body, "event: run.started") {
		t.Errorf("expected body to contain 'event: run.started', got: %s", body)
	}
	if !strings.Contains(body, "id: "+eventID.String()) {
		t.Errorf("expected body to contain event ID, got: %s", body)
	}
	if !strings.Contains(body, `"key":"value"`) {
		t.Errorf("expected body to contain payload, got: %s", body)
	}
}

func TestSSEHandler_ReplayEvents(t *testing.T) {
	sub := newMockEventSubscriber()
	puRepo := newMockProjectUserRepoForSSE()

	projectID := uuid.New()
	userID := uuid.New()
	puRepo.members[puRepo.key(projectID, userID)] = true

	// Create two replay events
	event1 := &model.Event{
		ID:         uuid.New(),
		ProjectID:  projectID,
		EntityType: "step",
		EntityID:   uuid.New(),
		Action:     "completed",
		Payload:    json.RawMessage(`{"step":1}`),
		CreatedAt:  time.Now().UTC().Add(-2 * time.Second),
	}
	event2 := &model.Event{
		ID:         uuid.New(),
		ProjectID:  projectID,
		EntityType: "run",
		EntityID:   uuid.New(),
		Action:     "failed",
		Payload:    json.RawMessage(`{"step":2}`),
		CreatedAt:  time.Now().UTC().Add(-1 * time.Second),
	}
	repo := &mockEventRepository{events: []*model.Event{event1, event2}}

	h := newTestSSEHandler(sub, repo, puRepo)

	lastEventID := uuid.New()
	req := makeAuthenticatedRequest(http.MethodGet, "/api/v1/events/stream?project_id="+projectID.String(), userID, model.RoleUser)
	req.Header.Set("Last-Event-ID", lastEventID.String())

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	// Wait for subscription to be established
	waitForSubscription(t, sub, projectID)

	// Cancel context to stop the handler
	cancel()
	<-done

	body := rec.Body.String()

	// Both replay events should appear before any live events
	if !strings.Contains(body, "event: step.completed") {
		t.Errorf("expected replay event step.completed in body, got: %s", body)
	}
	if !strings.Contains(body, "event: run.failed") {
		t.Errorf("expected replay event run.failed in body, got: %s", body)
	}
	if !strings.Contains(body, "id: "+event1.ID.String()) {
		t.Errorf("expected event1 ID in body, got: %s", body)
	}
	if !strings.Contains(body, "id: "+event2.ID.String()) {
		t.Errorf("expected event2 ID in body, got: %s", body)
	}

	// Verify replay order: step.completed appears before run.failed
	idx1 := strings.Index(body, "event: step.completed")
	idx2 := strings.Index(body, "event: run.failed")
	if idx1 >= idx2 {
		t.Errorf("expected step.completed before run.failed in replay, body: %s", body)
	}
}

func TestSSEHandler_AdminBypassesProjectCheck(t *testing.T) {
	sub := newMockEventSubscriber()
	repo := &mockEventRepository{}
	puRepo := newMockProjectUserRepoForSSE()
	h := newTestSSEHandler(sub, repo, puRepo)

	projectID := uuid.New()
	userID := uuid.New()
	// User is NOT a member, but is admin
	req := makeAuthenticatedRequest(http.MethodGet, "/api/v1/events/stream?project_id="+projectID.String(), userID, model.RoleAdmin)

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	// Wait for subscription to be established
	waitForSubscription(t, sub, projectID)

	cancel()
	<-done

	// Should NOT be 403 — admin bypasses membership check
	if rec.Code == http.StatusForbidden {
		t.Errorf("admin should bypass project membership check, got 403")
	}
}

func TestSSEHandler_ChannelClosed(t *testing.T) {
	sub := newMockEventSubscriber()
	repo := &mockEventRepository{}
	puRepo := newMockProjectUserRepoForSSE()
	h := newTestSSEHandler(sub, repo, puRepo)

	projectID := uuid.New()
	userID := uuid.New()
	puRepo.members[puRepo.key(projectID, userID)] = true

	req := makeAuthenticatedRequest(http.MethodGet, "/api/v1/events/stream?project_id="+projectID.String(), userID, model.RoleUser)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		h.ServeHTTP(rec, req)
		close(done)
	}()

	// Wait for subscription to be established
	waitForSubscription(t, sub, projectID)

	// Close the channel to simulate EventBus shutdown
	sub.closeChan(projectID)

	// Handler should exit gracefully
	select {
	case <-done:
		// Good — handler returned
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not exit after channel closed")
	}
}

// waitForSubscription polls until the mock subscriber has a channel for the project.
func waitForSubscription(t *testing.T, sub *mockEventSubscriber, projectID uuid.UUID) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		sub.mu.Lock()
		_, ok := sub.channels[projectID]
		sub.mu.Unlock()
		if ok {
			return
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for subscription to be established")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}
