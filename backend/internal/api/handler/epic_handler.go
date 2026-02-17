package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// EpicHandler implements epic-related HTTP handlers.
type EpicHandler struct {
	service *service.EpicService
}

// NewEpicHandler creates a new EpicHandler.
func NewEpicHandler(svc *service.EpicService) *EpicHandler {
	return &EpicHandler{service: svc}
}

// ListEpics handles GET /projects/{projectId}/epics.
func (h *EpicHandler) ListEpics(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListEpicsParams) {
	page := 1
	perPage := 20
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}
	if params.PerPage != nil && *params.PerPage > 0 {
		perPage = *params.PerPage
	}

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
		resp.Data[i] = toAPIEpic(e)
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreateEpic handles POST /projects/{projectId}/epics.
// Only admin users can create epics.
func (h *EpicHandler) CreateEpic(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	var req CreateEpicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
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

	writeJSON(w, http.StatusCreated, toAPIEpic(epic))
}

// GetEpic handles GET /projects/{projectId}/epics/{epicId}.
func (h *EpicHandler) GetEpic(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, epicID EpicIdPath) {
	epic, err := h.service.GetByID(r.Context(), epicID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIEpic(epic))
}

// UpdateEpic handles PUT /projects/{projectId}/epics/{epicId}.
// Only admin users can update epics.
func (h *EpicHandler) UpdateEpic(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, epicID EpicIdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	var req UpdateEpicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
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

	writeJSON(w, http.StatusOK, toAPIEpic(epic))
}

// DeleteEpic handles DELETE /projects/{projectId}/epics/{epicId}.
// Only admin users can delete epics.
func (h *EpicHandler) DeleteEpic(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, epicID EpicIdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	if err := h.service.Delete(r.Context(), epicID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// toAPIEpic converts a domain Epic to the API Epic type.
func toAPIEpic(e *model.Epic) Epic {
	epic := Epic{
		Id:        e.ID,
		ProjectId: e.ProjectID,
		Name:      e.Name,
		Status:    EpicStatus(e.Status),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
	if e.Description != nil {
		epic.Description = e.Description
	}
	return epic
}
