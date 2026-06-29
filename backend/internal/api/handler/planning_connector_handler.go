package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// PlanningConnectorHandler implements the persisted planning connector endpoints
// (config for status write-back) and the live status-options probe. Every handler is
// gated to the project OWNER or a global admin.
type PlanningConnectorHandler struct {
	svc *service.PlanningConnectorService
}

// NewPlanningConnectorHandler creates a new PlanningConnectorHandler.
func NewPlanningConnectorHandler(svc *service.PlanningConnectorService) *PlanningConnectorHandler {
	return &PlanningConnectorHandler{svc: svc}
}

// GetPlanningConnector handles GET /projects/{projectId}/planning/connector.
func (h *PlanningConnectorHandler) GetPlanningConnector(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if !h.authorize(w, r, projectID) {
		return
	}
	conn, err := h.svc.GetConnector(r.Context(), projectID)
	if err != nil {
		writeErrorResponse(w, err) // not-found -> 404 (no connector configured yet)
		return
	}
	writeJSON(w, http.StatusOK, toAPIPlanningConnector(conn))
}

// SetPlanningConnector handles PUT /projects/{projectId}/planning/connector.
func (h *PlanningConnectorHandler) SetPlanningConnector(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if !h.authorize(w, r, projectID) {
		return
	}
	var req SetPlanningConnectorRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	in := service.SetConnectorInput{
		Source:        string(req.Source),
		ProjectURL:    req.ProjectUrl,
		StatusMapping: fromAPIPlanningStatusMapping(req.StatusMapping),
	}
	if req.StatusField != nil {
		in.StatusField = *req.StatusField
	}
	if req.EpicIssueType != nil {
		in.EpicIssueType = *req.EpicIssueType
	}
	if req.DoneOptions != nil {
		in.DoneOptions = *req.DoneOptions
	}
	if req.WritebackEnabled != nil {
		in.WritebackEnabled = *req.WritebackEnabled
	}
	if req.PostRunComment != nil {
		in.PostRunComment = *req.PostRunComment
	}

	conn, err := h.svc.SetConnector(r.Context(), projectID, in)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIPlanningConnector(conn))
}

// GetPlanningStatusOptions handles GET /projects/{projectId}/planning/connector/status-options.
func (h *PlanningConnectorHandler) GetPlanningStatusOptions(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetPlanningStatusOptionsParams) {
	if !h.authorize(w, r, projectID) {
		return
	}
	opts, err := h.svc.StatusOptions(r.Context(), projectID, params.ProjectUrl, params.StatusField)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIPlanningStatusOptions(opts))
}

// authorize loads the project (404 if absent) and enforces owner-or-admin (403).
func (h *PlanningConnectorHandler) authorize(w http.ResponseWriter, r *http.Request, projectID uuid.UUID) bool {
	project, err := h.svc.LoadProject(r.Context(), projectID)
	if err != nil {
		writeErrorResponse(w, err)
		return false
	}
	return requireProjectOwnerOrAdmin(w, r, project)
}

// ─── mappers (domain <-> generated API types) ───────────────────────────────────

func toAPIPlanningConnector(c *model.PlanningConnector) PlanningConnector {
	out := PlanningConnector{
		ProjectId:        c.ProjectID,
		Source:           PlanningConnectorSource(c.Source),
		ProjectUrl:       c.ProjectURL,
		WritebackEnabled: c.WritebackEnabled,
		PostRunComment:   c.PostRunComment,
		StatusMapping:    toAPIPlanningStatusMapping(c.StatusMapping),
	}
	statusField := c.StatusField
	out.StatusField = &statusField
	epicType := c.EpicIssueType
	out.EpicIssueType = &epicType
	if c.DoneOptions != nil {
		opts := append([]string(nil), c.DoneOptions...)
		out.DoneOptions = &opts
	}
	createdAt := c.CreatedAt
	out.CreatedAt = &createdAt
	updatedAt := c.UpdatedAt
	out.UpdatedAt = &updatedAt
	return out
}

func toAPIPlanningStatusMapping(m model.PlanningStatusMapping) *PlanningStatusMapping {
	return &PlanningStatusMapping{
		Backlog: m.Backlog,
		Running: m.Running,
		Done:    m.Done,
		Failed:  m.Failed,
	}
}

func fromAPIPlanningStatusMapping(m *PlanningStatusMapping) model.PlanningStatusMapping {
	if m == nil {
		return model.PlanningStatusMapping{}
	}
	return model.PlanningStatusMapping{
		Backlog: m.Backlog,
		Running: m.Running,
		Done:    m.Done,
		Failed:  m.Failed,
	}
}

func toAPIPlanningStatusOptions(o port.PlanningStatusOptions) PlanningStatusOptions {
	out := PlanningStatusOptions{
		FieldName: o.FieldName,
		Options:   make([]PlanningStatusOption, len(o.Options)),
	}
	if o.FieldID != "" {
		fieldID := o.FieldID
		out.FieldId = &fieldID
	}
	for i, opt := range o.Options {
		out.Options[i] = PlanningStatusOption{Id: opt.ID, Name: opt.Name}
	}
	return out
}
