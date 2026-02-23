package action_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/action"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// --- Human-specific mocks ---

type humanMockHITLRepo struct {
	createFn func(ctx context.Context, req *model.HITLRequest) (*model.HITLRequest, error)
	created  []*model.HITLRequest
}

func (m *humanMockHITLRepo) Create(_ context.Context, req *model.HITLRequest) (*model.HITLRequest, error) {
	m.created = append(m.created, req)
	if m.createFn != nil {
		return m.createFn(context.Background(), req)
	}
	return req, nil
}

func (m *humanMockHITLRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.HITLRequest, error) {
	return nil, apperrors.NewNotFound("hitl_request", uuid.Nil)
}

func (m *humanMockHITLRepo) GetByRunStepID(_ context.Context, _ uuid.UUID) (*model.HITLRequest, error) {
	return nil, apperrors.NewNotFound("hitl_request", uuid.Nil)
}

func (m *humanMockHITLRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ model.HITLStatus, _ *uuid.UUID, _ *string, _ time.Time) (*model.HITLRequest, error) {
	return nil, nil
}

func (m *humanMockHITLRepo) ListPendingByProject(_ context.Context, _ uuid.UUID) ([]*model.PendingHITLRequest, error) {
	return nil, nil
}

func (m *humanMockHITLRepo) CountPendingByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *humanMockHITLRepo) ListFiltered(_ context.Context, _ *string, _, _ int32) ([]*model.HITLRequest, error) {
	return nil, nil
}

func (m *humanMockHITLRepo) CountFiltered(_ context.Context, _ *string) (int64, error) {
	return 0, nil
}

type humanStepStatusCall struct {
	ID     uuid.UUID
	Status model.StepStatus
}

type humanMockRunRepo struct {
	updateRunStepStatusFn func(ctx context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.RunStep, error)
	statusCalls           []humanStepStatusCall
}

func (m *humanMockRunRepo) CreateRun(_ context.Context, run *model.Run) (*model.Run, error) {
	return run, nil
}
func (m *humanMockRunRepo) GetRun(_ context.Context, _ uuid.UUID) (*model.Run, error) {
	return nil, apperrors.NewNotFound("run", uuid.Nil)
}
func (m *humanMockRunRepo) GetActiveRunByStory(_ context.Context, _ uuid.UUID) (*model.Run, error) {
	return nil, nil
}
func (m *humanMockRunRepo) ListRunsByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *humanMockRunRepo) ListRunsByStory(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *humanMockRunRepo) UpdateRunStatus(_ context.Context, id uuid.UUID, _ model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
	return &model.Run{ID: id}, nil
}
func (m *humanMockRunRepo) CountRunsByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *humanMockRunRepo) CountRunsByStory(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *humanMockRunRepo) CreateRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}
func (m *humanMockRunRepo) GetRunStep(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
	return &model.RunStep{ID: id}, nil
}
func (m *humanMockRunRepo) ListRunStepsByRun(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}
func (m *humanMockRunRepo) UpdateRunStepStatus(ctx context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.RunStep, error) {
	m.statusCalls = append(m.statusCalls, humanStepStatusCall{ID: id, Status: status})
	if m.updateRunStepStatusFn != nil {
		return m.updateRunStepStatusFn(ctx, id, status, startedAt, completedAt, errorMsg)
	}
	return &model.RunStep{ID: id, Status: status}, nil
}
func (m *humanMockRunRepo) UpdateRunStepContainerInfo(_ context.Context, id uuid.UUID, _ *string, _ *string) (*model.RunStep, error) {
	return &model.RunStep{ID: id}, nil
}
func (m *humanMockRunRepo) CreateRetryRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}
func (m *humanMockRunRepo) ListRetryStepsByParent(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}

type humanMockEventPublisher struct {
	publishFn func(ctx context.Context, event model.Event) error
	events    []model.Event
}

func (m *humanMockEventPublisher) Publish(_ context.Context, event model.Event) error {
	m.events = append(m.events, event)
	if m.publishFn != nil {
		return m.publishFn(context.Background(), event)
	}
	return nil
}

type humanMockStoryRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Story, error)
}

func (m *humanMockStoryRepo) Create(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *humanMockStoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, apperrors.NewNotFound("story", id)
}
func (m *humanMockStoryRepo) GetByKey(_ context.Context, _ uuid.UUID, _ string) (*model.Story, error) {
	return nil, nil
}
func (m *humanMockStoryRepo) ListByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *humanMockStoryRepo) ListByStatus(_ context.Context, _ uuid.UUID, _ []string, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *humanMockStoryRepo) ListByEpic(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *humanMockStoryRepo) CountByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *humanMockStoryRepo) CountByStatus(_ context.Context, _ uuid.UUID, _ []string) (int64, error) {
	return 0, nil
}
func (m *humanMockStoryRepo) Update(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *humanMockStoryRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

// --- Helpers ---

func buildHumanRunCtx(config map[string]string, metadata map[string]any) *model.RunContext {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()
	storyID := uuid.New()

	return &model.RunContext{
		Run: &model.Run{
			ID:        runID,
			ProjectID: projectID,
			StoryID:   storyID,
			Status:    model.RunStatusRunning,
		},
		RunStep: &model.RunStep{
			ID:       stepID,
			RunID:    runID,
			StepName: "review-plan",
			Action:   "human",
			Status:   model.StepStatusRunning,
			Config:   config,
		},
		ProjectID: projectID,
		StoryID:   storyID,
		Metadata:  metadata,
	}
}

func humanStory(storyID uuid.UUID) *model.Story {
	return &model.Story{
		ID:    storyID,
		Key:   "S-05",
		Title: "Test Story",
	}
}

// --- Tests ---

func TestHumanAction_Name(t *testing.T) {
	a := action.NewHumanAction(nil, nil, nil, nil, testLogger())
	if a.Name() != "human" {
		t.Fatalf("expected Name() = %q, got %q", "human", a.Name())
	}
}

func TestHumanAction_Execute_HappyPath(t *testing.T) {
	hitlRepo := &humanMockHITLRepo{}
	runRepo := &humanMockRunRepo{}
	eventPub := &humanMockEventPublisher{}
	storyRepo := &humanMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return humanStory(id), nil
		},
	}

	a := action.NewHumanAction(hitlRepo, runRepo, storyRepo, eventPub, testLogger())

	cfg := map[string]string{
		"message":      "Please review the generated plan for {story_key}",
		"instructions": "Check that all acceptance criteria are addressed",
	}
	runCtx := buildHumanRunCtx(cfg, map[string]any{"branch_name": "feat/S-05-plan"})

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Verify HITL request created with GateType "human" and status "pending"
	if len(hitlRepo.created) != 1 {
		t.Fatalf("expected 1 HITL request created, got %d", len(hitlRepo.created))
	}
	req := hitlRepo.created[0]
	if req.GateType != "human" {
		t.Fatalf("expected gate_type %q, got %q", "human", req.GateType)
	}
	if req.Status != model.HITLStatusPending {
		t.Fatalf("expected status %q, got %q", model.HITLStatusPending, req.Status)
	}
	if req.DiffContent != nil {
		t.Fatal("expected nil DiffContent for human gate")
	}
	if req.Message == nil {
		t.Fatal("expected non-nil Message")
	}
	expectedMsg := "Please review the generated plan for S-05"
	if *req.Message != expectedMsg {
		t.Fatalf("expected message %q, got %q", expectedMsg, *req.Message)
	}

	// Verify step transitioned to waiting_approval
	if len(runRepo.statusCalls) != 1 {
		t.Fatalf("expected 1 status update call, got %d", len(runRepo.statusCalls))
	}
	if runRepo.statusCalls[0].Status != model.StepStatusWaitingApproval {
		t.Fatalf("expected status %q, got %q", model.StepStatusWaitingApproval, runRepo.statusCalls[0].Status)
	}

	// Verify event published
	if len(eventPub.events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(eventPub.events))
	}
	if eventPub.events[0].EventName() != "human.pending" {
		t.Fatalf("expected event name %q, got %q", "human.pending", eventPub.events[0].EventName())
	}

	// Verify event payload contains expected fields
	var payload map[string]string
	if err := json.Unmarshal(eventPub.events[0].Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal event payload: %v", err)
	}
	if payload["story_key"] != "S-05" {
		t.Fatalf("expected story_key %q in payload, got %q", "S-05", payload["story_key"])
	}
	if payload["message"] != expectedMsg {
		t.Fatalf("expected message %q in payload, got %q", expectedMsg, payload["message"])
	}
	if payload["instructions"] != "Check that all acceptance criteria are addressed" {
		t.Fatalf("expected instructions in payload, got %q", payload["instructions"])
	}
}

func TestHumanAction_Execute_MessageTemplateRendering(t *testing.T) {
	hitlRepo := &humanMockHITLRepo{}
	runRepo := &humanMockRunRepo{}
	eventPub := &humanMockEventPublisher{}
	storyRepo := &humanMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return humanStory(id), nil
		},
	}

	a := action.NewHumanAction(hitlRepo, runRepo, storyRepo, eventPub, testLogger())

	cfg := map[string]string{
		"message": "Approve work on {story_key} — branch: {branch_name}",
	}
	runCtx := buildHumanRunCtx(cfg, map[string]any{"branch_name": "feat/S-05-plan"})

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(hitlRepo.created) != 1 {
		t.Fatalf("expected 1 HITL request, got %d", len(hitlRepo.created))
	}
	req := hitlRepo.created[0]
	if req.Message == nil {
		t.Fatal("expected non-nil Message")
	}
	expected := "Approve work on S-05 — branch: feat/S-05-plan"
	if *req.Message != expected {
		t.Fatalf("expected message %q, got %q", expected, *req.Message)
	}
}

func TestHumanAction_Execute_DefaultMessage(t *testing.T) {
	hitlRepo := &humanMockHITLRepo{}
	runRepo := &humanMockRunRepo{}
	eventPub := &humanMockEventPublisher{}
	storyRepo := &humanMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return humanStory(id), nil
		},
	}

	a := action.NewHumanAction(hitlRepo, runRepo, storyRepo, eventPub, testLogger())

	// No "message" key in config
	runCtx := buildHumanRunCtx(nil, map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(hitlRepo.created) != 1 {
		t.Fatalf("expected 1 HITL request, got %d", len(hitlRepo.created))
	}
	req := hitlRepo.created[0]
	if req.Message == nil {
		t.Fatal("expected non-nil Message")
	}
	expected := "Human approval required for step review-plan"
	if *req.Message != expected {
		t.Fatalf("expected message %q, got %q", expected, *req.Message)
	}
}

func TestHumanAction_Execute_HITLRepoCreateFails(t *testing.T) {
	hitlRepo := &humanMockHITLRepo{
		createFn: func(_ context.Context, _ *model.HITLRequest) (*model.HITLRequest, error) {
			return nil, fmt.Errorf("db connection lost")
		},
	}
	runRepo := &humanMockRunRepo{}
	eventPub := &humanMockEventPublisher{}
	storyRepo := &humanMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return humanStory(id), nil
		},
	}

	a := action.NewHumanAction(hitlRepo, runRepo, storyRepo, eventPub, testLogger())

	runCtx := buildHumanRunCtx(nil, map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when HITLRepository.Create fails")
	}
	if !strings.Contains(err.Error(), "create HITL request") {
		t.Fatalf("expected error containing %q, got %q", "create HITL request", err.Error())
	}

	// Verify step was NOT transitioned
	if len(runRepo.statusCalls) != 0 {
		t.Fatalf("expected no status updates when create fails, got %d", len(runRepo.statusCalls))
	}

	// Verify no events published
	if len(eventPub.events) != 0 {
		t.Fatalf("expected no events when create fails, got %d", len(eventPub.events))
	}
}

func TestHumanAction_Execute_UpdateStepStatusFails(t *testing.T) {
	hitlRepo := &humanMockHITLRepo{}
	runRepo := &humanMockRunRepo{
		updateRunStepStatusFn: func(_ context.Context, _ uuid.UUID, _ model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
			return nil, fmt.Errorf("db write failed")
		},
	}
	eventPub := &humanMockEventPublisher{}
	storyRepo := &humanMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return humanStory(id), nil
		},
	}

	a := action.NewHumanAction(hitlRepo, runRepo, storyRepo, eventPub, testLogger())

	runCtx := buildHumanRunCtx(nil, map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when UpdateRunStepStatus fails")
	}
	if !strings.Contains(err.Error(), "update step to waiting_approval") {
		t.Fatalf("expected error containing %q, got %q", "update step to waiting_approval", err.Error())
	}

	// HITL request was already created
	if len(hitlRepo.created) != 1 {
		t.Fatalf("expected 1 HITL request created before status update, got %d", len(hitlRepo.created))
	}

	// No event should be published
	if len(eventPub.events) != 0 {
		t.Fatalf("expected no events when step status update fails, got %d", len(eventPub.events))
	}
}

func TestHumanAction_Execute_EventPublisherFailure(t *testing.T) {
	hitlRepo := &humanMockHITLRepo{}
	runRepo := &humanMockRunRepo{}
	eventPub := &humanMockEventPublisher{
		publishFn: func(_ context.Context, _ model.Event) error {
			return fmt.Errorf("event bus down")
		},
	}
	storyRepo := &humanMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return humanStory(id), nil
		},
	}

	a := action.NewHumanAction(hitlRepo, runRepo, storyRepo, eventPub, testLogger())

	runCtx := buildHumanRunCtx(nil, map[string]any{})

	// Event failure should be non-fatal
	err := a.Execute(context.Background(), runCtx)
	if err != nil {
		t.Fatalf("expected nil error (event failure non-fatal), got %v", err)
	}

	// HITL request and status update should still succeed
	if len(hitlRepo.created) != 1 {
		t.Fatalf("expected 1 HITL request created, got %d", len(hitlRepo.created))
	}
	if len(runRepo.statusCalls) != 1 {
		t.Fatalf("expected 1 status update, got %d", len(runRepo.statusCalls))
	}
}

func TestHumanAction_Execute_StoryFetchFails(t *testing.T) {
	hitlRepo := &humanMockHITLRepo{}
	runRepo := &humanMockRunRepo{}
	eventPub := &humanMockEventPublisher{}
	storyRepo := &humanMockStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return nil, apperrors.NewNotFound("story", id)
		},
	}

	a := action.NewHumanAction(hitlRepo, runRepo, storyRepo, eventPub, testLogger())

	runCtx := buildHumanRunCtx(nil, map[string]any{})

	err := a.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatal("expected error when story fetch fails")
	}
	if !strings.Contains(err.Error(), "fetch story") {
		t.Fatalf("expected error containing %q, got %q", "fetch story", err.Error())
	}

	// Nothing else should be called
	if len(hitlRepo.created) != 0 {
		t.Fatalf("expected no HITL requests when story fetch fails, got %d", len(hitlRepo.created))
	}
	if len(runRepo.statusCalls) != 0 {
		t.Fatalf("expected no status updates when story fetch fails, got %d", len(runRepo.statusCalls))
	}
}
