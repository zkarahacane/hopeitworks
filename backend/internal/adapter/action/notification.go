package action

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// NotificationAction implements model.Action for publishing a notification event.
// It renders a configurable message template with RunContext variables and publishes
// a "notification.sent" event via EventPublisher. Failures are non-fatal.
type NotificationAction struct {
	eventPub  port.EventPublisher
	storyRepo port.StoryRepository
	logger    *slog.Logger
}

// NewNotificationAction creates a new NotificationAction.
func NewNotificationAction(
	eventPub port.EventPublisher,
	storyRepo port.StoryRepository,
	logger *slog.Logger,
) *NotificationAction {
	return &NotificationAction{eventPub: eventPub, storyRepo: storyRepo, logger: logger}
}

// Name returns the action name.
func (a *NotificationAction) Name() string { return "notification" }

// Execute publishes a notification event with a rendered message template.
func (a *NotificationAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
	// Read message template from metadata (populated from pipeline config)
	msgTemplate, ok := runCtx.Metadata["message"].(string)
	if !ok || msgTemplate == "" {
		msgTemplate = "Pipeline step {step_name} completed"
	}

	storyKey := ""
	story, err := a.storyRepo.GetByID(ctx, runCtx.StoryID)
	if err != nil {
		a.logger.Warn("notification: failed to fetch story, proceeding with empty story_key",
			"story_id", runCtx.StoryID, "error", err)
	} else {
		storyKey = story.Key
	}

	branchName, _ := runCtx.Metadata["branch_name"].(string)
	prURL, _ := runCtx.Metadata["pr_url"].(string)

	message := renderTemplate(msgTemplate, map[string]string{
		"story_key":   storyKey,
		"step_name":   runCtx.RunStep.StepName,
		"run_id":      runCtx.Run.ID.String(),
		"branch_name": branchName,
		"pr_url":      prURL,
	})

	payload := map[string]string{
		"message":   message,
		"step_id":   runCtx.RunStep.ID.String(),
		"run_id":    runCtx.Run.ID.String(),
		"story_key": storyKey,
	}
	payloadJSON, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		a.logger.Warn("notification: failed to marshal payload", "error", marshalErr)
		return nil // non-fatal
	}

	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  runCtx.ProjectID,
		EntityType: "notification",
		EntityID:   runCtx.RunStep.ID,
		Action:     "sent",
		Payload:    payloadJSON,
	}

	if pubErr := a.eventPub.Publish(ctx, event); pubErr != nil {
		a.logger.Warn("notification: failed to publish event",
			"story_key", storyKey,
			"run_id", runCtx.Run.ID,
			"error", pubErr,
		)
	}

	return nil
}
