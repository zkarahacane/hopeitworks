package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockEpicRunRepo implements port.EpicRunRepository for testing.
type mockEpicRunRepo struct {
	createEpicRunFn            func(ctx context.Context, run *model.EpicRun) (*model.EpicRun, error)
	getEpicRunFn               func(ctx context.Context, id uuid.UUID) (*model.EpicRun, error)
	updateEpicRunStatusFn      func(ctx context.Context, id uuid.UUID, status model.EpicRunStatus, completedAt *time.Time) (*model.EpicRun, error)
	insertEpicRunStoryFn       func(ctx context.Context, story model.EpicRunStory) error
	updateEpicRunStoryStatusFn func(ctx context.Context, epicRunID, storyID uuid.UUID, status string, runID *uuid.UUID) error
	listEpicRunStoriesFn       func(ctx context.Context, epicRunID uuid.UUID) ([]model.EpicRunStory, error)
}

func (m *mockEpicRunRepo) CreateEpicRun(ctx context.Context, run *model.EpicRun) (*model.EpicRun, error) {
	if m.createEpicRunFn != nil {
		return m.createEpicRunFn(ctx, run)
	}
	run.ID = uuid.New()
	run.CreatedAt = time.Now()
	return run, nil
}

func (m *mockEpicRunRepo) GetEpicRun(ctx context.Context, id uuid.UUID) (*model.EpicRun, error) {
	if m.getEpicRunFn != nil {
		return m.getEpicRunFn(ctx, id)
	}
	return nil, errors.NewNotFound("epic_run", id)
}

func (m *mockEpicRunRepo) UpdateEpicRunStatus(ctx context.Context, id uuid.UUID, status model.EpicRunStatus, completedAt *time.Time) (*model.EpicRun, error) {
	if m.updateEpicRunStatusFn != nil {
		return m.updateEpicRunStatusFn(ctx, id, status, completedAt)
	}
	return &model.EpicRun{ID: id, Status: status}, nil
}

func (m *mockEpicRunRepo) InsertEpicRunStory(ctx context.Context, story model.EpicRunStory) error {
	if m.insertEpicRunStoryFn != nil {
		return m.insertEpicRunStoryFn(ctx, story)
	}
	return nil
}

func (m *mockEpicRunRepo) UpdateEpicRunStoryStatus(ctx context.Context, epicRunID, storyID uuid.UUID, status string, runID *uuid.UUID) error {
	if m.updateEpicRunStoryStatusFn != nil {
		return m.updateEpicRunStoryStatusFn(ctx, epicRunID, storyID, status, runID)
	}
	return nil
}

func (m *mockEpicRunRepo) ListEpicRunStories(ctx context.Context, epicRunID uuid.UUID) ([]model.EpicRunStory, error) {
	if m.listEpicRunStoriesFn != nil {
		return m.listEpicRunStoriesFn(ctx, epicRunID)
	}
	return nil, nil
}

// mockEpicRepoForEpicRun implements port.EpicRepository for testing.
type mockEpicRepoForEpicRun struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Epic, error)
}

func (m *mockEpicRepoForEpicRun) Create(_ context.Context, epic *model.Epic) (*model.Epic, error) {
	return epic, nil
}
func (m *mockEpicRepoForEpicRun) GetByID(ctx context.Context, id uuid.UUID) (*model.Epic, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.NewNotFound("epic", id)
}
func (m *mockEpicRepoForEpicRun) ListByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Epic, error) {
	return nil, nil
}
func (m *mockEpicRepoForEpicRun) CountByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockEpicRepoForEpicRun) Update(_ context.Context, epic *model.Epic) (*model.Epic, error) {
	return epic, nil
}
func (m *mockEpicRepoForEpicRun) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockEpicRepoForEpicRun) GetBySourceRef(_ context.Context, _ uuid.UUID, _, _ string) (*model.Epic, error) {
	return nil, nil
}
func (m *mockEpicRepoForEpicRun) GetByName(_ context.Context, _ uuid.UUID, _ string) (*model.Epic, error) {
	return nil, nil
}
func (m *mockEpicRepoForEpicRun) CreateFromImport(_ context.Context, e *model.Epic) (*model.Epic, error) {
	return e, nil
}
func (m *mockEpicRepoForEpicRun) UpdateFromImport(_ context.Context, e *model.Epic) (*model.Epic, error) {
	return e, nil
}

// mockStoryRepoForEpicRun implements port.StoryRepository for testing.
type mockStoryRepoForEpicRun struct {
	listByEpicFn func(ctx context.Context, epicID uuid.UUID, limit, offset int32) ([]*model.Story, error)
}

func (m *mockStoryRepoForEpicRun) Create(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *mockStoryRepoForEpicRun) GetByID(_ context.Context, id uuid.UUID) (*model.Story, error) {
	return nil, errors.NewNotFound("story", id)
}
func (m *mockStoryRepoForEpicRun) GetByKey(_ context.Context, _ uuid.UUID, _ string) (*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForEpicRun) ListByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForEpicRun) ListByStatus(_ context.Context, _ uuid.UUID, _ []string, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForEpicRun) ListByEpic(ctx context.Context, epicID uuid.UUID, limit, offset int32) ([]*model.Story, error) {
	if m.listByEpicFn != nil {
		return m.listByEpicFn(ctx, epicID, limit, offset)
	}
	return nil, nil
}
func (m *mockStoryRepoForEpicRun) CountByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockStoryRepoForEpicRun) CountByStatus(_ context.Context, _ uuid.UUID, _ []string) (int64, error) {
	return 0, nil
}
func (m *mockStoryRepoForEpicRun) Update(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *mockStoryRepoForEpicRun) UpdateStoryCurrentStage(_ context.Context, id uuid.UUID, currentStage *string) (*model.Story, error) {
	return &model.Story{ID: id, CurrentStage: currentStage}, nil
}
func (m *mockStoryRepoForEpicRun) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockStoryRepoForEpicRun) GetBySourceRef(_ context.Context, _ uuid.UUID, _, _ string) (*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForEpicRun) CreateFromImport(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *mockStoryRepoForEpicRun) UpdateFromImport(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *mockStoryRepoForEpicRun) UpdateProvenanceOnly(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}

func TestLaunchEpicRun_HappyPath(t *testing.T) {
	projectID := uuid.New()
	epicID := uuid.New()

	stories := []*model.Story{
		{ID: uuid.New(), ProjectID: projectID, EpicID: &epicID, Key: "S-01", Title: "Story 1", Status: "backlog"},
		{ID: uuid.New(), ProjectID: projectID, EpicID: &epicID, Key: "S-02", Title: "Story 2", Status: "backlog"},
	}

	var insertedStories []model.EpicRunStory
	epicRunRepo := &mockEpicRunRepo{
		insertEpicRunStoryFn: func(_ context.Context, story model.EpicRunStory) error {
			insertedStories = append(insertedStories, story)
			return nil
		},
	}
	epicRepo := &mockEpicRepoForEpicRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Epic, error) {
			if id == epicID {
				return &model.Epic{ID: epicID, ProjectID: projectID, Name: "Test Epic"}, nil
			}
			return nil, errors.NewNotFound("epic", id)
		},
	}
	storyRepo := &mockStoryRepoForEpicRun{
		listByEpicFn: func(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
			return stories, nil
		},
	}

	scheduler := NewSchedulerService()
	eventPub := newMockEventPublisher()
	logger := testLogger()

	// Create a stub executor with real dependencies so the background goroutine doesn't crash
	runRepo := &mockRunRepo{
		createRunFn: func(_ context.Context, run *model.Run) (*model.Run, error) {
			run.ID = uuid.New()
			return run, nil
		},
		getActiveRunByStoryFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
			return nil, nil
		},
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, Status: model.RunStatusPending}, nil
		},
		listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return []*model.RunStep{}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _ *time.Time, _ *time.Time, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, Status: status}, nil
		},
	}
	pRepo := newMockProjectRepoForService()
	pRepo.projects[projectID] = &model.Project{ID: projectID, Name: "Test"}
	mockPCR := &mockPipelineConfigRepoForRun{
		getByProjectIDFn: func(_ context.Context, _ uuid.UUID) (*model.PipelineConfig, error) {
			return &model.PipelineConfig{
				ProjectID:  projectID,
				ConfigYAML: `steps: [{name: "dev", action_type: "implement"}]`,
			}, nil
		},
	}
	sRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, ProjectID: projectID, Key: "S-test", Status: "backlog"}, nil
		},
	}
	runSvc := NewRunService(runRepo, pRepo, sRepo, mockPCR, &mockJobQueue{})
	actionReg := newMockActionRegistry()
	pipeExec := NewPipelineExecutor(runRepo, sRepo, actionReg, eventPub, logger)
	stubExecutor := NewParallelGroupExecutor(epicRunRepo, runSvc, pipeExec, eventPub, logger)

	svc := NewEpicRunService(epicRunRepo, storyRepo, epicRepo, scheduler, stubExecutor, eventPub, logger)

	epicRun, err := svc.LaunchEpicRun(context.Background(), projectID, epicID)
	if err != nil {
		t.Fatalf("LaunchEpicRun failed: %v", err)
	}

	if epicRun.Status != model.EpicRunStatusPending {
		t.Errorf("expected status pending, got %s", epicRun.Status)
	}
	if epicRun.ProjectID != projectID {
		t.Errorf("expected projectID %s, got %s", projectID, epicRun.ProjectID)
	}
	if epicRun.EpicID != epicID {
		t.Errorf("expected epicID %s, got %s", epicID, epicRun.EpicID)
	}
	if len(insertedStories) != 2 {
		t.Errorf("expected 2 inserted stories, got %d", len(insertedStories))
	}

	// Give the background goroutine time to complete
	time.Sleep(100 * time.Millisecond)
}

func TestLaunchEpicRun_EpicNotFound(t *testing.T) {
	projectID := uuid.New()
	epicID := uuid.New()

	epicRunRepo := &mockEpicRunRepo{}
	epicRepo := &mockEpicRepoForEpicRun{}
	storyRepo := &mockStoryRepoForEpicRun{}
	scheduler := NewSchedulerService()
	eventPub := newMockEventPublisher()
	logger := testLogger()
	stubExecutor := NewParallelGroupExecutor(epicRunRepo, nil, nil, eventPub, logger)

	svc := NewEpicRunService(epicRunRepo, storyRepo, epicRepo, scheduler, stubExecutor, eventPub, logger)

	_, err := svc.LaunchEpicRun(context.Background(), projectID, epicID)
	if err == nil {
		t.Fatal("expected error for non-existent epic")
	}
	domErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domErr.Category != errors.CategoryNotFound {
		t.Errorf("expected CategoryNotFound, got %s", domErr.Category)
	}
}

func TestLaunchEpicRun_EpicWrongProject(t *testing.T) {
	projectID := uuid.New()
	otherProjectID := uuid.New()
	epicID := uuid.New()

	epicRunRepo := &mockEpicRunRepo{}
	epicRepo := &mockEpicRepoForEpicRun{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Epic, error) {
			return &model.Epic{ID: epicID, ProjectID: otherProjectID, Name: "Test Epic"}, nil
		},
	}
	storyRepo := &mockStoryRepoForEpicRun{}
	scheduler := NewSchedulerService()
	eventPub := newMockEventPublisher()
	logger := testLogger()
	stubExecutor := NewParallelGroupExecutor(epicRunRepo, nil, nil, eventPub, logger)

	svc := NewEpicRunService(epicRunRepo, storyRepo, epicRepo, scheduler, stubExecutor, eventPub, logger)

	_, err := svc.LaunchEpicRun(context.Background(), projectID, epicID)
	if err == nil {
		t.Fatal("expected error for epic in wrong project")
	}
	domErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domErr.Category != errors.CategoryNotFound {
		t.Errorf("expected CategoryNotFound, got %s", domErr.Category)
	}
}

func TestLaunchEpicRun_NoStories(t *testing.T) {
	projectID := uuid.New()
	epicID := uuid.New()

	epicRunRepo := &mockEpicRunRepo{}
	epicRepo := &mockEpicRepoForEpicRun{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Epic, error) {
			return &model.Epic{ID: epicID, ProjectID: projectID, Name: "Test Epic"}, nil
		},
	}
	storyRepo := &mockStoryRepoForEpicRun{
		listByEpicFn: func(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
			return []*model.Story{}, nil
		},
	}
	scheduler := NewSchedulerService()
	eventPub := newMockEventPublisher()
	logger := testLogger()
	stubExecutor := NewParallelGroupExecutor(epicRunRepo, nil, nil, eventPub, logger)

	svc := NewEpicRunService(epicRunRepo, storyRepo, epicRepo, scheduler, stubExecutor, eventPub, logger)

	_, err := svc.LaunchEpicRun(context.Background(), projectID, epicID)
	if err == nil {
		t.Fatal("expected error for epic with no stories")
	}
	domErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domErr.Code != "EPIC_HAS_NO_STORIES" {
		t.Errorf("expected EPIC_HAS_NO_STORIES code, got %s", domErr.Code)
	}
}

func TestLaunchEpicRun_CycleDetected(t *testing.T) {
	projectID := uuid.New()
	epicID := uuid.New()

	stories := []*model.Story{
		{ID: uuid.New(), ProjectID: projectID, EpicID: &epicID, Key: "S-01", Title: "Story 1", Status: "backlog", DependsOn: []string{"S-02"}},
		{ID: uuid.New(), ProjectID: projectID, EpicID: &epicID, Key: "S-02", Title: "Story 2", Status: "backlog", DependsOn: []string{"S-01"}},
	}

	epicRunRepo := &mockEpicRunRepo{}
	epicRepo := &mockEpicRepoForEpicRun{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*model.Epic, error) {
			return &model.Epic{ID: epicID, ProjectID: projectID, Name: "Test Epic"}, nil
		},
	}
	storyRepo := &mockStoryRepoForEpicRun{
		listByEpicFn: func(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
			return stories, nil
		},
	}
	scheduler := NewSchedulerService()
	eventPub := newMockEventPublisher()
	logger := testLogger()
	stubExecutor := NewParallelGroupExecutor(epicRunRepo, nil, nil, eventPub, logger)

	svc := NewEpicRunService(epicRunRepo, storyRepo, epicRepo, scheduler, stubExecutor, eventPub, logger)

	_, err := svc.LaunchEpicRun(context.Background(), projectID, epicID)
	if err == nil {
		t.Fatal("expected error for cycle in dependencies")
	}
	domErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domErr.Code != "DAG_CYCLE_DETECTED" {
		t.Errorf("expected DAG_CYCLE_DETECTED code, got %s", domErr.Code)
	}
}

func TestGetEpicRun_HappyPath(t *testing.T) {
	epicRunID := uuid.New()
	projectID := uuid.New()
	epicID := uuid.New()
	storyID := uuid.New()

	epicRunRepo := &mockEpicRunRepo{
		getEpicRunFn: func(_ context.Context, id uuid.UUID) (*model.EpicRun, error) {
			if id == epicRunID {
				return &model.EpicRun{
					ID:        epicRunID,
					ProjectID: projectID,
					EpicID:    epicID,
					Status:    model.EpicRunStatusRunning,
					CreatedAt: time.Now(),
				}, nil
			}
			return nil, errors.NewNotFound("epic_run", id)
		},
		listEpicRunStoriesFn: func(_ context.Context, id uuid.UUID) ([]model.EpicRunStory, error) {
			return []model.EpicRunStory{
				{EpicRunID: id, StoryID: storyID, GroupIndex: 0, Status: "completed"},
			}, nil
		},
	}
	storyRepo := &mockStoryRepoForEpicRun{}
	epicRepo := &mockEpicRepoForEpicRun{}
	scheduler := NewSchedulerService()
	eventPub := newMockEventPublisher()
	logger := testLogger()
	stubExecutor := NewParallelGroupExecutor(epicRunRepo, nil, nil, eventPub, logger)

	svc := NewEpicRunService(epicRunRepo, storyRepo, epicRepo, scheduler, stubExecutor, eventPub, logger)

	epicRun, err := svc.GetEpicRun(context.Background(), epicRunID)
	if err != nil {
		t.Fatalf("GetEpicRun failed: %v", err)
	}

	if epicRun.ID != epicRunID {
		t.Errorf("expected ID %s, got %s", epicRunID, epicRun.ID)
	}
	if len(epicRun.Stories) != 1 {
		t.Errorf("expected 1 story, got %d", len(epicRun.Stories))
	}
	if epicRun.Stories[0].Status != "completed" {
		t.Errorf("expected status completed, got %s", epicRun.Stories[0].Status)
	}
}

func TestGetEpicRun_NotFound(t *testing.T) {
	epicRunRepo := &mockEpicRunRepo{}
	storyRepo := &mockStoryRepoForEpicRun{}
	epicRepo := &mockEpicRepoForEpicRun{}
	scheduler := NewSchedulerService()
	eventPub := newMockEventPublisher()
	logger := testLogger()
	stubExecutor := NewParallelGroupExecutor(epicRunRepo, nil, nil, eventPub, logger)

	svc := NewEpicRunService(epicRunRepo, storyRepo, epicRepo, scheduler, stubExecutor, eventPub, logger)

	_, err := svc.GetEpicRun(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent epic run")
	}
}

func (m *mockStoryRepoForEpicRun) CountByEpicGroupedByStatus(_ context.Context, _ uuid.UUID) (model.StoryCounts, error) {
	return model.StoryCounts{}, nil
}

func (m *mockStoryRepoForEpicRun) SetWritebackStatus(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
