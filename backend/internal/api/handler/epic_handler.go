package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// EpicHandler implements epic-related HTTP handlers.
type EpicHandler struct {
	service     *service.EpicService
	userService *service.ProjectUserService
}

// NewEpicHandler creates a new EpicHandler.
func NewEpicHandler(svc *service.EpicService, userSvc *service.ProjectUserService) *EpicHandler {
	return &EpicHandler{service: svc, userService: userSvc}
}

// checkProjectAccess verifies the current user has access to the given project.
func (h *EpicHandler) checkProjectAccess(r *http.Request, projectID uuid.UUID) error {
	if middleware.IsAdmin(r.Context()) {
		return nil
	}
	userID, _ := middleware.UserIDFromContext(r.Context())
	isMember, err := h.userService.IsUserInProject(r.Context(), projectID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.NewForbidden("You are not a member of this project")
	}
	return nil
}

// ListEpics handles GET /projects/{id}/epics.
func (h *EpicHandler) ListEpics(w http.ResponseWriter, r *http.Request, projectID IdPath, params ListEpicsParams) {
	if err := h.checkProjectAccess(r, projectID); err != nil {
		writeErrorResponse(w, err)
		return
	}

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

// CreateEpic handles POST /projects/{id}/epics.
func (h *EpicHandler) CreateEpic(w http.ResponseWriter, r *http.Request, projectID IdPath) {
	if err := h.checkProjectAccess(r, projectID); err != nil {
		writeErrorResponse(w, err)
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
		status := model.EpicStatus(*req.Status)
		params.Status = &status
	}

	epic, err := h.service.Create(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAPIEpic(epic))
}

// GetEpic handles GET /projects/{id}/epics/{epicId}.
func (h *EpicHandler) GetEpic(w http.ResponseWriter, r *http.Request, projectID IdPath, epicID EpicIdPath) {
	if err := h.checkProjectAccess(r, projectID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	epic, err := h.service.GetByID(r.Context(), epicID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	if epic.ProjectID != projectID {
		writeErrorResponse(w, errors.NewNotFound("epic", epicID))
		return
	}

	writeJSON(w, http.StatusOK, toAPIEpic(epic))
}

// UpdateEpic handles PUT /projects/{id}/epics/{epicId}.
func (h *EpicHandler) UpdateEpic(w http.ResponseWriter, r *http.Request, projectID IdPath, epicID EpicIdPath) {
	if err := h.checkProjectAccess(r, projectID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	// Verify the epic belongs to this project
	existing, err := h.service.GetByID(r.Context(), epicID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	if existing.ProjectID != projectID {
		writeErrorResponse(w, errors.NewNotFound("epic", epicID))
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
		status := model.EpicStatus(*req.Status)
		params.Status = &status
	}

	epic, err := h.service.Update(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIEpic(epic))
}

// DeleteEpic handles DELETE /projects/{id}/epics/{epicId}.
func (h *EpicHandler) DeleteEpic(w http.ResponseWriter, r *http.Request, projectID IdPath, epicID EpicIdPath) {
	if err := h.checkProjectAccess(r, projectID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	// Verify the epic belongs to this project
	existing, err := h.service.GetByID(r.Context(), epicID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	if existing.ProjectID != projectID {
		writeErrorResponse(w, errors.NewNotFound("epic", epicID))
		return
	}

	if err := h.service.Delete(r.Context(), epicID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
