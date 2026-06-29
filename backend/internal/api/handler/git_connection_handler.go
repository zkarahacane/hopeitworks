package handler

import (
	stderrors "errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// GitConnectionHandler implements the project git-connection HTTP handlers
// (encrypted PAT management). Every handler is gated to the project OWNER or a
// global admin; the secret is write-only and never returned.
type GitConnectionHandler struct {
	svc *service.GitConnectionService
}

// NewGitConnectionHandler creates a new GitConnectionHandler.
func NewGitConnectionHandler(svc *service.GitConnectionService) *GitConnectionHandler {
	return &GitConnectionHandler{svc: svc}
}

// GetProjectGitConnection handles GET /projects/{id}/git-connection.
func (h *GitConnectionHandler) GetProjectGitConnection(w http.ResponseWriter, r *http.Request, id IdPath) {
	if !h.authorize(w, r, id) {
		return
	}
	view, err := h.svc.Status(r.Context(), id)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIGitConnectionStatus(view))
}

// SetProjectGitConnection handles PUT /projects/{id}/git-connection.
func (h *GitConnectionHandler) SetProjectGitConnection(w http.ResponseWriter, r *http.Request, id IdPath) {
	if !h.authorize(w, r, id) {
		return
	}
	var req SetGitConnectionRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	in := service.SetGitConnectionInput{
		Provider: "github",
		Validate: true,
	}
	if req.Provider != nil {
		in.Provider = string(*req.Provider)
	}
	if req.Token != nil {
		in.Token = *req.Token
	}
	if req.Validate != nil {
		in.Validate = *req.Validate
	}

	actor, _ := middleware.UserIDFromContext(r.Context())
	view, err := h.svc.Set(r.Context(), id, actor, in)
	if err != nil {
		writeGitConnectionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIGitConnectionStatus(view))
}

// ClearProjectGitConnection handles DELETE /projects/{id}/git-connection.
func (h *GitConnectionHandler) ClearProjectGitConnection(w http.ResponseWriter, r *http.Request, id IdPath) {
	if !h.authorize(w, r, id) {
		return
	}
	actor, _ := middleware.UserIDFromContext(r.Context())
	if err := h.svc.Clear(r.Context(), id, actor); err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// TestProjectGitConnection handles POST /projects/{id}/git-connection/test.
func (h *GitConnectionHandler) TestProjectGitConnection(w http.ResponseWriter, r *http.Request, id IdPath) {
	if !h.authorize(w, r, id) {
		return
	}
	var req TestGitConnectionRequest
	// Body is optional; tolerate an empty/absent body.
	if r.Body != nil && r.ContentLength != 0 {
		if !decodeJSONBody(w, r, &req) {
			return
		}
	}

	actor, _ := middleware.UserIDFromContext(r.Context())
	result, err := h.svc.Test(r.Context(), id, actor, req.Token)
	if err != nil {
		writeGitConnectionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIGitConnectionTestResult(result))
}

// authorize loads the project (404 if absent) and enforces owner-or-admin (403).
func (h *GitConnectionHandler) authorize(w http.ResponseWriter, r *http.Request, projectID uuid.UUID) bool {
	project, err := h.svc.LoadProject(r.Context(), projectID)
	if err != nil {
		writeErrorResponse(w, err)
		return false
	}
	return requireProjectOwnerOrAdmin(w, r, project)
}

// requireProjectOwnerOrAdmin allows the request when the caller is a global admin or
// the project's owner; otherwise it writes a 403 and returns false.
func requireProjectOwnerOrAdmin(w http.ResponseWriter, r *http.Request, project *model.Project) bool {
	if middleware.IsAdmin(r.Context()) {
		return true
	}
	if uid, ok := middleware.UserIDFromContext(r.Context()); ok && project.OwnerID != nil && *project.OwnerID == uid {
		return true
	}
	writeErrorResponse(w, errors.NewForbidden("project owner or admin access required"))
	return false
}

// writeGitConnectionError maps the transient-probe sentinel to HTTP 503; all other
// errors flow through the standard DomainError mapping.
func writeGitConnectionError(w http.ResponseWriter, err error) {
	if stderrors.Is(err, service.ErrGitConnectionProbeUnavailable) {
		writeJSON(w, http.StatusServiceUnavailable, Error{
			Error: struct {
				Code    string                  `json:"code"`
				Details *map[string]interface{} `json:"details,omitempty"`
				Message string                  `json:"message"`
			}{
				Code:    service.CodeGitConnectionProbeUnavailable,
				Message: "the provider was unreachable or rate-limited; the stored connection was left unchanged",
			},
		})
		return
	}
	writeErrorResponse(w, err)
}

// ─── mappers (domain view -> generated API types) ───────────────────────────────

func toAPIGitConnectionStatus(v *service.GitConnectionView) GitConnectionStatus {
	out := GitConnectionStatus{
		Configured:      v.Configured,
		Kind:            GitConnectionStatusKind(v.Kind),
		Provider:        GitConnectionStatusProvider(v.Provider),
		Status:          GitConnectionStatusStatus(v.Status),
		AccountLogin:    v.AccountLogin,
		SecretLast4:     v.SecretLast4,
		ExpiresAt:       v.ExpiresAt,
		LastValidatedAt: v.LastValidatedAt,
		ValidationError: v.ValidationError,
	}
	if v.Source != "" {
		src := GitConnectionStatusSource(v.Source)
		out.Source = &src
	}
	if v.TokenType != nil {
		tt := GitConnectionStatusTokenType(*v.TokenType)
		out.TokenType = &tt
	}
	if len(v.Scopes) > 0 {
		scopes := append([]string(nil), v.Scopes...)
		out.Scopes = &scopes
	}
	return out
}

func toAPIGitConnectionTestResult(v *service.GitConnectionTestView) GitConnectionTestResult {
	out := GitConnectionTestResult{
		Ok:           v.Ok,
		Status:       GitConnectionTestResultStatus(v.Status),
		TokenType:    GitConnectionTestResultTokenType(v.TokenType),
		AccountLogin: v.AccountLogin,
		ExpiresAt:    v.ExpiresAt,
	}
	if v.Message != "" {
		msg := v.Message
		out.Message = &msg
	}
	if len(v.Scopes) > 0 {
		scopes := append([]string(nil), v.Scopes...)
		out.Scopes = &scopes
	}
	if len(v.MissingScopes) > 0 {
		missing := append([]string(nil), v.MissingScopes...)
		out.MissingScopes = &missing
	}
	return out
}
