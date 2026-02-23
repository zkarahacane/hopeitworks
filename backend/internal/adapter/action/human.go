package action

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// HumanAction implements model.Action for suspending a pipeline step
// pending explicit human approval. Unlike HITLGateAction, it does not
// fetch a PR diff — it presents a configurable message and optional instructions
// to the reviewer.
type HumanAction struct {
	hitlRepo  port.HITLRepository
	runRepo   port.RunRepository
	storyRepo port.StoryRepository
	eventPub  port.EventPublisher
	logger    *slog.Logger
}

// NewHumanAction creates a new human approval action.
func NewHumanAction(
	hitlRepo port.HITLRepository,
	runRepo port.RunRepository,
	storyRepo port.StoryRepository,
	eventPub port.EventPublisher,
	logger *slog.Logger,
) *HumanAction {
	return &HumanAction{
		hitlRepo:  hitlRepo,
		runRepo:   runRepo,
		storyRepo: storyRepo,
		eventPub:  eventPub,
		logger:    logger,
	}
}

// Name returns the action identifier.
func (a *HumanAction) Name() string { return "human" }

// Execute creates a HITL request with GateType "human", transitions the step
// to waiting_approval, and publishes a human.pending event. Returns nil on
// success because suspension is not an error — it is the intended outcome.
func (a *HumanAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
	cfg := runCtx.RunStep.Config

	msgTemplate := cfg["message"]
	if msgTemplate == "" {
		msgTemplate = "Human approval required for step {step_name}"
	}
	instructions := cfg["instructions"]

	// Fetch story for event payload context
	story, err := a.storyRepo.GetByID(ctx, runCtx.StoryID)
	if err != nil {
		return fmt.Errorf("fetch story: %w", err)
	}

	// Build template variables from run context
	vars := map[string]string{
		"story_key": story.Key,
		"step_name": runCtx.RunStep.StepName,
	}
	if branchName, ok := runCtx.Metadata["branch_name"].(string); ok {
		vars["branch_name"] = branchName
	}
	if prURL, ok := runCtx.Metadata["pr_url"].(string); ok {
		vars["pr_url"] = prURL
	}

	message := renderHumanTemplate(msgTemplate, vars)
	renderedInstructions := renderHumanTemplate(instructions, vars)

	// Create HITL request
	req := &model.HITLRequest{
		ID:          uuid.New(),
		RunStepID:   runCtx.RunStep.ID,
		GateType:    "human",
		DiffContent: nil,
		Message:     &message,
		Status:      model.HITLStatusPending,
		CreatedAt:   time.Now(),
	}
	created, err := a.hitlRepo.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("create HITL request: %w", err)
	}

	// Transition step to waiting_approval
	now := time.Now()
	if _, err := a.runRepo.UpdateRunStepStatus(ctx, runCtx.RunStep.ID,
		model.StepStatusWaitingApproval, &now, nil, nil); err != nil {
		return fmt.Errorf("update step to waiting_approval: %w", err)
	}

	a.publishHumanPendingEvent(ctx, runCtx, story.Key, created.ID, message, renderedInstructions)

	return nil
}

// publishHumanPendingEvent publishes a human.pending event. Errors are logged
// but do not fail the step suspension.
func (a *HumanAction) publishHumanPendingEvent(
	ctx context.Context,
	runCtx *model.RunContext,
	storyKey string,
	hitlRequestID uuid.UUID,
	message, instructions string,
) {
	payload := map[string]string{
		"run_id":          runCtx.Run.ID.String(),
		"step_id":         runCtx.RunStep.ID.String(),
		"story_key":       storyKey,
		"hitl_request_id": hitlRequestID.String(),
		"message":         message,
		"instructions":    instructions,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		a.logger.Warn("failed to marshal human.pending payload", "error", err)
		return
	}

	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  runCtx.ProjectID,
		EntityType: "human",
		EntityID:   runCtx.RunStep.ID,
		Action:     "pending",
		Payload:    payloadJSON,
	}

	if err := a.eventPub.Publish(ctx, event); err != nil {
		a.logger.Warn("failed to publish human.pending event", "error", err)
	}
}

// renderHumanTemplate performs simple {key} replacement in a template string.
func renderHumanTemplate(tmpl string, vars map[string]string) string {
	result := tmpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	return result
}
