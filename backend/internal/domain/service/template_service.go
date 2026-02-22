package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Default template names used for fallback resolution.
const (
	TemplateNameImplement      = "implement"
	TemplateNameImplementRetry = "implement-retry"
	TemplateNameReview         = "review"
	TemplateNameMerge          = "merge"
	TemplateNameMergeConflict  = "merge-conflict"
)

// TemplateService resolves prompt templates from the database (with fallback to defaults)
// and renders them with story context using a TemplateRenderer.
type TemplateService struct {
	templateRepo port.PromptTemplateRepository
	renderer     port.TemplateRenderer
	logger       *slog.Logger
}

// NewTemplateService creates a new TemplateService.
func NewTemplateService(
	templateRepo port.PromptTemplateRepository,
	renderer port.TemplateRenderer,
	logger *slog.Logger,
) *TemplateService {
	return &TemplateService{
		templateRepo: templateRepo,
		renderer:     renderer,
		logger:       logger,
	}
}

// RenderForStory resolves a template by name for a project, falls back to defaults, and renders it.
func (s *TemplateService) RenderForStory(
	ctx context.Context,
	projectID uuid.UUID,
	templateName string,
	tmplCtx *model.TemplateContext,
) (string, error) {
	tmpl, err := s.templateRepo.GetByProjectAndName(ctx, projectID, templateName)
	if err != nil {
		// Check if the error is a not-found error
		domainErr, ok := err.(*errors.DomainError)
		if !ok || domainErr.Category != errors.CategoryNotFound {
			return "", err
		}

		// Template not found in DB, try default
		s.logger.Debug("template not found in DB, trying default",
			"project_id", projectID,
			"template_name", templateName,
		)

		defaultContent := getDefaultTemplate(templateName)
		if defaultContent == "" {
			return "", &errors.DomainError{
				Category: errors.CategoryNotFound,
				Code:     "TEMPLATE_NOT_FOUND",
				Message:  "template '" + templateName + "' not found in database and no default exists",
			}
		}

		return s.renderer.Render(defaultContent, tmplCtx)
	}

	return s.renderer.Render(tmpl.TemplateContent, tmplCtx)
}

// getDefaultTemplate returns hardcoded default template content for known template names.
// Returns empty string if no default exists for the given name.
func getDefaultTemplate(name string) string {
	defaults := map[string]string{
		TemplateNameImplement: `Implement story {{story_key}}: {{story_title}}

## Objective
{{story_objective}}

## Target Files
{{#each target_files}}
- {{this}}
{{/each}}

## Acceptance Criteria
{{acceptance_criteria}}

## Branch
{{branch_name}}`,

		TemplateNameImplementRetry: `Retry implementation for {{story_key}}: {{story_title}}

## Previous Error
{{error_context}}

## Log Tail
{{log_tail}}

## Existing Changes
{{diff_content}}

## Objective
{{story_objective}}

Fix the issues described above while preserving the existing changes.`,

		TemplateNameReview: `Review changes for {{story_key}}: {{story_title}}

## Story Context
**Objective:** {{story_objective}}

**Acceptance Criteria:**
{{acceptance_criteria}}

## Changes to Review
{{diff_content}}

## Review Instructions
- Verify all acceptance criteria are met
- Check code quality and adherence to project conventions
- Flag any issues or suggest improvements`,

		TemplateNameMerge: `Merge changes for {{story_key}}: {{story_title}}

## Story Context
**Objective:** {{story_objective}}

## Merge Instructions
- Create a pull request for the feature branch
- Ensure CI checks pass before merging
- Use squash merge to maintain clean commit history`,

		TemplateNameMergeConflict: `Resolve merge conflict for {{story_key}}: {{story_title}}

## Story Context
**Objective:** {{story_objective}}

## Conflict Details
{{error_context}}

## Current Changes
{{diff_content}}

## Resolution Instructions
- Review the conflict markers in the diff
- Resolve conflicts while preserving the story objective
- Ensure all acceptance criteria remain satisfied after resolution`,
	}
	return defaults[name]
}
