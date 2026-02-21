package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// EpicRunService provides business logic for epic run operations.
type EpicRunService struct {
	epicRunRepo port.EpicRunRepository
	storyRepo   port.StoryRepository
	epicRepo    port.EpicRepository
	scheduler   *SchedulerService
	executor    *ParallelGroupExecutor
	eventPub    port.EventPublisher
	logger      *slog.Logger
}

// NewEpicRunService creates a new EpicRunService.
func NewEpicRunService(
	epicRunRepo port.EpicRunRepository,
	storyRepo port.StoryRepository,
	epicRepo port.EpicRepository,
	scheduler *SchedulerService,
	executor *ParallelGroupExecutor,
	eventPub port.EventPublisher,
	logger *slog.Logger,
) *EpicRunService {
	return &EpicRunService{
		epicRunRepo: epicRunRepo,
		storyRepo:   storyRepo,
		epicRepo:    epicRepo,
		scheduler:   scheduler,
		executor:    executor,
		eventPub:    eventPub,
		logger:      logger,
	}
}

// LaunchEpicRun creates an epic run, validates the DAG, and launches async execution.
func (s *EpicRunService) LaunchEpicRun(ctx context.Context, projectID, epicID uuid.UUID) (*model.EpicRun, error) {
	// Verify epic exists and belongs to the project
	epic, err := s.epicRepo.GetByID(ctx, epicID)
	if err != nil {
		return nil, err
	}
	if epic.ProjectID != projectID {
		return nil, errors.NewNotFound("epic", epicID)
	}

	// Fetch all stories for the epic
	stories, err := s.storyRepo.ListByEpic(ctx, epicID, 10000, 0)
	if err != nil {
		return nil, err
	}
	if len(stories) == 0 {
		return nil, &errors.DomainError{
			Category: errors.CategoryValidation,
			Code:     "EPIC_HAS_NO_STORIES",
			Message:  "epic has no stories to run",
		}
	}

	// Dereference story pointers for DAG computation
	storyValues := make([]model.Story, len(stories))
	for i, sp := range stories {
		storyValues[i] = *sp
	}

	// Build DAG — returns DAG_CYCLE_DETECTED error on cycle
	dag, err := s.scheduler.BuildDAG(storyValues)
	if err != nil {
		return nil, err
	}

	// Create EpicRun record with status pending
	epicRun := &model.EpicRun{
		ProjectID: projectID,
		EpicID:    epicID,
		Status:    model.EpicRunStatusPending,
	}
	epicRun, err = s.epicRunRepo.CreateEpicRun(ctx, epicRun)
	if err != nil {
		return nil, err
	}

	// Insert one EpicRunStory row per story with correct group_index
	var allStories []model.EpicRunStory
	for groupIdx, group := range dag.Groups {
		for _, story := range group {
			ers := model.EpicRunStory{
				EpicRunID:  epicRun.ID,
				StoryID:    story.ID,
				GroupIndex: groupIdx,
				Status:     "pending",
			}
			if err := s.epicRunRepo.InsertEpicRunStory(ctx, ers); err != nil {
				return nil, err
			}
			allStories = append(allStories, ers)
		}
	}
	epicRun.Stories = allStories

	// Launch executor in a detached goroutine (fire-and-forget)
	go func() {
		detachedCtx := context.WithoutCancel(ctx)
		if execErr := s.executor.Execute(detachedCtx, epicRun, dag); execErr != nil {
			s.logger.Error("epic run failed", "epic_run_id", epicRun.ID, "error", execErr)
		}
	}()

	return epicRun, nil
}

// GetEpicRun retrieves an epic run by ID with its stories populated.
func (s *EpicRunService) GetEpicRun(ctx context.Context, id uuid.UUID) (*model.EpicRun, error) {
	epicRun, err := s.epicRunRepo.GetEpicRun(ctx, id)
	if err != nil {
		return nil, err
	}

	stories, err := s.epicRunRepo.ListEpicRunStories(ctx, id)
	if err != nil {
		return nil, err
	}
	epicRun.Stories = stories

	return epicRun, nil
}
