package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// EpicRepository defines the interface for epic persistence operations.
type EpicRepository interface {
	Create(ctx context.Context, epic *model.Epic) (*model.Epic, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Epic, error)
	ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Epic, error)
	CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
	Update(ctx context.Context, epic *model.Epic) (*model.Epic, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// --- Planning import (provenance-aware upsert + source-guarded adoption) ---

	// GetBySourceRef resolves an epic by its stable provenance identity
	// (project, source, external_id). Returns a NotFound DomainError when absent.
	GetBySourceRef(ctx context.Context, projectID uuid.UUID, source, externalID string) (*model.Epic, error)
	// GetByName resolves an epic by its (project, name). Backs source-guarded
	// adoption so a markdown/github epic attaches to an existing same-name epic
	// instead of tripping epics_uq_project_name. Returns NotFound when absent.
	GetByName(ctx context.Context, projectID uuid.UUID, name string) (*model.Epic, error)
	// CreateFromImport inserts an epic with service-computed provenance + status.
	CreateFromImport(ctx context.Context, e *model.Epic) (*model.Epic, error)
	// UpdateFromImport overwrites the import-managed columns with the
	// service-computed merged values (preserve-on-absent description, promote-only
	// status, source-guarded provenance).
	UpdateFromImport(ctx context.Context, e *model.Epic) (*model.Epic, error)
}
