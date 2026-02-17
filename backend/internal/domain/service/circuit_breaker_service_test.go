package service

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

const circuitBreakerOpenCode = "CIRCUIT_BREAKER_OPEN"

// cbMockProjectRepo is a mock implementation of port.ProjectRepository for circuit breaker tests.
type cbMockProjectRepo struct {
	mu       sync.Mutex
	projects map[uuid.UUID]*model.Project
}

func newCBMockProjectRepo() *cbMockProjectRepo {
	return &cbMockProjectRepo{
		projects: make(map[uuid.UUID]*model.Project),
	}
}

func (m *cbMockProjectRepo) Create(_ context.Context, project *model.Project) (*model.Project, error) {
	project.ID = uuid.New()
	m.projects[project.ID] = project
	return project, nil
}

func (m *cbMockProjectRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return nil, errors.NewNotFound("project", id)
	}
	cp := *p
	return &cp, nil
}

func (m *cbMockProjectRepo) List(_ context.Context, _, _ int32) ([]*model.Project, error) {
	return nil, nil
}

func (m *cbMockProjectRepo) Count(_ context.Context) (int64, error) {
	return int64(len(m.projects)), nil
}

func (m *cbMockProjectRepo) Update(_ context.Context, project *model.Project) (*model.Project, error) {
	m.projects[project.ID] = project
	return project, nil
}

func (m *cbMockProjectRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.projects, id)
	return nil
}

func (m *cbMockProjectRepo) IncrementCircuitBreakerCount(_ context.Context, id uuid.UUID) (*model.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[id]
	if !ok {
		return nil, errors.NewNotFound("project", id)
	}
	p.CircuitBreakerCount++
	if p.CircuitBreakerCount >= p.CircuitBreakerMax {
		p.CircuitBreakerActive = true
	}
	cp := *p
	return &cp, nil
}

func (m *cbMockProjectRepo) ResetCircuitBreaker(_ context.Context, id uuid.UUID) (*model.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[id]
	if !ok {
		return nil, errors.NewNotFound("project", id)
	}
	p.CircuitBreakerCount = 0
	p.CircuitBreakerActive = false
	cp := *p
	return &cp, nil
}

// cbMockEventPublisher is a mock for EventPublisher in circuit breaker tests.
type cbMockEventPublisher struct {
	mu     sync.Mutex
	events []model.Event
}

func newCBMockEventPublisher() *cbMockEventPublisher {
	return &cbMockEventPublisher{}
}

func (p *cbMockEventPublisher) Publish(_ context.Context, event model.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
	return nil
}

func (p *cbMockEventPublisher) getEvents() []model.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]model.Event, len(p.events))
	copy(result, p.events)
	return result
}

func newTestProject(id uuid.UUID, maxFailures int) *model.Project {
	return &model.Project{
		ID:                   id,
		Name:                 "test-project",
		CircuitBreakerCount:  0,
		CircuitBreakerActive: false,
		CircuitBreakerMax:    maxFailures,
	}
}

func TestCircuitBreakerService_CheckCircuitBreaker_Closed(t *testing.T) {
	repo := newCBMockProjectRepo()
	eventPub := newCBMockEventPublisher()
	svc := NewCircuitBreakerService(repo, eventPub, testLogger())

	projectID := uuid.New()
	repo.projects[projectID] = newTestProject(projectID, 3)

	err := svc.CheckCircuitBreaker(context.Background(), projectID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCircuitBreakerService_CheckCircuitBreaker_Open(t *testing.T) {
	repo := newCBMockProjectRepo()
	eventPub := newCBMockEventPublisher()
	svc := NewCircuitBreakerService(repo, eventPub, testLogger())

	projectID := uuid.New()
	p := newTestProject(projectID, 3)
	p.CircuitBreakerActive = true
	p.CircuitBreakerCount = 3
	repo.projects[projectID] = p

	err := svc.CheckCircuitBreaker(context.Background(), projectID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != circuitBreakerOpenCode {
		t.Errorf("expected code CIRCUIT_BREAKER_OPEN, got %s", domainErr.Code)
	}
}

func TestCircuitBreakerService_CheckCircuitBreaker_ProjectNotFound(t *testing.T) {
	repo := newCBMockProjectRepo()
	eventPub := newCBMockEventPublisher()
	svc := NewCircuitBreakerService(repo, eventPub, testLogger())

	err := svc.CheckCircuitBreaker(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %s", domainErr.Category)
	}
}

func TestCircuitBreakerService_RecordFailure_IncrementsBelowThreshold(t *testing.T) {
	repo := newCBMockProjectRepo()
	eventPub := newCBMockEventPublisher()
	svc := NewCircuitBreakerService(repo, eventPub, testLogger())

	projectID := uuid.New()
	repo.projects[projectID] = newTestProject(projectID, 3)

	err := svc.RecordFailure(context.Background(), projectID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.projects[projectID].CircuitBreakerCount != 1 {
		t.Errorf("expected count 1, got %d", repo.projects[projectID].CircuitBreakerCount)
	}
	if repo.projects[projectID].CircuitBreakerActive {
		t.Error("circuit breaker should not be active after 1 failure")
	}

	// No event should be published when below threshold
	events := eventPub.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestCircuitBreakerService_RecordFailure_TripsAtThreshold(t *testing.T) {
	repo := newCBMockProjectRepo()
	eventPub := newCBMockEventPublisher()
	svc := NewCircuitBreakerService(repo, eventPub, testLogger())

	projectID := uuid.New()
	p := newTestProject(projectID, 3)
	p.CircuitBreakerCount = 2 // one more failure will trip it
	repo.projects[projectID] = p

	err := svc.RecordFailure(context.Background(), projectID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.projects[projectID].CircuitBreakerCount != 3 {
		t.Errorf("expected count 3, got %d", repo.projects[projectID].CircuitBreakerCount)
	}
	if !repo.projects[projectID].CircuitBreakerActive {
		t.Error("circuit breaker should be active after reaching threshold")
	}

	// circuit_breaker.triggered event should be published
	events := eventPub.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventName() != "circuit_breaker.triggered" {
		t.Errorf("expected circuit_breaker.triggered event, got %s", events[0].EventName())
	}
}

func TestCircuitBreakerService_RecordSuccess_ResetsCount(t *testing.T) {
	repo := newCBMockProjectRepo()
	eventPub := newCBMockEventPublisher()
	svc := NewCircuitBreakerService(repo, eventPub, testLogger())

	projectID := uuid.New()
	p := newTestProject(projectID, 3)
	p.CircuitBreakerCount = 2 // some failures recorded
	repo.projects[projectID] = p

	err := svc.RecordSuccess(context.Background(), projectID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.projects[projectID].CircuitBreakerCount != 0 {
		t.Errorf("expected count 0, got %d", repo.projects[projectID].CircuitBreakerCount)
	}
	if repo.projects[projectID].CircuitBreakerActive {
		t.Error("circuit breaker should not be active after success")
	}
}

func TestCircuitBreakerService_RecordSuccess_NoopWhenCountZero(t *testing.T) {
	repo := newCBMockProjectRepo()
	eventPub := newCBMockEventPublisher()
	svc := NewCircuitBreakerService(repo, eventPub, testLogger())

	projectID := uuid.New()
	repo.projects[projectID] = newTestProject(projectID, 3)

	err := svc.RecordSuccess(context.Background(), projectID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Count should still be zero (no reset call needed)
	if repo.projects[projectID].CircuitBreakerCount != 0 {
		t.Errorf("expected count 0, got %d", repo.projects[projectID].CircuitBreakerCount)
	}
}

func TestCircuitBreakerService_Reset(t *testing.T) {
	repo := newCBMockProjectRepo()
	eventPub := newCBMockEventPublisher()
	svc := NewCircuitBreakerService(repo, eventPub, testLogger())

	projectID := uuid.New()
	p := newTestProject(projectID, 3)
	p.CircuitBreakerCount = 3
	p.CircuitBreakerActive = true
	repo.projects[projectID] = p

	err := svc.Reset(context.Background(), projectID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.projects[projectID].CircuitBreakerCount != 0 {
		t.Errorf("expected count 0, got %d", repo.projects[projectID].CircuitBreakerCount)
	}
	if repo.projects[projectID].CircuitBreakerActive {
		t.Error("circuit breaker should not be active after reset")
	}

	// circuit_breaker.reset event should be published
	events := eventPub.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventName() != "circuit_breaker.reset" {
		t.Errorf("expected circuit_breaker.reset event, got %s", events[0].EventName())
	}
}

func TestCircuitBreakerService_Reset_NoopWhenAlreadyClear(t *testing.T) {
	repo := newCBMockProjectRepo()
	eventPub := newCBMockEventPublisher()
	svc := NewCircuitBreakerService(repo, eventPub, testLogger())

	projectID := uuid.New()
	repo.projects[projectID] = newTestProject(projectID, 3)

	err := svc.Reset(context.Background(), projectID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// No events should be published since there was nothing to reset
	events := eventPub.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestCircuitBreakerService_Reset_ProjectNotFound(t *testing.T) {
	repo := newCBMockProjectRepo()
	eventPub := newCBMockEventPublisher()
	svc := NewCircuitBreakerService(repo, eventPub, testLogger())

	err := svc.Reset(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %s", domainErr.Category)
	}
}

func TestCircuitBreakerService_FullLifecycle(t *testing.T) {
	repo := newCBMockProjectRepo()
	eventPub := newCBMockEventPublisher()
	svc := NewCircuitBreakerService(repo, eventPub, testLogger())

	projectID := uuid.New()
	repo.projects[projectID] = newTestProject(projectID, 3)

	ctx := context.Background()

	// 1. Circuit breaker starts closed
	if err := svc.CheckCircuitBreaker(ctx, projectID); err != nil {
		t.Fatalf("step 1: expected circuit breaker closed, got %v", err)
	}

	// 2. Record 3 failures to trip the breaker
	for i := 0; i < 3; i++ {
		if err := svc.RecordFailure(ctx, projectID); err != nil {
			t.Fatalf("step 2: failure %d: %v", i+1, err)
		}
	}

	// 3. Circuit breaker should now be open
	err := svc.CheckCircuitBreaker(ctx, projectID)
	if err == nil {
		t.Fatal("step 3: expected circuit breaker open, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("step 3: expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != circuitBreakerOpenCode {
		t.Errorf("step 3: expected code CIRCUIT_BREAKER_OPEN, got %s", domainErr.Code)
	}

	// 4. Reset the circuit breaker
	if err := svc.Reset(ctx, projectID); err != nil {
		t.Fatalf("step 4: %v", err)
	}

	// 5. Circuit breaker should be closed again
	if err := svc.CheckCircuitBreaker(ctx, projectID); err != nil {
		t.Fatalf("step 5: expected circuit breaker closed after reset, got %v", err)
	}

	// 6. A success resets the count
	if err := svc.RecordFailure(ctx, projectID); err != nil {
		t.Fatalf("step 6a: %v", err)
	}
	if err := svc.RecordSuccess(ctx, projectID); err != nil {
		t.Fatalf("step 6b: %v", err)
	}
	if repo.projects[projectID].CircuitBreakerCount != 0 {
		t.Errorf("step 6: expected count 0, got %d", repo.projects[projectID].CircuitBreakerCount)
	}
}
