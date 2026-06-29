package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure the repos implement their ports at compile time.
var (
	_ port.PlanningConnectorRepository = (*PlanningConnectorRepo)(nil)
	_ port.PlanningWriteBackRepository = (*PlanningWriteBackRepo)(nil)
)

// PlanningConnectorRepo persists one planning connector per project using sqlc.
type PlanningConnectorRepo struct {
	queries *Queries
}

// NewPlanningConnectorRepository creates a new PlanningConnectorRepo.
func NewPlanningConnectorRepository(queries *Queries) *PlanningConnectorRepo {
	return &PlanningConnectorRepo{queries: queries}
}

// Get returns the project's connector, or a not-found DomainError when absent.
func (r *PlanningConnectorRepo) Get(ctx context.Context, projectID uuid.UUID) (*model.PlanningConnector, error) {
	row, err := r.queries.GetPlanningConnector(ctx, projectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("planning_connector", projectID)
		}
		return nil, apperrors.NewInternal("failed to get planning connector", err)
	}
	return toDomainPlanningConnector(row)
}

// Upsert inserts or replaces the project's connector, returning the stored row.
func (r *PlanningConnectorRepo) Upsert(ctx context.Context, c *model.PlanningConnector) (*model.PlanningConnector, error) {
	doneOptions, err := json.Marshal(nonNilStrings(c.DoneOptions))
	if err != nil {
		return nil, apperrors.NewInternal("failed to marshal done_options", err)
	}
	statusMapping, err := json.Marshal(c.StatusMapping)
	if err != nil {
		return nil, apperrors.NewInternal("failed to marshal status_mapping", err)
	}
	row, err := r.queries.UpsertPlanningConnector(ctx, UpsertPlanningConnectorParams{
		ProjectID:        c.ProjectID,
		Source:           c.Source,
		ProjectUrl:       textFromStringPtr(c.ProjectURL),
		StatusField:      c.StatusField,
		DoneOptions:      doneOptions,
		EpicIssueType:    c.EpicIssueType,
		StatusMapping:    statusMapping,
		WritebackEnabled: c.WritebackEnabled,
		PostRunComment:   c.PostRunComment,
	})
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, apperrors.NewNotFound("project", c.ProjectID)
		}
		return nil, apperrors.NewInternal("failed to upsert planning connector", err)
	}
	return toDomainPlanningConnector(row)
}

// toDomainPlanningConnector maps a sqlc row to the domain model.
func toDomainPlanningConnector(c PlanningConnector) (*model.PlanningConnector, error) {
	var doneOptions []string
	if len(c.DoneOptions) > 0 {
		if err := json.Unmarshal(c.DoneOptions, &doneOptions); err != nil {
			return nil, apperrors.NewInternal("failed to unmarshal done_options", err)
		}
	}
	var mapping model.PlanningStatusMapping
	if len(c.StatusMapping) > 0 {
		if err := json.Unmarshal(c.StatusMapping, &mapping); err != nil {
			return nil, apperrors.NewInternal("failed to unmarshal status_mapping", err)
		}
	}
	return &model.PlanningConnector{
		ProjectID:        c.ProjectID,
		Source:           c.Source,
		ProjectURL:       stringPtrFromText(c.ProjectUrl),
		StatusField:      c.StatusField,
		DoneOptions:      doneOptions,
		EpicIssueType:    c.EpicIssueType,
		StatusMapping:    mapping,
		WritebackEnabled: c.WritebackEnabled,
		PostRunComment:   c.PostRunComment,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}, nil
}

// PlanningWriteBackRepo appends audit rows for every write-back attempt.
type PlanningWriteBackRepo struct {
	queries *Queries
}

// NewPlanningWriteBackRepository creates a new PlanningWriteBackRepo.
func NewPlanningWriteBackRepository(queries *Queries) *PlanningWriteBackRepo {
	return &PlanningWriteBackRepo{queries: queries}
}

// Create appends one audit row and returns the stored row.
func (r *PlanningWriteBackRepo) Create(ctx context.Context, w *model.PlanningWriteBack) (*model.PlanningWriteBack, error) {
	row, err := r.queries.CreatePlanningWriteBack(ctx, CreatePlanningWriteBackParams{
		ProjectID:      w.ProjectID,
		StoryID:        w.StoryID,
		RunID:          uuidFromPtr(w.RunID),
		Source:         textFromStringPtr(w.Source),
		ExternalID:     textFromStringPtr(w.ExternalID),
		InternalStatus: textFromStringPtr(w.InternalStatus),
		RemoteStatus:   textFromStringPtr(w.RemoteStatus),
		Success:        w.Success,
		ErrorCode:      textFromStringPtr(w.ErrorCode),
		ErrorMessage:   textFromStringPtr(w.ErrorMessage),
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to create planning write-back audit row", err)
	}
	return toDomainPlanningWriteBack(row), nil
}

// ListByStory returns a story's most recent write-back attempts (newest first).
func (r *PlanningWriteBackRepo) ListByStory(ctx context.Context, storyID uuid.UUID, limit int32) ([]*model.PlanningWriteBack, error) {
	rows, err := r.queries.ListPlanningWriteBacksByStory(ctx, ListPlanningWriteBacksByStoryParams{
		StoryID: storyID,
		Limit:   limit,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list planning write-backs", err)
	}
	out := make([]*model.PlanningWriteBack, len(rows))
	for i, row := range rows {
		out[i] = toDomainPlanningWriteBack(row)
	}
	return out, nil
}

func toDomainPlanningWriteBack(w PlanningWriteBack) *model.PlanningWriteBack {
	out := &model.PlanningWriteBack{
		ID:             w.ID,
		ProjectID:      w.ProjectID,
		StoryID:        w.StoryID,
		Source:         stringPtrFromText(w.Source),
		ExternalID:     stringPtrFromText(w.ExternalID),
		InternalStatus: stringPtrFromText(w.InternalStatus),
		RemoteStatus:   stringPtrFromText(w.RemoteStatus),
		Success:        w.Success,
		ErrorCode:      stringPtrFromText(w.ErrorCode),
		ErrorMessage:   stringPtrFromText(w.ErrorMessage),
		CreatedAt:      w.CreatedAt,
	}
	if w.RunID.Valid {
		runID := uuid.UUID(w.RunID.Bytes)
		out.RunID = &runID
	}
	return out
}
