package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// CostRepository defines the interface for cost data persistence.
type CostRepository interface {
	// InsertCostRecord persists a single cost record.
	InsertCostRecord(ctx context.Context, record *model.CostRecord) (*model.CostRecord, error)

	// GetCostByRunStep retrieves the cost record for a run step.
	GetCostByRunStep(ctx context.Context, runStepID uuid.UUID) (*model.CostRecord, error)

	// SumCostByProject returns aggregated cost totals for a project since the given time.
	SumCostByProject(ctx context.Context, projectID uuid.UUID, since time.Time) (totalCost float64, totalInput, totalOutput int64, err error)

	// SumCostByRun returns the total cost for a run.
	SumCostByRun(ctx context.Context, runID uuid.UUID) (float64, error)

	// SumCostByStory returns aggregated cost totals for a story across all runs.
	SumCostByStory(ctx context.Context, storyID uuid.UUID) (totalCost float64, totalInput, totalOutput int64, runCount int, err error)

	// ListCostsByProjectByStory returns cost breakdown by story for a project since the given time.
	ListCostsByProjectByStory(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.StoryCostBreakdown, error)

	// ListCostsByProjectByRun returns cost breakdown by run for a project since the given time.
	ListCostsByProjectByRun(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.RunCostBreakdown, error)

	// ListCostsByProjectByModel returns cost breakdown by model for a project since the given time.
	ListCostsByProjectByModel(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.CostByModel, error)

	// ListStepCostsByRun returns per-step cost breakdown for a run.
	ListStepCostsByRun(ctx context.Context, runID uuid.UUID) ([]model.StepCostBreakdown, error)
}
