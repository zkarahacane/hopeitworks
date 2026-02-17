package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// DefaultPipelineConfigYAML is the default pipeline configuration seeded on project creation.
const DefaultPipelineConfigYAML = `steps:
  - name: agent_run
    action: agent_run
    model: claude-opus-4-6
    retry_policy:
      max_retries: 3
      strategy: exponential
  - name: hitl_gate
    action: hitl_gate
    auto_approve: false
  - name: git_create_pr
    action: git_create_pr
  - name: git_merge
    action: git_merge
`

// PipelineConfigService provides business logic for pipeline config operations.
type PipelineConfigService struct {
	repo port.PipelineConfigRepository
}

// NewPipelineConfigService creates a new PipelineConfigService.
func NewPipelineConfigService(repo port.PipelineConfigRepository) *PipelineConfigService {
	return &PipelineConfigService{repo: repo}
}

// GetByProjectID retrieves the pipeline config for a project.
func (s *PipelineConfigService) GetByProjectID(ctx context.Context, projectID uuid.UUID) (*model.PipelineConfig, error) {
	return s.repo.GetByProjectID(ctx, projectID)
}

// Upsert validates and saves a pipeline config for a project.
func (s *PipelineConfigService) Upsert(ctx context.Context, projectID uuid.UUID, configYAML string) (*model.PipelineConfig, error) {
	if err := validatePipelineConfigYAML(configYAML); err != nil {
		return nil, err
	}

	config := &model.PipelineConfig{
		ProjectID:  projectID,
		ConfigYAML: configYAML,
	}
	return s.repo.Upsert(ctx, config)
}

// SeedDefault creates a default pipeline config for a new project.
func (s *PipelineConfigService) SeedDefault(ctx context.Context, projectID uuid.UUID) (*model.PipelineConfig, error) {
	config := &model.PipelineConfig{
		ProjectID:  projectID,
		ConfigYAML: DefaultPipelineConfigYAML,
	}
	return s.repo.Upsert(ctx, config)
}

// validatePipelineConfigYAML parses and validates the pipeline config YAML.
func validatePipelineConfigYAML(configYAML string) error {
	if configYAML == "" {
		return errors.NewValidation("config_yaml", "is required")
	}

	var parsed model.PipelineConfigYAML
	if err := yaml.Unmarshal([]byte(configYAML), &parsed); err != nil {
		return &errors.DomainError{
			Category: errors.CategoryValidation,
			Code:     "INVALID_PIPELINE_CONFIG",
			Message:  fmt.Sprintf("invalid YAML: %v", err),
		}
	}

	if len(parsed.Steps) == 0 {
		return &errors.DomainError{
			Category: errors.CategoryValidation,
			Code:     "INVALID_PIPELINE_CONFIG",
			Message:  "pipeline config must have at least one step",
		}
	}

	for _, step := range parsed.Steps {
		if step.Name == "" {
			return &errors.DomainError{
				Category: errors.CategoryValidation,
				Code:     "INVALID_PIPELINE_CONFIG",
				Message:  "each step must have a name",
			}
		}
		if step.Action == "" {
			return &errors.DomainError{
				Category: errors.CategoryValidation,
				Code:     "INVALID_PIPELINE_CONFIG",
				Message:  fmt.Sprintf("step '%s' must have an action", step.Name),
			}
		}
		// TODO(S-3-3): Replace with ActionRegistry validation once implemented.
		if !model.ValidActions[step.Action] {
			return &errors.DomainError{
				Category: errors.CategoryValidation,
				Code:     "INVALID_PIPELINE_CONFIG",
				Message:  fmt.Sprintf("invalid action name: %s", step.Action),
			}
		}
	}

	return nil
}
