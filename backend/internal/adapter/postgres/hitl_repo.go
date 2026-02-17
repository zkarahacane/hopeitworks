package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure HITLRepo implements port.HITLRepository at compile time.
var _ port.HITLRepository = (*HITLRepo)(nil)

// HITLRepo implements port.HITLRepository using sqlc-generated queries.
type HITLRepo struct {
	queries *Queries
}

// NewHITLRepo creates a new HITLRepo.
func NewHITLRepo(queries *Queries) *HITLRepo {
	return &HITLRepo{queries: queries}
}

// Create persists a new HITL request.
func (r *HITLRepo) Create(ctx context.Context, req *model.HITLRequest) (*model.HITLRequest, error) {
	params := CreateHITLRequestParams{
		ID:        req.ID,
		RunStepID: req.RunStepID,
		GateType:  req.GateType,
		Status:    string(req.Status),
	}
	if req.DiffContent != nil {
		params.DiffContent = pgtype.Text{String: *req.DiffContent, Valid: true}
	}

	row, err := r.queries.CreateHITLRequest(ctx, params)
	if err != nil {
		return nil, apperrors.NewInternal("failed to create HITL request", err)
	}
	return toDomainHITLRequest(row), nil
}

// GetByID returns a HITL request by its ID.
func (r *HITLRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.HITLRequest, error) {
	row, err := r.queries.GetHITLRequest(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("hitl_request", id)
		}
		return nil, apperrors.NewInternal("failed to get HITL request", err)
	}
	return toDomainHITLRequest(row), nil
}

// GetByRunStepID returns the HITL request for the given run step.
func (r *HITLRepo) GetByRunStepID(ctx context.Context, runStepID uuid.UUID) (*model.HITLRequest, error) {
	row, err := r.queries.GetHITLRequestByRunStepID(ctx, runStepID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("hitl_request", runStepID)
		}
		return nil, apperrors.NewInternal("failed to get HITL request by run step ID", err)
	}
	return toDomainHITLRequest(row), nil
}

// UpdateStatus transitions the HITL request to approved or rejected.
func (r *HITLRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.HITLStatus, resolvedBy *uuid.UUID, rejectionReason *string, resolvedAt time.Time) (*model.HITLRequest, error) {
	params := UpdateHITLRequestStatusParams{
		ID:         id,
		Status:     string(status),
		ResolvedAt: pgtype.Timestamptz{Time: resolvedAt, Valid: true},
	}
	if resolvedBy != nil {
		params.ResolvedBy = pgtype.UUID{Bytes: *resolvedBy, Valid: true}
	}
	if rejectionReason != nil {
		params.RejectionReason = pgtype.Text{String: *rejectionReason, Valid: true}
	}

	row, err := r.queries.UpdateHITLRequestStatus(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("hitl_request", id)
		}
		return nil, apperrors.NewInternal("failed to update HITL request status", err)
	}
	return toDomainHITLRequest(row), nil
}

// ListPendingByProject returns all pending HITL requests for a project.
func (r *HITLRepo) ListPendingByProject(ctx context.Context, projectID uuid.UUID) ([]*model.PendingHITLRequest, error) {
	rows, err := r.queries.ListPendingHITLRequestsByProject(ctx, projectID)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list pending HITL requests", err)
	}
	result := make([]*model.PendingHITLRequest, len(rows))
	for i, row := range rows {
		result[i] = &model.PendingHITLRequest{
			ID:        row.ID,
			RunID:     row.RunID,
			StepID:    row.StepID,
			StoryKey:  row.StoryKey,
			CreatedAt: row.CreatedAt,
		}
	}
	return result, nil
}

// CountPendingByProject returns the count of pending HITL requests for a project.
func (r *HITLRepo) CountPendingByProject(ctx context.Context, projectID uuid.UUID) (int64, error) {
	count, err := r.queries.CountPendingHITLRequestsByProject(ctx, projectID)
	if err != nil {
		return 0, apperrors.NewInternal("failed to count pending HITL requests", err)
	}
	return count, nil
}

// toDomainHITLRequest maps a sqlc-generated HitlRequest to a domain HITLRequest.
func toDomainHITLRequest(row HitlRequest) *model.HITLRequest {
	req := &model.HITLRequest{
		ID:        row.ID,
		RunStepID: row.RunStepID,
		GateType:  row.GateType,
		Status:    model.HITLStatus(row.Status),
		CreatedAt: row.CreatedAt,
	}
	if row.DiffContent.Valid {
		req.DiffContent = &row.DiffContent.String
	}
	if row.ResolvedAt.Valid {
		req.ResolvedAt = &row.ResolvedAt.Time
	}
	if row.ResolvedBy.Valid {
		id := uuid.UUID(row.ResolvedBy.Bytes)
		req.ResolvedBy = &id
	}
	if row.RejectionReason.Valid {
		req.RejectionReason = &row.RejectionReason.String
	}
	return req
}
