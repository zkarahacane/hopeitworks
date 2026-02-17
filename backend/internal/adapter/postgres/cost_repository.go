package postgres

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure CostRepo implements port.CostRepository at compile time.
var _ port.CostRepository = (*CostRepo)(nil)

// CostRepo implements port.CostRepository using sqlc-generated queries.
type CostRepo struct {
	queries *Queries
}

// NewCostRepo creates a new CostRepo.
func NewCostRepo(queries *Queries) *CostRepo {
	return &CostRepo{queries: queries}
}

func (r *CostRepo) InsertCostRecord(ctx context.Context, record *model.CostRecord) (*model.CostRecord, error) {
	params := InsertCostRecordParams{
		RunStepID:    record.RunStepID,
		ProjectID:    record.ProjectID,
		TokensInput:  record.TokensInput,
		TokensOutput: record.TokensOutput,
		CostUsd:      float64ToNumeric(record.CostUSD),
		Model:        record.Model,
	}

	row, err := r.queries.InsertCostRecord(ctx, params)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, apperrors.NewNotFound("run_step", record.RunStepID)
		}
		return nil, apperrors.NewInternal("failed to insert cost record", err)
	}
	return toDomainCostRecord(row), nil
}

func (r *CostRepo) GetCostByRunStep(ctx context.Context, runStepID uuid.UUID) (*model.CostRecord, error) {
	row, err := r.queries.GetCostByRunStep(ctx, runStepID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("cost_record", runStepID)
		}
		return nil, apperrors.NewInternal("failed to get cost record by run step", err)
	}
	return toDomainCostRecord(row), nil
}

func (r *CostRepo) SumCostByProject(ctx context.Context, projectID uuid.UUID, since time.Time) (float64, int64, int64, error) {
	params := SumCostByProjectParams{
		ProjectID: projectID,
		CreatedAt: since,
	}
	row, err := r.queries.SumCostByProject(ctx, params)
	if err != nil {
		return 0, 0, 0, apperrors.NewInternal("failed to sum cost by project", err)
	}

	totalCost := numericToFloat64(row.TotalCost)
	totalInput := toInt64(row.TotalInput)
	totalOutput := toInt64(row.TotalOutput)

	return totalCost, totalInput, totalOutput, nil
}

func (r *CostRepo) SumCostByRun(ctx context.Context, runID uuid.UUID) (float64, error) {
	result, err := r.queries.SumCostByRun(ctx, runID)
	if err != nil {
		return 0, apperrors.NewInternal("failed to sum cost by run", err)
	}
	return numericToFloat64(result), nil
}

// toDomainCostRecord converts a sqlc CostRecord to a domain model CostRecord.
func toDomainCostRecord(r CostRecord) *model.CostRecord {
	return &model.CostRecord{
		ID:           r.ID,
		RunStepID:    r.RunStepID,
		ProjectID:    r.ProjectID,
		TokensInput:  r.TokensInput,
		TokensOutput: r.TokensOutput,
		CostUSD:      numericToFloat64(r.CostUsd),
		Model:        r.Model,
		CreatedAt:    r.CreatedAt,
	}
}

// float64ToNumeric converts a float64 cost value to pgtype.Numeric with 6 decimal places.
func float64ToNumeric(f float64) pgtype.Numeric {
	// Multiply by 1_000_000 to represent 6 decimal places, store with exponent -6.
	millionths := int64(f * 1_000_000)
	return pgtype.Numeric{
		Int:   big.NewInt(millionths),
		Exp:   -6,
		Valid: true,
	}
}

// toInt64 converts an any value returned by sqlc COALESCE to int64.
func toInt64(v any) int64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}
