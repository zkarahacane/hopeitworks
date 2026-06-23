package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// HaltRaiser is the slice of HITLService the watchdog needs to park a breached
// step at its durable stage. Defined as an interface so the watchdog depends on
// behaviour, not the concrete service.
type HaltRaiser interface {
	RaiseProbeHalt(ctx context.Context, stepID uuid.UUID, reason model.HaltReason) (*model.HITLRequest, error)
}

// CostSummer is the slice of CostRepository the watchdog needs to evaluate the
// cost_batch probe. Narrowed to the single method so the watchdog is decoupled
// from the full cost repository surface.
type CostSummer interface {
	SumCostByRun(ctx context.Context, runID uuid.UUID) (float64, error)
}

// Watchdog is the out-of-band guard evaluator (INC 4a). On a fixed interval it
// scans every running step, looks up the guards configured on its stage/step in
// the project pipeline, and evaluates the board-side probes (log_silence,
// wallclock, cost_batch) against signals the runtime already emits. On breach it
// applies the guard's on_fail action — by default halt-gate, which parks the run
// with a reason for a human to resolve.
//
// It deliberately lives out of band (a ticker, like TimeoutEnforcer) rather than
// inside pipeline_executor.go, so it never bloats the hot execution path.
type Watchdog struct {
	watchdogRepo   port.WatchdogRepository
	pipelineRepo   port.PipelineConfigRepository
	costRepo       CostSummer
	runRepo        port.RunRepository
	haltRaiser     HaltRaiser
	logger         *slog.Logger
	checkInterval  time.Duration
	configCacheTTL time.Duration

	// configCache memoizes parsed pipeline configs per project across a single
	// tick batch to avoid re-parsing YAML for every running step.
	configCache map[uuid.UUID]*cachedConfig
}

type cachedConfig struct {
	cfg     *model.PipelineConfigYAML
	fetched time.Time
}

// NewWatchdog creates a Watchdog. checkInterval is how often the scan runs
// (default 30s if zero/negative).
func NewWatchdog(
	watchdogRepo port.WatchdogRepository,
	pipelineRepo port.PipelineConfigRepository,
	costRepo CostSummer,
	runRepo port.RunRepository,
	haltRaiser HaltRaiser,
	logger *slog.Logger,
	checkInterval time.Duration,
) *Watchdog {
	if checkInterval <= 0 {
		checkInterval = 30 * time.Second
	}
	return &Watchdog{
		watchdogRepo:   watchdogRepo,
		pipelineRepo:   pipelineRepo,
		costRepo:       costRepo,
		runRepo:        runRepo,
		haltRaiser:     haltRaiser,
		logger:         logger,
		checkInterval:  checkInterval,
		configCacheTTL: 30 * time.Second,
		configCache:    make(map[uuid.UUID]*cachedConfig),
	}
}

// Start runs the watchdog scan loop until the context is cancelled.
func (w *Watchdog) Start(ctx context.Context) error {
	w.logger.Info("guard watchdog started", "check_interval", w.checkInterval)

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("guard watchdog stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := w.Check(ctx); err != nil {
				w.logger.Error("guard watchdog check failed", "error", err)
			}
		}
	}
}

// Check performs a single scan: evaluate every running step against its guards.
func (w *Watchdog) Check(ctx context.Context) error {
	steps, err := w.watchdogRepo.ListRunningSteps(ctx)
	if err != nil {
		return err
	}
	// Reset the per-tick config cache.
	w.configCache = make(map[uuid.UUID]*cachedConfig)

	now := time.Now()
	for _, step := range steps {
		guards := w.guardsForStep(ctx, step)
		if len(guards) == 0 {
			continue
		}
		for _, g := range guards {
			reason, breached := w.evaluate(ctx, step, g, now)
			if !breached {
				continue
			}
			w.applyOnFail(ctx, step, g, reason)
			// One breach per step per tick is enough — it is now halted/failed.
			break
		}
	}
	return nil
}

// guardsForStep resolves the effective guards for a running step from its
// project's pipeline config. We match guards by stage rather than step ID,
// because a run step's name maps to the pipeline step; the config's
// GuardsForStep keys on the pipeline step ID. We therefore fall back to matching
// the step by name within its stage.
func (w *Watchdog) guardsForStep(ctx context.Context, step *model.RunningStep) []model.Guard {
	cfg := w.pipelineConfig(ctx, step.ProjectID)
	if cfg == nil {
		return nil
	}
	for _, grp := range cfg.Groups {
		if grp.ID != step.StageID && grp.Name != step.StageName {
			continue
		}
		var guards []model.Guard
		for _, s := range grp.Steps {
			if s.ID == step.StepName || s.Name == step.StepName {
				guards = append(guards, s.Guards...)
			}
		}
		guards = append(guards, grp.Guards...)
		return guards
	}
	return nil
}

// pipelineConfig returns the parsed pipeline config for a project, memoized for
// the duration of a tick batch.
func (w *Watchdog) pipelineConfig(ctx context.Context, projectID uuid.UUID) *model.PipelineConfigYAML {
	if c, ok := w.configCache[projectID]; ok {
		return c.cfg
	}
	raw, err := w.pipelineRepo.GetByProjectID(ctx, projectID)
	if err != nil || raw == nil {
		w.configCache[projectID] = &cachedConfig{cfg: nil, fetched: time.Now()}
		return nil
	}
	parsed, err := model.ParsePipelineConfigYAML([]byte(raw.ConfigYAML))
	if err != nil {
		w.logger.Warn("watchdog could not parse pipeline config", "project_id", projectID, "error", err)
		w.configCache[projectID] = &cachedConfig{cfg: nil, fetched: time.Now()}
		return nil
	}
	w.configCache[projectID] = &cachedConfig{cfg: parsed, fetched: time.Now()}
	return parsed
}

// evaluate checks a single guard against a running step and returns the breach
// reason when the guard is breached.
func (w *Watchdog) evaluate(ctx context.Context, step *model.RunningStep, g model.Guard, now time.Time) (model.HaltReason, bool) {
	switch g.Kind {
	case model.GuardLogSilence:
		threshold := float64(g.Threshold)
		if threshold <= 0 {
			return model.HaltReason{}, false
		}
		// Baseline: last log; fall back to start time when no log emitted yet.
		baseline := step.LastLogAt
		if baseline == nil {
			baseline = step.StartedAt
		}
		if baseline == nil {
			return model.HaltReason{}, false
		}
		silence := now.Sub(*baseline).Seconds()
		if silence > threshold {
			return model.HaltReason{
				Probe:     model.GuardLogSilence,
				OnFail:    g.OnFail,
				Observed:  silence,
				Threshold: threshold,
				Unit:      "seconds",
			}, true
		}

	case model.GuardWallclock:
		if g.Max <= 0 || step.StartedAt == nil {
			return model.HaltReason{}, false
		}
		elapsed := now.Sub(*step.StartedAt).Seconds()
		if elapsed > g.Max {
			return model.HaltReason{
				Probe:     model.GuardWallclock,
				OnFail:    g.OnFail,
				Observed:  elapsed,
				Threshold: g.Max,
				Unit:      "seconds",
			}, true
		}

	case model.GuardCostBatch:
		if g.Max <= 0 {
			return model.HaltReason{}, false
		}
		cost, err := w.costRepo.SumCostByRun(ctx, step.RunID)
		if err != nil {
			w.logger.Warn("watchdog could not sum run cost", "run_id", step.RunID, "error", err)
			return model.HaltReason{}, false
		}
		if cost > g.Max {
			return model.HaltReason{
				Probe:     model.GuardCostBatch,
				OnFail:    g.OnFail,
				Observed:  cost,
				Threshold: g.Max,
				Unit:      "usd",
			}, true
		}
	}
	return model.HaltReason{}, false
}

// applyOnFail executes the guard's on_fail action for a breach.
func (w *Watchdog) applyOnFail(ctx context.Context, step *model.RunningStep, g model.Guard, reason model.HaltReason) {
	action := g.OnFail
	if action == "" {
		action = model.GuardOnFailHaltGate
	}
	w.logger.Warn("guard breached",
		"step_id", step.StepID, "run_id", step.RunID, "probe", reason.Probe,
		"observed", reason.Observed, "threshold", reason.Threshold, "on_fail", action)

	switch action {
	case model.GuardOnFailHaltGate:
		if _, err := w.haltRaiser.RaiseProbeHalt(ctx, step.StepID, reason); err != nil {
			w.logger.Error("failed to raise probe halt", "step_id", step.StepID, "error", err)
		}
	case model.GuardOnFailFail, model.GuardOnFailRetry:
		// fail: hard-fail the step and run. retry is treated as fail here in INC 4a
		// (the step-level retry_policy in the action layer owns bounded retries);
		// the run is failed so the epic layer can fail-fast dependents.
		now := time.Now()
		msg := probeHaltMessage(reason)
		if _, err := w.runRepo.UpdateRunStepStatus(ctx, step.StepID, model.StepStatusFailed, nil, &now, &msg); err != nil {
			w.logger.Error("failed to fail breached step", "step_id", step.StepID, "error", err)
		}
		if _, err := w.runRepo.UpdateRunStatus(ctx, step.RunID, model.RunStatusFailed, nil, &now, nil, &msg); err != nil {
			w.logger.Error("failed to fail breached run", "run_id", step.RunID, "error", err)
		}
	default:
		w.logger.Warn("unknown guard on_fail action, defaulting to halt-gate", "on_fail", action)
		if _, err := w.haltRaiser.RaiseProbeHalt(ctx, step.StepID, reason); err != nil {
			w.logger.Error("failed to raise probe halt", "step_id", step.StepID, "error", err)
		}
	}
}
