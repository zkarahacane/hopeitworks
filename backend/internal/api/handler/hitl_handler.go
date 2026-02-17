package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// HITLHandler implements HITL-related HTTP handlers.
type HITLHandler struct {
	service *service.HITLService
}

// NewHITLHandler creates a new HITLHandler.
func NewHITLHandler(svc *service.HITLService) *HITLHandler {
	return &HITLHandler{service: svc}
}

// ListPendingHITLRequests handles GET /projects/{projectId}/hitl/pending.
func (h *HITLHandler) ListPendingHITLRequests(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	pending, total, err := h.service.ListPendingByProject(r.Context(), projectID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	items := make([]PendingHITLRequestItem, len(pending))
	for i, p := range pending {
		items[i] = PendingHITLRequestItem{
			Id:        p.ID,
			RunId:     p.RunID,
			StepId:    p.StepID,
			StoryKey:  p.StoryKey,
			CreatedAt: p.CreatedAt,
		}
		if p.DiffURL != nil {
			items[i].DiffUrl = p.DiffURL
		}
	}

	resp := PendingHITLRequestList{
		Data:  items,
		Total: int(total),
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetHITLRequest handles GET /hitl-requests/{hitlRequestId}.
func (h *HITLHandler) GetHITLRequest(w http.ResponseWriter, r *http.Request, hitlRequestID HITLRequestIdPath) {
	req, err := h.service.GetByID(r.Context(), hitlRequestID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIHITLRequest(req))
}

// ApproveHITLRequest handles POST /hitl-requests/{hitlRequestId}/approve.
func (h *HITLHandler) ApproveHITLRequest(w http.ResponseWriter, r *http.Request, hitlRequestID HITLRequestIdPath) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, errors.NewUnauthorized("authentication required"))
		return
	}

	req, err := h.service.Approve(r.Context(), hitlRequestID, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIHITLRequest(req))
}

// RejectHITLRequest handles POST /hitl-requests/{hitlRequestId}/reject.
func (h *HITLHandler) RejectHITLRequest(w http.ResponseWriter, r *http.Request, hitlRequestID HITLRequestIdPath) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, errors.NewUnauthorized("authentication required"))
		return
	}

	var body RejectHITLRequestJSONRequestBody
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
			return
		}
	}

	req, err := h.service.Reject(r.Context(), hitlRequestID, userID, body.Reason)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIHITLRequest(req))
}

// toAPIHITLRequest converts a domain HITLRequest to the API HITLRequest type.
func toAPIHITLRequest(req *model.HITLRequest) HITLRequest {
	r := HITLRequest{
		Id:        req.ID,
		RunStepId: req.RunStepID,
		GateType:  req.GateType,
		Status:    HITLRequestStatus(req.Status),
		CreatedAt: req.CreatedAt,
	}
	if req.DiffContent != nil {
		r.DiffContent = req.DiffContent
	}
	if req.ResolvedAt != nil {
		r.ResolvedAt = req.ResolvedAt
	}
	if req.ResolvedBy != nil {
		r.ResolvedBy = req.ResolvedBy
	}
	if req.RejectionReason != nil {
		r.RejectionReason = req.RejectionReason
	}
	return r
}
