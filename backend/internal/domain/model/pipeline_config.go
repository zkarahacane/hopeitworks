package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// PipelineConfig represents a pipeline configuration for a project.
type PipelineConfig struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	ConfigYAML string
	Version    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// PipelineStep represents a single step in the pipeline YAML.
type PipelineStep struct {
	ID          string            `yaml:"id"          json:"id"`
	Name        string            `yaml:"name"        json:"name"`
	ActionType  string            `yaml:"action_type" json:"action_type"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	AgentID     string            `yaml:"agent_id,omitempty"    json:"agent_id,omitempty"`
	Model       string            `yaml:"model,omitempty"       json:"model,omitempty"`
	AutoApprove bool              `yaml:"auto_approve"          json:"auto_approve"`
	RetryPolicy RetryPolicy       `yaml:"retry_policy"          json:"retry_policy"`
	Config      map[string]string `yaml:"config,omitempty"      json:"config,omitempty"`
	// Guards are step-level safety probes (INC 4a). Step guards extend the stage
	// guards of the group this step belongs to. Optional.
	Guards []Guard `yaml:"guards,omitempty" json:"guards,omitempty"`
}

// Guard probe kinds (INC 4a, board-side — no runtime change). They consume
// signals the runtime already emits today.
const (
	// GuardLogSilence breaches when no log.emitted event has been seen for a
	// step within Threshold seconds (a heartbeat-via-logs liveness check).
	GuardLogSilence = "log_silence"
	// GuardWallclock breaches when a step has been running longer than Max seconds.
	GuardWallclock = "wallclock"
	// GuardCostBatch breaches when the cumulative cost of the run exceeds Max USD.
	GuardCostBatch = "cost_batch"
)

// Guard on_fail action values. halt-gate is the default: pause the run at its
// durable stage and raise a probe_halt HITL the human can resolve.
const (
	GuardOnFailHaltGate = "halt-gate"
	GuardOnFailFail     = "fail"
	GuardOnFailRetry    = "retry"
)

// Guard is a single safety probe attached to a stage (group) or step. A guard
// observes a running step and, on breach, applies OnFail. INC 4a covers the
// board-side probes (log_silence, wallclock, cost_batch) that need no runtime
// change. The semantic probes (blast-radius, loop) are INC 4b.
type Guard struct {
	// Kind is the probe kind: log_silence | wallclock | cost_batch.
	Kind string `yaml:"kind" json:"kind"`
	// Threshold is the breach threshold in seconds for time-based probes
	// (log_silence). Either Threshold or Max applies depending on the kind.
	Threshold int `yaml:"threshold,omitempty" json:"threshold,omitempty"`
	// Max is the breach ceiling: seconds for wallclock, USD for cost_batch.
	Max float64 `yaml:"max,omitempty" json:"max,omitempty"`
	// OnFail is the action to take on breach: halt-gate (default) | fail | retry.
	OnFail string `yaml:"on_fail,omitempty" json:"on_fail,omitempty"`
}

// normalizeGuards defaults an empty OnFail to halt-gate (the conservative
// "park with a reason" default from the safety model).
func normalizeGuards(guards []Guard) {
	for i := range guards {
		if guards[i].OnFail == "" {
			guards[i].OnFail = GuardOnFailHaltGate
		}
	}
}

// RetryPolicy defines retry behavior for a pipeline step.
type RetryPolicy struct {
	MaxRetries int    `yaml:"max_retries" json:"max_retries"`
	RetryType  string `yaml:"retry_type"  json:"retry_type"` // none, on-failure, always
}

// Stage transition policy values. A group's Transition controls how a card
// leaves the stage. INC 1 carries the field only (default "auto"); enforcement
// of manual/gate is a later increment.
const (
	TransitionAuto   = "auto"
	TransitionManual = "manual"
	TransitionGate   = "gate"
)

// PipelineGroup represents a named group of steps in the pipeline YAML.
// A group is the durable unit the board treats as a "stage".
type PipelineGroup struct {
	ID    string         `yaml:"id"    json:"id"`
	Name  string         `yaml:"name"  json:"name"`
	Steps []PipelineStep `yaml:"steps" json:"steps"`
	// Transition is the stage's exit policy: auto | manual | gate. Defaults to
	// "auto" when empty. Carried end-to-end but NOT enforced in INC 1.
	Transition string `yaml:"transition,omitempty" json:"transition,omitempty"`
	// Guards are stage-level safety probes (INC 4a). Every step in the group is
	// observed against these guards while it runs. Optional.
	Guards []Guard `yaml:"guards,omitempty" json:"guards,omitempty"`
}

// StepWithStage is a pipeline step paired with the identity of the group (stage)
// it originates from. Produced by FlatStepsWithStage so step creation can stamp
// stage_id/stage_name on each run_step instead of discarding the group identity.
type StepWithStage struct {
	Step      PipelineStep
	GroupID   string
	GroupName string
}

// PipelineConfigYAML represents the parsed YAML structure.
// Always uses groups. Legacy flat-steps YAML is auto-wrapped into a single
// "Default" group by ParsePipelineConfigYAML.
type PipelineConfigYAML struct {
	Groups []PipelineGroup `yaml:"groups" json:"groups"`
}

// pipelineConfigRawYAML is an intermediate struct for unmarshalling that
// handles both the new groups format and the legacy flat steps format.
type pipelineConfigRawYAML struct {
	Groups []PipelineGroup `yaml:"groups"`
	Steps  []PipelineStep  `yaml:"steps"` // legacy flat format
}

// ParsePipelineConfigYAML parses pipeline config YAML with backward
// compatibility. If the YAML has a top-level "steps:" array (old format),
// the steps are automatically wrapped in a single PipelineGroup named "Default".
func ParsePipelineConfigYAML(data []byte) (*PipelineConfigYAML, error) {
	var raw pipelineConfigRawYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	cfg := &PipelineConfigYAML{}

	if len(raw.Groups) > 0 {
		cfg.Groups = raw.Groups
	} else if len(raw.Steps) > 0 {
		// Legacy: wrap flat steps in a single default group
		cfg.Groups = []PipelineGroup{
			{ID: "default", Name: "Default", Steps: raw.Steps},
		}
	}

	// Normalize the transition policy: an unset transition defaults to "auto"
	// (the current, always-advancing behaviour). Carried only in INC 1.
	// Normalize guard on_fail (defaults to halt-gate) at both stage and step level.
	for i := range cfg.Groups {
		if cfg.Groups[i].Transition == "" {
			cfg.Groups[i].Transition = TransitionAuto
		}
		normalizeGuards(cfg.Groups[i].Guards)
		for j := range cfg.Groups[i].Steps {
			normalizeGuards(cfg.Groups[i].Steps[j].Guards)
		}
	}

	return cfg, nil
}

// TransitionForStage returns the exit transition policy (auto|manual|gate) of the
// group whose ID matches stageID. It defaults to "auto" when the stage is unknown
// or its transition is unset, matching ParsePipelineConfigYAML's normalization. The
// executor uses this to enforce manual/gate policies at stage boundaries.
func (c *PipelineConfigYAML) TransitionForStage(stageID string) string {
	for _, g := range c.Groups {
		if g.ID == stageID {
			if g.Transition == "" {
				return TransitionAuto
			}
			return g.Transition
		}
	}
	return TransitionAuto
}

// GuardsForStep returns the effective guards observing the given step ID: the
// step's own guards followed by the guards of the stage (group) it belongs to.
// Step guards take precedence implicitly by appearing first. Returns nil when no
// guards apply. Used by the watchdog to evaluate a running step.
func (c *PipelineConfigYAML) GuardsForStep(stepID string) []Guard {
	for _, g := range c.Groups {
		for _, s := range g.Steps {
			if s.ID == stepID {
				if len(s.Guards) == 0 && len(g.Guards) == 0 {
					return nil
				}
				guards := make([]Guard, 0, len(s.Guards)+len(g.Guards))
				guards = append(guards, s.Guards...)
				guards = append(guards, g.Guards...)
				return guards
			}
		}
	}
	return nil
}

// FlatSteps returns all steps across all groups in order.
func (c *PipelineConfigYAML) FlatSteps() []PipelineStep {
	var steps []PipelineStep
	for _, g := range c.Groups {
		steps = append(steps, g.Steps...)
	}
	return steps
}

// FlatStepsWithStage returns all steps across all groups in order, each paired
// with the identity (ID, Name) of the group it came from. Unlike FlatSteps it
// preserves stage identity so the executor can stamp stage_id/stage_name on each
// run_step. Step ordering matches FlatSteps exactly.
func (c *PipelineConfigYAML) FlatStepsWithStage() []StepWithStage {
	var steps []StepWithStage
	for _, g := range c.Groups {
		for _, s := range g.Steps {
			steps = append(steps, StepWithStage{
				Step:      s,
				GroupID:   g.ID,
				GroupName: g.Name,
			})
		}
	}
	return steps
}

// ValidActionTypes defines the set of valid pipeline step action_type values.
// These match the PipelineStepActionType enum in the OpenAPI spec.
var ValidActionTypes = map[string]bool{
	"agent_run":    true,
	"git_branch":   true,
	"git_pr":       true,
	"notification": true,
	"human":        true,
	"ci_poll":      true,
	"hitl_gate":    true,
	// Legacy action types kept for backward compatibility with stored YAML.
	// TODO(R-1-3): Remove once migration auto-wraps legacy configs.
	"implement": true,
	"review":    true,
	"merge":     true,
	"test":      true,
	"custom":    true,
}
