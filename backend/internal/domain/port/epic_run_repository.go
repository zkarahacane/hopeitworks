package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// EpicRunRepository defines the interface for epic run persistence operations.
type EpicRunRepository interface {
	// CreateEpicRun creates a new epic run record.
	CreateEpicRun(ctx context.Context, run *model.EpicRun) (*model.EpicRun, error)
	// GetEpicRun retrieves an epic run by ID.
	GetEpicRun(ctx context.Context, id uuid.UUID) (*model.EpicRun, error)
	// UpdateEpicRunStatus updates the status and optional completion time.
	UpdateEpicRunStatus(ctx context.Context, id uuid.UUID, status model.EpicRunStatus, completedAt *time.Time) (*model.EpicRun, error)
	// InsertEpicRunStory inserts a story association for an epic run.
	InsertEpicRunStory(ctx context.Context, story model.EpicRunStory) error
	// UpdateEpicRunStoryStatus updates the status and run ID for a story in an epic run.
	UpdateEpicRunStoryStatus(ctx context.Context, epicRunID, storyID uuid.UUID, status string, runID *uuid.UUID) error
	// ListEpicRunStories returns all stories for an epic run ordered by group index.
	ListEpicRunStories(ctx context.Context, epicRunID uuid.UUID) ([]model.EpicRunStory, error)
}
