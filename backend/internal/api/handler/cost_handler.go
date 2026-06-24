package handler

import (
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
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

// GetProjectCostSummary handles GET /projects/{projectId}/costs/summary.
func (h *CostHandler) GetProjectCostSummary(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetProjectCostSummaryParams) {
	period := "7d"
	if params.Period != nil {
		period = string(*params.Period)
	}

	summary, err := h.service.GetProjectCostSummary(r.Context(), projectID, period)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := CostSummary{
		TotalCostUsd:       summary.TotalCostUSD,
		TotalCostWeekUsd:   &summary.TotalCostWeekUSD,
		TotalCostMonthUsd:  &summary.TotalCostMonthUSD,
		AvgCostPerStoryUsd: summary.AvgCostPerStory,
		BudgetLimitUsd:     summary.BudgetLimitUSD,
		PeriodStart:        summary.PeriodStart.UTC(),
		PeriodEnd:          summary.PeriodEnd.UTC(),
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetProjectCostChart handles GET /projects/{projectId}/costs/chart.
func (h *CostHandler) GetProjectCostChart(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetProjectCostChartParams) {
	period := "7d"
	if params.Period != nil {
		period = string(*params.Period)
	}

	points, err := h.service.GetProjectCostChart(r.Context(), projectID, period)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := make([]CostDataPoint, len(points))
	for i, p := range points {
		t, _ := time.Parse("2006-01-02", p.Date)
		resp[i] = CostDataPoint{
			Date:         openapi_types.Date{Time: t},
			TotalCostUsd: p.TotalCostUSD,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetProjectCostRuns handles GET /projects/{projectId}/costs/runs.
func (h *CostHandler) GetProjectCostRuns(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetProjectCostRunsParams) {
	period := "7d"
	if params.Period != nil {
		period = string(*params.Period)
	}

	page, perPage := paginationDefaults(params.Page, params.PerPage)

	rows, total, err := h.service.GetProjectCostRuns(r.Context(), projectID, period, page, perPage)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	data := make([]RunCostRow, len(rows))
	for i, row := range rows {
		tokensInput := row.TokensInput
		tokensOutput := row.TokensOutput
		data[i] = RunCostRow{
			RunId:        row.RunID,
			StoryKey:     row.StoryKey,
			Status:       row.Status,
			StartedAt:    row.StartedAt,
			TotalCostUsd: row.TotalCostUSD,
			TokensInput:  &tokensInput,
			TokensOutput: &tokensOutput,
		}
	}

	resp := struct {
		Data       []RunCostRow `json:"data"`
		Pagination Pagination   `json:"pagination"`
	}{
		Data: data,
		Pagination: Pagination{
			Page:    page,
			PerPage: perPage,
			Total:   int(total),
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetProjectCostsByAgent handles GET /projects/{projectId}/costs/agents.
func (h *CostHandler) GetProjectCostsByAgent(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	breakdowns, err := h.service.GetProjectCostsByAgent(r.Context(), projectID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := make([]AgentCostBreakdown, len(breakdowns))
	for i, b := range breakdowns {
		resp[i] = AgentCostBreakdown{
			AgentId:      b.AgentID,
			AgentName:    b.AgentName,
			TokensInput:  b.TokensInput,
			TokensOutput: b.TokensOutput,
			CostUsd:      b.CostUSD,
			RunsCount:    b.RunsCount,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetProjectCostsByRole handles GET /projects/{projectId}/costs/by-role.
func (h *CostHandler) GetProjectCostsByRole(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	breakdown, err := h.service.GetProjectCostsByRole(r.Context(), projectID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := ProjectCostByRole{
		TotalCost:         breakdown.TotalCost,
		TotalTokensInput:  breakdown.TotalInput,
		TotalTokensOutput: breakdown.TotalOutput,
		Roles:             make([]ProjectRoleCostBreakdown, len(breakdown.Roles)),
	}

	for i, role := range breakdown.Roles {
		resp.Roles[i] = ProjectRoleCostBreakdown{
			Role:         role.Role,
			TokensInput:  role.TokensInput,
			TokensOutput: role.TokensOutput,
			CostUsd:      role.CostUSD,
			RunsCount:    role.RunsCount,
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

// GetRunCostsByRole handles GET /projects/{projectId}/runs/{runId}/costs/by-role.
func (h *CostHandler) GetRunCostsByRole(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
	detail, err := h.service.GetRunCostsByRole(r.Context(), projectID, runID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := RunCostByRole{
		RunId:             detail.RunID,
		TotalCost:         detail.TotalCost,
		TotalTokensInput:  detail.TotalInput,
		TotalTokensOutput: detail.TotalOutput,
		Roles:             make([]RoleCostBreakdown, len(detail.Roles)),
	}

	for i, role := range detail.Roles {
		resp.Roles[i] = RoleCostBreakdown{
			Role:         role.Role,
			TokensInput:  role.TokensInput,
			TokensOutput: role.TokensOutput,
			CostUsd:      role.CostUSD,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
