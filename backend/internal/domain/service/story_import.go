package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// ImportStoryInput represents a single parsed story to be imported.
// This is the input DTO that the handler maps from the markdown adapter's ParsedStory.
type ImportStoryInput struct {
	Key                string
	Title              string
	Epic               string
	DependsOn          []string
	Scope              string
	Status             string
	AcceptanceCriteria string
	ParseError         error
}

// ImportResult holds the aggregated result of a markdown import operation.
type ImportResult struct {
	Imported int
	Updated  int
	Failed   int
	Errors   []ImportStoryError
}

// ImportStoryError describes a per-story import failure.
type ImportStoryError struct {
	Key     string
	Message string
	Code    string
}

// Import processes a slice of parsed stories, creating or updating each one.
// Stories with parse errors or missing required fields are recorded in the errors list.
// Each story is processed independently (no transaction) to allow partial success.
func (s *StoryService) Import(ctx context.Context, projectID uuid.UUID, stories []ImportStoryInput) (*ImportResult, error) {
	result := &ImportResult{
		Errors: make([]ImportStoryError, 0),
	}

	for _, input := range stories {
		if input.ParseError != nil {
			result.Failed++
			result.Errors = append(result.Errors, ImportStoryError{
				Key:     input.Key,
				Message: fmt.Sprintf("invalid YAML frontmatter: %v", input.ParseError),
				Code:    "YAML_PARSE_ERROR",
			})
			continue
		}

		if input.Key == "" {
			result.Failed++
			result.Errors = append(result.Errors, ImportStoryError{
				Key:     "",
				Message: "key is required",
				Code:    "VALIDATION_ERROR",
			})
			continue
		}

		if input.Title == "" {
			result.Failed++
			result.Errors = append(result.Errors, ImportStoryError{
				Key:     input.Key,
				Message: "title is required",
				Code:    "VALIDATION_ERROR",
			})
			continue
		}

		existing, err := s.repo.GetByKey(ctx, projectID, input.Key)
		if err != nil {
			domainErr, ok := err.(*errors.DomainError)
			if !ok || domainErr.Category != errors.CategoryNotFound {
				result.Failed++
				result.Errors = append(result.Errors, ImportStoryError{
					Key:     input.Key,
					Message: fmt.Sprintf("failed to check existing story: %v", err),
					Code:    "IMPORT_ERROR",
				})
				continue
			}
			// Not found — create new story
			story := buildStoryFromInput(projectID, input)
			_, createErr := s.repo.Create(ctx, story)
			if createErr != nil {
				result.Failed++
				result.Errors = append(result.Errors, ImportStoryError{
					Key:     input.Key,
					Message: fmt.Sprintf("failed to create story: %v", createErr),
					Code:    "IMPORT_ERROR",
				})
				continue
			}
			result.Imported++
			continue
		}

		// Found — update existing story
		updateStoryFromInput(existing, input)
		_, updateErr := s.repo.Update(ctx, existing)
		if updateErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, ImportStoryError{
				Key:     input.Key,
				Message: fmt.Sprintf("failed to update story: %v", updateErr),
				Code:    "IMPORT_ERROR",
			})
			continue
		}
		result.Updated++
	}

	return result, nil
}

func buildStoryFromInput(projectID uuid.UUID, input ImportStoryInput) *model.Story {
	story := &model.Story{
		ProjectID: projectID,
		Key:       input.Key,
		Title:     input.Title,
		DependsOn: input.DependsOn,
		Status:    model.StoryStatusBacklog,
	}

	if input.Scope != "" {
		story.Scope = &input.Scope
	}
	if input.Status != "" {
		story.Status = input.Status
	}
	if input.AcceptanceCriteria != "" {
		story.AcceptanceCriteria = &input.AcceptanceCriteria
	}

	return story
}

func updateStoryFromInput(existing *model.Story, input ImportStoryInput) {
	existing.Title = input.Title
	existing.DependsOn = input.DependsOn

	if input.Scope != "" {
		existing.Scope = &input.Scope
	}
	if input.Status != "" {
		existing.Status = input.Status
	}
	if input.AcceptanceCriteria != "" {
		existing.AcceptanceCriteria = &input.AcceptanceCriteria
	}
}
