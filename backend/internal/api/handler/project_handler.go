package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// ProjectHandler implements project-related HTTP handlers.
type ProjectHandler struct {
	service *service.ProjectService
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{service: svc}
}

// ListProjects handles GET /projects.
// Any authenticated user can list projects.
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request, params ListProjectsParams) {
	page := 1
	perPage := 20
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}
	if params.PerPage != nil && *params.PerPage > 0 {
		perPage = *params.PerPage
	}

	result, err := h.service.List(r.Context(), page, perPage)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := ProjectList{
		Data: make([]Project, len(result.Projects)),
		Pagination: Pagination{
			Total:   int(result.Total),
			Page:    page,
			PerPage: perPage,
		},
	}
	for i, p := range result.Projects {
		resp.Data[i] = toAPIProject(p)
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreateProject handles POST /projects.
// Only admin users can create projects.
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	userID, _ := middleware.UserIDFromContext(r.Context())

	params := service.CreateProjectParams{
		Name:        req.Name,
		Description: req.Description,
	}
	if userID != uuid.Nil {
		params.OwnerID = &userID
	}

	project, err := h.service.Create(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAPIProject(project))
}

// GetProject handles GET /projects/{id}.
// Any authenticated user can get a project.
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request, id IdPath) {
	project, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIProject(project))
}

// UpdateProject handles PUT /projects/{id}.
// Only admin users can update projects.
func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request, id IdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	params := service.UpdateProjectParams{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	}

	project, err := h.service.Update(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIProject(project))
}

// DeleteProject handles DELETE /projects/{id}.
// Only admin users can delete projects.
func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request, id IdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
