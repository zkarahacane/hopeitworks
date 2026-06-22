package handler

import (
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// AgentHandler implements agent-related HTTP handlers.
type AgentHandler struct {
	service *service.AgentService
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(svc *service.AgentService) *AgentHandler {
	return &AgentHandler{service: svc}
}

// ListGlobalAgents handles GET /agents (global scope only).
func (h *AgentHandler) ListGlobalAgents(w http.ResponseWriter, r *http.Request, params ListGlobalAgentsParams) {
	result, err := h.service.ListGlobal(r.Context())
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	page, perPage := paginationDefaults(params.Page, params.PerPage)

	resp := struct {
		Data       []Agent    `json:"data"`
		Pagination Pagination `json:"pagination"`
	}{
		Data: make([]Agent, len(result.Agents)),
		Pagination: Pagination{
			Total:   result.Total,
			Page:    page,
			PerPage: perPage,
		},
	}
	for i, a := range result.Agents {
		resp.Data[i] = toAPIAgent(a)
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListProjectAgents handles GET /projects/{projectId}/agents (project + global).
func (h *AgentHandler) ListProjectAgents(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListProjectAgentsParams) {
	result, err := h.service.ListMerged(r.Context(), projectID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	page, perPage := paginationDefaults(params.Page, params.PerPage)

	resp := struct {
		Data       []Agent    `json:"data"`
		Pagination Pagination `json:"pagination"`
	}{
		Data: make([]Agent, len(result.Agents)),
		Pagination: Pagination{
			Total:   result.Total,
			Page:    page,
			PerPage: perPage,
		},
	}
	for i, a := range result.Agents {
		resp.Data[i] = toAPIAgent(a)
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreateAgent handles POST /projects/{projectId}/agents.
// Only admin users can create agents.
func (h *AgentHandler) CreateAgent(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if !requireAdmin(w, r) {
		return
	}

	var req CreateAgentRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	scope := "project"
	if req.Scope != nil {
		scope = string(*req.Scope)
	}

	provider := ""
	if req.Provider != nil {
		provider = string(*req.Provider)
	}

	pid := projectID
	params := service.CreateAgentParams{
		ProjectID:       &pid,
		Name:            req.Name,
		Model:           req.Model,
		Image:           req.Image,
		StackRef:        req.StackRef,
		TemplateContent: req.TemplateContent,
		Scope:           scope,
		Provider:        provider,
	}

	agent, err := h.service.Create(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAPIAgent(agent))
}

// GetAgent handles GET /projects/{projectId}/agents/{agentId}.
func (h *AgentHandler) GetAgent(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, agentID AgentIdPath) {
	agent, err := h.service.GetByID(r.Context(), agentID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIAgent(agent))
}

// UpdateAgent handles PUT /projects/{projectId}/agents/{agentId}.
// Only admin users can update agents.
func (h *AgentHandler) UpdateAgent(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, agentID AgentIdPath) {
	if !requireAdmin(w, r) {
		return
	}

	var req UpdateAgentRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	var provider *string
	if req.Provider != nil {
		p := string(*req.Provider)
		provider = &p
	}

	params := service.UpdateAgentParams{
		ID:              agentID,
		Name:            req.Name,
		Model:           req.Model,
		Image:           req.Image,
		StackRef:        req.StackRef,
		TemplateContent: req.TemplateContent,
		Provider:        provider,
	}

	agent, err := h.service.Update(r.Context(), params)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIAgent(agent))
}

// DeleteAgent handles DELETE /projects/{projectId}/agents/{agentId}.
// Only admin users can delete agents.
func (h *AgentHandler) DeleteAgent(w http.ResponseWriter, r *http.Request, _ ProjectIdPath, agentID AgentIdPath) {
	if !requireAdmin(w, r) {
		return
	}

	if err := h.service.Delete(r.Context(), agentID); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// toAPIAgent converts a domain Agent to the API Agent type.
func toAPIAgent(a *model.Agent) Agent {
	return Agent{
		Id:              a.ID,
		Name:            a.Name,
		Model:           a.Model,
		Image:           a.Image,
		StackRef:        a.StackRef,
		TemplateContent: a.TemplateContent,
		Scope:           AgentScope(a.Scope),
		Provider:        AgentProvider(a.Provider),
		ProjectId:       a.ProjectID,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}
}
