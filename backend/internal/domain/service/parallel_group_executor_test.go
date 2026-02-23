package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

func TestParallelGroupExecutor_Execute_HappyPath(t *testing.T) {
	projectID := uuid.New()
	epicID := uuid.New()
	epicRunID := uuid.New()

	var statusMu sync.Mutex
	var statusUpdates []model.EpicRunStatus

	epicRunRepo := &mockEpicRunRepo{
		updateEpicRunStatusFn: func(_ context.Context, _ uuid.UUID, status model.EpicRunStatus, _ *time.Time) (*model.EpicRun, error) {
			statusMu.Lock()
			statusUpdates = append(statusUpdates, status)
			statusMu.Unlock()
			return &model.EpicRun{ID: epicRunID, Status: status}, nil
		},
	}

	dag := model.DAGResult{
		Groups: [][]model.Story{
			{{ID: uuid.New(), Key: "S-01", ProjectID: projectID}},
			{{ID: uuid.New(), Key: "S-02", ProjectID: projectID}, {ID: uuid.New(), Key: "S-03", ProjectID: projectID}},
		},
	}

	epicRun := &model.EpicRun{
		ID:        epicRunID,
		ProjectID: projectID,
		EpicID:    epicID,
		Status:    model.EpicRunStatusPending,
	}

	logger := testLogger()
	eventPub := newMockEventPublisher()

	// Create RunService and PipelineExecutor with mocks that make LaunchRun and ExecuteRun succeed
	runRepo := &mockRunRepo{
		createRunFn: func(_ context.Context, run *model.Run) (*model.Run, error) {
			run.ID = uuid.New()
			return run, nil
		},
		getActiveRunByStoryFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
			return nil, nil
		},
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, Status: model.RunStatusPending}, nil
		},
		listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return []*model.RunStep{}, nil // No steps means ExecuteRun completes immediately
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _ *time.Time, _ *time.Time, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, Status: status}, nil
		},
	}

	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, ProjectID: projectID, Key: "S-mock", Status: "backlog"}, nil
		},
	}
	projectRepo := newMockProjectRepoForService()
	projectRepo.projects[projectID] = &model.Project{ID: projectID, Name: "Test"}
	parallelAgentID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	mockPCR := &mockPipelineConfigRepoForRun{
		getByProjectIDFn: func(_ context.Context, _ uuid.UUID) (*model.PipelineConfig, error) {
			return &model.PipelineConfig{
				ProjectID:  projectID,
				ConfigYAML: `steps: [{name: "implement", action_type: "implement", agent_id: "00000000-0000-0000-0000-000000000010", auto_approve: false}]`,
			}, nil
		},
	}

	agentRepo := newMockAgentRepo()
	agentRepo.agents[parallelAgentID] = &model.Agent{
		ID:    parallelAgentID,
		Model: "claude-sonnet-4-6",
		Image: "hopeitworks/agent:latest",
	}

	runSvc := NewRunService(runRepo, projectRepo, storyRepo, mockPCR, &mockJobQueue{})
	runSvc.SetAgentRepo(agentRepo)
	actionReg := newMockActionRegistry()
	pipeExec := NewPipelineExecutor(runRepo, storyRepo, actionReg, eventPub, logger)
	executor := NewParallelGroupExecutor(epicRunRepo, runSvc, pipeExec, eventPub, logger)

	err := executor.Execute(context.Background(), epicRun, dag)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify status transitions
	statusMu.Lock()
	defer statusMu.Unlock()

	if len(statusUpdates) < 2 {
		t.Fatalf("expected at least 2 status updates, got %d", len(statusUpdates))
	}
	if statusUpdates[0] != model.EpicRunStatusRunning {
		t.Errorf("expected first status Running, got %s", statusUpdates[0])
	}
	if statusUpdates[len(statusUpdates)-1] != model.EpicRunStatusCompleted {
		t.Errorf("expected last status Completed, got %s", statusUpdates[len(statusUpdates)-1])
	}
}

func TestParallelGroupExecutor_Execute_FailFast(t *testing.T) {
	projectID := uuid.New()
	epicID := uuid.New()
	epicRunID := uuid.New()

	var statusMu sync.Mutex
	var statusUpdates []model.EpicRunStatus

	epicRunRepo := &mockEpicRunRepo{
		updateEpicRunStatusFn: func(_ context.Context, _ uuid.UUID, status model.EpicRunStatus, _ *time.Time) (*model.EpicRun, error) {
			statusMu.Lock()
			statusUpdates = append(statusUpdates, status)
			statusMu.Unlock()
			return &model.EpicRun{ID: epicRunID, Status: status}, nil
		},
	}

	story1 := model.Story{ID: uuid.New(), Key: "S-01", ProjectID: projectID}
	story2 := model.Story{ID: uuid.New(), Key: "S-02", ProjectID: projectID}

	dag := model.DAGResult{
		Groups: [][]model.Story{
			{story1},
			{story2}, // Should never be reached
		},
	}

	epicRun := &model.EpicRun{
		ID:        epicRunID,
		ProjectID: projectID,
		EpicID:    epicID,
		Status:    model.EpicRunStatusPending,
	}

	logger := testLogger()
	eventPub := newMockEventPublisher()

	// Make ExecuteRun fail by having GetRun return an error
	runRepo := &mockRunRepo{
		createRunFn: func(_ context.Context, run *model.Run) (*model.Run, error) {
			run.ID = uuid.New()
			return run, nil
		},
		getActiveRunByStoryFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
			return nil, nil
		},
		getRunFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
			return nil, fmt.Errorf("simulated execution failure")
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _ *time.Time, _ *time.Time, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, Status: status}, nil
		},
	}

	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, ProjectID: projectID, Key: "S-mock", Status: "backlog"}, nil
		},
	}
	projectRepo := newMockProjectRepoForService()
	projectRepo.projects[projectID] = &model.Project{ID: projectID, Name: "Test"}
	failFastAgentID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	mockPCR := &mockPipelineConfigRepoForRun{
		getByProjectIDFn: func(_ context.Context, _ uuid.UUID) (*model.PipelineConfig, error) {
			return &model.PipelineConfig{
				ProjectID:  projectID,
				ConfigYAML: `steps: [{name: "implement", action_type: "implement", agent_id: "00000000-0000-0000-0000-000000000020", auto_approve: false}]`,
			}, nil
		},
	}

	failFastAgentRepo := newMockAgentRepo()
	failFastAgentRepo.agents[failFastAgentID] = &model.Agent{
		ID:    failFastAgentID,
		Model: "claude-sonnet-4-6",
		Image: "hopeitworks/agent:latest",
	}

	runSvc := NewRunService(runRepo, projectRepo, storyRepo, mockPCR, &mockJobQueue{})
	runSvc.SetAgentRepo(failFastAgentRepo)
	actionReg := newMockActionRegistry()
	pipeExec := NewPipelineExecutor(runRepo, storyRepo, actionReg, eventPub, logger)
	executor := NewParallelGroupExecutor(epicRunRepo, runSvc, pipeExec, eventPub, logger)

	err := executor.Execute(context.Background(), epicRun, dag)
	if err == nil {
		t.Fatal("expected error on fail-fast")
	}

	// Verify epic run was marked as failed
	statusMu.Lock()
	defer statusMu.Unlock()

	var foundFailed bool
	for _, status := range statusUpdates {
		if status == model.EpicRunStatusFailed {
			foundFailed = true
			break
		}
	}
	if !foundFailed {
		t.Errorf("expected epic run status to be updated to Failed, got: %v", statusUpdates)
	}
}

func TestEpicRunStatusTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    model.EpicRunStatus
		to      model.EpicRunStatus
		wantErr bool
	}{
		{"pending -> running", model.EpicRunStatusPending, model.EpicRunStatusRunning, false},
		{"running -> completed", model.EpicRunStatusRunning, model.EpicRunStatusCompleted, false},
		{"running -> failed", model.EpicRunStatusRunning, model.EpicRunStatusFailed, false},
		{"running -> paused", model.EpicRunStatusRunning, model.EpicRunStatusPaused, false},
		{"pending -> completed (invalid)", model.EpicRunStatusPending, model.EpicRunStatusCompleted, true},
		{"completed -> running (invalid)", model.EpicRunStatusCompleted, model.EpicRunStatusRunning, true},
		{"failed -> running (invalid)", model.EpicRunStatusFailed, model.EpicRunStatusRunning, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := model.ValidateEpicRunTransition(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEpicRunTransition(%s, %s) error = %v, wantErr %v", tt.from, tt.to, err, tt.wantErr)
			}
		})
	}
}
