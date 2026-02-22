package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// RunHandler implements run-related HTTP handlers.
type RunHandler struct {
	service *service.RunService
}

// NewRunHandler creates a new RunHandler.
func NewRunHandler(svc *service.RunService) *RunHandler {
	return &RunHandler{service: svc}
}

// ListRunsByProject handles GET /projects/{projectId}/runs.
func (h *RunHandler) ListRunsByProject(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListRunsByProjectParams) {
	page, perPage := paginationDefaults(params.Page, params.PerPage)

	result, err := h.service.ListRunsByProject(r.Context(), projectID, page, perPage)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := RunList{
		Data: make([]Run, len(result.Runs)),
		Pagination: Pagination{
			Total:   int(result.Total),
			Page:    page,
			PerPage: perPage,
		},
	}
	for i, run := range result.Runs {
		resp.Data[i] = toAPIRun(run)
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreateRun handles POST /projects/{projectId}/runs.
func (h *RunHandler) CreateRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	var req CreateRunRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	configJSON, err := json.Marshal(req.PipelineConfig)
	if err != nil {
		writeErrorResponse(w, errors.NewValidation("pipeline_config", "failed to marshal config"))
		return
	}

	params := service.CreateRunParams{
		ProjectID:      projectID,
		StoryID:        req.StoryId,
		PipelineConfig: configJSON,
	}

	run, err := h.service.CreateRun(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAPIRunWithSteps(run))
}

// ListRunsByStory handles GET /stories/{storyId}/runs.
func (h *RunHandler) ListRunsByStory(w http.ResponseWriter, r *http.Request, storyID StoryIdPath, params ListRunsByStoryParams) {
	page, perPage := paginationDefaults(params.Page, params.PerPage)

	result, err := h.service.ListRunsByStory(r.Context(), storyID, page, perPage)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := RunList{
		Data: make([]Run, len(result.Runs)),
		Pagination: Pagination{
			Total:   int(result.Total),
			Page:    page,
			PerPage: perPage,
		},
	}
	for i, run := range result.Runs {
		resp.Data[i] = toAPIRun(run)
	}

	writeJSON(w, http.StatusOK, resp)
}

// LaunchRun handles POST /projects/{projectId}/stories/{storyId}/runs.
func (h *RunHandler) LaunchRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, storyID StoryIdPath) {
	run, err := h.service.LaunchRun(r.Context(), projectID, storyID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAPIRunWithSteps(run))
}

// GetRun handles GET /runs/{runId}.
func (h *RunHandler) GetRun(w http.ResponseWriter, r *http.Request, runID RunIdPath) {
	run, err := h.service.GetRun(r.Context(), runID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIRunWithSteps(run))
}

// PauseRun handles POST /projects/{projectId}/runs/{runId}/pause.
func (h *RunHandler) PauseRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
	run, err := h.service.PauseRun(r.Context(), projectID, runID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIRun(run))
}

// ResumeRun handles POST /projects/{projectId}/runs/{runId}/resume.
func (h *RunHandler) ResumeRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
	run, err := h.service.ResumeRun(r.Context(), projectID, runID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIRun(run))
}

// PauseEpicRun handles POST /projects/{projectId}/epics/{epicId}/runs/{runId}/pause.
func (h *RunHandler) PauseEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath, runID RunIdPath) {
	run, err := h.service.PauseEpicRun(r.Context(), projectID, epicID, runID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIRun(run))
}

// ResumeEpicRun handles POST /projects/{projectId}/epics/{epicId}/runs/{runId}/resume.
func (h *RunHandler) ResumeEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath, runID RunIdPath) {
	run, err := h.service.ResumeEpicRun(r.Context(), projectID, epicID, runID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIRun(run))
}

// toAPIRun converts a domain Run to the API Run type.
func toAPIRun(r *model.Run) Run {
	run := Run{
		Id:        r.ID,
		ProjectId: r.ProjectID,
		StoryId:   r.StoryID,
		Status:    RunStatus(r.Status),
		Progress:  r.Progress,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
	run.PipelineConfigSnapshot = rawJSONToMap(r.PipelineConfigSnapshot)
	if r.StartedAt != nil {
		run.StartedAt = r.StartedAt
	}
	if r.CompletedAt != nil {
		run.CompletedAt = r.CompletedAt
	}
	if r.ErrorMessage != nil {
		run.ErrorMessage = r.ErrorMessage
	}
	if r.StoryKey != "" {
		run.StoryKey = &r.StoryKey
	}
	return run
}

// RetryStep handles POST /runs/{runId}/steps/{stepId}/retry.
func (h *RunHandler) RetryStep(w http.ResponseWriter, r *http.Request, runID RunIdPath, stepID StepIdPath) {
	run, err := h.service.RetryStep(r.Context(), runID, stepID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIRunWithSteps(run))
}

// toAPIRunStep converts a domain RunStep to the API RunStep type.
func toAPIRunStep(s *model.RunStep) RunStep {
	step := RunStep{
		Id:        s.ID,
		RunId:     s.RunID,
		StepName:  s.StepName,
		StepOrder: s.StepOrder,
		Action:    s.Action,
		Status:    RunStepStatus(s.Status),
		CreatedAt: s.CreatedAt,
	}
	if s.StartedAt != nil {
		step.StartedAt = s.StartedAt
	}
	if s.CompletedAt != nil {
		step.CompletedAt = s.CompletedAt
	}
	if s.ErrorMessage != nil {
		step.ErrorMessage = s.ErrorMessage
	}
	if s.ContainerID != nil {
		step.ContainerId = s.ContainerID
	}
	if s.LogTail != nil {
		step.LogTail = s.LogTail
	}
	if s.ParentStepID != nil {
		step.ParentStepId = s.ParentStepID
	}
	if s.RetryCount > 0 {
		rc := s.RetryCount
		step.RetryCount = &rc
	}
	if s.RetryType != nil {
		rt := RunStepRetryType(*s.RetryType)
		step.RetryType = &rt
	}
	return step
}

// toAPIRunWithSteps converts a domain Run (with steps) to the API RunWithSteps type.
func toAPIRunWithSteps(r *model.Run) RunWithSteps {
	rws := RunWithSteps{
		Id:        r.ID,
		ProjectId: r.ProjectID,
		StoryId:   r.StoryID,
		Status:    RunWithStepsStatus(r.Status),
		Progress:  r.Progress,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		Steps:     make([]RunStep, len(r.Steps)),
	}
	rws.PipelineConfigSnapshot = rawJSONToMap(r.PipelineConfigSnapshot)
	if r.StartedAt != nil {
		rws.StartedAt = r.StartedAt
	}
	if r.CompletedAt != nil {
		rws.CompletedAt = r.CompletedAt
	}
	if r.ErrorMessage != nil {
		rws.ErrorMessage = r.ErrorMessage
	}
	if r.StoryKey != "" {
		rws.StoryKey = &r.StoryKey
	}
	for i := range r.Steps {
		rws.Steps[i] = toAPIRunStep(&r.Steps[i])
	}
	return rws
}

// rawJSONToMap converts json.RawMessage to *map[string]interface{} for the API type.
func rawJSONToMap(raw json.RawMessage) *map[string]interface{} {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return &m
}
