package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// PromptTemplateService provides business logic for prompt template operations.
type PromptTemplateService struct {
	repo port.PromptTemplateRepository
}

// NewPromptTemplateService creates a new PromptTemplateService.
func NewPromptTemplateService(repo port.PromptTemplateRepository) *PromptTemplateService {
	return &PromptTemplateService{repo: repo}
}

// CreatePromptTemplateParams holds parameters for creating a prompt template.
type CreatePromptTemplateParams struct {
	ProjectID       uuid.UUID
	Name            string
	TemplateContent string
	Type            string
}

// Create validates inputs and creates a new prompt template.
func (s *PromptTemplateService) Create(ctx context.Context, params CreatePromptTemplateParams) (*model.PromptTemplate, error) {
	if params.Name == "" {
		return nil, errors.NewValidation("name", "is required")
	}
	if len(params.Name) > 255 {
		return nil, errors.NewValidation("name", "must be 255 characters or less")
	}
	if params.TemplateContent == "" {
		return nil, errors.NewValidation("template_content", "is required")
	}
	if params.ProjectID == uuid.Nil {
		return nil, errors.NewValidation("project_id", "is required")
	}
	if !isValidTemplateType(params.Type) {
		return nil, errors.NewValidation("type", "must be one of: implement, retry, review, merge, custom")
	}

	tmpl := &model.PromptTemplate{
		ProjectID:       params.ProjectID,
		Name:            params.Name,
		TemplateContent: params.TemplateContent,
		Type:            model.TemplateType(params.Type),
	}

	return s.repo.Create(ctx, tmpl)
}

// GetByID retrieves a prompt template by ID.
func (s *PromptTemplateService) GetByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	return s.repo.GetByID(ctx, id)
}

// PromptTemplateListResult holds the result of a paginated list operation.
type PromptTemplateListResult struct {
	Templates []*model.PromptTemplate
	Total     int64
}

// ListByProject retrieves a paginated list of prompt templates for a project.
func (s *PromptTemplateService) ListByProject(ctx context.Context, projectID uuid.UUID, page, perPage int) (*PromptTemplateListResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := int32((page - 1) * perPage)
	limit := int32(perPage)

	templates, err := s.repo.ListByProject(ctx, projectID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &PromptTemplateListResult{
		Templates: templates,
		Total:     total,
	}, nil
}

// UpdatePromptTemplateParams holds parameters for updating a prompt template.
type UpdatePromptTemplateParams struct {
	ID              uuid.UUID
	Name            *string
	TemplateContent *string
	Type            *string
}

// Update validates inputs and updates an existing prompt template.
func (s *PromptTemplateService) Update(ctx context.Context, params UpdatePromptTemplateParams) (*model.PromptTemplate, error) {
	existing, err := s.repo.GetByID(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if params.Name != nil {
		if *params.Name == "" {
			return nil, errors.NewValidation("name", "must not be empty")
		}
		if len(*params.Name) > 255 {
			return nil, errors.NewValidation("name", "must be 255 characters or less")
		}
		existing.Name = *params.Name
	}
	if params.TemplateContent != nil {
		if *params.TemplateContent == "" {
			return nil, errors.NewValidation("template_content", "must not be empty")
		}
		existing.TemplateContent = *params.TemplateContent
	}
	if params.Type != nil {
		if !isValidTemplateType(*params.Type) {
			return nil, errors.NewValidation("type", "must be one of: implement, retry, review, merge, custom")
		}
		existing.Type = model.TemplateType(*params.Type)
	}

	return s.repo.Update(ctx, existing)
}

// Delete removes a prompt template by ID.
func (s *PromptTemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func isValidTemplateType(t string) bool {
	switch model.TemplateType(t) {
	case model.TemplateTypeImplement, model.TemplateTypeRetry, model.TemplateTypeReview, model.TemplateTypeMerge, model.TemplateTypeCustom:
		return true
	default:
		return false
	}
}
