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
		CreatedAt: req.CreatedAt,
	}
	if req.DiffContent != nil {
		params.DiffContent = pgtype.Text{String: *req.DiffContent, Valid: true}
	}

	row, err := r.queries.CreateHITLRequest(ctx, params)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, apperrors.NewNotFound("run_step", req.RunStepID)
		}
		return nil, apperrors.NewInternal("failed to create HITL request", err)
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
		return nil, apperrors.NewInternal("failed to get HITL request by run step", err)
	}
	return toDomainHITLRequest(row), nil
}

// GetPendingByRunID returns the pending HITL request for a given run.
func (r *HITLRepo) GetPendingByRunID(ctx context.Context, runID uuid.UUID) (*model.HITLRequest, error) {
	row, err := r.queries.GetPendingHITLRequestByRunID(ctx, runID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("hitl_request", runID)
		}
		return nil, apperrors.NewInternal("failed to get pending HITL request by run", err)
	}
	return toDomainHITLRequest(row), nil
}

// UpdateStatus transitions a HITL request to approved or rejected.
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

// toDomainHITLRequest maps a sqlc-generated HitlRequest to a domain HITLRequest.
func toDomainHITLRequest(r HitlRequest) *model.HITLRequest {
	req := &model.HITLRequest{
		ID:        r.ID,
		RunStepID: r.RunStepID,
		GateType:  r.GateType,
		Status:    model.HITLStatus(r.Status),
		CreatedAt: r.CreatedAt,
	}
	if r.DiffContent.Valid {
		req.DiffContent = &r.DiffContent.String
	}
	if r.ResolvedAt.Valid {
		req.ResolvedAt = &r.ResolvedAt.Time
	}
	if r.ResolvedBy.Valid {
		id := uuid.UUID(r.ResolvedBy.Bytes)
		req.ResolvedBy = &id
	}
	if r.RejectionReason.Valid {
		req.RejectionReason = &r.RejectionReason.String
	}
	return req
}
