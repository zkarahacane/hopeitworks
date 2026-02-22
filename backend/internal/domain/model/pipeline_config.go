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
// The ActionType field uses the same values as the API enum (implement, review, merge, test, custom).
type PipelineStep struct {
	ID          string      `yaml:"id"`
	Name        string      `yaml:"name"`
	ActionType  string      `yaml:"action_type"`
	Model       string      `yaml:"model,omitempty"`
	AutoApprove bool        `yaml:"auto_approve"`
	RetryPolicy RetryPolicy `yaml:"retry_policy"`
}

// RetryPolicy defines retry behavior for a pipeline step.
type RetryPolicy struct {
	MaxRetries int    `yaml:"max_retries"`
	RetryType  string `yaml:"retry_type"` // none, on-failure, always
}

// PipelineConfigYAML represents the parsed YAML structure.
type PipelineConfigYAML struct {
	Steps []PipelineStep `yaml:"steps"`
}

// ValidActionTypes defines the set of valid pipeline step action_type values.
// These match the PipelineStepActionType enum in the OpenAPI spec.
var ValidActionTypes = map[string]bool{
	"implement": true,
	"review":    true,
	"merge":     true,
	"test":      true,
	"custom":    true,
	"ci_poll":   true,
	"hitl_gate": true,
}
