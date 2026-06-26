package handler

import (
	"net/http"

	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// PlanningHandler implements planning-source import HTTP handlers.
//
// Phase 0 ships a stub. Declaring the OpenAPI path
// POST /projects/{projectId}/planning/import forces the generated
// ServerInterface to require an ImportPlanning method, so the binary must
// compile against it ahead of the real implementation. Phase 1 replaces the
// body below with the PlanningImportService call. No business logic lives here.
type PlanningHandler struct{}

// NewPlanningHandler creates a new PlanningHandler.
func NewPlanningHandler() *PlanningHandler {
	return &PlanningHandler{}
}

// ImportPlanning handles POST /projects/{projectId}/planning/import.
//
// Stub (Phase 0): returns a not-implemented internal error via the standard
// DomainError mapping. Phase 1 wires the real import service + adapters.
func (h *PlanningHandler) ImportPlanning(w http.ResponseWriter, _ *http.Request, _ ProjectIdPath) {
	writeErrorResponse(w, errors.NewInternal("planning import not implemented", nil))
}
