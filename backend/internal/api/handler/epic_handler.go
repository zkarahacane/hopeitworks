package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// EpicHandler implements epic-related HTTP handlers.
type EpicHandler struct {
	service   *service.EpicService
	scheduler *service.SchedulerService
	storyRepo port.StoryRepository
	runRepo   port.RunRepository
}

// NewEpicHandler creates a new EpicHandler.
func NewEpicHandler(svc *service.EpicService, scheduler *service.SchedulerService, storyRepo port.StoryRepository, runRepo port.RunRepository) *EpicHandler {
	return &EpicHandler{service: svc, scheduler: scheduler, storyRepo: storyRepo, runRepo: runRepo}
}

// ListEpics handles GET /projects/{projectId}/epics.
func (h *EpicHandler) ListEpics(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListEpicsParams) {
	page, perPage := paginationDefaults(params.Page, params.PerPage)

	result, err := h.service.ListByProject(r.Context(), projectID, page, perPage)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := EpicList{
		Data: make([]Epic, len(result.Epics)),
		Pagination: Pagination{
			Total:   int(result.Total),
			Page:    page,
			PerPage: perPage,
		},
	}
	for i, e := range result.Epics {
		resp.Data[i] = h.toAPIEpic(r.Context(), e)
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreateEpic handles POST /projects/{projectId}/epics.
// Only admin users can create epics.
func (h *EpicHandler) CreateEpic(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if !requireAdmin(w, r) {
		return
	}

	var req CreateEpicRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	params := service.CreateEpicParams{
		ProjectID:   projectID,
		Name:        req.Name,
		Description: req.Description,
	}
	if req.Status != nil {
		params.Status = string(*req.Status)
	}

	epic, err := h.service.Create(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, h.toAPIEpic(r.Context(), epic))
}

// GetEpic handles GET /projects/{projectId}/epics/{epicId}.
func (h *EpicHandler) GetEpic(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, epicID EpicIdPath) {
	epic, err := h.service.GetByID(r.Context(), epicID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, h.toAPIEpic(r.Context(), epic))
}

// UpdateEpic handles PUT /projects/{projectId}/epics/{epicId}.
// Only admin users can update epics.
func (h *EpicHandler) UpdateEpic(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, epicID EpicIdPath) {
	if !requireAdmin(w, r) {
		return
	}

	var req UpdateEpicRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	params := service.UpdateEpicParams{
		ID:          epicID,
		Name:        req.Name,
		Description: req.Description,
	}
	if req.Status != nil {
		s := string(*req.Status)
		params.Status = &s
	}

	epic, err := h.service.Update(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, h.toAPIEpic(r.Context(), epic))
}

// DeleteEpic handles DELETE /projects/{projectId}/epics/{epicId}.
// Only admin users can delete epics.
func (h *EpicHandler) DeleteEpic(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, epicID EpicIdPath) {
	if !requireAdmin(w, r) {
		return
	}

	if err := h.service.Delete(r.Context(), epicID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetEpicDAG handles GET /projects/{projectId}/epics/{epicId}/dag.
func (h *EpicHandler) GetEpicDAG(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, epicID EpicIdPath) {
	stories, err := h.storyRepo.ListByEpic(r.Context(), epicID, 500, 0)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	storyValues := make([]model.Story, len(stories))
	for i, s := range stories {
		storyValues[i] = *s
	}

	dag, err := h.scheduler.BuildDAG(storyValues)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	// Best-effort enrichment of each node with its story's latest-run data
	// (run id/status, container id, cost). Enrichment errors are non-fatal: the
	// DAG still renders with bare nodes.
	storyIDs := make([]uuid.UUID, len(storyValues))
	for i, s := range storyValues {
		storyIDs[i] = s.ID
	}
	runInfo, err := h.runRepo.GetDAGNodeRunInfoByStories(r.Context(), storyIDs)
	if err != nil {
		runInfo = map[uuid.UUID]model.DAGNodeRunInfo{}
	}

	writeJSON(w, http.StatusOK, toEpicDAGResponse(dag, runInfo))
}

// toEpicDAGResponse converts a DAGResult to the API EpicDAGResponse type,
// enriching each node with its story's latest-run data when available.
func toEpicDAGResponse(dag model.DAGResult, runInfo map[uuid.UUID]model.DAGNodeRunInfo) EpicDAGResponse {
	nodes := make([]EpicDAGNode, 0)
	edges := make([]EpicDAGEdge, 0)

	for layer, group := range dag.Groups {
		for _, s := range group {
			node := EpicDAGNode{
				Id:     s.ID,
				Key:    s.Key,
				Title:  s.Title,
				Status: s.Status,
				Layer:  layer,
			}
			if info, ok := runInfo[s.ID]; ok {
				runID := info.RunID
				runStatus := info.RunStatus
				costUSD := info.CostUSD
				node.RunId = &runID
				node.RunStatus = &runStatus
				node.ContainerId = info.ContainerID
				node.CostUsd = &costUSD
			}
			nodes = append(nodes, node)
			for _, dep := range s.DependsOn {
				edges = append(edges, EpicDAGEdge{Source: dep, Target: s.Key})
			}
		}
	}

	return EpicDAGResponse{Nodes: nodes, Edges: edges}
}

// toAPIEpic converts a domain Epic to the API Epic type, populating story_counts
// (per story.status) via a single grouped query. Count errors are non-fatal:
// the epic is still returned with zeroed counts.
func (h *EpicHandler) toAPIEpic(ctx context.Context, e *model.Epic) Epic {
	counts, err := h.storyRepo.CountByEpicGroupedByStatus(ctx, e.ID)
	if err != nil {
		counts = model.StoryCounts{}
	}
	return buildAPIEpic(e, counts)
}

// buildAPIEpic converts a domain Epic plus its story counts to the API Epic type.
func buildAPIEpic(e *model.Epic, counts model.StoryCounts) Epic {
	epic := Epic{
		Id:        e.ID,
		ProjectId: e.ProjectID,
		Name:      e.Name,
		Status:    EpicStatus(e.Status),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		StoryCounts: StoryCounts{
			Backlog: counts.Backlog,
			Running: counts.Running,
			Done:    counts.Done,
			Failed:  counts.Failed,
		},
	}
	if e.Description != nil {
		epic.Description = e.Description
	}
	// Planning provenance (read-only).
	if e.Source != "" {
		src := EpicSource(e.Source)
		epic.Source = &src
	}
	epic.ExternalId = e.ExternalID
	epic.SourceUrl = e.SourceURL
	epic.SyncedAt = e.SyncedAt
	return epic
}
