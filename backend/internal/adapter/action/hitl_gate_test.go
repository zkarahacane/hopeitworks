package action_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/action"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// --- HITL-specific mocks ---

type hitlMockHITLRepo struct {
	mu       sync.Mutex
	createFn func(ctx context.Context, req *model.HITLRequest) (*model.HITLRequest, error)
	created  []*model.HITLRequest
}

func (m *hitlMockHITLRepo) Create(ctx context.Context, req *model.HITLRequest) (*model.HITLRequest, error) {
	m.mu.Lock()
	m.created = append(m.created, req)
	m.mu.Unlock()
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return req, nil
}

func (m *hitlMockHITLRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.HITLRequest, error) {
	return nil, apperrors.NewNotFound("hitl_request", uuid.Nil)
}

func (m *hitlMockHITLRepo) GetByRunStepID(_ context.Context, _ uuid.UUID) (*model.HITLRequest, error) {
	return nil, apperrors.NewNotFound("hitl_request", uuid.Nil)
}

func (m *hitlMockHITLRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ model.HITLStatus, _ *uuid.UUID, _ *string, _ time.Time) (*model.HITLRequest, error) {
	return nil, nil
}

func (m *hitlMockHITLRepo) UpdateResolution(_ context.Context, _ uuid.UUID, _ model.HITLStatus, _ *uuid.UUID, _ string, _ time.Time) (*model.HITLRequest, error) {
	return nil, nil
}

func (m *hitlMockHITLRepo) ListProbeHalts(_ context.Context, _ *uuid.UUID) ([]*model.ProbeHalt, error) {
	return nil, nil
}

func (m *hitlMockHITLRepo) ListPendingByProject(_ context.Context, _ uuid.UUID) ([]*model.PendingHITLRequest, error) {
	return nil, nil
}

func (m *hitlMockHITLRepo) CountPendingByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *hitlMockHITLRepo) ListFiltered(_ context.Context, _ *string, _, _ int32) ([]*model.HITLRequest, error) {
	return nil, nil
}

func (m *hitlMockHITLRepo) CountFiltered(_ context.Context, _ *string) (int64, error) {
	return 0, nil
}

func (m *hitlMockHITLRepo) getCreated() []*model.HITLRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*model.HITLRequest, len(m.created))
	copy(result, m.created)
	return result
}

type hitlStepStatusCall struct {
	ID     uuid.UUID
	Status model.StepStatus
}

type hitlMockRunRepo struct {
	mu                    sync.Mutex
	updateRunStepStatusFn func(ctx context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.RunStep, error)
	statusCalls           []hitlStepStatusCall
}

func (m *hitlMockRunRepo) CreateRun(_ context.Context, run *model.Run) (*model.Run, error) {
	return run, nil
}
func (m *hitlMockRunRepo) GetRun(_ context.Context, _ uuid.UUID) (*model.Run, error) {
	return nil, apperrors.NewNotFound("run", uuid.Nil)
}
func (m *hitlMockRunRepo) GetActiveRunByStory(_ context.Context, _ uuid.UUID) (*model.Run, error) {
	return nil, nil
}
func (m *hitlMockRunRepo) ListRunsByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *hitlMockRunRepo) ListRunsByStory(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *hitlMockRunRepo) ListRunsByStatus(_ context.Context, _ model.RunStatus) ([]*model.Run, error) {
	return nil, nil
}
func (m *hitlMockRunRepo) MarkRunOrphanedIfRunning(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (bool, error) {
	return false, nil
}
func (m *hitlMockRunRepo) UpdateRunStatus(_ context.Context, id uuid.UUID, _ model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
	return &model.Run{ID: id}, nil
}
func (m *hitlMockRunRepo) CountRunsByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *hitlMockRunRepo) CountRunsByStory(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *hitlMockRunRepo) CreateRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}
func (m *hitlMockRunRepo) GetRunStep(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
	return &model.RunStep{ID: id}, nil
}
func (m *hitlMockRunRepo) ListRunStepsByRun(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}
func (m *hitlMockRunRepo) UpdateRunStepStatus(ctx context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.RunStep, error) {
	m.mu.Lock()
	m.statusCalls = append(m.statusCalls, hitlStepStatusCall{ID: id, Status: status})
	m.mu.Unlock()
	if m.updateRunStepStatusFn != nil {
		return m.updateRunStepStatusFn(ctx, id, status, startedAt, completedAt, errorMsg)
	}
	return &model.RunStep{ID: id, Status: status}, nil
}
func (m *hitlMockRunRepo) UpdateRunStepContainerInfo(_ context.Context, id uuid.UUID, _ *string, _ *string) (*model.RunStep, error) {
	return &model.RunStep{ID: id}, nil
}

func (m *hitlMockRunRepo) CreateRetryRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}

func (m *hitlMockRunRepo) ListRetryStepsByParent(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}

func (m *hitlMockRunRepo) getStatusCalls() []hitlStepStatusCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]hitlStepStatusCall, len(m.statusCalls))
	copy(result, m.statusCalls)
	return result
}

type hitlMockGitProvider struct {
	mu          sync.Mutex
	getPRDiffFn func(ctx context.Context, prURL string) (string, error)
	diffCalls   []string
}

func (m *hitlMockGitProvider) CloneRepo(_ context.Context, _ string, _ string) error { return nil }
func (m *hitlMockGitProvider) CreateBranch(_ context.Context, _ string, _ string) error {
	return nil
}
func (m *hitlMockGitProvider) CreateRemoteBranch(_ context.Context, _ string, _ string, _ string) error {
	return nil
}
func (m *hitlMockGitProvider) Push(_ context.Context, _ string, _ string) error { return nil }
func (m *hitlMockGitProvider) CreatePR(_ context.Context, _ string, _ string, _ string, _ string) (string, error) {
	return "", nil
}
func (m *hitlMockGitProvider) CreateRemotePR(_ context.Context, _ string, _ string, _ string, _ string, _ string) (string, error) {
	return "", nil
}
func (m *hitlMockGitProvider) MergePR(_ context.Context, _ string, _ string) error { return nil }
func (m *hitlMockGitProvider) GetCIStatus(_ context.Context, _ string) (string, error) {
	return ciStatusPass, nil
}
func (m *hitlMockGitProvider) GetRemoteCIStatus(_ context.Context, _ string) (string, error) {
	return ciStatusPass, nil
}
func (m *hitlMockGitProvider) GetPRDiff(ctx context.Context, prURL string) (string, error) {
	m.mu.Lock()
	m.diffCalls = append(m.diffCalls, prURL)
	m.mu.Unlock()
	if m.getPRDiffFn != nil {
		return m.getPRDiffFn(ctx, prURL)
	}
	return "diff --git a/file.go b/file.go\n+added line", nil
}

func (m *hitlMockGitProvider) getDiffCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.diffCalls))
	copy(result, m.diffCalls)
	return result
}

type hitlMockGitProviderFactory struct {
	provider port.GitProvider
	err      error
}

func (m *hitlMockGitProviderFactory) ForProjectID(_ context.Context, _ uuid.UUID) (port.GitProvider, error) {
	return m.provider, m.err
}

type hitlMockEventPublisher struct {
	mu     sync.Mutex
	events []model.Event
}

func (m *hitlMockEventPublisher) Publish(_ context.Context, event model.Event) error {
	m.mu.Lock()
	m.events = append(m.events, event)
	m.mu.Unlock()
	return nil
}

func (m *hitlMockEventPublisher) getEvents() []model.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]model.Event, len(m.events))
	copy(result, m.events)
	return result
}

type hitlMockStoryRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Story, error)
}

func (m *hitlMockStoryRepo) Create(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *hitlMockStoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, apperrors.NewNotFound("story", id)
}
func (m *hitlMockStoryRepo) GetByKey(_ context.Context, _ uuid.UUID, _ string) (*model.Story, error) {
	return nil, nil
}
func (m *hitlMockStoryRepo) ListByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *hitlMockStoryRepo) ListByStatus(_ context.Context, _ uuid.UUID, _ []string, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *hitlMockStoryRepo) ListByEpic(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *hitlMockStoryRepo) CountByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *hitlMockStoryRepo) CountByStatus(_ context.Context, _ uuid.UUID, _ []string) (int64, error) {
	return 0, nil
}
func (m *hitlMockStoryRepo) Update(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *hitlMockStoryRepo) UpdateStoryCurrentStage(_ context.Context, id uuid.UUID, currentStage *string) (*model.Story, error) {
	return &model.Story{ID: id, CurrentStage: currentStage}, nil
}
func (m *hitlMockStoryRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

// --- Helpers ---

func buildRunCtx(metadata map[string]any) *model.RunContext {
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
			Action: "hitl_gate",
			Status: model.StepStatusRunning,
		},
		ProjectID: projectID,
		StoryID:   storyID,
		Metadata:  metadata,
	}
}

func defaultStory(storyID uuid.UUID) *model.Story {
	return &model.Story{
		ID:    storyID,
		Key:   "S-01",
		Title: "Test Story",
	}
}

// --- Tests ---

func TestHITLGateAction_Name(t *testing.T) {
	a := action.NewHITLGateAction(nil, nil, &hitlMockGitProviderFactory{}, nil, nil, testLogger())
	if a.Name() != "hitl_gate" {
		t.Fatalf("expected Name() = %q, got %q", "hitl_gate", a.Name())
	}
}

func TestHITLGateAction_Execute_HappyPathWithPRURL(t *testing.T) {
	hitlRepo := &hitlMockHITLRepo{}
	runRepo := &hitlMockRunRepo{}
	gitProvider := &hitlMockGitProvider{}
	factory := &hitlMockGitProviderFactory{provider: gitProvider}
	eventPub := &hitlMockEventPublisher{}
	storyRepo := &hitlMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return defaultStory(id), nil
		},
	}

	a := action.NewHITLGateAction(hitlRepo, runRepo, factory, eventPub, storyRepo, testLogger())

	prURL := "https://github.com/owner/repo/pull/42"
	runCtx := buildRunCtx(map[string]any{"pr_url": prURL})

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Verify GetPRDiff was called
	diffCalls := gitProvider.getDiffCalls()
	if len(diffCalls) != 1 || diffCalls[0] != prURL {
		t.Fatalf("expected 1 GetPRDiff call with %q, got %v", prURL, diffCalls)
	}

	// Verify HITL request was created with non-nil diff
	created := hitlRepo.getCreated()
	if len(created) != 1 {
		t.Fatalf("expected 1 HITL request created, got %d", len(created))
	}
	if created[0].DiffContent == nil {
		t.Fatal("expected non-nil DiffContent")
	}
	if created[0].Status != model.HITLStatusPending {
		t.Fatalf("expected status %q, got %q", model.HITLStatusPending, created[0].Status)
	}
	if created[0].GateType != "approval" {
		t.Fatalf("expected gate_type %q, got %q", "approval", created[0].GateType)
	}

	// Verify step was transitioned to waiting_approval
	statusCalls := runRepo.getStatusCalls()
	if len(statusCalls) != 1 {
		t.Fatalf("expected 1 status update call, got %d", len(statusCalls))
	}
	if statusCalls[0].Status != model.StepStatusWaitingApproval {
		t.Fatalf("expected status %q, got %q", model.StepStatusWaitingApproval, statusCalls[0].Status)
	}

	// Verify event was published
	events := eventPub.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(events))
	}
	if events[0].EventName() != "hitl_gate.pending" {
		t.Fatalf("expected event name %q, got %q", "hitl_gate.pending", events[0].EventName())
	}
}

func TestHITLGateAction_Execute_NoPRURL(t *testing.T) {
	hitlRepo := &hitlMockHITLRepo{}
	runRepo := &hitlMockRunRepo{}
	gitProvider := &hitlMockGitProvider{}
	factory := &hitlMockGitProviderFactory{provider: gitProvider}
	eventPub := &hitlMockEventPublisher{}
	storyRepo := &hitlMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return defaultStory(id), nil
		},
	}

	a := action.NewHITLGateAction(hitlRepo, runRepo, factory, eventPub, storyRepo, testLogger())

	runCtx := buildRunCtx(map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Verify GetPRDiff was NOT called
	diffCalls := gitProvider.getDiffCalls()
	if len(diffCalls) != 0 {
		t.Fatalf("expected no GetPRDiff calls, got %d", len(diffCalls))
	}

	// Verify HITL request was created with nil diff
	created := hitlRepo.getCreated()
	if len(created) != 1 {
		t.Fatalf("expected 1 HITL request created, got %d", len(created))
	}
	if created[0].DiffContent != nil {
		t.Fatal("expected nil DiffContent when no PR URL")
	}
}

func TestHITLGateAction_Execute_GetPRDiffFails(t *testing.T) {
	hitlRepo := &hitlMockHITLRepo{}
	runRepo := &hitlMockRunRepo{}
	gitProvider := &hitlMockGitProvider{
		getPRDiffFn: func(_ context.Context, _ string) (string, error) {
			return "", fmt.Errorf("gh pr diff failed: network error")
		},
	}
	factory := &hitlMockGitProviderFactory{provider: gitProvider}
	eventPub := &hitlMockEventPublisher{}
	storyRepo := &hitlMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return defaultStory(id), nil
		},
	}

	a := action.NewHITLGateAction(hitlRepo, runRepo, factory, eventPub, storyRepo, testLogger())

	prURL := "https://github.com/owner/repo/pull/42"
	runCtx := buildRunCtx(map[string]any{"pr_url": prURL})

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error (non-fatal diff failure), got %v", err)
	}

	// Verify HITL request was created with nil diff (since fetch failed)
	created := hitlRepo.getCreated()
	if len(created) != 1 {
		t.Fatalf("expected 1 HITL request created, got %d", len(created))
	}
	if created[0].DiffContent != nil {
		t.Fatal("expected nil DiffContent when diff fetch fails")
	}

	// Verify step was still transitioned and event published
	statusCalls := runRepo.getStatusCalls()
	if len(statusCalls) != 1 || statusCalls[0].Status != model.StepStatusWaitingApproval {
		t.Fatalf("expected waiting_approval status update, got %v", statusCalls)
	}
	events := eventPub.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestHITLGateAction_Execute_HITLRepoCreateFails(t *testing.T) {
	hitlRepo := &hitlMockHITLRepo{
		createFn: func(_ context.Context, _ *model.HITLRequest) (*model.HITLRequest, error) {
			return nil, fmt.Errorf("db connection lost")
		},
	}
	runRepo := &hitlMockRunRepo{}
	gitProvider := &hitlMockGitProvider{}
	factory := &hitlMockGitProviderFactory{provider: gitProvider}
	eventPub := &hitlMockEventPublisher{}
	storyRepo := &hitlMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return defaultStory(id), nil
		},
	}

	a := action.NewHITLGateAction(hitlRepo, runRepo, factory, eventPub, storyRepo, testLogger())

	runCtx := buildRunCtx(map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when HITLRepository.Create fails")
	}
	if !strings.Contains(err.Error(), "create HITL request") {
		t.Fatalf("expected error containing %q, got %q", "create HITL request", err.Error())
	}

	// Verify step was NOT transitioned (error before status update)
	statusCalls := runRepo.getStatusCalls()
	if len(statusCalls) != 0 {
		t.Fatalf("expected no status updates when create fails, got %d", len(statusCalls))
	}

	// Verify no events published
	events := eventPub.getEvents()
	if len(events) != 0 {
		t.Fatalf("expected no events when create fails, got %d", len(events))
	}
}

func TestHITLGateAction_Execute_UpdateStepStatusFails(t *testing.T) {
	hitlRepo := &hitlMockHITLRepo{}
	runRepo := &hitlMockRunRepo{
		updateRunStepStatusFn: func(_ context.Context, _ uuid.UUID, _ model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
			return nil, fmt.Errorf("db write failed")
		},
	}
	gitProvider := &hitlMockGitProvider{}
	factory := &hitlMockGitProviderFactory{provider: gitProvider}
	eventPub := &hitlMockEventPublisher{}
	storyRepo := &hitlMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return defaultStory(id), nil
		},
	}

	a := action.NewHITLGateAction(hitlRepo, runRepo, factory, eventPub, storyRepo, testLogger())

	runCtx := buildRunCtx(map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when UpdateRunStepStatus fails")
	}
	if !strings.Contains(err.Error(), "update step to waiting_approval") {
		t.Fatalf("expected error containing %q, got %q", "update step to waiting_approval", err.Error())
	}

	// HITL request was already created in DB — verify Create was called
	created := hitlRepo.getCreated()
	if len(created) != 1 {
		t.Fatalf("expected 1 HITL request created before status update, got %d", len(created))
	}

	// No event should be published since we errored before publishing
	events := eventPub.getEvents()
	if len(events) != 0 {
		t.Fatalf("expected no events when step status update fails, got %d", len(events))
	}
}

func TestHITLGateAction_Execute_StoryFetchFails(t *testing.T) {
	hitlRepo := &hitlMockHITLRepo{}
	runRepo := &hitlMockRunRepo{}
	gitProvider := &hitlMockGitProvider{}
	factory := &hitlMockGitProviderFactory{provider: gitProvider}
	eventPub := &hitlMockEventPublisher{}
	storyRepo := &hitlMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return nil, apperrors.NewNotFound("story", id)
		},
	}

	a := action.NewHITLGateAction(hitlRepo, runRepo, factory, eventPub, storyRepo, testLogger())

	runCtx := buildRunCtx(map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when story fetch fails")
	}
	if !strings.Contains(err.Error(), "fetch story") {
		t.Fatalf("expected error containing %q, got %q", "fetch story", err.Error())
	}
}

func (m *hitlMockStoryRepo) CountByEpicGroupedByStatus(_ context.Context, _ uuid.UUID) (model.StoryCounts, error) {
	return model.StoryCounts{}, nil
}

func (m *hitlMockRunRepo) GetLatestRunByStory(_ context.Context, _ uuid.UUID) (*model.LatestRun, error) {
	return nil, nil
}

func (m *hitlMockRunRepo) GetLatestRunsByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*model.LatestRun, error) {
	return map[uuid.UUID]*model.LatestRun{}, nil
}

func (m *hitlMockRunRepo) GetDAGNodeRunInfoByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]model.DAGNodeRunInfo, error) {
	return map[uuid.UUID]model.DAGNodeRunInfo{}, nil
}

func (m *hitlMockRunRepo) UpdateRunMetadata(_ context.Context, _ uuid.UUID, _ map[string]interface{}) error {
	return nil
}

func (m *hitlMockRunRepo) AppendStepLogTail(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
