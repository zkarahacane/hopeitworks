package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure PromptTemplateRepo implements port.PromptTemplateRepository at compile time.
var _ port.PromptTemplateRepository = (*PromptTemplateRepo)(nil)

// PromptTemplateRepo implements port.PromptTemplateRepository using sqlc-generated queries.
type PromptTemplateRepo struct {
	queries *Queries
}

// NewPromptTemplateRepo creates a new PromptTemplateRepo.
func NewPromptTemplateRepo(queries *Queries) *PromptTemplateRepo {
	return &PromptTemplateRepo{queries: queries}
}

// Create inserts a new prompt template.
func (r *PromptTemplateRepo) Create(ctx context.Context, tmpl *model.PromptTemplate) (*model.PromptTemplate, error) {
	params := CreatePromptTemplateParams{
		ProjectID:       tmpl.ProjectID,
		Name:            tmpl.Name,
		TemplateContent: tmpl.TemplateContent,
		Type:            string(tmpl.Type),
	}

	row, err := r.queries.CreatePromptTemplate(ctx, params)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("prompt_template", tmpl.Name)
		}
		return nil, apperrors.NewInternal("failed to create prompt template", err)
	}
	return toDomainPromptTemplate(row), nil
}

// GetByID retrieves a prompt template by ID.
func (r *PromptTemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	row, err := r.queries.GetPromptTemplate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("prompt_template", id)
		}
		return nil, apperrors.NewInternal("failed to get prompt template", err)
	}
	return toDomainPromptTemplate(row), nil
}

// GetByProjectAndName retrieves a prompt template by project ID and name.
func (r *PromptTemplateRepo) GetByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*model.PromptTemplate, error) {
	row, err := r.queries.GetPromptTemplateByProjectAndName(ctx, GetPromptTemplateByProjectAndNameParams{
		ProjectID: projectID,
		Name:      name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("prompt_template", name)
		}
		return nil, apperrors.NewInternal("failed to get prompt template by project and name", err)
	}
	return toDomainPromptTemplate(row), nil
}

// ListByProject retrieves prompt templates for a project with pagination.
func (r *PromptTemplateRepo) ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.PromptTemplate, error) {
	rows, err := r.queries.ListPromptTemplatesByProject(ctx, ListPromptTemplatesByProjectParams{
		ProjectID: projectID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list prompt templates", err)
	}
	templates := make([]*model.PromptTemplate, len(rows))
	for i, row := range rows {
		templates[i] = toDomainPromptTemplate(row)
	}
	return templates, nil
}

// CountByProject counts prompt templates for a project.
func (r *PromptTemplateRepo) CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error) {
	count, err := r.queries.CountPromptTemplatesByProject(ctx, projectID)
	if err != nil {
		return 0, apperrors.NewInternal("failed to count prompt templates", err)
	}
	return count, nil
}

// Update updates an existing prompt template.
func (r *PromptTemplateRepo) Update(ctx context.Context, tmpl *model.PromptTemplate) (*model.PromptTemplate, error) {
	params := UpdatePromptTemplateParams{
		ID:              tmpl.ID,
		Name:            textFromStringPtr(&tmpl.Name),
		TemplateContent: textFromStringPtr(&tmpl.TemplateContent),
		Type:            pgtype.Text{String: string(tmpl.Type), Valid: true},
	}

	row, err := r.queries.UpdatePromptTemplate(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("prompt_template", tmpl.ID)
		}
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("prompt_template", tmpl.Name)
		}
		return nil, apperrors.NewInternal("failed to update prompt template", err)
	}
	return toDomainPromptTemplate(row), nil
}

// Delete removes a prompt template by ID.
func (r *PromptTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.queries.DeletePromptTemplate(ctx, id)
	if err != nil {
		return apperrors.NewInternal("failed to delete prompt template", err)
	}
	return nil
}

// toDomainPromptTemplate maps a sqlc-generated PromptTemplate to a domain PromptTemplate.
func toDomainPromptTemplate(p PromptTemplate) *model.PromptTemplate {
	return &model.PromptTemplate{
		ID:              p.ID,
		ProjectID:       p.ProjectID,
		Name:            p.Name,
		TemplateContent: p.TemplateContent,
		Type:            model.TemplateType(p.Type),
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}
