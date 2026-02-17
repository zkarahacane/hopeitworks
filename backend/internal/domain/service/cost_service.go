package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// CostService provides business logic for cost operations.
type CostService struct {
	costRepo    port.CostRepository
	projectRepo port.ProjectRepository
	storyRepo   port.StoryRepository
	runRepo     port.RunRepository
	logger      *slog.Logger
}

// NewCostService creates a new CostService.
func NewCostService(
	costRepo port.CostRepository,
	projectRepo port.ProjectRepository,
	storyRepo port.StoryRepository,
	runRepo port.RunRepository,
	logger *slog.Logger,
) *CostService {
	return &CostService{
		costRepo:    costRepo,
		projectRepo: projectRepo,
		storyRepo:   storyRepo,
		runRepo:     runRepo,
		logger:      logger,
	}
}

// RecordStepCost aggregates cost events and inserts a single cost record for a step.
func (s *CostService) RecordStepCost(ctx context.Context, stepID, projectID uuid.UUID, events []model.CostEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Aggregate all events
	var totalInput, totalOutput int64
	modelName := events[0].Model
	for _, e := range events {
		totalInput += e.InputTokens
		totalOutput += e.OutputTokens
		if e.Model != "" {
			modelName = e.Model
		}
	}

	// Compute cost
	costUSD, known := model.ComputeCostUSD(modelName, totalInput, totalOutput)
	if !known {
		s.logger.Warn("unknown model for cost computation, cost set to zero",
			"model", modelName,
			"step_id", stepID,
		)
	}

	record := &model.CostRecord{
		RunStepID:    stepID,
		ProjectID:    projectID,
		TokensInput:  totalInput,
		TokensOutput: totalOutput,
		CostUSD:      costUSD,
		Model:        modelName,
	}

	_, err := s.costRepo.InsertCostRecord(ctx, record)
	return err
}

// parsePeriod converts a period string to a time.Time representing the start of the period.
func parsePeriod(period string) (time.Time, error) {
	now := time.Now().UTC()
	switch period {
	case "7d":
		return now.AddDate(0, 0, -7), nil
	case "30d":
		return now.AddDate(0, 0, -30), nil
	case "90d":
		return now.AddDate(0, 0, -90), nil
	default:
		return time.Time{}, errors.NewValidation("period", fmt.Sprintf("invalid period: %s, must be one of: 7d, 30d, 90d", period))
	}
}

// GetProjectCosts returns aggregated cost data for a project over a time period.
func (s *CostService) GetProjectCosts(ctx context.Context, projectID uuid.UUID, period string) (*model.ProjectCostSummary, error) {
	// Verify project exists
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if period == "" {
		period = "7d"
	}

	since, err := parsePeriod(period)
	if err != nil {
		return nil, err
	}

	// Fetch aggregated totals
	totalCost, totalInput, totalOutput, err := s.costRepo.SumCostByProject(ctx, projectID, since)
	if err != nil {
		return nil, err
	}

	// Fetch breakdowns in parallel would be nice, but keep it simple for MVP
	byStory, err := s.costRepo.ListCostsByProjectByStory(ctx, projectID, since)
	if err != nil {
		return nil, err
	}

	byRun, err := s.costRepo.ListCostsByProjectByRun(ctx, projectID, since)
	if err != nil {
		return nil, err
	}

	byModel, err := s.costRepo.ListCostsByProjectByModel(ctx, projectID, since)
	if err != nil {
		return nil, err
	}

	return &model.ProjectCostSummary{
		TotalCost:   totalCost,
		TotalInput:  totalInput,
		TotalOutput: totalOutput,
		MaxBudget:   project.MaxBudget,
		ByStory:     byStory,
		ByRun:       byRun,
		ByModel:     byModel,
	}, nil
}

// GetStoryCosts returns aggregated cost data for a story across all runs.
func (s *CostService) GetStoryCosts(ctx context.Context, projectID, storyID uuid.UUID) (*model.StoryCostSummary, error) {
	// Verify story exists and belongs to project
	story, err := s.storyRepo.GetByID(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if story.ProjectID != projectID {
		return nil, errors.NewNotFound("story", storyID)
	}

	totalCost, totalInput, totalOutput, runCount, err := s.costRepo.SumCostByStory(ctx, storyID)
	if err != nil {
		return nil, err
	}

	return &model.StoryCostSummary{
		StoryID:     storyID,
		TotalCost:   totalCost,
		TotalInput:  totalInput,
		TotalOutput: totalOutput,
		RunCount:    runCount,
	}, nil
}

// GetRunCosts returns cost data for a run with per-step breakdown.
func (s *CostService) GetRunCosts(ctx context.Context, projectID, runID uuid.UUID) (*model.RunCostDetail, error) {
	// Verify run exists and belongs to project
	run, err := s.runRepo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.ProjectID != projectID {
		return nil, errors.NewNotFound("run", runID)
	}

	totalCost, err := s.costRepo.SumCostByRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	steps, err := s.costRepo.ListStepCostsByRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	return &model.RunCostDetail{
		RunID:     runID,
		TotalCost: totalCost,
		Steps:     steps,
	}, nil
}
