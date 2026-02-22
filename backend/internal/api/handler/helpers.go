package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// decodeJSONBody decodes the request body into dst and writes a validation
// error response if the body contains invalid JSON. Returns true on success.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeErrorResponse(w, errors.NewValidation("body", "invalid JSON"))
		return false
	}
	return true
}

// requireAdmin checks whether the current user is an admin and writes a
// forbidden error response if not. Returns true when the user is an admin.
func requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !middleware.IsAdmin(r.Context()) {
		writeErrorResponse(w, errors.NewForbidden("Admin access required"))
		return false
	}
	return true
}

// paginationDefaults extracts page and perPage from optional pointer params,
// defaulting to page=1 and perPage=20 when nil or non-positive.
func paginationDefaults(page, perPage *int) (int, int) {
	p := 1
	pp := 20
	if page != nil && *page > 0 {
		p = *page
	}
	if perPage != nil && *perPage > 0 {
		pp = *perPage
	}
	return p, pp
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErrorResponse maps a domain error to an HTTP error response.
func writeErrorResponse(w http.ResponseWriter, err error) {
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		domainErr = errors.NewInternal("unexpected error", err)
	}

	status := mapCategoryToStatus(domainErr.Category)
	resp := Error{
		Error: struct {
			Code    string                  `json:"code"`
			Details *map[string]interface{} `json:"details,omitempty"`
			Message string                  `json:"message"`
		}{
			Code:    domainErr.Code,
			Message: domainErr.Message,
		},
	}

	writeJSON(w, status, resp)
}

func mapCategoryToStatus(cat errors.ErrorCategory) int {
	switch cat {
	case errors.CategoryNotFound:
		return http.StatusNotFound
	case errors.CategoryValidation:
		return http.StatusBadRequest
	case errors.CategoryConflict:
		return http.StatusConflict
	case errors.CategoryUnauthorized:
		return http.StatusUnauthorized
	case errors.CategoryForbidden:
		return http.StatusForbidden
	case errors.CategoryInvalidState:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// toAPIProject converts a domain Project to the API Project type.
func toAPIProject(p *model.Project) Project {
	gitProvider := ProjectGitProvider(p.GitProvider)
	agentRuntime := ProjectAgentRuntime(p.AgentRuntime)

	proj := Project{
		Id:                   p.ID,
		Name:                 p.Name,
		OwnerId:              uuid.Nil,
		GitProvider:          &gitProvider,
		AgentRuntime:         &agentRuntime,
		MaxBudget:            p.MaxBudget,
		CircuitBreakerCount:  p.CircuitBreakerCount,
		CircuitBreakerActive: p.CircuitBreakerActive,
		CircuitBreakerMax:    p.CircuitBreakerMax,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
	if p.Description != nil {
		proj.Description = p.Description
	}
	if p.OwnerID != nil {
		proj.OwnerId = *p.OwnerID
	}
	if p.RepoURL != nil {
		proj.RepoUrl = p.RepoURL
	}
	if p.GitTokenEnv != nil {
		proj.GitTokenEnv = p.GitTokenEnv
	}
	if p.DefaultModel != nil {
		proj.DefaultModel = p.DefaultModel
	}
	return proj
}

// toAPIUser converts a domain User to the API User type.
func toAPIUser(u *model.User) User {
	return User{
		Id:        u.ID,
		Email:     openapi_types.Email(u.Email),
		Name:      u.Name,
		Role:      UserRole(u.Role),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
