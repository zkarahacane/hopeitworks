package action_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/action"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

const (
	ciPollActionName = "ci_poll"
	ciStatusPass     = "pass"
	ciStatusPending  = "pending"
	ciStatusFail     = "fail"
)

// --- Mocks for CIPollAction ---

type ciMockGitProvider struct {
	mu            sync.Mutex
	getCIStatusFn func(ctx context.Context, workDir string) (string, error)
	calls         int
}

func (m *ciMockGitProvider) GetCIStatus(ctx context.Context, workDir string) (string, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	return m.getCIStatusFn(ctx, workDir)
}

func (m *ciMockGitProvider) CloneRepo(_ context.Context, _ string, _ string) error    { return nil }
func (m *ciMockGitProvider) CreateBranch(_ context.Context, _ string, _ string) error { return nil }
func (m *ciMockGitProvider) Push(_ context.Context, _ string, _ string) error         { return nil }
func (m *ciMockGitProvider) CreatePR(_ context.Context, _ string, _ string, _ string, _ string) (string, error) {
	return "", nil
}
func (m *ciMockGitProvider) MergePR(_ context.Context, _ string, _ string) error   { return nil }
func (m *ciMockGitProvider) GetPRDiff(_ context.Context, _ string) (string, error) { return "", nil }

func (m *ciMockGitProvider) getCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

type ciMockGitProviderFactory struct {
	provider port.GitProvider
	err      error
}

func (m *ciMockGitProviderFactory) ForProjectID(_ context.Context, _ uuid.UUID) (port.GitProvider, error) {
	return m.provider, m.err
}

type ciMockEventPublisher struct {
	mu        sync.Mutex
	Published []model.Event
}

func (m *ciMockEventPublisher) Publish(_ context.Context, e model.Event) error {
	m.mu.Lock()
	m.Published = append(m.Published, e)
	m.mu.Unlock()
	return nil
}

func (m *ciMockEventPublisher) getPublished() []model.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]model.Event, len(m.Published))
	copy(result, m.Published)
	return result
}

// --- Helpers ---

// fastCIPollConfig returns a config with very short intervals for tests.
func fastCIPollConfig() action.CIPollConfig {
	return action.CIPollConfig{
		DefaultPollInterval: 1 * time.Millisecond,
		DefaultTimeout:      5 * time.Second,
	}
}

func buildCIRunCtx(metadata map[string]any) *model.RunContext {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()
	storyID := uuid.New()

	return &model.RunContext{
		Run: &model.Run{
			ID:        runID,
			ProjectID: projectID,
			StoryID:   storyID,
			Status:    model.RunStatusRunning,
		},
		RunStep: &model.RunStep{
			ID:     stepID,
			RunID:  runID,
			Action: ciPollActionName,
			Status: model.StepStatusRunning,
		},
		ProjectID: projectID,
		StoryID:   storyID,
		Metadata:  metadata,
	}
}

// --- Tests ---

func TestCIPollAction_Name(t *testing.T) {
	a := action.NewCIPollAction(nil, nil, fastCIPollConfig(), testLogger())
	if a.Name() != ciPollActionName {
		t.Fatalf("expected Name() = %q, got %q", ciPollActionName, a.Name())
	}
}

func TestCIPollAction_Execute_MissingPRURL(t *testing.T) {
	gitProvider := &ciMockGitProvider{}
	factory := &ciMockGitProviderFactory{provider: gitProvider}
	eventPub := &ciMockEventPublisher{}

	a := action.NewCIPollAction(factory, eventPub, fastCIPollConfig(), testLogger())
	runCtx := buildCIRunCtx(map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when pr_url is missing")
	}
	if !strings.Contains(err.Error(), "CI_POLL_MISSING_PR_URL") {
		t.Fatalf("expected error to contain %q, got %q", "CI_POLL_MISSING_PR_URL", err.Error())
	}

	// GetCIStatus should never be called
	if gitProvider.getCalls() != 0 {
		t.Fatalf("expected 0 GetCIStatus calls, got %d", gitProvider.getCalls())
	}
}

func TestCIPollAction_Execute_HappyPath_PassOnFirstTick(t *testing.T) {
	gitProvider := &ciMockGitProvider{
		getCIStatusFn: func(_ context.Context, _ string) (string, error) {
			return ciStatusPass, nil
		},
	}
	factory := &ciMockGitProviderFactory{provider: gitProvider}
	eventPub := &ciMockEventPublisher{}

	a := action.NewCIPollAction(factory, eventPub, fastCIPollConfig(), testLogger())
	runCtx := buildCIRunCtx(map[string]any{
		"pr_url": "https://github.com/owner/repo/pull/1",
	})

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// One GetCIStatus call for the "pass"
	if gitProvider.getCalls() != 1 {
		t.Fatalf("expected 1 GetCIStatus call, got %d", gitProvider.getCalls())
	}

	// One event published (for the "pass" status)
	events := eventPub.getPublished()
	if len(events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(events))
	}
	if events[0].EntityType != ciPollActionName {
		t.Fatalf("expected entity_type %q, got %q", ciPollActionName, events[0].EntityType)
	}
	if events[0].Action != "checking" {
		t.Fatalf("expected action %q, got %q", "checking", events[0].Action)
	}
}

func TestCIPollAction_Execute_PendingThenPass(t *testing.T) {
	callCount := 0
	gitProvider := &ciMockGitProvider{
		getCIStatusFn: func(_ context.Context, _ string) (string, error) {
			callCount++
			if callCount < 3 {
				return ciStatusPending, nil
			}
			return ciStatusPass, nil
		},
	}
	factory := &ciMockGitProviderFactory{provider: gitProvider}
	eventPub := &ciMockEventPublisher{}

	a := action.NewCIPollAction(factory, eventPub, fastCIPollConfig(), testLogger())
	runCtx := buildCIRunCtx(map[string]any{
		"pr_url": "https://github.com/owner/repo/pull/2",
	})

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// 3 calls: pending, pending, pass
	if gitProvider.getCalls() != 3 {
		t.Fatalf("expected 3 GetCIStatus calls, got %d", gitProvider.getCalls())
	}

	// 3 events published (2 pending + 1 pass)
	events := eventPub.getPublished()
	if len(events) != 3 {
		t.Fatalf("expected 3 events published, got %d", len(events))
	}
	for _, e := range events {
		if e.EntityType != ciPollActionName || e.Action != "checking" {
			t.Fatalf("unexpected event: entity_type=%q action=%q", e.EntityType, e.Action)
		}
	}
}

func TestCIPollAction_Execute_CIFailure(t *testing.T) {
	prURL := "https://github.com/owner/repo/pull/3"
	gitProvider := &ciMockGitProvider{
		getCIStatusFn: func(_ context.Context, _ string) (string, error) {
			return ciStatusFail, nil
		},
	}
	factory := &ciMockGitProviderFactory{provider: gitProvider}
	eventPub := &ciMockEventPublisher{}

	a := action.NewCIPollAction(factory, eventPub, fastCIPollConfig(), testLogger())
	runCtx := buildCIRunCtx(map[string]any{"pr_url": prURL})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when CI fails")
	}
	if !strings.Contains(err.Error(), "CI_POLL_FAILED") {
		t.Fatalf("expected error to contain %q, got %q", "CI_POLL_FAILED", err.Error())
	}
	if !strings.Contains(err.Error(), prURL) {
		t.Fatalf("expected error to contain PR URL %q, got %q", prURL, err.Error())
	}
}

func TestCIPollAction_Execute_Timeout(t *testing.T) {
	gitProvider := &ciMockGitProvider{
		getCIStatusFn: func(_ context.Context, _ string) (string, error) {
			return ciStatusPending, nil
		},
	}
	factory := &ciMockGitProviderFactory{provider: gitProvider}
	eventPub := &ciMockEventPublisher{}

	// 1ms timeout — expires almost immediately
	cfg := action.CIPollConfig{
		DefaultPollInterval: 1 * time.Millisecond,
		DefaultTimeout:      1 * time.Millisecond,
	}
	a := action.NewCIPollAction(factory, eventPub, cfg, testLogger())
	runCtx := buildCIRunCtx(map[string]any{
		"pr_url": "https://github.com/owner/repo/pull/4",
	})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error on timeout")
	}
	if !strings.Contains(err.Error(), "CI_POLL_TIMEOUT") {
		t.Fatalf("expected error to contain %q, got %q", "CI_POLL_TIMEOUT", err.Error())
	}
}

func TestCIPollAction_Execute_ContextCancellation(t *testing.T) {
	ready := make(chan struct{})
	gitProvider := &ciMockGitProvider{
		getCIStatusFn: func(ctx context.Context, _ string) (string, error) {
			// Signal that we're inside the poll loop, then block.
			select {
			case ready <- struct{}{}:
			default:
			}
			// Block until context is cancelled.
			<-ctx.Done()
			return "", ctx.Err()
		},
	}
	factory := &ciMockGitProviderFactory{provider: gitProvider}
	eventPub := &ciMockEventPublisher{}

	cfg := action.CIPollConfig{
		DefaultPollInterval: 1 * time.Millisecond,
		DefaultTimeout:      30 * time.Second,
	}
	a := action.NewCIPollAction(factory, eventPub, cfg, testLogger())
	runCtx := buildCIRunCtx(map[string]any{
		"pr_url": "https://github.com/owner/repo/pull/5",
	})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Execute(ctx, runCtx)
	}()

	// Wait for the first poll attempt then cancel.
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for first poll")
	}
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error on context cancellation")
		}
		if !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("expected context.Canceled error, got %q", err.Error())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Execute to return after cancel")
	}
}
