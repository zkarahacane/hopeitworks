package action_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"sort"
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

const (
	testNetwork      = "test-network"
	testStoryKey     = "S-42"
	testAgentRepoURL = "https://github.com/test/repo"
)

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

func (m *mockContainerManager) DisconnectContainer(_ context.Context, _, _ string) error {
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
func (m *mockStoryRepo) GetBySourceRef(_ context.Context, _ uuid.UUID, _, _ string) (*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepo) CreateFromImport(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *mockStoryRepo) UpdateFromImport(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *mockStoryRepo) UpdateProvenanceOnly(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
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
func (m *mockStoryRepo) UpdateStoryCurrentStage(_ context.Context, id uuid.UUID, currentStage *string) (*model.Story, error) {
	return &model.Story{ID: id, CurrentStage: currentStage}, nil
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
func (m *mockRunRepo) ListRunsByStatus(_ context.Context, _ model.RunStatus) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepo) MarkRunOrphanedIfRunning(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (bool, error) {
	return false, nil
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

// mockEnvironmentRepo is an EnvironmentRepository mock. By default GetByProjectID
// returns NotFound, modelling a project with no Environment (legacy behaviour).
type mockEnvironmentRepo struct {
	getByProjectIDFn func(ctx context.Context, projectID uuid.UUID) (*model.Environment, error)
}

func (m *mockEnvironmentRepo) Create(_ context.Context, e *model.Environment) (*model.Environment, error) {
	return e, nil
}
func (m *mockEnvironmentRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Environment, error) {
	return nil, errors.NewNotFound("environment", id)
}
func (m *mockEnvironmentRepo) GetByProjectID(ctx context.Context, projectID uuid.UUID) (*model.Environment, error) {
	if m.getByProjectIDFn != nil {
		return m.getByProjectIDFn(ctx, projectID)
	}
	return nil, errors.NewNotFound("environment", projectID)
}
func (m *mockEnvironmentRepo) Update(_ context.Context, e *model.Environment) (*model.Environment, error) {
	return e, nil
}
func (m *mockEnvironmentRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

// mockSidecarManager is a SidecarManager mock. launchFn defaults to a guard that
// FAILS the test if Launch is ever called: the back-compat golden test relies on
// Launch staying untouched when a project has no Environment.
type mockSidecarManager struct {
	t *testing.T

	launchFn func(ctx context.Context, runID uuid.UUID, env *model.Environment) (*port.SidecarContext, error)

	mu           sync.Mutex
	launchCalls  int
	cleanupCalls int
	stopCalls    int
}

func (m *mockSidecarManager) Launch(ctx context.Context, runID uuid.UUID, env *model.Environment) (*port.SidecarContext, error) {
	m.mu.Lock()
	m.launchCalls++
	m.mu.Unlock()
	if m.launchFn != nil {
		return m.launchFn(ctx, runID, env)
	}
	if m.t != nil {
		m.t.Errorf("SidecarManager.Launch must not be called when project has no Environment")
	}
	return nil, fmt.Errorf("unexpected Launch call")
}

func (m *mockSidecarManager) Stop(_ context.Context, _ *port.SidecarContext) error {
	m.mu.Lock()
	m.stopCalls++
	m.mu.Unlock()
	return nil
}

func (m *mockSidecarManager) Cleanup(_ context.Context, _ *port.SidecarContext) error {
	m.mu.Lock()
	m.cleanupCalls++
	m.mu.Unlock()
	return nil
}

func (m *mockSidecarManager) ListOrphanNetworks(_ context.Context) ([]model.NetworkInfo, error) {
	return nil, nil
}

func (m *mockSidecarManager) GC(_ context.Context, _ time.Duration) error { return nil }

func (m *mockSidecarManager) getLaunchCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.launchCalls
}

func (m *mockSidecarManager) getCleanupCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cleanupCalls
}

// mockStackRepo is a StackRepository mock. By default GetByKey returns a stack
// with no image, so callers fall back to the agent image unless getByKeyFn is
// set to return a real image_ref.
type mockStackRepo struct {
	getByKeyFn func(ctx context.Context, key string) (*model.Stack, error)
}

func (m *mockStackRepo) List(_ context.Context) ([]*model.Stack, error) { return nil, nil }
func (m *mockStackRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.Stack, error) {
	return nil, nil
}
func (m *mockStackRepo) GetByKey(ctx context.Context, key string) (*model.Stack, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, errors.NewNotFound("stack", key)
}
func (m *mockStackRepo) Upsert(_ context.Context, s *model.Stack) (*model.Stack, error) {
	return s, nil
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

func (m *mockCostRepo) ListByProjectByRole(_ context.Context, _ uuid.UUID) ([]model.ProjectRoleCostBreakdown, error) {
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

	containerMgr    *mockContainerManager
	logStreamer     *mockLogStreamer
	eventPub        *mockEventPublisher
	storyRepo       *mockStoryRepo
	projectRepo     *mockProjectRepo
	runRepo         *mockRunRepo
	environmentRepo *mockEnvironmentRepo
	sidecarMgr      *mockSidecarManager
	stackRepo       *mockStackRepo
	costSvc         *service.CostService

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
	repoURL := testAgentRepoURL
	f.story = &model.Story{
		ID:                 f.storyID,
		ProjectID:          f.projectID,
		Key:                testStoryKey,
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

	// Default: project has no Environment (GetByProjectID -> NotFound) and the
	// SidecarManager guards against any Launch call. This is the legacy path the
	// back-compat golden test pins.
	f.environmentRepo = &mockEnvironmentRepo{}
	f.sidecarMgr = &mockSidecarManager{t: t}
	f.stackRepo = &mockStackRepo{}

	// Create a real TemplateRenderer
	renderer := &mockTemplateRenderer{}

	// Create a CostService with a no-op mock repository
	f.costSvc = service.NewCostService(&mockCostRepo{}, nil, nil, nil, testLogger())

	agentCfg := action.AgentConfig{
		DefaultMemory: 4294967296,
		DefaultCPUs:   2.0,
		NetworkName:   testNetwork,
		LogTailLines:  50,
	}

	f.action = action.NewAgentRunAction(
		f.containerMgr,
		f.logStreamer,
		f.eventPub,
		f.storyRepo,
		f.projectRepo,
		f.runRepo,
		f.environmentRepo,
		f.sidecarMgr,
		f.stackRepo,
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

// setIsolateRuns rebuilds f.action with the given East-West isolation flag,
// reusing the same mocks. It keeps the legacy path (no AgentRuntime injected) so
// createContainer is exercised — the second security-critical network path.
func (f *agentRunFixture) setIsolateRuns(t *testing.T, isolate bool) {
	t.Helper()
	agentCfg := action.AgentConfig{
		DefaultMemory: 4294967296,
		DefaultCPUs:   2.0,
		NetworkName:   testNetwork,
		IsolateRuns:   isolate,
		LogTailLines:  50,
	}
	f.action = action.NewAgentRunAction(
		f.containerMgr,
		f.logStreamer,
		f.eventPub,
		f.storyRepo,
		f.projectRepo,
		f.runRepo,
		f.environmentRepo,
		f.sidecarMgr,
		f.stackRepo,
		&mockTemplateRenderer{},
		f.costSvc,
		agentCfg,
		testLogger(),
		nil, // apiKeySvc - legacy mode
		nil, // tokenStore - legacy mode
		nil, // statusStore - legacy mode
		"",  // callbackURL - legacy mode
	)
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
	if opts.NetworkName != testNetwork {
		t.Errorf("expected network %q, got %q", testNetwork, opts.NetworkName)
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
	if opts.Labels["story_key"] != testStoryKey {
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
	if envMap["REPO_URL"] != testAgentRepoURL {
		t.Errorf("expected REPO_URL, got %q", envMap["REPO_URL"])
	}
	if envMap["BRANCH_NAME"] != "feat/s-42-test" {
		t.Errorf("expected BRANCH_NAME feat/s-42-test, got %q", envMap["BRANCH_NAME"])
	}
	if envMap["STORY_KEY"] != testStoryKey {
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

// TestAgentRunAction_NoEnvironment_GoldenBackCompat is the cardinal back-compat
// gate. A run whose project has NO Environment (GetByProjectID -> NotFound) must
// produce ContainerOpts STRICTLY identical to the pre-P2c2c behaviour: no extra
// connection-string env, no ExtraNetworks, and the SidecarManager is never asked
// to Launch. The guard mockSidecarManager fails the test if Launch is called.
func TestAgentRunAction_NoEnvironment_GoldenBackCompat(t *testing.T) {
	f := newAgentRunFixture(t)
	runCtx := f.newRunContext()

	if err := f.action.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// SidecarManager.Launch must NOT have been called.
	if got := f.sidecarMgr.getLaunchCalls(); got != 0 {
		t.Fatalf("expected 0 Launch calls for a project with no Environment, got %d", got)
	}
	// No Environment -> no sidecar teardown either.
	if got := f.sidecarMgr.getCleanupCalls(); got != 0 {
		t.Fatalf("expected 0 Cleanup calls for a project with no Environment, got %d", got)
	}

	f.containerMgr.mu.Lock()
	createCalls := f.containerMgr.createCalls
	f.containerMgr.mu.Unlock()
	if len(createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(createCalls))
	}
	opts := createCalls[0]

	// Golden invariant 1: ExtraNetworks empty (container is single-homed exactly
	// like before P2c2c).
	if len(opts.ExtraNetworks) != 0 {
		t.Errorf("expected empty ExtraNetworks, got %v", opts.ExtraNetworks)
	}
	// Golden invariant 2: NetworkName unchanged (the shared agent network only).
	if opts.NetworkName != testNetwork {
		t.Errorf("expected NetworkName test-network, got %q", opts.NetworkName)
	}
	// Golden invariant 3: no connection-string env leaked in.
	for _, key := range []string{"DATABASE_URL", "REDIS_URL", "MONGODB_URL", "ELASTICSEARCH_URL", "SMTP_HOST", "SMTP_PORT"} {
		for _, e := range opts.Env {
			if strings.HasPrefix(e, key+"=") {
				t.Errorf("expected no %s in env for a project with no Environment, got %q", key, e)
			}
		}
	}

	// Golden invariant 4: the env slice equals the legacy set, field-by-field.
	// Build the expected legacy env exactly as createContainer would with no
	// extraEnv, sort both, and compare element-by-element.
	wantEnv := []string{
		"REPO_URL=" + testAgentRepoURL,
		"BRANCH_NAME=feat/s-42-test",
		"STORY_KEY=S-42",
		"PROMPT_CONTENT=" + envValue(opts.Env, "PROMPT_CONTENT"),
		"PROMPT=" + envValue(opts.Env, "PROMPT"),
		"GIT_TOKEN=" + envValue(opts.Env, "GIT_TOKEN"),
		"GIT_PROVIDER=" + envValue(opts.Env, "GIT_PROVIDER"),
		"GITHUB_TOKEN=" + envValue(opts.Env, "GITHUB_TOKEN"),
		"CLAUDE_MD_CONTENT=" + envValue(opts.Env, "CLAUDE_MD_CONTENT"),
		"CLAUDE_CODE_OAUTH_TOKEN=" + envValue(opts.Env, "CLAUDE_CODE_OAUTH_TOKEN"),
	}
	gotEnv := append([]string(nil), opts.Env...)
	sort.Strings(gotEnv)
	sort.Strings(wantEnv)
	if len(gotEnv) != len(wantEnv) {
		t.Fatalf("env length mismatch: got %d (%v), want %d (%v)", len(gotEnv), gotEnv, len(wantEnv), wantEnv)
	}
	for i := range gotEnv {
		if gotEnv[i] != wantEnv[i] {
			t.Errorf("env[%d] mismatch:\n got=%q\nwant=%q", i, gotEnv[i], wantEnv[i])
		}
	}

	// Golden invariant 5: labels unchanged.
	if opts.Labels["managed_by"] != "hopeitworks" ||
		opts.Labels["run_id"] != f.runID.String() ||
		opts.Labels["step_id"] != f.stepID.String() ||
		opts.Labels["story_key"] != testStoryKey {
		t.Errorf("labels changed: %v", opts.Labels)
	}
	// Golden invariant 6: resource limits unchanged.
	if opts.Memory != 4294967296 || opts.CPUs != 2.0 {
		t.Errorf("resource limits changed: memory=%d cpus=%f", opts.Memory, opts.CPUs)
	}
}

// TestAgentRunAction_WithEnvironment_SidecarWiring proves the live path: a project
// WITH an Environment (one postgres service) launches sidecars, injects
// DATABASE_URL, dual-homes the agent container on the run network, and tears the
// sidecars down via the deferred Cleanup even when a later step fails.
func TestAgentRunAction_WithEnvironment_SidecarWiring(t *testing.T) {
	f := newAgentRunFixture(t)

	runNetwork := "hopeitworks-run-" + f.runID.String()
	env := &model.Environment{
		ProjectID: f.projectID,
		Services: []model.EnvironmentService{
			{
				Name:  "db",
				Image: "postgres:16",
				Env: map[string]string{
					"POSTGRES_USER":     "app",
					"POSTGRES_PASSWORD": "secret",
					"POSTGRES_DB":       "appdb",
				},
			},
		},
	}
	f.environmentRepo.getByProjectIDFn = func(_ context.Context, _ uuid.UUID) (*model.Environment, error) {
		return env, nil
	}
	f.sidecarMgr.launchFn = func(_ context.Context, runID uuid.UUID, gotEnv *model.Environment) (*port.SidecarContext, error) {
		if gotEnv != env {
			t.Errorf("Launch received unexpected environment")
		}
		return &port.SidecarContext{
			RunID:        runID,
			NetworkName:  runNetwork,
			ContainerIDs: map[string]string{"db": "sidecar-db"},
			ServiceAddrs: map[string]string{"db": "db"},
		}, nil
	}

	// Make a later step fail to prove the deferred Cleanup still runs.
	f.containerMgr.startFn = func(_ context.Context, _ string) error {
		return fmt.Errorf("docker start error")
	}

	runCtx := f.newRunContext()
	if err := f.action.Execute(context.Background(), runCtx); err == nil {
		t.Fatal("expected error from start failure, got nil")
	}

	// Launch was called exactly once.
	if got := f.sidecarMgr.getLaunchCalls(); got != 1 {
		t.Fatalf("expected 1 Launch call, got %d", got)
	}
	// Cleanup runs even though a later step failed (deferred teardown).
	if got := f.sidecarMgr.getCleanupCalls(); got != 1 {
		t.Fatalf("expected 1 Cleanup call (deferred), got %d", got)
	}

	f.containerMgr.mu.Lock()
	createCalls := f.containerMgr.createCalls
	f.containerMgr.mu.Unlock()
	if len(createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(createCalls))
	}
	opts := createCalls[0]

	// DATABASE_URL injected from the postgres service.
	wantURL := "postgres://app:secret@db:5432/appdb"
	if got := envValue(opts.Env, "DATABASE_URL"); got != wantURL {
		t.Errorf("expected DATABASE_URL=%q, got %q", wantURL, got)
	}

	// Dual-homing: keeps shared NetworkName AND attaches to the run network.
	if opts.NetworkName != testNetwork {
		t.Errorf("expected shared NetworkName test-network, got %q", opts.NetworkName)
	}
	if len(opts.ExtraNetworks) != 1 || opts.ExtraNetworks[0] != runNetwork {
		t.Errorf("expected ExtraNetworks=[%s], got %v", runNetwork, opts.ExtraNetworks)
	}
}

// TestAgentRunAction_Legacy_RunNetworkIsolation is the action-package pendant of
// the docker-package TestRuntime_Launch_Isolated_RunNetworkPrimary: it covers the
// SECOND security-critical path, the legacy createContainer (a.runtime == nil),
// for both isolation states. It closes the silent-regression hole where a future
// re-dual-home of the isolated branch would break East-West isolation without
// failing any test.
//
// Both cases run the live legacy path (the fixture injects no AgentRuntime) with a
// project that HAS an Environment, so sidecarCtx.NetworkName is non-empty.
func TestAgentRunAction_Legacy_RunNetworkIsolation(t *testing.T) {
	tests := []struct {
		name        string
		isolateRuns bool
		// wantPrimary is the expected opts.NetworkName.
		wantPrimary func(runNet string) string
		// wantExtra is the expected opts.ExtraNetworks.
		wantExtra func(runNet string) []string
	}{
		{
			name:        "default dual-homes on shared network",
			isolateRuns: false,
			wantPrimary: func(_ string) string { return testNetwork },
			wantExtra:   func(runNet string) []string { return []string{runNet} },
		},
		{
			name:        "isolated single-homes on per-run network",
			isolateRuns: true,
			wantPrimary: func(runNet string) string { return runNet },
			wantExtra:   func(_ string) []string { return nil },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAgentRunFixture(t)
			f.setIsolateRuns(t, tt.isolateRuns)

			runNetwork := "hopeitworks-run-" + f.runID.String()
			env := &model.Environment{
				ProjectID: f.projectID,
				Services: []model.EnvironmentService{
					{Name: "db", Image: "postgres:16", Env: map[string]string{"POSTGRES_PASSWORD": "x"}},
				},
			}
			f.environmentRepo.getByProjectIDFn = func(_ context.Context, _ uuid.UUID) (*model.Environment, error) {
				return env, nil
			}
			f.sidecarMgr.launchFn = func(_ context.Context, runID uuid.UUID, _ *model.Environment) (*port.SidecarContext, error) {
				return &port.SidecarContext{
					RunID:        runID,
					NetworkName:  runNetwork,
					ContainerIDs: map[string]string{"db": "sidecar-db"},
					ServiceAddrs: map[string]string{"db": "db"},
				}, nil
			}

			if err := f.action.Execute(context.Background(), f.newRunContext()); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			f.containerMgr.mu.Lock()
			createCalls := f.containerMgr.createCalls
			f.containerMgr.mu.Unlock()
			if len(createCalls) != 1 {
				t.Fatalf("expected 1 create call, got %d", len(createCalls))
			}
			opts := createCalls[0]

			if got, want := opts.NetworkName, tt.wantPrimary(runNetwork); got != want {
				t.Errorf("primary NetworkName: got %q, want %q", got, want)
			}
			wantExtra := tt.wantExtra(runNetwork)
			if len(opts.ExtraNetworks) != len(wantExtra) {
				t.Fatalf("ExtraNetworks: got %v, want %v", opts.ExtraNetworks, wantExtra)
			}
			for i := range wantExtra {
				if opts.ExtraNetworks[i] != wantExtra[i] {
					t.Errorf("ExtraNetworks[%d]: got %q, want %q", i, opts.ExtraNetworks[i], wantExtra[i])
				}
			}

			// Under isolation the SHARED network must appear NOWHERE (primary or extra):
			// that is the East-West invariant the agent is single-homed on its run net.
			if tt.isolateRuns {
				if opts.NetworkName == testNetwork {
					t.Errorf("isolated agent must NOT be on the shared network %q", testNetwork)
				}
				for _, n := range opts.ExtraNetworks {
					if n == testNetwork {
						t.Errorf("shared network %q leaked into ExtraNetworks: %v", testNetwork, opts.ExtraNetworks)
					}
				}
			}
		})
	}
}

// TestAgentRunAction_ConnString_EscapesCredentials proves the connection-string
// builder percent-encodes user-controlled credentials. A POSTGRES_PASSWORD made
// of URL-reserved characters (@ : / # ?) must yield a DATABASE_URL that parses
// cleanly with url.Parse and whose userinfo round-trips back to the exact
// password — no broken URL, no injected components.
func TestAgentRunAction_ConnString_EscapesCredentials(t *testing.T) {
	f := newAgentRunFixture(t)

	const rawPass = "p@s:w/rd#?x"
	env := &model.Environment{
		ProjectID: f.projectID,
		Services: []model.EnvironmentService{
			{
				Name:  "db",
				Image: "postgres:16",
				Env: map[string]string{
					"POSTGRES_USER":     "app",
					"POSTGRES_PASSWORD": rawPass,
					"POSTGRES_DB":       "appdb",
				},
			},
		},
	}
	f.environmentRepo.getByProjectIDFn = func(_ context.Context, _ uuid.UUID) (*model.Environment, error) {
		return env, nil
	}
	f.sidecarMgr.launchFn = func(_ context.Context, runID uuid.UUID, _ *model.Environment) (*port.SidecarContext, error) {
		return &port.SidecarContext{
			RunID:        runID,
			NetworkName:  "hopeitworks-run-" + runID.String(),
			ContainerIDs: map[string]string{"db": "sidecar-db"},
			ServiceAddrs: map[string]string{"db": "db"},
		}, nil
	}

	if err := f.action.Execute(context.Background(), f.newRunContext()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	f.containerMgr.mu.Lock()
	createCalls := f.containerMgr.createCalls
	f.containerMgr.mu.Unlock()
	if len(createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(createCalls))
	}

	raw := envValue(createCalls[0].Env, "DATABASE_URL")
	if raw == "" {
		t.Fatal("expected DATABASE_URL to be set")
	}
	// The raw env value must NOT contain the unescaped password verbatim — the
	// reserved characters must be percent-encoded.
	if strings.Contains(raw, rawPass) {
		t.Errorf("expected password to be percent-encoded, but raw URL contains it verbatim: %q", raw)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("DATABASE_URL must be parsable, got error: %v (url=%q)", err, raw)
	}
	if parsed.Scheme != "postgres" {
		t.Errorf("expected scheme postgres, got %q", parsed.Scheme)
	}
	if parsed.Host != "db:5432" {
		t.Errorf("expected host db:5432, got %q", parsed.Host)
	}
	if parsed.Path != "/appdb" {
		t.Errorf("expected path /appdb, got %q", parsed.Path)
	}
	if parsed.User == nil {
		t.Fatal("expected userinfo in DATABASE_URL")
	}
	if parsed.User.Username() != "app" {
		t.Errorf("expected username app, got %q", parsed.User.Username())
	}
	gotPass, ok := parsed.User.Password()
	if !ok {
		t.Fatal("expected a password in userinfo")
	}
	if gotPass != rawPass {
		t.Errorf("password did not round-trip: got %q, want %q", gotPass, rawPass)
	}
}

// envValue returns the value of the first KEY=value entry matching key, or "".
func envValue(env []string, key string) string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return e[len(prefix):]
		}
	}
	return ""
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

// --- Substrate dispatch (Stage 2) test infrastructure ---

// testSubstrateHandleID is the run handle the fake substrate returns from Launch.
// Defined once so the assertions (persistContainerID, streamAndWait target, Stop
// arg) all reference the same literal (goconst).
const testSubstrateHandleID = "sub-123"

// mockAgentRuntime is a fake port.AgentRuntime that records the RunSpec it was
// asked to Launch and counts Launch/Wait/Stop, so the dispatch path can be
// asserted without any real container backend. waitExitCode lets a test prove
// that the runtime's own Wait is IGNORED as the outcome source in callback mode.
type mockAgentRuntime struct {
	mu sync.Mutex

	launchSpecs []port.RunSpec
	waitCalls   int
	stopHandles []port.RunHandle

	launchErr    error
	waitExitCode int
	waitErr      error
	// waitDelay, when > 0, makes Wait sleep before returning so a test can order it
	// relative to the callback-status arrival (crash-detection reconciliation). Wait
	// still honours ctx cancellation while sleeping, so it never leaks.
	waitDelay time.Duration
}

func (m *mockAgentRuntime) Provision(_ context.Context, _ model.CapabilitySpec) (model.ProvisionResult, error) {
	return model.ProvisionResult{}, nil
}

func (m *mockAgentRuntime) Launch(_ context.Context, spec port.RunSpec) (port.RunHandle, error) {
	m.mu.Lock()
	m.launchSpecs = append(m.launchSpecs, spec)
	m.mu.Unlock()
	if m.launchErr != nil {
		return port.RunHandle{}, m.launchErr
	}
	return port.RunHandle{ID: testSubstrateHandleID}, nil
}

func (m *mockAgentRuntime) Wait(ctx context.Context, _ port.RunHandle) (port.RunResult, error) {
	m.mu.Lock()
	m.waitCalls++
	delay := m.waitDelay
	waitErr := m.waitErr
	exitCode := m.waitExitCode
	m.mu.Unlock()
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return port.RunResult{}, ctx.Err()
		}
	}
	if waitErr != nil {
		return port.RunResult{}, waitErr
	}
	return port.RunResult{ExitCode: exitCode}, nil
}

func (m *mockAgentRuntime) Stop(_ context.Context, h port.RunHandle) error {
	m.mu.Lock()
	m.stopHandles = append(m.stopHandles, h)
	m.mu.Unlock()
	return nil
}

func (m *mockAgentRuntime) SupportedCapabilities() model.CapabilitySet {
	return model.CapabilitySet{}
}

func (m *mockAgentRuntime) launchCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.launchSpecs)
}

func (m *mockAgentRuntime) stopCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.stopHandles)
}

func (m *mockAgentRuntime) lastSpec(t *testing.T) port.RunSpec {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.launchSpecs) == 0 {
		t.Fatalf("Launch was never called")
	}
	return m.launchSpecs[len(m.launchSpecs)-1]
}

// mockCallbackStatusStore is a fake port.CallbackStatusStore. WaitForStatus
// returns the configured exit code / error message, recording that it was
// consulted — used to prove the callback channel (not the runtime's Wait) is the
// outcome source in callback mode.
type mockCallbackStatusStore struct {
	mu sync.Mutex

	waitCalls int
	exitCode  int
	errMsg    string
	waitErr   error
	// block, when true, makes WaitForStatus park on ctx.Done() and return ctx.Err()
	// — modelling a status that NEVER arrives, so the substrate process exit drives
	// the outcome (crash detection).
	block bool
	// delay, when > 0, holds the status back for that long before returning it —
	// modelling a callback that lands AFTER the process exit but within the grace.
	delay time.Duration
}

func (m *mockCallbackStatusStore) WaitForStatus(ctx context.Context, _ uuid.UUID, _ time.Duration) (int, string, error) {
	m.mu.Lock()
	m.waitCalls++
	block := m.block
	delay := m.delay
	waitErr := m.waitErr
	exitCode := m.exitCode
	errMsg := m.errMsg
	m.mu.Unlock()
	if block {
		<-ctx.Done()
		return -1, "", ctx.Err()
	}
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return -1, "", ctx.Err()
		}
	}
	if waitErr != nil {
		return -1, "", waitErr
	}
	return exitCode, errMsg, nil
}

func (m *mockCallbackStatusStore) SetStatus(_ context.Context, _ uuid.UUID, _ int, _ string) error {
	return nil
}

func (m *mockCallbackStatusStore) waitCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.waitCalls
}

// envHas reports whether env contains a KEY=value entry for the given key.
func envHas(env []string, key string) bool {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}

// TestAgentRunAction_RuntimeDispatch_Legacy proves the legacy-mode run is realised
// THROUGH the injected port.AgentRuntime: Launch is called once with a RunSpec
// carrying the buildAgentEnv output / labels / resources / empty Network, the
// substrate handle is persisted as the container id, the outcome comes from
// streamAndWait (legacy, statusStore nil), Stop is deferred, and the Docker
// ContainerManager.Create is NEVER called directly on this path.
func TestAgentRunAction_RuntimeDispatch_Legacy(t *testing.T) {
	f := newAgentRunFixture(t)
	fakeRT := &mockAgentRuntime{}

	// Rebuild the action with the runtime injected (legacy mode: statusStore nil,
	// no runtime_kind metadata → isCallbackMode false → streamAndWait).
	action := newRuntimeAction(f, nil, nil, "", fakeRT)

	runCtx := f.newRunContext()
	if err := action.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Launch called exactly once.
	if got := fakeRT.launchCount(); got != 1 {
		t.Fatalf("expected 1 Launch call, got %d", got)
	}
	spec := fakeRT.lastSpec(t)

	// The Docker ContainerManager.Create must NOT be called on the runtime path.
	f.containerMgr.mu.Lock()
	createCalls := len(f.containerMgr.createCalls)
	f.containerMgr.mu.Unlock()
	if createCalls != 0 {
		t.Fatalf("expected 0 direct containerMgr.Create calls on the runtime path, got %d", createCalls)
	}

	// RunSpec.Env carries what buildAgentEnv produced (base block present).
	for _, key := range []string{"REPO_URL", "STORY_KEY", "PROMPT", "CLAUDE_MD_CONTENT"} {
		if !envHas(spec.Env, key) {
			t.Errorf("expected RunSpec.Env to contain %s=, env=%v", key, spec.Env)
		}
	}

	// RunSpec.Labels match buildAgentLabels.
	if spec.Labels["managed_by"] != model.LabelManagedByValue ||
		spec.Labels["run_id"] != f.runID.String() ||
		spec.Labels["step_id"] != f.stepID.String() ||
		spec.Labels["story_key"] != testStoryKey {
		t.Errorf("RunSpec.Labels mismatch: %v", spec.Labels)
	}

	// Resources copied from AgentConfig.
	if spec.Memory != 4294967296 || spec.CPUs != 2.0 {
		t.Errorf("RunSpec resources mismatch: memory=%d cpus=%f", spec.Memory, spec.CPUs)
	}

	// No Environment → zero RunNetwork (byte-identical to single-homed legacy).
	if spec.Network.Name != "" {
		t.Errorf("expected empty RunNetwork.Name with no Environment, got %q", spec.Network.Name)
	}

	// Image threaded through.
	if spec.Image != "hopeitworks/agent:latest" {
		t.Errorf("expected RunSpec.Image hopeitworks/agent:latest, got %q", spec.Image)
	}

	// persistContainerID received the substrate handle id.
	f.runRepo.mu.Lock()
	var persistedHandle bool
	for _, c := range f.runRepo.containerInfoCalls {
		if c.ContainerID != nil && *c.ContainerID == testSubstrateHandleID {
			persistedHandle = true
		}
	}
	f.runRepo.mu.Unlock()
	if !persistedHandle {
		t.Errorf("expected persistContainerID(%q) to be called with the substrate handle", testSubstrateHandleID)
	}

	// Stop deferred once on the handle.
	if got := fakeRT.stopCount(); got != 1 {
		t.Errorf("expected 1 Stop call (deferred teardown), got %d", got)
	}
	if len(fakeRT.stopHandles) == 1 && fakeRT.stopHandles[0].ID != testSubstrateHandleID {
		t.Errorf("expected Stop on handle %q, got %q", testSubstrateHandleID, fakeRT.stopHandles[0].ID)
	}
}

// TestAgentRunAction_RuntimeDispatch_CallbackWaitNotSkipped is the anti-drift
// guard. In callback mode the outcome MUST come from the callback channel
// (statusStore.WaitForStatus), NOT from the runtime's own Wait — proving the
// rejected P3c fork (read result.ExitCode off the substrate, skipping
// callback-wait + token-revoke) has not been reintroduced.
//
// The fake runtime's Wait is rigged to return a NON-zero exit code; the callback
// reports 0. The run must succeed (callback wins) and statusStore.WaitForStatus
// must have been consulted.
func TestAgentRunAction_RuntimeDispatch_CallbackWaitNotSkipped(t *testing.T) {
	f := newAgentRunFixture(t)

	// Rig the runtime so its OWN Wait would fail the run if it were the outcome
	// source (exit 1). It must be ignored in callback mode.
	fakeRT := &mockAgentRuntime{waitExitCode: 1, waitErr: nil}

	// Callback channel says exit 0 (success).
	statusStore := &mockCallbackStatusStore{exitCode: 0}

	// apiKeySvc is nil: the fixture runCtx carries no UserID (uuid.Nil), so
	// buildAgentEnv never resolves an API key. tokenStore IS exercised (the token
	// mint stays in buildAgentEnv).
	tokenStore := &mockTokenStore{}

	action := newCallbackRuntimeAction(f, statusStore, tokenStore, fakeRT)

	runCtx := f.newRunContext()
	// claude_code runtime_kind → callback mode (with statusStore set).
	runCtx.Metadata["runtime_kind"] = model.RuntimeKindClaudeCode

	if err := action.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("expected success (callback reports exit 0), got %v", err)
	}

	// The callback channel WAS consulted: outcome came from the callback.
	if got := statusStore.waitCount(); got != 1 {
		t.Fatalf("expected statusStore.WaitForStatus to be consulted once (callback = outcome source), got %d", got)
	}

	// The runtime's own Wait exit code was NOT used as the outcome source. Stage 3b
	// DOES consult runtime.Wait concurrently for crash detection, so it may be called
	// — but its non-zero exit code must NOT decide the outcome. The success above
	// (with waitExitCode=1) already proves it; here we only assert the status, not
	// the process exit, drove the result.
	if statusStore.exitCode != 0 {
		t.Fatalf("test misconfigured: status exit code must be 0 to prove callback wins")
	}

	// Launch + Stop still happen exactly once (substrate lifecycle preserved).
	if got := fakeRT.launchCount(); got != 1 {
		t.Errorf("expected 1 Launch call, got %d", got)
	}
	if got := fakeRT.stopCount(); got != 1 {
		t.Errorf("expected 1 Stop call (deferred teardown), got %d", got)
	}
}

// testCrashGrace is the short grace the Stage 3b reconciliation tests configure so
// a crash is declared (or a late status is awaited) in milliseconds, not the 5s
// production default.
const testCrashGrace = 50 * time.Millisecond

// TestAgentRunAction_RuntimeDispatch_StatusWinsOverProcessExit is the anti-drift
// guard under concurrency: even when the substrate process is OBSERVED to exit
// first (runtime.Wait returns quickly with a non-zero exit), the callback status is
// the source of truth. The status reports exit 0, so the run SUCCEEDS — the
// process exit code is never read as the outcome, only used for crash detection.
func TestAgentRunAction_RuntimeDispatch_StatusWinsOverProcessExit(t *testing.T) {
	f := newAgentRunFixture(t)

	// Process "finished" fast with a non-zero exit; if it were the outcome the run
	// would fail. The status arrives and reports success.
	fakeRT := &mockAgentRuntime{waitExitCode: 1}
	statusStore := &mockCallbackStatusStore{exitCode: 0}
	tokenStore := &mockTokenStore{}

	act := newCrashGraceRuntimeAction(f, statusStore, tokenStore, fakeRT, testCrashGrace)

	runCtx := f.newRunContext()
	runCtx.Metadata["runtime_kind"] = model.RuntimeKindClaudeCode

	if err := act.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("expected success (callback status reports exit 0), got %v", err)
	}

	// The callback channel WAS the outcome source.
	if got := statusStore.waitCount(); got != 1 {
		t.Errorf("expected statusStore.WaitForStatus consulted once, got %d", got)
	}
	// Substrate lifecycle preserved: Launch + Stop once each. (runtime.Wait may or
	// may not have returned before status won — both are fine and leak-free.)
	if got := fakeRT.launchCount(); got != 1 {
		t.Errorf("expected 1 Launch call, got %d", got)
	}
	if got := fakeRT.stopCount(); got != 1 {
		t.Errorf("expected 1 Stop call (deferred teardown), got %d", got)
	}
	// Token still revoked exactly once.
	if revoked := tokenStore.revokedTokenList(); len(revoked) != 1 || revoked[0] != testMintedToken {
		t.Errorf("expected exactly 1 revoke of %q, got %v", testMintedToken, revoked)
	}
}

// TestAgentRunAction_RuntimeDispatch_CrashWhenProcessExitsWithoutStatus proves the
// crash-detection path (ADR §2d): the substrate process exits (runtime.Wait returns
// fast) and NO callback status ever arrives (WaitForStatus parks on ctx). After the
// short grace the run is declared a CRASH — an error — instead of blocking on the
// 2h status timeout. The test also bounds the wall time to ~grace, proving it does
// not wait 2h.
func TestAgentRunAction_RuntimeDispatch_CrashWhenProcessExitsWithoutStatus(t *testing.T) {
	f := newAgentRunFixture(t)

	// Process exits fast (exit 1); status NEVER arrives (blocks until ctx done).
	fakeRT := &mockAgentRuntime{waitExitCode: 1}
	statusStore := &mockCallbackStatusStore{block: true}
	tokenStore := &mockTokenStore{}

	act := newCrashGraceRuntimeAction(f, statusStore, tokenStore, fakeRT, testCrashGrace)

	runCtx := f.newRunContext()
	runCtx.Metadata["runtime_kind"] = model.RuntimeKindClaudeCode

	start := time.Now()
	err := act.Execute(context.Background(), runCtx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a crash error when the process exits without a callback status, got nil")
	}
	if !strings.Contains(err.Error(), "without reporting a callback status") {
		t.Errorf("expected a crash error message, got: %v", err)
	}
	// It must resolve in roughly the grace, NOT the 2h status timeout.
	if elapsed > 5*time.Second {
		t.Errorf("expected crash detection to resolve within the grace, took %v", elapsed)
	}

	// Substrate lifecycle preserved and token revoked despite the crash.
	if got := fakeRT.stopCount(); got != 1 {
		t.Errorf("expected 1 Stop call (deferred teardown), got %d", got)
	}
	if revoked := tokenStore.revokedTokenList(); len(revoked) != 1 || revoked[0] != testMintedToken {
		t.Errorf("expected exactly 1 revoke of %q even on crash, got %v", testMintedToken, revoked)
	}
}

// TestAgentRunAction_RuntimeDispatch_StatusAfterProcessExit_WithinGrace proves the
// reverse ordering inside the grace window: the substrate process exits first, then
// the callback status lands a moment later but BEFORE the grace elapses. The outcome
// is the status (success), NOT a crash — the in-flight callback is honoured.
func TestAgentRunAction_RuntimeDispatch_StatusAfterProcessExit_WithinGrace(t *testing.T) {
	f := newAgentRunFixture(t)

	// Process exits immediately; the status arrives shortly after but well within
	// the grace (delay < grace).
	fakeRT := &mockAgentRuntime{waitExitCode: 1}
	statusStore := &mockCallbackStatusStore{exitCode: 0, delay: testCrashGrace / 5}
	tokenStore := &mockTokenStore{}

	act := newCrashGraceRuntimeAction(f, statusStore, tokenStore, fakeRT, testCrashGrace)

	runCtx := f.newRunContext()
	runCtx.Metadata["runtime_kind"] = model.RuntimeKindClaudeCode

	if err := act.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("expected success (status arrives within grace), got %v", err)
	}

	// The status WAS the outcome source — not a crash.
	if got := statusStore.waitCount(); got != 1 {
		t.Errorf("expected statusStore.WaitForStatus consulted once, got %d", got)
	}
	if got := fakeRT.stopCount(); got != 1 {
		t.Errorf("expected 1 Stop call (deferred teardown), got %d", got)
	}
	if revoked := tokenStore.revokedTokenList(); len(revoked) != 1 || revoked[0] != testMintedToken {
		t.Errorf("expected exactly 1 revoke of %q, got %v", testMintedToken, revoked)
	}
}

// TestAgentRunAction_RuntimeDispatch_CleanExitWaitsForStatus proves the ADR §2d
// exit-0 rule: a substrate process that finishes CLEANLY (exit 0) without a status
// is NOT a crash — it is a delayed/in-flight callback. Even when the status lands
// only AFTER the crash grace would have elapsed, the run waits for the authoritative
// status (bounded by the 2h WaitForStatus, not the grace) and SUCCEEDS. A non-zero
// exit on the same timing would have been declared a crash; exit 0 must not be.
func TestAgentRunAction_RuntimeDispatch_CleanExitWaitsForStatus(t *testing.T) {
	f := newAgentRunFixture(t)

	// Process exits CLEANLY (exit 0) right away; the status arrives well AFTER the
	// grace (3× grace) and reports success.
	fakeRT := &mockAgentRuntime{waitExitCode: 0}
	statusStore := &mockCallbackStatusStore{exitCode: 0, delay: testCrashGrace * 3}
	tokenStore := &mockTokenStore{}

	act := newCrashGraceRuntimeAction(f, statusStore, tokenStore, fakeRT, testCrashGrace)

	runCtx := f.newRunContext()
	runCtx.Metadata["runtime_kind"] = model.RuntimeKindClaudeCode

	if err := act.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("expected success: a clean exit-0 without a status must wait for the status, not crash; got %v", err)
	}

	// The status WAS the outcome source — waited beyond the grace, no crash.
	if got := statusStore.waitCount(); got != 1 {
		t.Errorf("expected statusStore.WaitForStatus consulted once, got %d", got)
	}
	if got := fakeRT.stopCount(); got != 1 {
		t.Errorf("expected 1 Stop call (deferred teardown), got %d", got)
	}
	if revoked := tokenStore.revokedTokenList(); len(revoked) != 1 || revoked[0] != testMintedToken {
		t.Errorf("expected exactly 1 revoke of %q, got %v", testMintedToken, revoked)
	}
}

// newCrashGraceRuntimeAction is newCallbackRuntimeAction with an explicit short
// CrashGrace, so the Stage 3b reconciliation tests resolve in milliseconds.
func newCrashGraceRuntimeAction(
	f *agentRunFixture,
	statusStore port.CallbackStatusStore,
	tokenStore port.ContainerTokenStore,
	rt port.AgentRuntime,
	crashGrace time.Duration,
) *action.AgentRunAction {
	agentCfg := action.AgentConfig{
		DefaultMemory: 4294967296,
		DefaultCPUs:   2.0,
		NetworkName:   testNetwork,
		LogTailLines:  50,
		CrashGrace:    crashGrace,
	}
	return action.NewAgentRunAction(
		f.containerMgr,
		f.logStreamer,
		f.eventPub,
		f.storyRepo,
		f.projectRepo,
		f.runRepo,
		f.environmentRepo,
		f.sidecarMgr,
		f.stackRepo,
		&mockTemplateRenderer{},
		f.costSvc,
		agentCfg,
		testLogger(),
		nil, // apiKeySvc — not needed: fixture runCtx has no UserID
		tokenStore,
		statusStore,
		"http://callback",
		action.WithAgentRuntime(rt),
	)
}

// newRuntimeAction builds an AgentRunAction from the fixture's mocks plus the
// supplied auth/callback wiring and an injected runtime. Keeps the long positional
// constructor in one place for the Stage 2 runtime-path tests.
func newRuntimeAction(
	f *agentRunFixture,
	apiKeySvc *service.APIKeyService,
	tokenStore port.ContainerTokenStore,
	callbackURL string,
	rt port.AgentRuntime,
) *action.AgentRunAction {
	agentCfg := action.AgentConfig{
		DefaultMemory: 4294967296,
		DefaultCPUs:   2.0,
		NetworkName:   testNetwork,
		LogTailLines:  50,
	}
	return action.NewAgentRunAction(
		f.containerMgr,
		f.logStreamer,
		f.eventPub,
		f.storyRepo,
		f.projectRepo,
		f.runRepo,
		f.environmentRepo,
		f.sidecarMgr,
		f.stackRepo,
		&mockTemplateRenderer{},
		f.costSvc,
		agentCfg,
		testLogger(),
		apiKeySvc,
		tokenStore,
		nil, // statusStore set by withStatusStore when callback mode is wanted
		callbackURL,
		action.WithAgentRuntime(rt),
	)
}

// newCallbackRuntimeAction builds an AgentRunAction wired for callback mode (a
// status store set so isCallbackMode is satisfied) with a runtime injected. Used
// by the anti-drift test that proves callback-wait is the outcome source.
func newCallbackRuntimeAction(
	f *agentRunFixture,
	statusStore port.CallbackStatusStore,
	tokenStore port.ContainerTokenStore,
	rt port.AgentRuntime,
) *action.AgentRunAction {
	agentCfg := action.AgentConfig{
		DefaultMemory: 4294967296,
		DefaultCPUs:   2.0,
		NetworkName:   testNetwork,
		LogTailLines:  50,
	}
	return action.NewAgentRunAction(
		f.containerMgr,
		f.logStreamer,
		f.eventPub,
		f.storyRepo,
		f.projectRepo,
		f.runRepo,
		f.environmentRepo,
		f.sidecarMgr,
		f.stackRepo,
		&mockTemplateRenderer{},
		f.costSvc,
		agentCfg,
		testLogger(),
		nil, // apiKeySvc — not needed: fixture runCtx has no UserID
		tokenStore,
		statusStore,
		"http://callback",
		action.WithAgentRuntime(rt),
	)
}

// mockTokenStore is a minimal port.ContainerTokenStore for the callback-mode
// runtime-dispatch tests. Create returns a fixed token and remembers it; Revoke
// records every token it was asked to revoke, so a test can prove the run revokes
// the SAME token Create minted (the dead-revoke fix).
type mockTokenStore struct {
	mu            sync.Mutex
	createdCalls  int
	createdTokens []string
	revokedTokens []string
}

const testMintedToken = "token-abc"

func (m *mockTokenStore) Create(_ context.Context, _, _, _ uuid.UUID, _ string, _ time.Duration) (string, error) {
	m.mu.Lock()
	m.createdCalls++
	m.createdTokens = append(m.createdTokens, testMintedToken)
	m.mu.Unlock()
	return testMintedToken, nil
}

func (m *mockTokenStore) Validate(_ context.Context, _ string) (*model.ContainerToken, error) {
	return nil, nil
}

func (m *mockTokenStore) Revoke(_ context.Context, token string) error {
	m.mu.Lock()
	m.revokedTokens = append(m.revokedTokens, token)
	m.mu.Unlock()
	return nil
}

func (m *mockTokenStore) createCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.createdCalls
}

func (m *mockTokenStore) revokedTokenList() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.revokedTokens))
	copy(out, m.revokedTokens)
	return out
}

// TestAgentRunAction_CallbackToken_MintedOnceAndRevoked is the Stage 3a proof that
// the dead token-revoke is fixed. A callback-mode run via the runtime path must:
//   - mint the callback token EXACTLY once (Create), and
//   - revoke EXACTLY that token once (Revoke) after the callback resolves.
//
// Before Stage 3a the revoke never fired: it resolved the token through a stub
// that always returned none, so the token only died at its 2h TTL.
func TestAgentRunAction_CallbackToken_MintedOnceAndRevoked(t *testing.T) {
	f := newAgentRunFixture(t)

	fakeRT := &mockAgentRuntime{}
	statusStore := &mockCallbackStatusStore{exitCode: 0}
	tokenStore := &mockTokenStore{}

	act := newCallbackRuntimeAction(f, statusStore, tokenStore, fakeRT)

	runCtx := f.newRunContext()
	runCtx.Metadata["runtime_kind"] = model.RuntimeKindClaudeCode

	if err := act.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	// Mint exactly once.
	if got := tokenStore.createCount(); got != 1 {
		t.Fatalf("expected Create to be called once, got %d", got)
	}

	// Revoke exactly once, with the SAME token Create minted.
	revoked := tokenStore.revokedTokenList()
	if len(revoked) != 1 {
		t.Fatalf("expected exactly 1 Revoke call, got %d (%v)", len(revoked), revoked)
	}
	if revoked[0] != testMintedToken {
		t.Errorf("expected Revoke to use the minted token %q, got %q", testMintedToken, revoked[0])
	}

	// The minted token must also reach the harness via RunSpec.Env (AUTH_TOKEN) and
	// the typed RunSpec.Callback mirror.
	spec := fakeRT.lastSpec(t)
	if got := envValue(spec.Env, "AUTH_TOKEN"); got != testMintedToken {
		t.Errorf("expected AUTH_TOKEN=%q in RunSpec.Env, got %q", testMintedToken, got)
	}
	if spec.Callback == nil {
		t.Fatal("expected RunSpec.Callback to be set in callback mode")
	}
	if spec.Callback.AuthToken != testMintedToken {
		t.Errorf("expected RunSpec.Callback.AuthToken=%q, got %q", testMintedToken, spec.Callback.AuthToken)
	}
	if spec.Callback.RunID != f.runID || spec.Callback.StepID != f.stepID {
		t.Errorf("RunSpec.Callback ids mismatch: run=%s step=%s", spec.Callback.RunID, spec.Callback.StepID)
	}
}

// TestAgentRunAction_CallbackToken_RevokedOnWaitError proves the revoke fires even
// when WaitForStatus FAILS (error / timeout / cancel). The revoke is deferred
// before the wait, so a failed run still revokes its token with the SAME value
// Create minted — the token never lingers to its 2h TTL on the error path.
func TestAgentRunAction_CallbackToken_RevokedOnWaitError(t *testing.T) {
	f := newAgentRunFixture(t)

	fakeRT := &mockAgentRuntime{}
	// WaitForStatus returns an error: the run must fail, yet the token is revoked.
	statusStore := &mockCallbackStatusStore{waitErr: fmt.Errorf("status wait timed out")}
	tokenStore := &mockTokenStore{}

	act := newCallbackRuntimeAction(f, statusStore, tokenStore, fakeRT)

	runCtx := f.newRunContext()
	runCtx.Metadata["runtime_kind"] = model.RuntimeKindClaudeCode

	if err := act.Execute(context.Background(), runCtx); err == nil {
		t.Fatal("expected error from WaitForStatus failure, got nil")
	}

	// Minted once.
	if got := tokenStore.createCount(); got != 1 {
		t.Fatalf("expected Create to be called once, got %d", got)
	}

	// Revoked once with the minted token, despite the wait error (deferred revoke).
	revoked := tokenStore.revokedTokenList()
	if len(revoked) != 1 {
		t.Fatalf("expected exactly 1 Revoke call on the error path, got %d (%v)", len(revoked), revoked)
	}
	if revoked[0] != testMintedToken {
		t.Errorf("expected Revoke to use the minted token %q on the error path, got %q", testMintedToken, revoked[0])
	}
}

// TestAgentRunAction_AuditsPromptOnEventBus proves the prompt is auditable on the
// substrate-agnostic event bus: a run publishes exactly one LogEvent of Type
// "prompt" whose Message is the rendered prompt, independent of the substrate.
func TestAgentRunAction_AuditsPromptOnEventBus(t *testing.T) {
	f := newAgentRunFixture(t)
	runCtx := f.newRunContext()

	if err := f.action.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// The mockTemplateRenderer renders "rendered: <first 20 chars of template>".
	wantPrompt := "rendered: " + runCtx.Metadata["template_content"].(string)[:20]

	var promptEvents []model.LogEvent
	for _, e := range f.eventPub.getEvents() {
		var le model.LogEvent
		if err := json.Unmarshal(e.Payload, &le); err != nil {
			continue
		}
		if le.Type == "prompt" {
			promptEvents = append(promptEvents, le)
		}
	}

	if len(promptEvents) != 1 {
		t.Fatalf("expected exactly 1 prompt-audit event, got %d", len(promptEvents))
	}
	if promptEvents[0].Message != wantPrompt {
		t.Errorf("prompt-audit Message mismatch:\n got=%q\nwant=%q", promptEvents[0].Message, wantPrompt)
	}
}

// TestAgentRunAction_NoPromptAudit_WhenEmpty proves an empty rendered prompt is
// NOT audited (no empty event pollutes the bus).
func TestAgentRunAction_NoPromptAudit_WhenEmpty(t *testing.T) {
	f := newAgentRunFixture(t)
	runCtx := f.newRunContext()
	delete(runCtx.Metadata, "template_content") // empty prompt

	if err := f.action.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for _, e := range f.eventPub.getEvents() {
		var le model.LogEvent
		if err := json.Unmarshal(e.Payload, &le); err != nil {
			continue
		}
		if le.Type == "prompt" {
			t.Errorf("expected no prompt-audit event for an empty prompt, got %q", le.Message)
		}
	}
}
