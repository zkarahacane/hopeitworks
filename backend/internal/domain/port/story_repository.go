package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// StoryRepository defines the interface for story persistence operations.
type StoryRepository interface {
	Create(ctx context.Context, story *model.Story) (*model.Story, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error)
	GetByKey(ctx context.Context, projectID uuid.UUID, key string) (*model.Story, error)
	ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Story, error)
	ListByStatus(ctx context.Context, projectID uuid.UUID, statuses []string, limit, offset int32) ([]*model.Story, error)
	ListByEpic(ctx context.Context, epicID uuid.UUID, limit, offset int32) ([]*model.Story, error)
	CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
	CountByStatus(ctx context.Context, projectID uuid.UUID, statuses []string) (int64, error)
	// CountByEpicGroupedByStatus returns story counts per lifecycle status for an epic
	// in a single GROUP BY query.
	CountByEpicGroupedByStatus(ctx context.Context, epicID uuid.UUID) (model.StoryCounts, error)
	Update(ctx context.Context, story *model.Story) (*model.Story, error)
	// UpdateStoryCurrentStage sets the story's current_stage. A nil currentStage
	// clears the stage (NULL). Advanced by the executor at stage boundaries.
	UpdateStoryCurrentStage(ctx context.Context, id uuid.UUID, currentStage *string) (*model.Story, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// --- Planning import (provenance-aware upsert) ---

	// GetBySourceRef resolves a remote-sourced story by its stable provenance
	// identity (project, source, external_id). Returns a NotFound DomainError when
	// absent. Markdown resolution uses GetByKey; this is the github_projects path.
	GetBySourceRef(ctx context.Context, projectID uuid.UUID, source, externalID string) (*model.Story, error)
	// CreateFromImport inserts a story whose import-managed columns the service has
	// already computed. Never writes target_files/current_stage (executor-owned).
	CreateFromImport(ctx context.Context, s *model.Story) (*model.Story, error)
	// UpdateFromImport overwrites the import-managed columns of an UNLOCKED story
	// with the service-computed merged values (advances last_import_hash).
	UpdateFromImport(ctx context.Context, s *model.Story) (*model.Story, error)
	// UpdateProvenanceOnly refreshes a LOCKED story's cosmetic title + provenance
	// only. It deliberately does NOT advance last_import_hash.
	UpdateProvenanceOnly(ctx context.Context, s *model.Story) (*model.Story, error)

	// SetWritebackStatus sets the story's outbound write-back state
	// (disabled|pending|synced|failed). Managed solely by the write-back path.
	SetWritebackStatus(ctx context.Context, id uuid.UUID, status string) error
}
