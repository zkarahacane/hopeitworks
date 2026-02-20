package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// HITLHandler implements HITL approval/rejection HTTP handlers.
type HITLHandler struct {
	service     *service.HITLService
	userService *service.ProjectUserService
}

// NewHITLHandler creates a new HITLHandler.
func NewHITLHandler(svc *service.HITLService, userSvc *service.ProjectUserService) *HITLHandler {
	return &HITLHandler{service: svc, userService: userSvc}
}

// checkProjectAccess verifies the current user has access to the given project.
func (h *HITLHandler) checkProjectAccess(r *http.Request, projectID uuid.UUID) error {
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

// ApproveHITLGate handles POST /projects/{projectId}/runs/{runId}/hitl/approve.
func (h *HITLHandler) ApproveHITLGate(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
	if err := h.checkProjectAccess(r, projectID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	reviewerID, _ := middleware.UserIDFromContext(r.Context())
	result, err := h.service.Approve(r.Context(), projectID, runID, reviewerID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := HITLActionResponse{
		RunId:         openapi_types.UUID(result.RunID),
		HitlRequestId: openapi_types.UUID(result.HITLRequestID),
		Status:        result.Status,
	}
	writeJSON(w, http.StatusAccepted, resp)
}

// RejectHITLGate handles POST /projects/{projectId}/runs/{runId}/hitl/reject.
func (h *HITLHandler) RejectHITLGate(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
	if err := h.checkProjectAccess(r, projectID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
			return
		}
	}

	reviewerID, _ := middleware.UserIDFromContext(r.Context())
	result, err := h.service.Reject(r.Context(), projectID, runID, reviewerID, body.Reason)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	resp := HITLActionResponse{
		RunId:         openapi_types.UUID(result.RunID),
		HitlRequestId: openapi_types.UUID(result.HITLRequestID),
		Status:        result.Status,
	}
	writeJSON(w, http.StatusAccepted, resp)
}
