package model

import (
	"time"

	"github.com/google/uuid"
)

// CostRecord represents a persisted cost record for a single run step.
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

// modelPricing maps model names to [inputPerMTok, outputPerMTok] pricing in USD.
// Per million tokens.
var modelPricing = map[string][2]float64{
	"claude-opus-4-6":   {15.0, 75.0},
	"claude-sonnet-4-5": {3.0, 15.0},
	"claude-haiku-4-3":  {0.25, 1.25},
}

// ComputeCostUSD returns the cost in USD for the given model and token counts.
// The second return value is false when the model is not recognized, in which
// case the cost is 0.
func ComputeCostUSD(modelName string, inputTokens, outputTokens int64) (float64, bool) {
	pricing, ok := modelPricing[modelName]
	if !ok {
		return 0, false
	}
	cost := (float64(inputTokens)/1_000_000)*pricing[0] +
		(float64(outputTokens)/1_000_000)*pricing[1]
	return cost, true
}
