package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// CostService provides business logic for recording and querying agent run costs.
type CostService struct {
	costRepo port.CostRepository
	logger   *slog.Logger
}

// NewCostService creates a new CostService.
func NewCostService(costRepo port.CostRepository, logger *slog.Logger) *CostService {
	return &CostService{
		costRepo: costRepo,
		logger:   logger,
	}
}

// RecordStepCost aggregates all cost events for a step and persists a single cost record.
// If events is empty, this is a no-op. Cost recording failures are propagated to the caller.
func (s *CostService) RecordStepCost(ctx context.Context, stepID, projectID uuid.UUID, events []model.CostEvent) error {
	if len(events) == 0 {
		return nil
	}

	var totalInput, totalOutput int64
	var totalCost float64
	// Use the model from the first event as the primary model; aggregate tokens across all events.
	primaryModel := events[0].Model

	for _, e := range events {
		totalInput += e.InputTokens
		totalOutput += e.OutputTokens
	}

	cost, known := model.ComputeCostUSD(primaryModel, totalInput, totalOutput)
	if !known {
		s.logger.Warn("unknown model for cost computation",
			"unknown_model", primaryModel,
			"step_id", stepID,
		)
		totalCost = 0
	} else {
		totalCost = cost
	}

	record := &model.CostRecord{
		RunStepID:    stepID,
		ProjectID:    projectID,
		TokensInput:  totalInput,
		TokensOutput: totalOutput,
		CostUSD:      totalCost,
		Model:        primaryModel,
	}

	_, err := s.costRepo.InsertCostRecord(ctx, record)
	return err
}
