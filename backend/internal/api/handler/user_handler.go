package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// UserHandler implements user management HTTP handlers.
type UserHandler struct {
	service *service.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

// ListUsers handles GET /users.
// Only admin users can list users.
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request, params ListUsersParams) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	page, perPage := paginationDefaults(params.Page, params.PerPage)

	result, err := h.service.List(r.Context(), page, perPage)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := UserList{
		Data: make([]User, len(result.Users)),
		Pagination: Pagination{
			Total:   int(result.Total),
			Page:    page,
			PerPage: perPage,
		},
	}
	for i, u := range result.Users {
		resp.Data[i] = toAPIUser(u)
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetUser handles GET /users/{id}.
// Only admin users can get user details.
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request, id IdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	user, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIUser(user))
}

// UpdateUser handles PUT /users/{id}.
// Only admin users can update users.
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request, id IdPath) {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	params := service.UpdateUserParams{
		ID: id,
	}
	if req.Name != nil {
		params.Name = req.Name
	}
	if req.Email != nil {
		email := string(*req.Email)
		params.Email = &email
	}
	if req.Role != nil {
		role := model.Role(*req.Role)
		params.Role = &role
	}

	user, err := h.service.Update(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIUser(user))
}

// DeleteUser handles DELETE /users/{id}.
// Only admin users can delete users.
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request, id IdPath) {
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
