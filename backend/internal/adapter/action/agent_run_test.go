package action_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/action"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

const testContainerID = "container-123"

// testLogger creates a silent logger for tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func strPtr(s string) *string { return &s }

// --- Mock implementations ---

type mockContainerManager struct {
	createFn         func(ctx context.Context, opts model.ContainerOpts) (string, error)
	startFn          func(ctx context.Context, containerID string) error
	stopFn           func(ctx context.Context, containerID string) error
	removeFn         func(ctx context.Context, containerID string) error
	waitFn           func(ctx context.Context, containerID string) (int, error)
	listContainersFn func(ctx context.Context, labels map[string]string) ([]port.ContainerInfo, error)

	mu          sync.Mutex
	createCalls []model.ContainerOpts
	startCalls  []string
	stopCalls   []string
	removeCalls []string
}

func (m *mockContainerManager) Create(ctx context.Context, opts model.ContainerOpts) (string, error) {
	m.mu.Lock()
	m.createCalls = append(m.createCalls, opts)
	m.mu.Unlock()
	if m.createFn != nil {
		return m.createFn(ctx, opts)
	}
	return testContainerID, nil
}

func (m *mockContainerManager) Start(ctx context.Context, containerID string) error {
	m.mu.Lock()
	m.startCalls = append(m.startCalls, containerID)
	m.mu.Unlock()
	if m.startFn != nil {
		return m.startFn(ctx, containerID)
	}
	return nil
}

func (m *mockContainerManager) Stop(ctx context.Context, containerID string) error {
	m.mu.Lock()
	m.stopCalls = append(m.stopCalls, containerID)
	m.mu.Unlock()
	if m.stopFn != nil {
		return m.stopFn(ctx, containerID)
	}
	return nil
}

func (m *mockContainerManager) Remove(ctx context.Context, containerID string) error {
	m.mu.Lock()
	m.removeCalls = append(m.removeCalls, containerID)
	m.mu.Unlock()
	if m.removeFn != nil {
		return m.removeFn(ctx, containerID)
	}
	return nil
}

func (m *mockContainerManager) Wait(ctx context.Context, containerID string) (int, error) {
	if m.waitFn != nil {
		return m.waitFn(ctx, containerID)
	}
	return 0, nil
}

func (m *mockContainerManager) ListContainers(ctx context.Context, labels map[string]string) ([]port.ContainerInfo, error) {
	if m.listContainersFn != nil {
		return m.listContainersFn(ctx, labels)
	}
	return nil, nil
}

func (m *mockContainerManager) ListRunningContainers(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
	return nil, nil
}

func (m *mockContainerManager) CreateNetwork(_ context.Context, _ string, _ map[string]string) (string, error) {
	return "", nil
}

func (m *mockContainerManager) RemoveNetwork(_ context.Context, _ string) error {
	return nil
}

func (m *mockContainerManager) ConnectContainer(_ context.Context, _, _ string, _ []string) error {
	return nil
}

func (m *mockContainerManager) ListNetworks(_ context.Context, _ map[string]string) ([]model.NetworkInfo, error) {
	return nil, nil
}

func (m *mockContainerManager) InspectHealth(_ context.Context, _ string) (string, error) {
	return model.HealthRunning, nil
}

type mockLogStreamer struct {
	streamLogsFn func(ctx context.Context, containerID, runID, stepID string) (<-chan model.LogEvent, <-chan int, error)
}

func (m *mockLogStreamer) StreamLogs(ctx context.Context, containerID, runID, stepID string) (<-chan model.LogEvent, <-chan int, error) {
	if m.streamLogsFn != nil {
		return m.streamLogsFn(ctx, containerID, runID, stepID)
	}
	logCh := make(chan model.LogEvent)
	doneCh := make(chan int, 1)
	close(logCh)
	doneCh <- 0
	return logCh, doneCh, nil
}

type mockEventPublisher struct {
	mu     sync.Mutex
	events []model.Event
}

func (m *mockEventPublisher) Publish(_ context.Context, event model.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockEventPublisher) getEvents() []model.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]model.Event, len(m.events))
	copy(result, m.events)
	return result
}

type mockStoryRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Story, error)
}

func (m *mockStoryRepo) Create(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *mockStoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.NewNotFound("story", id)
}
func (m *mockStoryRepo) GetByKey(_ context.Context, _ uuid.UUID, _ string) (*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepo) ListByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepo) ListByStatus(_ context.Context, _ uuid.UUID, _ []string, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepo) ListByEpic(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepo) CountByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockStoryRepo) CountByStatus(_ context.Context, _ uuid.UUID, _ []string) (int64, error) {
	return 0, nil
}
func (m *mockStoryRepo) Update(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *mockStoryRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

type mockProjectRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Project, error)
}

func (m *mockProjectRepo) Create(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}
func (m *mockProjectRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.NewNotFound("project", id)
}
func (m *mockProjectRepo) List(_ context.Context, _, _ int32) ([]*model.Project, error) {
	return nil, nil
}
func (m *mockProjectRepo) Count(_ context.Context) (int64, error) { return 0, nil }
func (m *mockProjectRepo) Update(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}
func (m *mockProjectRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockProjectRepo) IncrementCircuitBreakerCount(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return &model.Project{}, nil
}
func (m *mockProjectRepo) ResetCircuitBreaker(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return &model.Project{}, nil
}

type mockRunRepo struct {
	updateRunStepContainerInfoFn func(ctx context.Context, id uuid.UUID, containerID *string, logTail *string) (*model.RunStep, error)

	mu                    sync.Mutex
	containerInfoCalls    []containerInfoCall
	createRunFn           func(ctx context.Context, run *model.Run) (*model.Run, error)
	getRunFn              func(ctx context.Context, id uuid.UUID) (*model.Run, error)
	getActiveRunByStoryFn func(ctx context.Context, storyID uuid.UUID) (*model.Run, error)
	listRunsByProjectFn   func(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Run, error)
	listRunsByStoryFn     func(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]*model.Run, error)
	updateRunStatusFn     func(ctx context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt, pausedAt *time.Time, errorMsg *string) (*model.Run, error)
	countRunsByProjectFn  func(ctx context.Context, projectID uuid.UUID) (int64, error)
	countRunsByStoryFn    func(ctx context.Context, storyID uuid.UUID) (int64, error)
	createRunStepFn       func(ctx context.Context, step *model.RunStep) (*model.RunStep, error)
	getRunStepFn          func(ctx context.Context, id uuid.UUID) (*model.RunStep, error)
	listRunStepsByRunFn   func(ctx context.Context, runID uuid.UUID) ([]*model.RunStep, error)
	updateRunStepStatusFn func(ctx context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.RunStep, error)
}

type containerInfoCall struct {
	ID          uuid.UUID
	ContainerID *string
	LogTail     *string
}

func (m *mockRunRepo) UpdateRunStepContainerInfo(ctx context.Context, id uuid.UUID, containerID *string, logTail *string) (*model.RunStep, error) {
	m.mu.Lock()
	m.containerInfoCalls = append(m.containerInfoCalls, containerInfoCall{ID: id, ContainerID: containerID, LogTail: logTail})
	m.mu.Unlock()
	if m.updateRunStepContainerInfoFn != nil {
		return m.updateRunStepContainerInfoFn(ctx, id, containerID, logTail)
	}
	return &model.RunStep{ID: id}, nil
}

func (m *mockRunRepo) CreateRun(ctx context.Context, run *model.Run) (*model.Run, error) {
	if m.createRunFn != nil {
		return m.createRunFn(ctx, run)
	}
	return run, nil
}
func (m *mockRunRepo) GetRun(ctx context.Context, id uuid.UUID) (*model.Run, error) {
	if m.getRunFn != nil {
		return m.getRunFn(ctx, id)
	}
	return nil, errors.NewNotFound("run", id)
}
func (m *mockRunRepo) GetActiveRunByStory(ctx context.Context, storyID uuid.UUID) (*model.Run, error) {
	if m.getActiveRunByStoryFn != nil {
		return m.getActiveRunByStoryFn(ctx, storyID)
	}
	return nil, nil
}
func (m *mockRunRepo) ListRunsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Run, error) {
	if m.listRunsByProjectFn != nil {
		return m.listRunsByProjectFn(ctx, projectID, limit, offset)
	}
	return nil, nil
}
func (m *mockRunRepo) ListRunsByStory(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]*model.Run, error) {
	if m.listRunsByStoryFn != nil {
		return m.listRunsByStoryFn(ctx, storyID, limit, offset)
	}
	return nil, nil
}
func (m *mockRunRepo) UpdateRunStatus(ctx context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt, pausedAt *time.Time, errorMsg *string) (*model.Run, error) {
	if m.updateRunStatusFn != nil {
		return m.updateRunStatusFn(ctx, id, status, startedAt, completedAt, pausedAt, errorMsg)
	}
	return &model.Run{ID: id, Status: status}, nil
}
func (m *mockRunRepo) CountRunsByProject(ctx context.Context, projectID uuid.UUID) (int64, error) {
	if m.countRunsByProjectFn != nil {
		return m.countRunsByProjectFn(ctx, projectID)
	}
	return 0, nil
}
func (m *mockRunRepo) CountRunsByStory(ctx context.Context, storyID uuid.UUID) (int64, error) {
	if m.countRunsByStoryFn != nil {
		return m.countRunsByStoryFn(ctx, storyID)
	}
	return 0, nil
}
func (m *mockRunRepo) CreateRunStep(ctx context.Context, step *model.RunStep) (*model.RunStep, error) {
	if m.createRunStepFn != nil {
		return m.createRunStepFn(ctx, step)
	}
	return step, nil
}
func (m *mockRunRepo) GetRunStep(ctx context.Context, id uuid.UUID) (*model.RunStep, error) {
	if m.getRunStepFn != nil {
		return m.getRunStepFn(ctx, id)
	}
	return nil, errors.NewNotFound("run_step", id)
}
func (m *mockRunRepo) ListRunStepsByRun(ctx context.Context, runID uuid.UUID) ([]*model.RunStep, error) {
	if m.listRunStepsByRunFn != nil {
		return m.listRunStepsByRunFn(ctx, runID)
	}
	return nil, nil
}
func (m *mockRunRepo) UpdateRunStepStatus(ctx context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.RunStep, error) {
	if m.updateRunStepStatusFn != nil {
		return m.updateRunStepStatusFn(ctx, id, status, startedAt, completedAt, errorMsg)
	}
	return &model.RunStep{ID: id, Status: status}, nil
}

func (m *mockRunRepo) CreateRetryRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}

func (m *mockRunRepo) ListRetryStepsByParent(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}

func (m *mockRunRepo) getContainerInfoCalls() []containerInfoCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]containerInfoCall, len(m.containerInfoCalls))
	copy(result, m.containerInfoCalls)
	return result
}

type mockTemplateRenderer struct{}

func (m *mockTemplateRenderer) Render(templateContent string, _ *model.TemplateContext) (string, error) {
	return "rendered: " + templateContent[:min(20, len(templateContent))], nil
}

// --- Test fixture ---

// mockCostRepo is a no-op CostRepository for use in AgentRunAction tests.
type mockCostRepo struct{}

func (m *mockCostRepo) InsertCostRecord(_ context.Context, r *model.CostRecord) (*model.CostRecord, error) {
	return r, nil
}
func (m *mockCostRepo) GetCostByRunStep(_ context.Context, _ uuid.UUID) (*model.CostRecord, error) {
	return nil, nil
}
func (m *mockCostRepo) SumCostByProject(_ context.Context, _ uuid.UUID, _ time.Time) (float64, int64, int64, error) {
	return 0, 0, 0, nil
}
func (m *mockCostRepo) SumCostByRun(_ context.Context, _ uuid.UUID) (float64, error) {
	return 0, nil
}
func (m *mockCostRepo) SumCostByStory(_ context.Context, _ uuid.UUID) (float64, int64, int64, int, error) {
	return 0, 0, 0, 0, nil
}
func (m *mockCostRepo) ListCostsByProjectByStory(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.StoryCostBreakdown, error) {
	return nil, nil
}
func (m *mockCostRepo) ListCostsByProjectByRun(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.RunCostBreakdown, error) {
	return nil, nil
}
func (m *mockCostRepo) ListCostsByProjectByModel(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.CostByModel, error) {
	return nil, nil
}
func (m *mockCostRepo) ListStepCostsByRun(_ context.Context, _ uuid.UUID) ([]model.StepCostBreakdown, error) {
	return nil, nil
}
func (m *mockCostRepo) ListDailyCostsByProject(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.CostDataPoint, error) {
	return nil, nil
}
func (m *mockCostRepo) ListCostsByProjectByRunPaginated(_ context.Context, _ uuid.UUID, _ time.Time, _, _ int32) ([]model.RunCostRow, error) {
	return nil, nil
}
func (m *mockCostRepo) CountCostsByProjectByRun(_ context.Context, _ uuid.UUID, _ time.Time) (int64, error) {
	return 0, nil
}
func (m *mockCostRepo) ListByProjectByAgent(_ context.Context, _ uuid.UUID) ([]model.AgentCostBreakdown, error) {
	return nil, nil
}

func (m *mockCostRepo) ListCostsByRunByRole(_ context.Context, _ uuid.UUID) ([]model.RoleCostBreakdown, error) {
	return nil, nil
}

func (m *mockCostRepo) SumTokensByRun(_ context.Context, _ uuid.UUID) (int64, int64, error) {
	return 0, 0, nil
}

type agentRunFixture struct {
	projectID uuid.UUID
	storyID   uuid.UUID
	runID     uuid.UUID
	stepID    uuid.UUID

	story   *model.Story
	project *model.Project
	run     *model.Run
	runStep *model.RunStep

	containerMgr *mockContainerManager
	logStreamer  *mockLogStreamer
	eventPub     *mockEventPublisher
	storyRepo    *mockStoryRepo
	projectRepo  *mockProjectRepo
	runRepo      *mockRunRepo
	costSvc      *service.CostService

	action *action.AgentRunAction
}

func newAgentRunFixture(t *testing.T) *agentRunFixture {
	t.Helper()

	f := &agentRunFixture{
		projectID: uuid.New(),
		storyID:   uuid.New(),
		runID:     uuid.New(),
		stepID:    uuid.New(),
	}

	backendScope := "backend"
	repoURL := "https://github.com/test/repo"
	f.story = &model.Story{
		ID:                 f.storyID,
		ProjectID:          f.projectID,
		Key:                "S-42",
		Title:              "Test Story",
		Objective:          strPtr("Implement feature X"),
		Scope:              &backendScope,
		AcceptanceCriteria: strPtr("AC: feature works"),
		TargetFiles:        []string{"main.go", "handler.go"},
		Status:             "backlog",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	f.project = &model.Project{
		ID:      f.projectID,
		Name:    "Test Project",
		RepoURL: &repoURL,
	}

	f.run = &model.Run{
		ID:        f.runID,
		ProjectID: f.projectID,
		StoryID:   f.storyID,
		Status:    model.RunStatusRunning,
	}

	f.runStep = &model.RunStep{
		ID:        f.stepID,
		RunID:     f.runID,
		StepName:  "agent-run",
		StepOrder: 1,
		Action:    "agent_run",
		Status:    model.StepStatusRunning,
	}

	f.containerMgr = &mockContainerManager{
		createFn: func(_ context.Context, _ model.ContainerOpts) (string, error) {
			return testContainerID, nil
		},
	}

	f.logStreamer = &mockLogStreamer{
		streamLogsFn: func(_ context.Context, _, _, _ string) (<-chan model.LogEvent, <-chan int, error) {
			logCh := make(chan model.LogEvent, 3)
			doneCh := make(chan int, 1)
			go func() {
				logCh <- model.LogEvent{Message: "line 1", Level: "info", Timestamp: time.Now()}
				logCh <- model.LogEvent{Message: "line 2", Level: "info", Timestamp: time.Now()}
				logCh <- model.LogEvent{Message: "line 3", Level: "info", Timestamp: time.Now()}
				close(logCh)
				doneCh <- 0
			}()
			return logCh, doneCh, nil
		},
	}

	f.eventPub = &mockEventPublisher{}
	f.storyRepo = &mockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			if id == f.storyID {
				return f.story, nil
			}
			return nil, errors.NewNotFound("story", id)
		},
	}
	f.projectRepo = &mockProjectRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Project, error) {
			if id == f.projectID {
				return f.project, nil
			}
			return nil, errors.NewNotFound("project", id)
		},
	}
	f.runRepo = &mockRunRepo{}

	// Create a real TemplateRenderer
	renderer := &mockTemplateRenderer{}

	// Create a CostService with a no-op mock repository
	f.costSvc = service.NewCostService(&mockCostRepo{}, nil, nil, nil, testLogger())

	agentCfg := action.AgentConfig{
		DefaultMemory: 4294967296,
		DefaultCPUs:   2.0,
		NetworkName:   "test-network",
		LogTailLines:  50,
	}

	f.action = action.NewAgentRunAction(
		f.containerMgr,
		f.logStreamer,
		f.eventPub,
		f.storyRepo,
		f.projectRepo,
		f.runRepo,
		renderer,
		f.costSvc,
		agentCfg,
		testLogger(),
		nil, // apiKeySvc - not needed for legacy mode tests
		nil, // tokenStore - not needed for legacy mode tests
		nil, // statusStore - not needed for legacy mode tests
		"",  // callbackURL - not needed for legacy mode tests
	)

	return f
}

func (f *agentRunFixture) newRunContext() *model.RunContext {
	return &model.RunContext{
		Run:       f.run,
		RunStep:   f.runStep,
		ProjectID: f.projectID,
		StoryID:   f.storyID,
		Metadata: map[string]any{
			"branch_name":      "feat/s-42-test",
			"agent_image":      "hopeitworks/agent:latest",
			"template_content": "Implement story {{story_key}}: {{story_title}}",
		},
	}
}

// --- Tests ---

func TestAgentRunAction_Name(t *testing.T) {
	f := newAgentRunFixture(t)
	if f.action.Name() != "agent_run" {
		t.Errorf("expected action name %q, got %q", "agent_run", f.action.Name())
	}
}

func TestAgentRunAction_HappyPath(t *testing.T) {
	f := newAgentRunFixture(t)
	runCtx := f.newRunContext()

	err := f.action.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify container was created with correct opts
	f.containerMgr.mu.Lock()
	createCalls := f.containerMgr.createCalls
	f.containerMgr.mu.Unlock()
	if len(createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(createCalls))
	}

	opts := createCalls[0]
	if opts.Image != "hopeitworks/agent:latest" {
		t.Errorf("expected image %q, got %q", "hopeitworks/agent:latest", opts.Image)
	}
	if opts.NetworkName != "test-network" {
		t.Errorf("expected network %q, got %q", "test-network", opts.NetworkName)
	}
	if opts.Memory != 4294967296 {
		t.Errorf("expected memory 4294967296, got %d", opts.Memory)
	}
	if opts.CPUs != 2.0 {
		t.Errorf("expected CPUs 2.0, got %f", opts.CPUs)
	}

	// Verify labels
	if opts.Labels["managed_by"] != "hopeitworks" {
		t.Error("expected managed_by label")
	}
	if opts.Labels["run_id"] != f.runID.String() {
		t.Errorf("expected run_id label %s, got %s", f.runID, opts.Labels["run_id"])
	}
	if opts.Labels["step_id"] != f.stepID.String() {
		t.Errorf("expected step_id label %s, got %s", f.stepID, opts.Labels["step_id"])
	}
	if opts.Labels["story_key"] != "S-42" {
		t.Errorf("expected story_key label S-42, got %s", opts.Labels["story_key"])
	}

	// Verify env vars contain expected keys (including CLAUDE_MD_CONTENT)
	envMap := make(map[string]string)
	for _, env := range opts.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	if _, ok := envMap["CLAUDE_MD_CONTENT"]; !ok {
		t.Error("expected CLAUDE_MD_CONTENT env var to be set")
	}
	if !strings.Contains(envMap["CLAUDE_MD_CONTENT"], "Test Project") {
		t.Errorf("expected CLAUDE_MD_CONTENT to contain project name, got: %q", envMap["CLAUDE_MD_CONTENT"])
	}
	if envMap["REPO_URL"] != "https://github.com/test/repo" {
		t.Errorf("expected REPO_URL, got %q", envMap["REPO_URL"])
	}
	if envMap["BRANCH_NAME"] != "feat/s-42-test" {
		t.Errorf("expected BRANCH_NAME feat/s-42-test, got %q", envMap["BRANCH_NAME"])
	}
	if envMap["STORY_KEY"] != "S-42" {
		t.Errorf("expected STORY_KEY S-42, got %q", envMap["STORY_KEY"])
	}
	if _, ok := envMap["PROMPT_CONTENT"]; !ok {
		t.Error("expected PROMPT_CONTENT env var")
	}

	// Verify container was started
	f.containerMgr.mu.Lock()
	startCalls := f.containerMgr.startCalls
	f.containerMgr.mu.Unlock()
	if len(startCalls) != 1 || startCalls[0] != testContainerID {
		t.Errorf("expected start called with container-123, got %v", startCalls)
	}

	// Verify container ID was persisted
	infoCalls := f.runRepo.getContainerInfoCalls()
	var foundContainerIDPersist bool
	for _, call := range infoCalls {
		if call.ContainerID != nil && *call.ContainerID == testContainerID {
			foundContainerIDPersist = true
		}
	}
	if !foundContainerIDPersist {
		t.Error("expected container ID to be persisted")
	}

	// Verify log events were published
	events := f.eventPub.getEvents()
	if len(events) < 3 {
		t.Errorf("expected at least 3 log events published, got %d", len(events))
	}
	for _, e := range events {
		if e.EntityType != "log" || e.Action != "emitted" {
			t.Errorf("expected log.emitted event, got %s.%s", e.EntityType, e.Action)
		}
	}

	// Verify container was cleaned up (stop + remove)
	f.containerMgr.mu.Lock()
	stopCalls := f.containerMgr.stopCalls
	removeCalls := f.containerMgr.removeCalls
	f.containerMgr.mu.Unlock()
	if len(stopCalls) < 1 {
		t.Error("expected container stop called during cleanup")
	}
	if len(removeCalls) < 1 {
		t.Error("expected container remove called during cleanup")
	}
}

func TestAgentRunAction_MissingAgentImage(t *testing.T) {
	f := newAgentRunFixture(t)
	runCtx := f.newRunContext()
	delete(runCtx.Metadata, "agent_image")

	err := f.action.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error for missing agent_image, got nil")
	}
	if !strings.Contains(err.Error(), "agent_image is required") {
		t.Errorf("expected error about agent_image, got: %v", err)
	}

	// Verify no container was created
	f.containerMgr.mu.Lock()
	createCalls := f.containerMgr.createCalls
	f.containerMgr.mu.Unlock()
	if len(createCalls) > 0 {
		t.Error("expected no container to be created when agent_image is missing")
	}
}

func TestAgentRunAction_EmptyTemplateContent(t *testing.T) {
	f := newAgentRunFixture(t)
	runCtx := f.newRunContext()
	delete(runCtx.Metadata, "template_content")

	err := f.action.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected no error (empty prompt is ok), got %v", err)
	}

	// Verify PROMPT_CONTENT is empty
	f.containerMgr.mu.Lock()
	createCalls := f.containerMgr.createCalls
	f.containerMgr.mu.Unlock()

	envMap := make(map[string]string)
	for _, env := range createCalls[0].Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	if envMap["PROMPT_CONTENT"] != "" {
		t.Errorf("expected empty PROMPT_CONTENT when template_content is absent, got %q", envMap["PROMPT_CONTENT"])
	}
}

func TestAgentRunAction_AgentFailure(t *testing.T) {
	f := newAgentRunFixture(t)

	// Configure log streamer to send 10 events then exit with code 1
	f.logStreamer.streamLogsFn = func(_ context.Context, _, _, _ string) (<-chan model.LogEvent, <-chan int, error) {
		logCh := make(chan model.LogEvent, 10)
		doneCh := make(chan int, 1)
		go func() {
			for i := 0; i < 10; i++ {
				logCh <- model.LogEvent{
					Message:   fmt.Sprintf("log line %d", i),
					Level:     "info",
					Timestamp: time.Now(),
				}
			}
			close(logCh)
			doneCh <- 1 // non-zero exit code
		}()
		return logCh, doneCh, nil
	}

	runCtx := f.newRunContext()
	err := f.action.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error for non-zero exit code, got nil")
	}
	if !strings.Contains(err.Error(), "exited with code 1") {
		t.Errorf("expected error to contain 'exited with code 1', got: %v", err)
	}

	// Verify log tail was persisted
	infoCalls := f.runRepo.getContainerInfoCalls()
	var foundLogTail bool
	for _, call := range infoCalls {
		if call.LogTail != nil {
			foundLogTail = true
			tail := *call.LogTail
			// Should contain the last log lines
			if !strings.Contains(tail, "log line") {
				t.Errorf("expected log tail to contain log lines, got: %s", tail)
			}
		}
	}
	if !foundLogTail {
		t.Error("expected log tail to be persisted on failure")
	}

	// Verify container was cleaned up
	f.containerMgr.mu.Lock()
	removeCalls := f.containerMgr.removeCalls
	f.containerMgr.mu.Unlock()
	if len(removeCalls) < 1 {
		t.Error("expected container to be removed on failure")
	}
}

func TestAgentRunAction_ContainerCreateFailure(t *testing.T) {
	f := newAgentRunFixture(t)

	f.containerMgr.createFn = func(_ context.Context, _ model.ContainerOpts) (string, error) {
		return "", fmt.Errorf("docker error: no space left on device")
	}

	runCtx := f.newRunContext()
	err := f.action.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error for container create failure, got nil")
	}
	if !strings.Contains(err.Error(), "create container") {
		t.Errorf("expected error to wrap 'create container', got: %v", err)
	}

	// Verify Start and Wait were NOT called
	f.containerMgr.mu.Lock()
	startCalls := f.containerMgr.startCalls
	f.containerMgr.mu.Unlock()
	if len(startCalls) > 0 {
		t.Error("expected Start to NOT be called after create failure")
	}
}

func TestAgentRunAction_ContextCancellation(t *testing.T) {
	f := newAgentRunFixture(t)

	// Configure wait to block until context is cancelled.
	f.logStreamer.streamLogsFn = func(ctx context.Context, _, _, _ string) (<-chan model.LogEvent, <-chan int, error) {
		logCh := make(chan model.LogEvent)
		doneCh := make(chan int, 1)
		go func() {
			<-ctx.Done()
			close(logCh)
			close(doneCh) // no value — mimics real LogStreamer on cancellation
		}()
		return logCh, doneCh, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	runCtx := f.newRunContext()
	err := f.action.Execute(ctx, runCtx)
	if err == nil {
		t.Fatal("expected error for context cancellation, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context canceled error, got: %v", err)
	}

	// Verify container was cleaned up
	f.containerMgr.mu.Lock()
	stopCalls := f.containerMgr.stopCalls
	removeCalls := f.containerMgr.removeCalls
	f.containerMgr.mu.Unlock()
	if len(stopCalls) < 1 {
		t.Error("expected container stop called on cancellation")
	}
	if len(removeCalls) < 1 {
		t.Error("expected container remove called on cancellation")
	}
}

func TestAgentRunAction_StoryNotFound(t *testing.T) {
	f := newAgentRunFixture(t)

	f.storyRepo.getByIDFn = func(_ context.Context, id uuid.UUID) (*model.Story, error) {
		return nil, errors.NewNotFound("story", id)
	}

	runCtx := f.newRunContext()
	err := f.action.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error for story not found, got nil")
	}
	if !strings.Contains(err.Error(), "fetch story") {
		t.Errorf("expected error to wrap 'fetch story', got: %v", err)
	}

	// Verify no container was created
	f.containerMgr.mu.Lock()
	createCalls := f.containerMgr.createCalls
	f.containerMgr.mu.Unlock()
	if len(createCalls) > 0 {
		t.Error("expected no container to be created when story is not found")
	}
}

func TestAgentRunAction_ProjectNotFound(t *testing.T) {
	f := newAgentRunFixture(t)

	f.projectRepo.getByIDFn = func(_ context.Context, id uuid.UUID) (*model.Project, error) {
		return nil, errors.NewNotFound("project", id)
	}

	runCtx := f.newRunContext()
	err := f.action.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error for project not found, got nil")
	}
	if !strings.Contains(err.Error(), "fetch project") {
		t.Errorf("expected error to wrap 'fetch project', got: %v", err)
	}
}

func TestAgentRunAction_CleanupOnAllPaths(t *testing.T) {
	f := newAgentRunFixture(t)

	// Make start fail
	f.containerMgr.startFn = func(_ context.Context, _ string) error {
		return fmt.Errorf("docker start error")
	}

	runCtx := f.newRunContext()
	err := f.action.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error for start failure, got nil")
	}

	// Even though start failed, cleanup should still run (deferred)
	f.containerMgr.mu.Lock()
	removeCalls := f.containerMgr.removeCalls
	f.containerMgr.mu.Unlock()
	if len(removeCalls) < 1 {
		t.Error("expected container cleanup even when start fails")
	}
}

func TestAgentRunAction_BranchNameFromMetadata(t *testing.T) {
	f := newAgentRunFixture(t)
	runCtx := f.newRunContext()
	runCtx.Metadata["branch_name"] = "feat/runtime-4"

	err := f.action.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	f.containerMgr.mu.Lock()
	createCalls := f.containerMgr.createCalls
	f.containerMgr.mu.Unlock()
	if len(createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(createCalls))
	}

	envMap := make(map[string]string)
	for _, env := range createCalls[0].Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["BRANCH_NAME"] != "feat/runtime-4" {
		t.Errorf("expected BRANCH_NAME=feat/runtime-4, got %q", envMap["BRANCH_NAME"])
	}
}

func TestAgentRunAction_ModelFromMetadata(t *testing.T) {
	f := newAgentRunFixture(t)
	runCtx := f.newRunContext()
	runCtx.Metadata["model"] = "claude-opus-4-6"

	err := f.action.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	f.containerMgr.mu.Lock()
	createCalls := f.containerMgr.createCalls
	f.containerMgr.mu.Unlock()
	if len(createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(createCalls))
	}

	envMap := make(map[string]string)
	for _, env := range createCalls[0].Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["MODEL"] != "claude-opus-4-6" {
		t.Errorf("expected MODEL=claude-opus-4-6, got %q", envMap["MODEL"])
	}
}

func TestAgentRunAction_ModelFallback(t *testing.T) {
	f := newAgentRunFixture(t)
	runCtx := f.newRunContext()
	// Explicitly ensure no "model" in metadata
	delete(runCtx.Metadata, "model")

	err := f.action.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	f.containerMgr.mu.Lock()
	createCalls := f.containerMgr.createCalls
	f.containerMgr.mu.Unlock()
	if len(createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(createCalls))
	}

	for _, env := range createCalls[0].Env {
		if strings.HasPrefix(env, "MODEL=") {
			t.Errorf("expected no MODEL env var when model is not in metadata, got %q", env)
		}
	}
}

func (m *mockStoryRepo) CountByEpicGroupedByStatus(_ context.Context, _ uuid.UUID) (model.StoryCounts, error) {
	return model.StoryCounts{}, nil
}

func (m *mockRunRepo) GetLatestRunByStory(_ context.Context, _ uuid.UUID) (*model.LatestRun, error) {
	return nil, nil
}

func (m *mockRunRepo) GetLatestRunsByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*model.LatestRun, error) {
	return map[uuid.UUID]*model.LatestRun{}, nil
}

func (m *mockRunRepo) GetDAGNodeRunInfoByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]model.DAGNodeRunInfo, error) {
	return map[uuid.UUID]model.DAGNodeRunInfo{}, nil
}

func (m *mockRunRepo) UpdateRunMetadata(_ context.Context, _ uuid.UUID, _ map[string]interface{}) error {
	return nil
}

func (m *mockRunRepo) AppendStepLogTail(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
