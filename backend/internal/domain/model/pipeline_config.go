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
	Name        string                 `yaml:"name"`
	Action      string                 `yaml:"action"`
	Model       *string                `yaml:"model,omitempty"`
	AutoApprove *bool                  `yaml:"auto_approve,omitempty"`
	RetryPolicy *RetryPolicy           `yaml:"retry_policy,omitempty"`
	Params      map[string]interface{} `yaml:"params,omitempty"`
}

// RetryPolicy defines retry behavior for a pipeline step.
type RetryPolicy struct {
	MaxRetries int    `yaml:"max_retries"`
	Strategy   string `yaml:"strategy"` // fixed, exponential
}

// PipelineConfigYAML represents the parsed YAML structure.
type PipelineConfigYAML struct {
	Steps []PipelineStep `yaml:"steps"`
}

// ValidActions defines the set of valid pipeline step action names.
// TODO(S-3-3): Replace with ActionRegistry once implemented.
var ValidActions = map[string]bool{
	"agent_run":     true,
	"hitl_gate":     true,
	"git_create_pr": true,
	"git_merge":     true,
}
