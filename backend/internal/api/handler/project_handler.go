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
	service        *service.ProjectService
	userService    *service.ProjectUserService
	circuitBreaker *service.CircuitBreakerService
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(svc *service.ProjectService, userSvc *service.ProjectUserService, cbSvc *service.CircuitBreakerService) *ProjectHandler {
	return &ProjectHandler{service: svc, userService: userSvc, circuitBreaker: cbSvc}
}

// checkProjectAccess verifies the current user has access to the given project.
// Admins bypass the check; non-admins must be project members.
func (h *ProjectHandler) checkProjectAccess(r *http.Request, projectID uuid.UUID) error {
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

// ListProjects handles GET /projects.
// Admins see all projects; non-admins see only their assigned projects.
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request, params ListProjectsParams) {
	page := 1
	perPage := 20
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}
	if params.PerPage != nil && *params.PerPage > 0 {
		perPage = *params.PerPage
	}

	var result *service.ListResult
	var err error

	if middleware.IsAdmin(r.Context()) {
		result, err = h.service.List(r.Context(), page, perPage)
	} else {
		userID, _ := middleware.UserIDFromContext(r.Context())
		result, err = h.userService.ListProjectsForUser(r.Context(), userID, page, perPage)
	}
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
// Admins can access any project; non-admins must be project members.
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request, id IdPath) {
	if err := h.checkProjectAccess(r, id); err != nil {
		writeErrorResponse(w, err)
		return
	}

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

// ResetCircuitBreaker handles POST /projects/{id}/circuit-breaker/reset.
// Only admin users can reset the circuit breaker.
func (h *ProjectHandler) ResetCircuitBreaker(w http.ResponseWriter, r *http.Request, id IdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	if err := h.circuitBreaker.Reset(r.Context(), id); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
