package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// ProfileHandler handles self-service profile endpoints for the authenticated user.
type ProfileHandler struct {
	userService *service.UserService
}

// NewProfileHandler creates a new ProfileHandler.
func NewProfileHandler(svc *service.UserService) *ProfileHandler {
	return &ProfileHandler{userService: svc}
}

// GetMyProfile handles GET /users/me.
func (h *ProfileHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, errors.NewUnauthorized("authentication required"))
		return
	}

	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, errors.NewUnauthorized("user not found"))
		return
	}

	writeJSON(w, http.StatusOK, toAPIUser(user))
}

// UpdateMyProfile handles PUT /users/me.
func (h *ProfileHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, errors.NewUnauthorized("authentication required"))
		return
	}

	var req UpdateMyProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	if req.Name == nil && req.Email == nil {
		writeErrorResponse(w, errors.NewValidation("body", "at least one field (name or email) must be provided"))
		return
	}

	params := service.UpdateProfileParams{
		ID: userID,
	}
	if req.Name != nil {
		params.Name = req.Name
	}
	if req.Email != nil {
		email := string(*req.Email)
		params.Email = &email
	}

	user, err := h.userService.UpdateProfile(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIUser(user))
}

// ChangeMyPassword handles PUT /users/me/password.
func (h *ProfileHandler) ChangeMyPassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, errors.NewUnauthorized("authentication required"))
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	if req.CurrentPassword == "" {
		writeErrorResponse(w, errors.NewValidation("current_password", "must not be empty"))
		return
	}
	if req.NewPassword == "" {
		writeErrorResponse(w, errors.NewValidation("new_password", "must not be empty"))
		return
	}

	err := h.userService.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if err == service.ErrInvalidCurrentPassword {
			writeErrorResponse(w, &errors.DomainError{
				Category: errors.CategoryUnauthorized,
				Code:     "INVALID_CREDENTIALS",
				Message:  "current password is incorrect",
			})
			return
		}
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
