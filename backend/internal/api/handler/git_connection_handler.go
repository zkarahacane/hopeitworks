package handler

import (
	"net/http"

	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// GitConnectionHandler implements the project git-connection HTTP handlers
// (encrypted PAT management). This is a Phase 0 stub: the contract is frozen in
// api/openapi.yaml and the handler is wired for a green build, but every method
// returns "not implemented". Phase 1 replaces these stubs with the real service.
type GitConnectionHandler struct{}

// NewGitConnectionHandler creates a new GitConnectionHandler. Phase 1 will add a
// *service.GitConnectionService dependency; the stub takes none.
func NewGitConnectionHandler() *GitConnectionHandler {
	return &GitConnectionHandler{}
}

// errGitConnectionNotImplemented is the placeholder error returned by every stub
// method until Phase 1 wires the real service.
func errGitConnectionNotImplemented() error {
	return errors.NewDomainError(
		"NOT_IMPLEMENTED",
		"git connection management is not implemented yet",
		nil,
	)
}

// GetProjectGitConnection handles GET /projects/{id}/git-connection (stub).
func (h *GitConnectionHandler) GetProjectGitConnection(w http.ResponseWriter, _ *http.Request, _ IdPath) {
	writeErrorResponse(w, errGitConnectionNotImplemented())
}

// SetProjectGitConnection handles PUT /projects/{id}/git-connection (stub).
func (h *GitConnectionHandler) SetProjectGitConnection(w http.ResponseWriter, _ *http.Request, _ IdPath) {
	writeErrorResponse(w, errGitConnectionNotImplemented())
}

// ClearProjectGitConnection handles DELETE /projects/{id}/git-connection (stub).
func (h *GitConnectionHandler) ClearProjectGitConnection(w http.ResponseWriter, _ *http.Request, _ IdPath) {
	writeErrorResponse(w, errGitConnectionNotImplemented())
}

// TestProjectGitConnection handles POST /projects/{id}/git-connection/test (stub).
func (h *GitConnectionHandler) TestProjectGitConnection(w http.ResponseWriter, _ *http.Request, _ IdPath) {
	writeErrorResponse(w, errGitConnectionNotImplemented())
}
