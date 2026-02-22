package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// AddProjectUserRequest is the request body for adding a user to a project.
type AddProjectUserRequest struct {
	UserID uuid.UUID         `json:"user_id"`
	Role   model.ProjectRole `json:"role,omitempty"`
}

// ProjectMemberResponse is the response body for a project member.
type ProjectMemberResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	UserRole    string    `json:"user_role"`
	ProjectRole string    `json:"project_role"`
	AssignedAt  time.Time `json:"assigned_at"`
}

// ProjectUserHandler implements project-user association HTTP handlers.
type ProjectUserHandler struct {
	service *service.ProjectUserService
}

// NewProjectUserHandler creates a new ProjectUserHandler.
func NewProjectUserHandler(svc *service.ProjectUserService) *ProjectUserHandler {
	return &ProjectUserHandler{service: svc}
}

// AddUser handles POST /api/v1/projects/{id}/users (admin only).
func (h *ProjectUserHandler) AddUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	idStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(idStr)
	if err != nil {
		writeErrorResponse(w, errors.NewValidation("id", "invalid project ID format"))
		return
	}

	var req AddProjectUserRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	if req.UserID == uuid.Nil {
		writeErrorResponse(w, errors.NewValidation("user_id", "is required"))
		return
	}

	role := req.Role
	if role == "" {
		role = model.ProjectRoleMember
	}

	pu, err := h.service.AddUser(r.Context(), projectID, req.UserID, role)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, pu)
}

// RemoveUser handles DELETE /api/v1/projects/{id}/users/{user_id} (admin only).
func (h *ProjectUserHandler) RemoveUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	idStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(idStr)
	if err != nil {
		writeErrorResponse(w, errors.NewValidation("id", "invalid project ID format"))
		return
	}

	userIDStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeErrorResponse(w, errors.NewValidation("user_id", "invalid user ID format"))
		return
	}

	if err := h.service.RemoveUser(r.Context(), projectID, userID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListMembers handles GET /api/v1/projects/{id}/users (admin or assigned user).
func (h *ProjectUserHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(idStr)
	if err != nil {
		writeErrorResponse(w, errors.NewValidation("id", "invalid project ID format"))
		return
	}

	members, err := h.service.ListMembers(r.Context(), projectID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := make([]ProjectMemberResponse, len(members))
	for i, m := range members {
		resp[i] = ProjectMemberResponse{
			UserID:      m.UserID,
			Email:       m.Email,
			Name:        m.Name,
			UserRole:    string(m.UserRole),
			ProjectRole: string(m.ProjectRole),
			AssignedAt:  m.AssignedAt,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
