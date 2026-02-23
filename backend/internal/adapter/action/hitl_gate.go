package action

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// HITLGateAction implements model.Action for suspending a pipeline step
// pending human approval. It creates a HITL request and transitions the
// step to waiting_approval.
type HITLGateAction struct {
	hitlRepo           port.HITLRepository
	runRepo            port.RunRepository
	gitProviderFactory port.GitProviderFactory
	eventPub           port.EventPublisher
	storyRepo          port.StoryRepository
	logger             *slog.Logger
}

// NewHITLGateAction creates a new HITL gate action.
func NewHITLGateAction(
	hitlRepo port.HITLRepository,
	runRepo port.RunRepository,
	gitProviderFactory port.GitProviderFactory,
	eventPub port.EventPublisher,
	storyRepo port.StoryRepository,
	logger *slog.Logger,
) *HITLGateAction {
	return &HITLGateAction{
		hitlRepo:           hitlRepo,
		runRepo:            runRepo,
		gitProviderFactory: gitProviderFactory,
		eventPub:           eventPub,
		storyRepo:          storyRepo,
		logger:             logger,
	}
}

// Name returns the action identifier.
func (a *HITLGateAction) Name() string {
	return "hitl_gate"
}

// Execute creates a HITL request, transitions the step to waiting_approval,
// and publishes a hitl_gate.pending event. Returns nil on success because
// suspension is not an error — it is the intended outcome.
func (a *HITLGateAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
	// 1. Fetch story for context (story key for event payload)
	story, err := a.storyRepo.GetByID(ctx, runCtx.StoryID)
	if err != nil {
		return fmt.Errorf("fetch story: %w", err)
	}

	// 2. Attempt to fetch PR diff (non-fatal)
	var diffContent *string
	if prURL, ok := runCtx.Metadata["pr_url"].(string); ok && prURL != "" {
		gitProvider, factoryErr := a.gitProviderFactory.ForProjectID(ctx, runCtx.ProjectID)
		if factoryErr != nil {
			a.logger.Warn("failed to resolve git provider for PR diff, proceeding without diff",
				"error", factoryErr)
		}
		var diff string
		var diffErr error
		if gitProvider != nil {
			diff, diffErr = gitProvider.GetPRDiff(ctx, prURL)
		} else {
			diffErr = factoryErr
		}
		if diffErr != nil {
			a.logger.Warn("failed to fetch PR diff, proceeding without diff",
				"pr_url", prURL, "error", diffErr)
		} else {
			diffContent = &diff
		}
	}

	// 3. Create HITL request
	req := &model.HITLRequest{
		ID:          uuid.New(),
		RunStepID:   runCtx.RunStep.ID,
		GateType:    "approval",
		DiffContent: diffContent,
		Status:      model.HITLStatusPending,
		CreatedAt:   time.Now(),
	}
	created, err := a.hitlRepo.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("create HITL request: %w", err)
	}

	// 4. Transition step to waiting_approval
	now := time.Now()
	if _, err := a.runRepo.UpdateRunStepStatus(ctx, runCtx.RunStep.ID,
		model.StepStatusWaitingApproval, &now, nil, nil); err != nil {
		return fmt.Errorf("update step to waiting_approval: %w", err)
	}

	// 5. Publish hitl_gate.pending event
	a.publishPendingEvent(ctx, runCtx, story.Key, created.ID)

	return nil
}

// publishPendingEvent publishes a hitl_gate.pending event.
func (a *HITLGateAction) publishPendingEvent(ctx context.Context, runCtx *model.RunContext, storyKey string, hitlRequestID uuid.UUID) {
	payload := map[string]string{
		"run_id":          runCtx.Run.ID.String(),
		"step_id":         runCtx.RunStep.ID.String(),
		"story_key":       storyKey,
		"hitl_request_id": hitlRequestID.String(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		a.logger.Error("failed to marshal hitl_gate.pending payload", "error", err)
		return
	}

	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  runCtx.ProjectID,
		EntityType: "hitl_gate",
		EntityID:   runCtx.RunStep.ID,
		Action:     "pending",
		Payload:    payloadJSON,
	}

	if err := a.eventPub.Publish(ctx, event); err != nil {
		a.logger.Error("failed to publish hitl_gate.pending event", "error", err)
	}
}
