package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure EpicRepo implements port.EpicRepository at compile time.
var _ port.EpicRepository = (*EpicRepo)(nil)

// EpicRepo implements port.EpicRepository using sqlc-generated queries.
type EpicRepo struct {
	queries *Queries
}

// NewEpicRepo creates a new EpicRepo.
func NewEpicRepo(queries *Queries) *EpicRepo {
	return &EpicRepo{queries: queries}
}

func (r *EpicRepo) Create(ctx context.Context, epic *model.Epic) (*model.Epic, error) {
	params := CreateEpicParams{
		ProjectID:   epic.ProjectID,
		Name:        epic.Name,
		Description: textFromStringPtr(epic.Description),
		Status:      epic.Status,
	}

	row, err := r.queries.CreateEpic(ctx, params)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("epic", epic.Name)
		}
		return nil, apperrors.NewInternal("failed to create epic", err)
	}
	return toDomainEpic(row), nil
}

func (r *EpicRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Epic, error) {
	row, err := r.queries.GetEpic(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("epic", id)
		}
		return nil, apperrors.NewInternal("failed to get epic", err)
	}
	return toDomainEpic(row), nil
}

func (r *EpicRepo) ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Epic, error) {
	rows, err := r.queries.ListEpicsByProject(ctx, ListEpicsByProjectParams{
		ProjectID: projectID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list epics", err)
	}
	epics := make([]*model.Epic, len(rows))
	for i, row := range rows {
		epics[i] = toDomainEpic(row)
	}
	return epics, nil
}

func (r *EpicRepo) CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error) {
	count, err := r.queries.CountEpicsByProject(ctx, projectID)
	if err != nil {
		return 0, apperrors.NewInternal("failed to count epics", err)
	}
	return count, nil
}

func (r *EpicRepo) Update(ctx context.Context, epic *model.Epic) (*model.Epic, error) {
	params := UpdateEpicParams{
		ID:          epic.ID,
		Name:        textFromStringPtr(&epic.Name),
		Description: textFromStringPtr(epic.Description),
		Status:      pgtype.Text{String: epic.Status, Valid: true},
	}

	row, err := r.queries.UpdateEpic(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("epic", epic.ID)
		}
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("epic", epic.Name)
		}
		return nil, apperrors.NewInternal("failed to update epic", err)
	}
	return toDomainEpic(row), nil
}

func (r *EpicRepo) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.queries.DeleteEpic(ctx, id)
	if err != nil {
		return apperrors.NewInternal("failed to delete epic", err)
	}
	return nil
}

// toDomainEpic maps a sqlc-generated Epic to a domain Epic.
func toDomainEpic(e Epic) *model.Epic {
	epic := &model.Epic{
		ID:        e.ID,
		ProjectID: e.ProjectID,
		Name:      e.Name,
		Status:    e.Status,
		Source:    e.Source,
		SyncedAt:  timeFromTimestamptz(e.SyncedAt),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
	if e.Description.Valid {
		epic.Description = &e.Description.String
	}
	if e.ExternalID.Valid {
		epic.ExternalID = &e.ExternalID.String
	}
	if e.SourceUrl.Valid {
		epic.SourceURL = &e.SourceUrl.String
	}
	return epic
}

// GetBySourceRef resolves an epic by (project, source, external_id).
func (r *EpicRepo) GetBySourceRef(ctx context.Context, projectID uuid.UUID, source, externalID string) (*model.Epic, error) {
	row, err := r.queries.GetEpicBySourceRef(ctx, GetEpicBySourceRefParams{
		ProjectID:  projectID,
		Source:     source,
		ExternalID: pgtype.Text{String: externalID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("epic", externalID)
		}
		return nil, apperrors.NewInternal("failed to get epic by source ref", err)
	}
	return toDomainEpic(row), nil
}

// GetByName resolves an epic by (project, name) — backs source-guarded adoption.
func (r *EpicRepo) GetByName(ctx context.Context, projectID uuid.UUID, name string) (*model.Epic, error) {
	row, err := r.queries.GetEpicByName(ctx, GetEpicByNameParams{
		ProjectID: projectID,
		Name:      name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("epic", name)
		}
		return nil, apperrors.NewInternal("failed to get epic by name", err)
	}
	return toDomainEpic(row), nil
}

// CreateFromImport inserts an epic with service-computed provenance + status.
func (r *EpicRepo) CreateFromImport(ctx context.Context, epic *model.Epic) (*model.Epic, error) {
	row, err := r.queries.CreateEpicFromImport(ctx, CreateEpicFromImportParams{
		ProjectID:   epic.ProjectID,
		Name:        epic.Name,
		Description: textFromStringPtr(epic.Description),
		Status:      epic.Status,
		Source:      epic.Source,
		ExternalID:  textFromStringPtr(epic.ExternalID),
		SourceUrl:   textFromStringPtr(epic.SourceURL),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("epic", epic.Name)
		}
		return nil, apperrors.NewInternal("failed to create epic from import", err)
	}
	return toDomainEpic(row), nil
}

// UpdateFromImport overwrites the import-managed columns with merged values.
func (r *EpicRepo) UpdateFromImport(ctx context.Context, epic *model.Epic) (*model.Epic, error) {
	row, err := r.queries.UpdateEpicFromImport(ctx, UpdateEpicFromImportParams{
		ID:          epic.ID,
		Name:        epic.Name,
		Description: textFromStringPtr(epic.Description),
		Status:      epic.Status,
		Source:      epic.Source,
		ExternalID:  textFromStringPtr(epic.ExternalID),
		SourceUrl:   textFromStringPtr(epic.SourceURL),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("epic", epic.ID)
		}
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("epic", epic.Name)
		}
		return nil, apperrors.NewInternal("failed to update epic from import", err)
	}
	return toDomainEpic(row), nil
}
