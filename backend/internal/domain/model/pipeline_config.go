package model

import (
	"time"

	"github.com/google/uuid"
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
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	ActionType  string            `yaml:"action_type"`
	Model       string            `yaml:"model,omitempty"`
	AutoApprove bool              `yaml:"auto_approve"`
	RetryPolicy RetryPolicy       `yaml:"retry_policy"`
	Config      map[string]string `yaml:"config,omitempty"`
}

// RetryPolicy defines retry behavior for a pipeline step.
type RetryPolicy struct {
	MaxRetries int    `yaml:"max_retries"`
	RetryType  string `yaml:"retry_type"` // none, on-failure, always
}

// PipelineGroup represents a named group of steps in the pipeline YAML.
type PipelineGroup struct {
	ID    string         `yaml:"id"`
	Name  string         `yaml:"name"`
	Steps []PipelineStep `yaml:"steps"`
}

// PipelineConfigYAML represents the parsed YAML structure.
// Supports the groups-based format. The legacy flat steps format is handled
// via backward-compatible parsing in story R-1-3.
type PipelineConfigYAML struct {
	Groups []PipelineGroup `yaml:"groups"`
	// Steps is kept for backward-compatible parsing of legacy YAML that uses
	// a flat steps array. New configs always use groups.
	Steps []PipelineStep `yaml:"steps"`
}

// FlatSteps returns all steps across all groups in order.
// Falls back to the legacy Steps field if Groups is empty.
func (c *PipelineConfigYAML) FlatSteps() []PipelineStep {
	if len(c.Groups) > 0 {
		var steps []PipelineStep
		for _, g := range c.Groups {
			steps = append(steps, g.Steps...)
		}
		return steps
	}
	return c.Steps
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
