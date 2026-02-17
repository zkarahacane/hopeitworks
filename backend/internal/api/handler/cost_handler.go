package handler

import (
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// CostHandler implements cost-related HTTP handlers.
type CostHandler struct {
	service *service.CostService
}

// NewCostHandler creates a new CostHandler.
func NewCostHandler(svc *service.CostService) *CostHandler {
	return &CostHandler{service: svc}
}

// GetProjectCosts handles GET /projects/{projectId}/costs.
func (h *CostHandler) GetProjectCosts(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetProjectCostsParams) {
	period := "7d"
	if params.Period != nil {
		period = string(*params.Period)
	}

	summary, err := h.service.GetProjectCosts(r.Context(), projectID, period)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := ProjectCostSummary{
		TotalCost:   summary.TotalCost,
		TotalInput:  summary.TotalInput,
		TotalOutput: summary.TotalOutput,
		MaxBudget:   summary.MaxBudget,
		ByStory:     make([]StoryCostBreakdown, len(summary.ByStory)),
		ByRun:       make([]RunCostBreakdown, len(summary.ByRun)),
		ByModel:     make([]ModelCostBreakdown, len(summary.ByModel)),
	}

	for i, s := range summary.ByStory {
		resp.ByStory[i] = StoryCostBreakdown{
			StoryId:   s.StoryID,
			StoryKey:  s.StoryKey,
			TotalCost: s.TotalCost,
		}
	}

	for i, r := range summary.ByRun {
		resp.ByRun[i] = RunCostBreakdown{
			RunId:     r.RunID,
			StoryKey:  r.StoryKey,
			Status:    r.Status,
			TotalCost: r.TotalCost,
			CreatedAt: r.CreatedAt,
		}
	}

	for i, m := range summary.ByModel {
		resp.ByModel[i] = ModelCostBreakdown{
			Model:        m.Model,
			TotalCost:    m.TotalCost,
			TokensInput:  m.TokensInput,
			TokensOutput: m.TokensOutput,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetStoryCosts handles GET /projects/{projectId}/stories/{storyId}/costs.
func (h *CostHandler) GetStoryCosts(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, storyID StoryIdPath) {
	summary, err := h.service.GetStoryCosts(r.Context(), projectID, storyID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := StoryCostSummary{
		StoryId:     summary.StoryID,
		TotalCost:   summary.TotalCost,
		TotalInput:  summary.TotalInput,
		TotalOutput: summary.TotalOutput,
		RunCount:    summary.RunCount,
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetRunCosts handles GET /projects/{projectId}/runs/{runId}/costs.
func (h *CostHandler) GetRunCosts(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
	detail, err := h.service.GetRunCosts(r.Context(), projectID, runID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := RunCostDetail{
		RunId:     detail.RunID,
		TotalCost: detail.TotalCost,
		Steps:     make([]StepCostBreakdown, len(detail.Steps)),
	}

	for i, s := range detail.Steps {
		resp.Steps[i] = StepCostBreakdown{
			StepId:       s.StepID,
			StepName:     s.StepName,
			Model:        s.Model,
			TokensInput:  s.TokensInput,
			TokensOutput: s.TokensOutput,
			CostUsd:      s.CostUSD,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
