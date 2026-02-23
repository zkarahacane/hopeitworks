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
}

// RetryPolicy defines retry behavior for a pipeline step.
type RetryPolicy struct {
	MaxRetries int    `yaml:"max_retries" json:"max_retries"`
	RetryType  string `yaml:"retry_type"  json:"retry_type"` // none, on-failure, always
}

// PipelineGroup represents a named group of steps in the pipeline YAML.
type PipelineGroup struct {
	ID    string         `yaml:"id"    json:"id"`
	Name  string         `yaml:"name"  json:"name"`
	Steps []PipelineStep `yaml:"steps" json:"steps"`
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

	return cfg, nil
}

// FlatSteps returns all steps across all groups in order.
func (c *PipelineConfigYAML) FlatSteps() []PipelineStep {
	var steps []PipelineStep
	for _, g := range c.Groups {
		steps = append(steps, g.Steps...)
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
