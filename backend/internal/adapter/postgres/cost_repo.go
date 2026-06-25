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
		AgentID:      uuidFromPtr(record.AgentID),
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

// ListDailyCostsByProject returns daily cost data points for chart rendering.
func (r *CostRepo) ListDailyCostsByProject(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.CostDataPoint, error) {
	rows, err := r.queries.ListDailyCostsByProject(ctx, ListDailyCostsByProjectParams{
		ProjectID: projectID,
		CreatedAt: since,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list daily costs by project", err)
	}

	results := make([]model.CostDataPoint, len(rows))
	for i, row := range rows {
		results[i] = model.CostDataPoint{
			Date:         row.Date,
			TotalCostUSD: numericToFloat64(row.TotalCostUsd),
		}
	}
	return results, nil
}

// ListCostsByProjectByRunPaginated returns paginated run-level cost breakdown.
func (r *CostRepo) ListCostsByProjectByRunPaginated(ctx context.Context, projectID uuid.UUID, since time.Time, limit, offset int32) ([]model.RunCostRow, error) {
	rows, err := r.queries.ListCostsByProjectByRunPaginated(ctx, ListCostsByProjectByRunPaginatedParams{
		ProjectID: projectID,
		CreatedAt: since,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list costs by project by run paginated", err)
	}

	results := make([]model.RunCostRow, len(rows))
	for i, row := range rows {
		results[i] = model.RunCostRow{
			RunID:        row.RunID,
			StoryKey:     row.StoryKey,
			Status:       row.Status,
			StartedAt:    row.StartedAt,
			TotalCostUSD: numericToFloat64(row.TotalCostUsd),
			TokensInput:  row.TokensInput,
			TokensOutput: row.TokensOutput,
		}
	}
	return results, nil
}

// CountCostsByProjectByRun returns the count of distinct runs with costs.
func (r *CostRepo) CountCostsByProjectByRun(ctx context.Context, projectID uuid.UUID, since time.Time) (int64, error) {
	count, err := r.queries.CountCostsByProjectByRun(ctx, CountCostsByProjectByRunParams{
		ProjectID: projectID,
		CreatedAt: since,
	})
	if err != nil {
		return 0, apperrors.NewInternal("failed to count costs by project by run", err)
	}
	return count, nil
}

// toDomainCostRecord maps a sqlc-generated CostRecord to a domain CostRecord.
func toDomainCostRecord(c CostRecord) *model.CostRecord {
	return &model.CostRecord{
		ID:           c.ID,
		RunStepID:    c.RunStepID,
		ProjectID:    c.ProjectID,
		AgentID:      pgtypeToUUIDPtr(c.AgentID),
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

// ListByProjectByAgent returns cost breakdown by agent for a project.
func (r *CostRepo) ListByProjectByAgent(ctx context.Context, projectID uuid.UUID) ([]model.AgentCostBreakdown, error) {
	rows, err := r.queries.ListCostsByProjectByAgent(ctx, projectID)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list costs by project by agent", err)
	}

	results := make([]model.AgentCostBreakdown, len(rows))
	for i, row := range rows {
		agentID := row.AgentID.Bytes
		results[i] = model.AgentCostBreakdown{
			AgentID:      agentID,
			AgentName:    row.AgentName,
			TokensInput:  row.TokensInput,
			TokensOutput: row.TokensOutput,
			CostUSD:      numericToFloat64(row.CostUsd),
			RunsCount:    row.RunsCount,
		}
	}
	return results, nil
}

// ListByProjectByRole returns cost breakdown by role (agent type) for a project.
func (r *CostRepo) ListByProjectByRole(ctx context.Context, projectID uuid.UUID) ([]model.ProjectRoleCostBreakdown, error) {
	rows, err := r.queries.ListCostsByProjectByRole(ctx, projectID)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list costs by project by role", err)
	}

	results := make([]model.ProjectRoleCostBreakdown, len(rows))
	for i, row := range rows {
		results[i] = model.ProjectRoleCostBreakdown{
			Role:         row.Role,
			TokensInput:  row.TokensInput,
			TokensOutput: row.TokensOutput,
			CostUSD:      numericToFloat64(row.CostUsd),
			RunsCount:    row.RunsCount,
		}
	}
	return results, nil
}

// ListCostsByRunByRole returns per-role cost breakdown for a run.
func (r *CostRepo) ListCostsByRunByRole(ctx context.Context, runID uuid.UUID) ([]model.RoleCostBreakdown, error) {
	rows, err := r.queries.ListCostsByRunByRole(ctx, runID)
	if err != nil {
		return nil, apperrors.NewInternal("failed to list costs by run by role", err)
	}

	results := make([]model.RoleCostBreakdown, len(rows))
	for i, row := range rows {
		results[i] = model.RoleCostBreakdown{
			Role:         row.Role,
			TokensInput:  row.TokensInput,
			TokensOutput: row.TokensOutput,
			CostUSD:      numericToFloat64(row.CostUsd),
		}
	}
	return results, nil
}

// SumTokensByRun returns total input/output tokens for a run across all steps.
func (r *CostRepo) SumTokensByRun(ctx context.Context, runID uuid.UUID) (int64, int64, error) {
	row, err := r.queries.SumTokensByRun(ctx, runID)
	if err != nil {
		return 0, 0, apperrors.NewInternal("failed to sum tokens by run", err)
	}
	return row.TokensInput, row.TokensOutput, nil
}

// pgtypeToUUIDPtr converts a pgtype.UUID to a *uuid.UUID.
func pgtypeToUUIDPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
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
