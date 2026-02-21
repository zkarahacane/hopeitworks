package postgres

import (
	"context"
	"errors"
	"math"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure ProjectRepo implements port.ProjectRepository at compile time.
var _ port.ProjectRepository = (*ProjectRepo)(nil)

// ProjectRepo implements port.ProjectRepository using sqlc-generated queries.
type ProjectRepo struct {
	queries *Queries
}

// NewProjectRepo creates a new ProjectRepo.
func NewProjectRepo(queries *Queries) *ProjectRepo {
	return &ProjectRepo{queries: queries}
}

func (r *ProjectRepo) Create(ctx context.Context, project *model.Project) (*model.Project, error) {
	params := CreateProjectParams{
		Name:         project.Name,
		Description:  textFromStringPtr(project.Description),
		OwnerID:      uuidFromPtr(project.OwnerID),
		RepoUrl:      textFromStringPtr(project.RepoURL),
		GitProvider:  project.GitProvider,
		GitTokenEnv:  textFromStringPtr(project.GitTokenEnv),
		AgentRuntime: project.AgentRuntime,
		DefaultModel: textFromStringPtr(project.DefaultModel),
		MaxBudget:    numericFromFloat64Ptr(project.MaxBudget),
	}

	row, err := r.queries.CreateProject(ctx, params)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("project", project.Name)
		}
		return nil, apperrors.NewInternal("failed to create project", err)
	}
	return toDomainProject(row), nil
}

func (r *ProjectRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	row, err := r.queries.GetProject(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("project", id)
		}
		return nil, apperrors.NewInternal("failed to get project", err)
	}
	return toDomainProject(row), nil
}

func (r *ProjectRepo) List(ctx context.Context, limit, offset int32) ([]*model.Project, error) {
	rows, err := r.queries.ListProjects(ctx, ListProjectsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to list projects", err)
	}
	projects := make([]*model.Project, len(rows))
	for i, row := range rows {
		projects[i] = toDomainProject(row)
	}
	return projects, nil
}

func (r *ProjectRepo) Count(ctx context.Context) (int64, error) {
	count, err := r.queries.CountProjects(ctx)
	if err != nil {
		return 0, apperrors.NewInternal("failed to count projects", err)
	}
	return count, nil
}

func (r *ProjectRepo) Update(ctx context.Context, project *model.Project) (*model.Project, error) {
	params := UpdateProjectParams{
		ID:           project.ID,
		Name:         textFromStringPtr(&project.Name),
		Description:  textFromStringPtr(project.Description),
		OwnerID:      uuidFromPtr(project.OwnerID),
		RepoUrl:      textFromStringPtr(project.RepoURL),
		GitProvider:  textFromStringPtr(&project.GitProvider),
		GitTokenEnv:  textFromStringPtr(project.GitTokenEnv),
		AgentRuntime: textFromStringPtr(&project.AgentRuntime),
		DefaultModel: textFromStringPtr(project.DefaultModel),
		MaxBudget:    numericFromFloat64Ptr(project.MaxBudget),
	}

	row, err := r.queries.UpdateProject(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("project", project.ID)
		}
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflict("project", project.Name)
		}
		return nil, apperrors.NewInternal("failed to update project", err)
	}
	return toDomainProject(row), nil
}

func (r *ProjectRepo) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.queries.DeleteProject(ctx, id)
	if err != nil {
		return apperrors.NewInternal("failed to delete project", err)
	}
	return nil
}

// IncrementCircuitBreakerCount increments the circuit breaker failure count for a project.
// If the count reaches the max threshold, the circuit breaker is activated.
func (r *ProjectRepo) IncrementCircuitBreakerCount(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	row, err := r.queries.IncrementCircuitBreakerCount(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("project", id)
		}
		return nil, apperrors.NewInternal("failed to increment circuit breaker count", err)
	}
	return toDomainProject(row), nil
}

// ResetCircuitBreaker resets the circuit breaker state for a project.
func (r *ProjectRepo) ResetCircuitBreaker(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	row, err := r.queries.ResetCircuitBreaker(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("project", id)
		}
		return nil, apperrors.NewInternal("failed to reset circuit breaker", err)
	}
	return toDomainProject(row), nil
}

// toDomainProject maps a sqlc-generated Project to a domain Project.
func toDomainProject(p Project) *model.Project {
	project := &model.Project{
		ID:                   p.ID,
		Name:                 p.Name,
		GitProvider:          p.GitProvider,
		AgentRuntime:         p.AgentRuntime,
		CircuitBreakerCount:  int(p.CircuitBreakerCount),
		CircuitBreakerActive: p.CircuitBreakerActive,
		CircuitBreakerMax:    int(p.CircuitBreakerMax),
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
	if p.Description.Valid {
		project.Description = &p.Description.String
	}
	if p.OwnerID.Valid {
		id := uuid.UUID(p.OwnerID.Bytes)
		project.OwnerID = &id
	}
	if p.RepoUrl.Valid {
		project.RepoURL = &p.RepoUrl.String
	}
	if p.GitTokenEnv.Valid {
		project.GitTokenEnv = &p.GitTokenEnv.String
	}
	if p.DefaultModel.Valid {
		project.DefaultModel = &p.DefaultModel.String
	}
	if p.MaxBudget.Valid {
		f := numericToFloat64(p.MaxBudget)
		project.MaxBudget = &f
	}
	return project
}

func textFromStringPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func uuidFromPtr(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

func numericFromFloat64Ptr(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	// Convert float64 to pgtype.Numeric via big.Int representation
	// Multiply by 100 for 2 decimal places, store as integer with exponent -2
	cents := int64(math.Round(*f * 100))
	return pgtype.Numeric{
		Int:   big.NewInt(cents),
		Exp:   -2,
		Valid: true,
	}
}

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid || n.Int == nil {
		return 0
	}
	f, _ := new(big.Float).SetInt(n.Int).Float64()
	for i := int32(0); i < -n.Exp; i++ {
		f /= 10
	}
	for i := int32(0); i < n.Exp; i++ {
		f *= 10
	}
	return f
}

// isUniqueViolation checks if a pgx error is a unique constraint violation.
func isUniqueViolation(err error) bool {
	// pgx wraps postgres errors; check for SQLSTATE 23505
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
