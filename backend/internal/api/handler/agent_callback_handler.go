package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// LogCallbackRequest is the request body for the log callback endpoint.
type LogCallbackRequest struct {
	Lines []string `json:"lines"`
}

// CostCallbackRequest is the request body for the cost callback endpoint.
// InputTokens/OutputTokens are int64 to match agent-runtime's costPayload wire type.
type CostCallbackRequest struct {
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	Model        string  `json:"model"`
	CostUSD      float64 `json:"cost_usd"`
}

// StatusCallbackRequest is the request body for the status callback endpoint.
type StatusCallbackRequest struct {
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}

// AgentCallbackHandler handles HTTP callbacks from agent containers.
type AgentCallbackHandler struct {
	eventPub    port.EventPublisher
	costSvc     *service.CostService
	statusStore port.CallbackStatusStore
	runRepo     port.RunRepository
}

// NewAgentCallbackHandler creates a new handler for agent container callbacks.
func NewAgentCallbackHandler(
	eventPub port.EventPublisher,
	costSvc *service.CostService,
	statusStore port.CallbackStatusStore,
	runRepo port.RunRepository,
) *AgentCallbackHandler {
	return &AgentCallbackHandler{
		eventPub:    eventPub,
		costSvc:     costSvc,
		statusStore: statusStore,
		runRepo:     runRepo,
	}
}

// HandleLogs receives log lines from an agent container and publishes them to the event system.
// POST /internal/agent/callback/runs/{runId}/steps/{stepId}/logs
func (h *AgentCallbackHandler) HandleLogs(w http.ResponseWriter, r *http.Request) {
	runID, stepID, ok := h.parseRunStepIDs(w, r)
	if !ok {
		return
	}

	var req LogCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"INVALID_BODY","message":"invalid JSON body"}}`, http.StatusBadRequest)
		return
	}

	// Look up the run to get the project ID for the event
	run, err := h.runRepo.GetRun(r.Context(), runID)
	if err != nil {
		http.Error(w, `{"error":{"code":"RUN_NOT_FOUND","message":"run not found"}}`, http.StatusNotFound)
		return
	}

	for _, line := range req.Lines {
		logEvent := model.LogEvent{Message: line, Type: "stdout"}
		payload, marshalErr := json.Marshal(logEvent)
		if marshalErr != nil {
			continue
		}

		event := model.Event{
			ID:         uuid.New(),
			ProjectID:  run.ProjectID,
			EntityType: "log",
			EntityID:   stepID,
			Action:     "emitted",
			Payload:    payload,
		}
		_ = h.eventPub.Publish(r.Context(), event)
	}

	// Persist log lines to run_steps.log_tail (bounded tail, non-fatal on failure).
	if len(req.Lines) > 0 {
		joined := strings.Join(req.Lines, "\n") + "\n"
		if err := h.runRepo.AppendStepLogTail(r.Context(), stepID, joined); err != nil {
			slog.Warn("failed to persist log tail", "step_id", stepID, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// HandleCost receives a cost event from an agent container and records it.
// POST /internal/agent/callback/runs/{runId}/steps/{stepId}/cost
func (h *AgentCallbackHandler) HandleCost(w http.ResponseWriter, r *http.Request) {
	_, stepID, ok := h.parseRunStepIDs(w, r)
	if !ok {
		return
	}

	var req CostCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"INVALID_BODY","message":"invalid JSON body"}}`, http.StatusBadRequest)
		return
	}

	// Look up the step to get the run, then the run to get the project ID
	step, err := h.runRepo.GetRunStep(r.Context(), stepID)
	if err != nil {
		http.Error(w, `{"error":{"code":"STEP_NOT_FOUND","message":"step not found"}}`, http.StatusNotFound)
		return
	}

	run, err := h.runRepo.GetRun(r.Context(), step.RunID)
	if err != nil {
		http.Error(w, `{"error":{"code":"RUN_NOT_FOUND","message":"run not found"}}`, http.StatusNotFound)
		return
	}

	costEvents := []model.CostEvent{
		{
			InputTokens:  req.InputTokens,
			OutputTokens: req.OutputTokens,
			Model:        req.Model,
			CostUSD:      req.CostUSD,
		},
	}

	// Cost recording failure is non-fatal — log but continue
	_ = h.costSvc.RecordStepCost(r.Context(), stepID, run.ProjectID, costEvents, nil)

	w.WriteHeader(http.StatusOK)
}

// HandleStatus receives the final exit status from an agent container.
// POST /internal/agent/callback/runs/{runId}/steps/{stepId}/status
func (h *AgentCallbackHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	_, stepID, ok := h.parseRunStepIDs(w, r)
	if !ok {
		return
	}

	var req StatusCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"INVALID_BODY","message":"invalid JSON body"}}`, http.StatusBadRequest)
		return
	}

	if err := h.statusStore.SetStatus(r.Context(), stepID, req.ExitCode, req.Error); err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL","message":"failed to set status"}}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// parseRunStepIDs extracts and validates runId and stepId from URL parameters.
func (h *AgentCallbackHandler) parseRunStepIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	runIDStr := chi.URLParam(r, "runId")
	stepIDStr := chi.URLParam(r, "stepId")

	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		http.Error(w, `{"error":{"code":"INVALID_RUN_ID","message":"invalid run ID"}}`, http.StatusBadRequest)
		return uuid.Nil, uuid.Nil, false
	}

	stepID, err := uuid.Parse(stepIDStr)
	if err != nil {
		http.Error(w, `{"error":{"code":"INVALID_STEP_ID","message":"invalid step ID"}}`, http.StatusBadRequest)
		return uuid.Nil, uuid.Nil, false
	}

	return runID, stepID, true
}
