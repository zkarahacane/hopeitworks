package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// DefaultPipelineConfigYAML is the default pipeline configuration seeded on project creation.
// Uses the groups-based format with action_type enum values matching the OpenAPI spec.
const DefaultPipelineConfigYAML = `groups:
  - id: setup
    name: Setup
    steps:
      - id: 880e8400-e29b-41d4-a716-446655440001
        name: create-branch
        action_type: git_branch
        description: Create feature branch from base
        auto_approve: true
        config:
          base_branch: main
        retry_policy:
          max_retries: 1
          retry_type: on-failure
  - id: development
    name: Development
    steps:
      - id: 880e8400-e29b-41d4-a716-446655440002
        name: dev-agent
        action_type: agent_run
        description: Run development agent
        model: claude-opus-4-6
        auto_approve: false
        retry_policy:
          max_retries: 2
          retry_type: on-failure
  - id: review
    name: Review
    steps:
      - id: 880e8400-e29b-41d4-a716-446655440003
        name: review-agent
        action_type: agent_run
        description: Run code review agent
        model: claude-sonnet-4-6
        auto_approve: false
        retry_policy:
          max_retries: 1
          retry_type: on-failure
  - id: merge
    name: Merge
    steps:
      - id: 880e8400-e29b-41d4-a716-446655440004
        name: create-pr
        action_type: git_pr
        description: Create and merge pull request
        model: claude-sonnet-4-6
        auto_approve: true
        retry_policy:
          max_retries: 1
          retry_type: on-failure
  - id: delivery
    name: Delivery
    steps:
      - id: 880e8400-e29b-41d4-a716-446655440005
        name: poll-ci
        action_type: ci_poll
        description: Wait for CI to pass
        auto_approve: true
        config:
          timeout_minutes: "30"
      - id: 880e8400-e29b-41d4-a716-446655440006
        name: notify
        action_type: notification
        description: Send completion notification
        auto_approve: true
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
// Validates group structure (non-empty names, non-empty steps) and step-level
// fields (name, action_type). Uses ParsePipelineConfigYAML for backward-
// compatible parsing of both groups and legacy flat-steps formats.
func validatePipelineConfigYAML(configYAML string) error {
	if configYAML == "" {
		return errors.NewValidation("config_yaml", "is required")
	}

	parsed, err := model.ParsePipelineConfigYAML([]byte(configYAML))
	if err != nil {
		return &errors.DomainError{
			Category: errors.CategoryValidation,
			Code:     "INVALID_PIPELINE_CONFIG",
			Message:  err.Error(),
		}
	}

	if len(parsed.Groups) == 0 {
		return &errors.DomainError{
			Category: errors.CategoryValidation,
			Code:     "INVALID_PIPELINE_CONFIG",
			Message:  "pipeline config must have at least one group",
		}
	}

	for i, group := range parsed.Groups {
		if group.Name == "" {
			return &errors.DomainError{
				Category: errors.CategoryValidation,
				Code:     "INVALID_PIPELINE_CONFIG",
				Message:  fmt.Sprintf("groups[%d].name: group name is required", i),
			}
		}
		if len(group.Steps) == 0 {
			return &errors.DomainError{
				Category: errors.CategoryValidation,
				Code:     "INVALID_PIPELINE_CONFIG",
				Message:  fmt.Sprintf("groups[%d].steps: group must have at least one step", i),
			}
		}
		for j, step := range group.Steps {
			if step.Name == "" {
				return &errors.DomainError{
					Category: errors.CategoryValidation,
					Code:     "INVALID_PIPELINE_CONFIG",
					Message:  fmt.Sprintf("groups[%d].steps[%d].name: step name is required", i, j),
				}
			}
			if step.ActionType == "" {
				return &errors.DomainError{
					Category: errors.CategoryValidation,
					Code:     "INVALID_PIPELINE_CONFIG",
					Message:  fmt.Sprintf("groups[%d].steps[%d].action_type: action type is required for step '%s'", i, j, step.Name),
				}
			}
			if !model.ValidActionTypes[step.ActionType] {
				return &errors.DomainError{
					Category: errors.CategoryValidation,
					Code:     "INVALID_ACTION_TYPE",
					Message:  fmt.Sprintf("groups[%d].steps[%d].action_type: unknown action type: %s", i, j, step.ActionType),
				}
			}
		}
	}

	return nil
}
