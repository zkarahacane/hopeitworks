package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// AgentBundleHandler serves the fetch-at-startup capability bundle to agent containers.
//
// The endpoint is part of the internal, container-token-authenticated channel (the same
// auth as the log/cost/status callbacks). It is intentionally NOT part of the generated
// /api/v1 OpenAPI surface: those endpoints use cookie/JWT auth, whereas this one is
// authenticated by the short-lived container token (see middleware.InternalAuth) and is
// mounted directly under /internal/agent/callback alongside the other callbacks.
//
// SECURITY: the agent is identified from the validated container token, never from the
// request body or query — a container cannot fetch another agent's bundle or secrets.
type AgentBundleHandler struct {
	bundleSvc *service.BundleService
	logger    *slog.Logger
}

// NewAgentBundleHandler creates a new AgentBundleHandler.
func NewAgentBundleHandler(bundleSvc *service.BundleService, logger *slog.Logger) *AgentBundleHandler {
	return &AgentBundleHandler{bundleSvc: bundleSvc, logger: logger}
}

// HandleBundle returns the composed RuntimeBundle for the authenticated container.
// GET /internal/agent/callback/bundle
//
// On any composition error it logs and returns an empty bundle with 200 — the runtime
// must never be blocked from starting by a capability problem (warn+skip invariant).
func (h *AgentBundleHandler) HandleBundle(w http.ResponseWriter, r *http.Request) {
	ct, ok := middleware.ContainerTokenFromContext(r.Context())
	if !ok {
		// InternalAuth must run before this handler; absence means misconfiguration.
		http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"missing container token"}}`, http.StatusUnauthorized)
		return
	}

	bundle, err := h.bundleSvc.ComposeBundle(r.Context(), ct.AgentID)
	if err != nil {
		h.logger.Warn("bundle composition failed, returning empty bundle",
			"agent_id", ct.AgentID, "step_id", ct.StepID, "error", err)
		bundle = model.RuntimeBundle{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encErr := json.NewEncoder(w).Encode(bundle); encErr != nil {
		h.logger.Warn("failed to encode bundle response", "step_id", ct.StepID, "error", encErr)
	}
}
