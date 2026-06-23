package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// --- watchdog test doubles ---------------------------------------------------

type fakeWatchdogRepo struct {
	steps []*model.RunningStep
}

func (f *fakeWatchdogRepo) ListRunningSteps(_ context.Context) ([]*model.RunningStep, error) {
	return f.steps, nil
}

type fakePipelineRepo struct {
	cfgYAML string
}

func (f *fakePipelineRepo) GetByProjectID(_ context.Context, projectID uuid.UUID) (*model.PipelineConfig, error) {
	return &model.PipelineConfig{ProjectID: projectID, ConfigYAML: f.cfgYAML}, nil
}

func (f *fakePipelineRepo) Upsert(_ context.Context, cfg *model.PipelineConfig) (*model.PipelineConfig, error) {
	return cfg, nil
}

type fakeHaltRaiser struct {
	halts []haltCall
}

type haltCall struct {
	stepID uuid.UUID
	reason model.HaltReason
}

func (f *fakeHaltRaiser) RaiseProbeHalt(_ context.Context, stepID uuid.UUID, reason model.HaltReason) (*model.HITLRequest, error) {
	f.halts = append(f.halts, haltCall{stepID: stepID, reason: reason})
	return &model.HITLRequest{ID: uuid.New(), RunStepID: stepID}, nil
}

func newTestWatchdog(repo *fakeWatchdogRepo, pipe *fakePipelineRepo, cost *fakeCostSummer, run *mockRunRepoForHITL, raiser *fakeHaltRaiser) *Watchdog {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewWatchdog(repo, pipe, cost, run, raiser, logger, time.Second)
}

// fakeCostSummer implements the CostSummer slice the watchdog uses for the
// cost_batch probe.
type fakeCostSummer struct {
	costByRun map[uuid.UUID]float64
}

func (m *fakeCostSummer) SumCostByRun(_ context.Context, runID uuid.UUID) (float64, error) {
	return m.costByRun[runID], nil
}

// --- tests -------------------------------------------------------------------

func TestWatchdog_LogSilenceBreach(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()
	old := time.Now().Add(-5 * time.Minute)

	repo := &fakeWatchdogRepo{steps: []*model.RunningStep{{
		StepID:    stepID,
		RunID:     runID,
		StepName:  "dev-agent",
		StageID:   "dev",
		StageName: "Development",
		ProjectID: projectID,
		StartedAt: &old,
		LastLogAt: &old, // 5 minutes since last log
	}}}
	pipe := &fakePipelineRepo{cfgYAML: `groups:
  - id: dev
    name: Development
    steps:
      - id: dev-agent
        name: dev-agent
        action_type: agent_run
        guards:
          - kind: log_silence
            threshold: 120
`}
	raiser := &fakeHaltRaiser{}
	wd := newTestWatchdog(repo, pipe, &fakeCostSummer{}, newMockRunRepoForHITL(), raiser)

	if err := wd.Check(context.Background()); err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(raiser.halts) != 1 {
		t.Fatalf("expected 1 halt raised, got %d", len(raiser.halts))
	}
	if raiser.halts[0].reason.Probe != model.GuardLogSilence {
		t.Errorf("expected log_silence halt, got %q", raiser.halts[0].reason.Probe)
	}
	if raiser.halts[0].reason.Threshold != 120 {
		t.Errorf("expected threshold 120, got %v", raiser.halts[0].reason.Threshold)
	}
}

func TestWatchdog_WallclockBreach(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()
	started := time.Now().Add(-40 * time.Minute)
	recent := time.Now().Add(-10 * time.Second)

	repo := &fakeWatchdogRepo{steps: []*model.RunningStep{{
		StepID:    stepID,
		RunID:     runID,
		StepName:  "dev-agent",
		StageID:   "dev",
		StageName: "Development",
		ProjectID: projectID,
		StartedAt: &started, // running 40 min
		LastLogAt: &recent,  // logs are fresh, so log_silence won't fire
	}}}
	pipe := &fakePipelineRepo{cfgYAML: `groups:
  - id: dev
    name: Development
    guards:
      - kind: wallclock
        max: 1800
    steps:
      - id: dev-agent
        name: dev-agent
        action_type: agent_run
`}
	raiser := &fakeHaltRaiser{}
	wd := newTestWatchdog(repo, pipe, &fakeCostSummer{}, newMockRunRepoForHITL(), raiser)

	if err := wd.Check(context.Background()); err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(raiser.halts) != 1 || raiser.halts[0].reason.Probe != model.GuardWallclock {
		t.Fatalf("expected 1 wallclock halt, got %+v", raiser.halts)
	}
}

func TestWatchdog_CostBatchBreach(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()
	recent := time.Now().Add(-5 * time.Second)

	repo := &fakeWatchdogRepo{steps: []*model.RunningStep{{
		StepID:    stepID,
		RunID:     runID,
		StepName:  "dev-agent",
		StageID:   "dev",
		StageName: "Development",
		ProjectID: projectID,
		StartedAt: &recent,
		LastLogAt: &recent,
	}}}
	pipe := &fakePipelineRepo{cfgYAML: `groups:
  - id: dev
    name: Development
    guards:
      - kind: cost_batch
        max: 5
    steps:
      - id: dev-agent
        name: dev-agent
        action_type: agent_run
`}
	cost := &fakeCostSummer{costByRun: map[uuid.UUID]float64{runID: 7.5}} // over $5
	raiser := &fakeHaltRaiser{}
	wd := newTestWatchdog(repo, pipe, cost, newMockRunRepoForHITL(), raiser)

	if err := wd.Check(context.Background()); err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(raiser.halts) != 1 || raiser.halts[0].reason.Probe != model.GuardCostBatch {
		t.Fatalf("expected 1 cost_batch halt, got %+v", raiser.halts)
	}
	if raiser.halts[0].reason.Observed != 7.5 {
		t.Errorf("expected observed 7.5, got %v", raiser.halts[0].reason.Observed)
	}
}

func TestWatchdog_NoBreachWithinThresholds(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	recent := time.Now().Add(-10 * time.Second)

	repo := &fakeWatchdogRepo{steps: []*model.RunningStep{{
		StepID:    uuid.New(),
		RunID:     runID,
		StepName:  "dev-agent",
		StageID:   "dev",
		StageName: "Development",
		ProjectID: projectID,
		StartedAt: &recent,
		LastLogAt: &recent,
	}}}
	pipe := &fakePipelineRepo{cfgYAML: `groups:
  - id: dev
    name: Development
    guards:
      - kind: log_silence
        threshold: 120
      - kind: wallclock
        max: 1800
      - kind: cost_batch
        max: 5
    steps:
      - id: dev-agent
        name: dev-agent
        action_type: agent_run
`}
	cost := &fakeCostSummer{costByRun: map[uuid.UUID]float64{runID: 1.0}}
	raiser := &fakeHaltRaiser{}
	wd := newTestWatchdog(repo, pipe, cost, newMockRunRepoForHITL(), raiser)

	if err := wd.Check(context.Background()); err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(raiser.halts) != 0 {
		t.Fatalf("expected no halts within thresholds, got %+v", raiser.halts)
	}
}
