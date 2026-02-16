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
		Status:      string(epic.Status),
	}

	row, err := r.queries.CreateEpic(ctx, params)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("epic", epic.Name)
		}
		if isForeignKeyViolation(err) {
			return nil, apperrors.NewNotFound("project", epic.ProjectID)
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
		Status:      pgtype.Text{String: string(epic.Status), Valid: true},
	}

	row, err := r.queries.UpdateEpic(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("epic", epic.ID)
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
		Status:    model.EpicStatus(e.Status),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
	if e.Description.Valid {
		epic.Description = &e.Description.String
	}
	return epic
}
