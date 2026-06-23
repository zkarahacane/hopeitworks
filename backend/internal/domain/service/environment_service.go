package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// field name constants used in validation error messages.
const (
	envFieldSource = "source"
	envFieldStacks = "stacks"
)

// validSources is the set of allowed EnvironmentSource values.
var validSources = map[string]bool{
	model.EnvironmentSourceDevcontainer: true,
	model.EnvironmentSourceCompose:      true,
	model.EnvironmentSourceMakefile:     true,
	model.EnvironmentSourceDeclared:     true,
}

// validStacks is the set of allowed StackKey values.
var validStacks = map[string]bool{
	model.StackKeyGo:     true,
	model.StackKeyNode:   true,
	model.StackKeyPython: true,
	model.StackKeyGoNode: true,
}

// EnvService provides business logic for project execution environments.
// Exactly one Environment exists per project (upsert semantics).
type EnvService struct {
	repo port.EnvironmentRepository
}

// NewEnvironmentService creates a new EnvService.
func NewEnvironmentService(repo port.EnvironmentRepository) *EnvService {
	return &EnvService{repo: repo}
}

// GetByProject returns the project's execution environment, or NotFound when absent.
func (s *EnvService) GetByProject(ctx context.Context, projectID uuid.UUID) (*model.Environment, error) {
	return s.repo.GetByProjectID(ctx, projectID)
}

// UpsertEnvironmentInput carries the fields that can be set when creating or
// replacing a project's environment.
type UpsertEnvironmentInput struct {
	Stacks   []string
	Services []model.EnvironmentService
	Source   string
	Commands map[string]string
}

// Upsert creates the environment when absent or replaces it when present.
// An empty source defaults to "declared" before validation.
func (s *EnvService) Upsert(ctx context.Context, projectID uuid.UUID, input UpsertEnvironmentInput) (*model.Environment, error) {
	// Default empty source to "declared".
	if input.Source == "" {
		input.Source = model.EnvironmentSourceDeclared
	}

	if err := validateEnvironmentInput(input); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByProjectID(ctx, projectID)
	if err != nil {
		domErr, ok := err.(*errors.DomainError)
		if !ok || domErr.Category != errors.CategoryNotFound {
			return nil, err
		}
		// Not found — create a new environment.
		env := &model.Environment{
			ProjectID: projectID,
			Stacks:    input.Stacks,
			Services:  input.Services,
			Source:    input.Source,
			Commands:  input.Commands,
		}
		return s.repo.Create(ctx, env)
	}

	// Found — update in place.
	existing.Stacks = input.Stacks
	existing.Services = input.Services
	existing.Source = input.Source
	existing.Commands = input.Commands
	return s.repo.Update(ctx, existing)
}

// Delete removes the project's environment. Returns NotFound when absent.
func (s *EnvService) Delete(ctx context.Context, projectID uuid.UUID) error {
	existing, err := s.repo.GetByProjectID(ctx, projectID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, existing.ID)
}

// validateEnvironmentInput validates source and stacks fields.
func validateEnvironmentInput(input UpsertEnvironmentInput) error {
	if !validSources[input.Source] {
		return errors.NewValidation(envFieldSource, "must be one of: devcontainer, compose, makefile, declared")
	}
	for _, sk := range input.Stacks {
		if !validStacks[sk] {
			return errors.NewValidation(envFieldStacks, "unknown stack key: "+sk+"; must be one of: go, node, python, go-node")
		}
	}
	return nil
}
