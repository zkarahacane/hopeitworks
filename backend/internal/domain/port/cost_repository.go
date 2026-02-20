package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// CostRepository defines the interface for cost record persistence operations.
type CostRepository interface {
	// InsertCostRecord persists a new cost record and returns it with its generated ID.
	InsertCostRecord(ctx context.Context, record *model.CostRecord) (*model.CostRecord, error)

	// GetCostByRunStep retrieves the cost record for a given run step.
	GetCostByRunStep(ctx context.Context, runStepID uuid.UUID) (*model.CostRecord, error)

	// SumCostByProject returns aggregate cost and token totals for a project since the given time.
	SumCostByProject(ctx context.Context, projectID uuid.UUID, since time.Time) (totalCost float64, totalInput, totalOutput int64, err error)

	// SumCostByRun returns the total cost in USD across all steps of a run.
	SumCostByRun(ctx context.Context, runID uuid.UUID) (float64, error)
}
