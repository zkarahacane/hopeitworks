package postgres

import (
	"context"
	"errors"
	"math"
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

// InsertCostRecord persists a single cost record.
func (r *CostRepo) InsertCostRecord(ctx context.Context, record *model.CostRecord) (*model.CostRecord, error) {
	params := InsertCostRecordParams{
		RunStepID:    record.RunStepID,
		ProjectID:    record.ProjectID,
		TokensInput:  record.TokensInput,
		TokensOutput: record.TokensOutput,
		CostUsd:      numericFromFloat64Cost(record.CostUSD),
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

// GetCostByRunStep retrieves the cost record for a run step.
func (r *CostRepo) GetCostByRunStep(ctx context.Context, runStepID uuid.UUID) (*model.CostRecord, error) {
	row, err := r.queries.GetCostByRunStep(ctx, runStepID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("cost_record", runStepID)
		}
		return nil, apperrors.NewInternal("failed to get cost by run step", err)
	}
	return toDomainCostRecord(row), nil
}

// SumCostByProject returns aggregated cost totals for a project since the given time.
func (r *CostRepo) SumCostByProject(ctx context.Context, projectID uuid.UUID, since time.Time) (float64, int64, int64, error) {
	row, err := r.queries.SumCostByProject(ctx, SumCostByProjectParams{
		ProjectID: projectID,
		CreatedAt: since,
	})
	if err != nil {
		return 0, 0, 0, apperrors.NewInternal("failed to sum cost by project", err)
	}

	totalCost := numericToFloat64(row.TotalCost)
	totalInput := toInt64(row.TotalInput)
	totalOutput := toInt64(row.TotalOutput)

	return totalCost, totalInput, totalOutput, nil
}

// SumCostByRun returns the total cost for a run.
func (r *CostRepo) SumCostByRun(ctx context.Context, runID uuid.UUID) (float64, error) {
	result, err := r.queries.SumCostByRun(ctx, runID)
	if err != nil {
		return 0, apperrors.NewInternal("failed to sum cost by run", err)
	}
	return numericToFloat64(result), nil
}

// SumCostByStory returns aggregated cost totals for a story across all runs.
func (r *CostRepo) SumCostByStory(ctx context.Context, storyID uuid.UUID) (float64, int64, int64, int, error) {
	row, err := r.queries.SumCostByStory(ctx, storyID)
	if err != nil {
		return 0, 0, 0, 0, apperrors.NewInternal("failed to sum cost by story", err)
	}

	totalCost := numericToFloat64(row.TotalCost)
	totalInput := toInt64(row.TotalInput)
	totalOutput := toInt64(row.TotalOutput)

	return totalCost, totalInput, totalOutput, int(row.RunCount), nil
}

// ListCostsByProjectByStory returns cost breakdown by story for a project since the given time.
func (r *CostRepo) ListCostsByProjectByStory(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.StoryCostBreakdown, error) {
	rows, err := r.queries.ListCostsByProjectByStory(ctx, ListCostsByProjectByStoryParams{
		ProjectID: projectID,
		CreatedAt: since,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list costs by project by story", err)
	}

	results := make([]model.StoryCostBreakdown, len(rows))
	for i, row := range rows {
		results[i] = model.StoryCostBreakdown{
			StoryID:   row.StoryID,
			StoryKey:  row.StoryKey,
			TotalCost: numericToFloat64(row.TotalCost),
		}
	}
	return results, nil
}

// ListCostsByProjectByRun returns cost breakdown by run for a project since the given time.
func (r *CostRepo) ListCostsByProjectByRun(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.RunCostBreakdown, error) {
	rows, err := r.queries.ListCostsByProjectByRun(ctx, ListCostsByProjectByRunParams{
		ProjectID: projectID,
		CreatedAt: since,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list costs by project by run", err)
	}

	results := make([]model.RunCostBreakdown, len(rows))
	for i, row := range rows {
		results[i] = model.RunCostBreakdown{
			RunID:     row.RunID,
			StoryKey:  row.StoryKey,
			Status:    row.Status,
			TotalCost: numericToFloat64(row.TotalCost),
			CreatedAt: row.CreatedAt,
		}
	}
	return results, nil
}

// ListCostsByProjectByModel returns cost breakdown by model for a project since the given time.
func (r *CostRepo) ListCostsByProjectByModel(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.CostByModel, error) {
	rows, err := r.queries.ListCostsByProjectByModel(ctx, ListCostsByProjectByModelParams{
		ProjectID: projectID,
		CreatedAt: since,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list costs by project by model", err)
	}

	results := make([]model.CostByModel, len(rows))
	for i, row := range rows {
		results[i] = model.CostByModel{
			Model:        row.Model,
			TotalCost:    numericToFloat64(row.TotalCost),
			TokensInput:  toInt64(row.TokensInput),
			TokensOutput: toInt64(row.TokensOutput),
		}
	}
	return results, nil
}

// ListStepCostsByRun returns per-step cost breakdown for a run.
func (r *CostRepo) ListStepCostsByRun(ctx context.Context, runID uuid.UUID) ([]model.StepCostBreakdown, error) {
	rows, err := r.queries.ListStepCostsByRun(ctx, runID)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list step costs by run", err)
	}

	results := make([]model.StepCostBreakdown, len(rows))
	for i, row := range rows {
		results[i] = model.StepCostBreakdown{
			StepID:       row.StepID,
			StepName:     row.StepName,
			Model:        row.Model,
			TokensInput:  row.TokensInput,
			TokensOutput: row.TokensOutput,
			CostUSD:      numericToFloat64(row.CostUsd),
		}
	}
	return results, nil
}

// toDomainCostRecord maps a sqlc-generated CostRecord to a domain CostRecord.
func toDomainCostRecord(c CostRecord) *model.CostRecord {
	return &model.CostRecord{
		ID:           c.ID,
		RunStepID:    c.RunStepID,
		ProjectID:    c.ProjectID,
		TokensInput:  c.TokensInput,
		TokensOutput: c.TokensOutput,
		CostUSD:      numericToFloat64(c.CostUsd),
		Model:        c.Model,
		CreatedAt:    c.CreatedAt,
	}
}

// numericFromFloat64Cost converts a float64 cost value to pgtype.Numeric with 6 decimal places.
func numericFromFloat64Cost(f float64) pgtype.Numeric {
	// Multiply by 1_000_000 for 6 decimal places, store as integer with exponent -6
	micros := int64(math.Round(f * 1_000_000))
	return pgtype.Numeric{
		Int:   big.NewInt(micros),
		Exp:   -6,
		Valid: true,
	}
}

// toInt64 converts an interface{} from COALESCE aggregate results to int64.
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int32:
		return int64(val)
	case int:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}
