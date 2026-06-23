package postgres

import (
	"context"
	"encoding/json"
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
	if req.Message != nil {
		params.Message = pgtype.Text{String: *req.Message, Valid: true}
	}
	if req.HaltReason != nil {
		raw, marshalErr := json.Marshal(req.HaltReason)
		if marshalErr != nil {
			return nil, apperrors.NewInternal("failed to marshal halt reason", marshalErr)
		}
		params.HaltReason = raw
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

// UpdateResolution transitions a probe_halt HITL request to a terminal status
// and records the enriched resolution action taken alongside the resolving human.
func (r *HITLRepo) UpdateResolution(ctx context.Context, id uuid.UUID, status model.HITLStatus, resolvedBy *uuid.UUID, action string, resolvedAt time.Time) (*model.HITLRequest, error) {
	params := UpdateHITLRequestStatusParams{
		ID:               id,
		Status:           string(status),
		ResolvedAt:       pgtype.Timestamptz{Time: resolvedAt, Valid: true},
		ResolutionAction: pgtype.Text{String: action, Valid: action != ""},
	}
	if resolvedBy != nil {
		params.ResolvedBy = pgtype.UUID{Bytes: *resolvedBy, Valid: true}
	}

	row, err := r.queries.UpdateHITLRequestStatus(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("hitl_request", id)
		}
		return nil, apperrors.NewInternal("failed to update HITL request resolution", err)
	}
	return toDomainHITLRequest(row), nil
}

// ListProbeHalts returns pending probe_halt gates for batch triage.
func (r *HITLRepo) ListProbeHalts(ctx context.Context, projectID *uuid.UUID) ([]*model.ProbeHalt, error) {
	var arg pgtype.UUID
	if projectID != nil {
		arg = pgtype.UUID{Bytes: *projectID, Valid: true}
	}
	rows, err := r.queries.ListProbeHalts(ctx, arg)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list probe halts", err)
	}
	result := make([]*model.ProbeHalt, len(rows))
	for i, row := range rows {
		ph := &model.ProbeHalt{
			ID:         row.ID,
			RunStepID:  row.RunStepID,
			RunID:      row.RunID,
			ProjectID:  row.ProjectID,
			StoryKey:   row.StoryKey,
			StoryTitle: row.StoryTitle,
			StepName:   row.StepName,
			CreatedAt:  row.CreatedAt,
		}
		if row.StageName.Valid {
			ph.StageName = row.StageName.String
		}
		ph.HaltReason = unmarshalHaltReason(row.HaltReason)
		result[i] = ph
	}
	return result, nil
}

// unmarshalHaltReason decodes the halt_reason JSONB column; returns nil on empty
// or invalid payloads (a malformed reason must not block triage).
func unmarshalHaltReason(raw []byte) *model.HaltReason {
	if len(raw) == 0 {
		return nil
	}
	var hr model.HaltReason
	if err := json.Unmarshal(raw, &hr); err != nil {
		return nil
	}
	return &hr
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
		if row.DiffUrl.Valid {
			result[i].DiffURL = &row.DiffUrl.String
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

// ListFiltered returns HITL requests optionally filtered by status with pagination.
func (r *HITLRepo) ListFiltered(ctx context.Context, status *string, limit, offset int32) ([]*model.HITLRequest, error) {
	statusStr := ""
	if status != nil {
		statusStr = *status
	}
	rows, err := r.queries.ListHITLRequestsFiltered(ctx, ListHITLRequestsFilteredParams{
		Column1: statusStr,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list filtered HITL requests", err)
	}
	result := make([]*model.HITLRequest, len(rows))
	for i, row := range rows {
		result[i] = toDomainHITLRequest(row)
	}
	return result, nil
}

// CountFiltered returns the count of HITL requests optionally filtered by status.
func (r *HITLRepo) CountFiltered(ctx context.Context, status *string) (int64, error) {
	statusStr := ""
	if status != nil {
		statusStr = *status
	}
	count, err := r.queries.CountHITLRequestsFiltered(ctx, statusStr)
	if err != nil {
		return 0, apperrors.NewInternal("failed to count filtered HITL requests", err)
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
	if row.Message.Valid {
		req.Message = &row.Message.String
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
	if row.ResolutionAction.Valid {
		req.ResolutionAction = &row.ResolutionAction.String
	}
	req.HaltReason = unmarshalHaltReason(row.HaltReason)
	return req
}
