package model

import (
	"time"

	"github.com/google/uuid"
)

// CostRecord represents a cost record for a run step.
type CostRecord struct {
	ID           uuid.UUID
	RunStepID    uuid.UUID
	ProjectID    uuid.UUID
	TokensInput  int64
	TokensOutput int64
	// CostUSD is the total cost in US dollars for this step.
	CostUSD   float64
	Model     string
	CreatedAt time.Time
}

// CostEvent is an intermediate accumulation type parsed from agent NDJSON output.
type CostEvent struct {
	InputTokens  int64
	OutputTokens int64
	Model        string
}

// Pricing holds input and output pricing per million tokens.
type Pricing struct {
	InputPerMTok  float64
	OutputPerMTok float64
}

// modelPricingMap maps model names to their token pricing (USD per million tokens).
var modelPricingMap = map[string]Pricing{
	"claude-opus-4-6":   {InputPerMTok: 15.0, OutputPerMTok: 75.0},
	"claude-sonnet-4-6": {InputPerMTok: 3.0, OutputPerMTok: 15.0},
	"claude-haiku-4-5":  {InputPerMTok: 0.25, OutputPerMTok: 1.25},
}

// ComputeCostUSD computes the cost in USD for the given model and token counts.
// Returns (costUSD, known) where known is false for unrecognized models.
func ComputeCostUSD(model string, inputTokens, outputTokens int64) (float64, bool) {
	pricing, ok := modelPricingMap[model]
	if !ok {
		return 0, false
	}
	cost := (float64(inputTokens)/1_000_000)*pricing.InputPerMTok +
		(float64(outputTokens)/1_000_000)*pricing.OutputPerMTok
	return cost, true
}

// CostSummary holds the simplified cost summary returned by the /costs/summary endpoint.
type CostSummary struct {
	TotalCostUSD      float64
	TotalCostWeekUSD  float64
	TotalCostMonthUSD float64
	AvgCostPerStory   float64
	BudgetLimitUSD    *float64
	PeriodStart       time.Time
	PeriodEnd         time.Time
}

// ProjectCostSummary holds aggregated cost data for a project over a time period.
type ProjectCostSummary struct {
	TotalCost   float64
	TotalInput  int64
	TotalOutput int64
	MaxBudget   *float64
	ByStory     []StoryCostBreakdown
	ByRun       []RunCostBreakdown
	ByModel     []CostByModel
}

// StoryCostBreakdown holds cost data for a single story.
type StoryCostBreakdown struct {
	StoryID   uuid.UUID
	StoryKey  string
	TotalCost float64
}

// RunCostBreakdown holds cost data for a single run.
type RunCostBreakdown struct {
	RunID     uuid.UUID
	StoryKey  string
	Status    string
	TotalCost float64
	CreatedAt time.Time
}

// CostByModel holds cost data for a single model.
type CostByModel struct {
	Model        string
	TotalCost    float64
	TokensInput  int64
	TokensOutput int64
}

// StoryCostSummary holds aggregated cost data for a story.
type StoryCostSummary struct {
	StoryID     uuid.UUID
	TotalCost   float64
	TotalInput  int64
	TotalOutput int64
	RunCount    int
}

// RunCostDetail holds cost data for a run with per-step breakdown.
type RunCostDetail struct {
	RunID     uuid.UUID
	TotalCost float64
	Steps     []StepCostBreakdown
}

// StepCostBreakdown holds cost data for a single step.
type StepCostBreakdown struct {
	StepID       uuid.UUID
	StepName     string
	Model        string
	TokensInput  int64
	TokensOutput int64
	CostUSD      float64
}

// CostDataPoint represents a single day's cost total for chart rendering.
type CostDataPoint struct {
	Date         string // "YYYY-MM-DD"
	TotalCostUSD float64
}

// RunCostRow represents a run-level cost row for the paginated runs tab.
type RunCostRow struct {
	RunID        uuid.UUID
	StoryKey     string
	Status       string
	StartedAt    time.Time
	TotalCostUSD float64
}
