package action_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/action"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// --- Notification-specific mocks ---

type notificationMockEventPublisher struct {
	mu              sync.Mutex
	publishFn       func(ctx context.Context, event model.Event) error
	publishedEvents []model.Event
}

func (m *notificationMockEventPublisher) Publish(ctx context.Context, event model.Event) error {
	m.mu.Lock()
	m.publishedEvents = append(m.publishedEvents, event)
	m.mu.Unlock()
	if m.publishFn != nil {
		return m.publishFn(ctx, event)
	}
	return nil
}

type notificationMockStoryRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Story, error)
}

func (m *notificationMockStoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, apperrors.NewNotFound("story", id)
}

func (m *notificationMockStoryRepo) GetByKey(_ context.Context, _ uuid.UUID, _ string) (*model.Story, error) {
	return nil, apperrors.NewNotFound("story", uuid.Nil)
}

func (m *notificationMockStoryRepo) GetBySourceRef(_ context.Context, _ uuid.UUID, _, _ string) (*model.Story, error) {
	return nil, apperrors.NewNotFound("story", uuid.Nil)
}
func (m *notificationMockStoryRepo) CreateFromImport(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *notificationMockStoryRepo) UpdateFromImport(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *notificationMockStoryRepo) UpdateProvenanceOnly(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}

func (m *notificationMockStoryRepo) Create(_ context.Context, _ *model.Story) (*model.Story, error) {
	return nil, nil
}

func (m *notificationMockStoryRepo) Update(_ context.Context, _ *model.Story) (*model.Story, error) {
	return nil, nil
}

func (m *notificationMockStoryRepo) UpdateStoryCurrentStage(_ context.Context, id uuid.UUID, currentStage *string) (*model.Story, error) {
	return &model.Story{ID: id, CurrentStage: currentStage}, nil
}

func (m *notificationMockStoryRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *notificationMockStoryRepo) ListByProject(_ context.Context, _ uuid.UUID, _ int32, _ int32) ([]*model.Story, error) {
	return nil, nil
}

func (m *notificationMockStoryRepo) ListByStatus(_ context.Context, _ uuid.UUID, _ []string, _ int32, _ int32) ([]*model.Story, error) {
	return nil, nil
}

func (m *notificationMockStoryRepo) ListByEpic(_ context.Context, _ uuid.UUID, _ int32, _ int32) ([]*model.Story, error) {
	return nil, nil
}

func (m *notificationMockStoryRepo) CountByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *notificationMockStoryRepo) CountByStatus(_ context.Context, _ uuid.UUID, _ []string) (int64, error) {
	return 0, nil
}

// --- Tests ---

func TestNotificationAction_Name(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	eventPub := &notificationMockEventPublisher{}
	storyRepo := &notificationMockStoryRepo{}

	notif := action.NewNotificationAction(eventPub, storyRepo, logger)

	if got := notif.Name(); got != "notification" {
		t.Errorf("Name() = %q, want %q", got, "notification")
	}
}

func TestNotificationAction_Execute_HappyPath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	eventPub := &notificationMockEventPublisher{}
	storyRepo := &notificationMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:  id,
				Key: "S-01",
			}, nil
		},
	}

	notif := action.NewNotificationAction(eventPub, storyRepo, logger)

	projectID := uuid.New()
	storyID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()

	runCtx := &model.RunContext{
		ProjectID: projectID,
		StoryID:   storyID,
		Run: &model.Run{
			ID: runID,
		},
		RunStep: &model.RunStep{
			ID:       stepID,
			StepName: "deploy",
		},
		Metadata: map[string]interface{}{
			"message":     "Story {story_key} deployed successfully",
			"branch_name": "feat/S-01-login",
		},
	}

	err := notif.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Verify event published
	if len(eventPub.publishedEvents) != 1 {
		t.Fatalf("published %d events, want 1", len(eventPub.publishedEvents))
	}

	event := eventPub.publishedEvents[0]

	if event.EntityType != "notification" {
		t.Errorf("EntityType = %q, want %q", event.EntityType, "notification")
	}
	if event.Action != "sent" {
		t.Errorf("Action = %q, want %q", event.Action, "sent")
	}
	if event.EntityID != stepID {
		t.Errorf("EntityID = %v, want %v", event.EntityID, stepID)
	}
	if event.ProjectID != projectID {
		t.Errorf("ProjectID = %v, want %v", event.ProjectID, projectID)
	}

	// Verify payload
	var payload map[string]string
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	expectedMessage := "Story S-01 deployed successfully"
	if payload["message"] != expectedMessage {
		t.Errorf("payload[message] = %q, want %q", payload["message"], expectedMessage)
	}
	if payload["story_key"] != "S-01" {
		t.Errorf("payload[story_key] = %q, want %q", payload["story_key"], "S-01")
	}
	if payload["step_id"] != stepID.String() {
		t.Errorf("payload[step_id] = %q, want %q", payload["step_id"], stepID.String())
	}
	if payload["run_id"] != runID.String() {
		t.Errorf("payload[run_id] = %q, want %q", payload["run_id"], runID.String())
	}
}

func TestNotificationAction_Execute_TemplateRendering(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	eventPub := &notificationMockEventPublisher{}
	storyRepo := &notificationMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:  id,
				Key: "S-03",
			}, nil
		},
	}

	notif := action.NewNotificationAction(eventPub, storyRepo, logger)

	projectID := uuid.New()
	storyID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()

	runCtx := &model.RunContext{
		ProjectID: projectID,
		StoryID:   storyID,
		Run: &model.Run{
			ID: runID,
		},
		RunStep: &model.RunStep{
			ID:       stepID,
			StepName: "create-pr",
		},
		Metadata: map[string]interface{}{
			"message":     "Branch {branch_name} created for {story_key}",
			"branch_name": "feat/S-03-login",
		},
	}

	err := notif.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Verify rendered message
	event := eventPub.publishedEvents[0]
	var payload map[string]string
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	expectedMessage := "Branch feat/S-03-login created for S-03"
	if payload["message"] != expectedMessage {
		t.Errorf("payload[message] = %q, want %q", payload["message"], expectedMessage)
	}
}

func TestNotificationAction_Execute_MissingMessageConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	eventPub := &notificationMockEventPublisher{}
	storyRepo := &notificationMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:  id,
				Key: "S-02",
			}, nil
		},
	}

	notif := action.NewNotificationAction(eventPub, storyRepo, logger)

	projectID := uuid.New()
	storyID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()

	runCtx := &model.RunContext{
		ProjectID: projectID,
		StoryID:   storyID,
		Run: &model.Run{
			ID: runID,
		},
		RunStep: &model.RunStep{
			ID:       stepID,
			StepName: "notify",
		},
		Metadata: map[string]interface{}{}, // No message key
	}

	err := notif.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Verify default message was used
	if len(eventPub.publishedEvents) != 1 {
		t.Fatalf("published %d events, want 1", len(eventPub.publishedEvents))
	}

	event := eventPub.publishedEvents[0]
	var payload map[string]string
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	expectedMessage := "Pipeline step notify completed"
	if payload["message"] != expectedMessage {
		t.Errorf("payload[message] = %q, want %q", payload["message"], expectedMessage)
	}
}

func TestNotificationAction_Execute_EventPublisherFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	eventPub := &notificationMockEventPublisher{
		publishFn: func(_ context.Context, _ model.Event) error {
			return apperrors.NewInternal("publish failed", nil)
		},
	}
	storyRepo := &notificationMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:  id,
				Key: "S-04",
			}, nil
		},
	}

	notif := action.NewNotificationAction(eventPub, storyRepo, logger)

	projectID := uuid.New()
	storyID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()

	runCtx := &model.RunContext{
		ProjectID: projectID,
		StoryID:   storyID,
		Run: &model.Run{
			ID: runID,
		},
		RunStep: &model.RunStep{
			ID:       stepID,
			StepName: "notify",
		},
		Metadata: map[string]interface{}{
			"message": "Test message",
		},
	}

	// Should not return error even if publisher fails (non-fatal)
	err := notif.Execute(context.Background(), runCtx)
	if err != nil {
		t.Errorf("Execute() error = %v, want nil (EventPublisher failures are non-fatal)", err)
	}
}

func TestNotificationAction_Execute_StoryLookupFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	eventPub := &notificationMockEventPublisher{}
	storyRepo := &notificationMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return nil, apperrors.NewNotFound("story", id)
		},
	}

	notif := action.NewNotificationAction(eventPub, storyRepo, logger)

	projectID := uuid.New()
	storyID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()

	runCtx := &model.RunContext{
		ProjectID: projectID,
		StoryID:   storyID,
		Run: &model.Run{
			ID: runID,
		},
		RunStep: &model.RunStep{
			ID:       stepID,
			StepName: "notify",
		},
		Metadata: map[string]interface{}{
			"message": "Story {story_key} failed",
		},
	}

	// Should not return error even if story lookup fails
	err := notif.Execute(context.Background(), runCtx)
	if err != nil {
		t.Errorf("Execute() error = %v, want nil (story lookup failures are non-fatal)", err)
	}

	// Verify event was still published with empty story_key
	if len(eventPub.publishedEvents) != 1 {
		t.Fatalf("published %d events, want 1", len(eventPub.publishedEvents))
	}

	event := eventPub.publishedEvents[0]
	var payload map[string]string
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if payload["story_key"] != "" {
		t.Errorf("payload[story_key] = %q, want empty string", payload["story_key"])
	}

	// Message should have empty story_key placeholder
	expectedMessage := "Story  failed"
	if payload["message"] != expectedMessage {
		t.Errorf("payload[message] = %q, want %q", payload["message"], expectedMessage)
	}
}

func (m *notificationMockStoryRepo) CountByEpicGroupedByStatus(_ context.Context, _ uuid.UUID) (model.StoryCounts, error) {
	return model.StoryCounts{}, nil
}
