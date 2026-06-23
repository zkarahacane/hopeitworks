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

	req, err := h.service.Reject(r.Context(), hitlRequestID, userID, &body.Reason)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIHITLRequest(req))
}

// ResolveHITLRequest handles POST /hitl-requests/{hitlRequestId}/resolve.
// It closes a probe_halt gate with an enriched action (resume/override/send_back/
// skip/abort).
func (h *HITLHandler) ResolveHITLRequest(w http.ResponseWriter, r *http.Request, hitlRequestID HITLRequestIdPath) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, errors.NewUnauthorized("authentication required"))
		return
	}

	var body ResolveHITLRequestJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return
	}

	req, err := h.service.Resolve(r.Context(), hitlRequestID, userID, string(body.Action))
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIHITLRequest(req))
}

// ListProbeHalts handles GET /probe-halts — the batch triage inbox of pending
// probe_halt gates, optionally scoped to a project.
func (h *HITLHandler) ListProbeHalts(w http.ResponseWriter, r *http.Request, params ListProbeHaltsParams) {
	halts, err := h.service.ListProbeHalts(r.Context(), params.ProjectId)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	items := make([]ProbeHalt, len(halts))
	for i, ph := range halts {
		items[i] = toAPIProbeHalt(ph)
	}
	writeJSON(w, http.StatusOK, ProbeHaltList{Data: items, Total: len(items)})
}

// ListHITLRequests handles GET /hitl-requests with optional status filter and pagination.
func (h *HITLHandler) ListHITLRequests(w http.ResponseWriter, r *http.Request, params ListHITLRequestsParams) {
	page, perPage := paginationDefaults(params.Page, params.PerPage)

	var status *string
	if params.Status != nil {
		s := string(*params.Status)
		status = &s
	}

	items, total, err := h.service.ListAll(r.Context(), status, page, perPage)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	data := make([]HITLRequest, len(items))
	for i, item := range items {
		data[i] = toAPIHITLRequest(item)
	}

	resp := HITLRequestList{
		Data: data,
		Pagination: Pagination{
			Page:    page,
			PerPage: perPage,
			Total:   int(total),
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetHITLRequestByStep handles GET /hitl-requests/by-step/{stepId}.
func (h *HITLHandler) GetHITLRequestByStep(w http.ResponseWriter, r *http.Request, stepID StepIdPath) {
	req, err := h.service.GetByStepID(r.Context(), stepID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIHITLRequest(req))
}

// toAPIHITLRequest converts a domain HITLRequest to the API HITLRequest type.
func toAPIHITLRequest(req *model.HITLRequest) HITLRequest {
	r := HITLRequest{
		Id:         req.ID,
		RunStepId:  req.RunStepID,
		StepId:     req.RunStepID, // StepId maps to run_step_id in the domain model
		GateType:   req.GateType,
		Status:     HITLRequestStatus(req.Status),
		CreatedAt:  req.CreatedAt,
		StoryKey:   "",
		StoryTitle: "",
	}
	if req.DiffContent != nil {
		r.DiffContent = req.DiffContent
	}
	if req.ResolvedAt != nil {
		r.ResolvedAt = req.ResolvedAt
	}
	if req.ResolvedBy != nil {
		r.ReviewerId = req.ResolvedBy
	}
	if req.RejectionReason != nil {
		r.RejectionReason = req.RejectionReason
	}
	if req.ResolutionAction != nil {
		action := HITLRequestResolutionAction(*req.ResolutionAction)
		r.ResolutionAction = &action
	}
	r.HaltReason = toAPIHaltReason(req.HaltReason)
	return r
}

// toAPIHaltReason converts a domain HaltReason to the API type; nil-safe.
func toAPIHaltReason(hr *model.HaltReason) *HaltReason {
	if hr == nil {
		return nil
	}
	probe := HaltReasonProbe(hr.Probe)
	observed := hr.Observed
	threshold := hr.Threshold
	out := &HaltReason{
		Observed:  &observed,
		Threshold: &threshold,
	}
	if hr.Probe != "" {
		out.Probe = &probe
	}
	if hr.OnFail != "" {
		out.OnFail = &hr.OnFail
	}
	if hr.Unit != "" {
		out.Unit = &hr.Unit
	}
	if hr.Detail != "" {
		out.Detail = &hr.Detail
	}
	return out
}

// toAPIProbeHalt converts a domain ProbeHalt to the API type.
func toAPIProbeHalt(ph *model.ProbeHalt) ProbeHalt {
	out := ProbeHalt{
		Id:         ph.ID,
		RunStepId:  ph.RunStepID,
		RunId:      ph.RunID,
		ProjectId:  ph.ProjectID,
		StoryKey:   ph.StoryKey,
		StoryTitle: ph.StoryTitle,
		StepName:   ph.StepName,
		CreatedAt:  ph.CreatedAt,
		HaltReason: toAPIHaltReason(ph.HaltReason),
	}
	if ph.StageName != "" {
		out.StageName = &ph.StageName
	}
	return out
}
