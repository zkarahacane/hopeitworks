package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// 422 error codes surfaced by the connector PUT / status-options (mirror openapi).
const (
	// CodePlanningConnectorNoGitConnection is returned when write-back is enabled but
	// no usable git connection (stored PAT or env token) is configured.
	CodePlanningConnectorNoGitConnection = "PLANNING_CONNECTOR_NO_GIT_CONNECTION"
	// CodePlanningConnectorInvalidMapping is returned when write-back is enabled but
	// the status mapping has no usable target option.
	CodePlanningConnectorInvalidMapping = "PLANNING_CONNECTOR_INVALID_MAPPING"
	// CodePlanningStatusFieldNotFound is returned when the tracker is reachable but the
	// status single-select field could not be resolved (status-options endpoint).
	CodePlanningStatusFieldNotFound = "PLANNING_STATUS_FIELD_NOT_FOUND"

	// defaultStatusField is the single-select field name used when none is configured.
	defaultStatusField = "Status"
)

// PlanningConnectorService owns the persisted connector config (status field, done
// options, status mapping, write-back toggles) and the live status-options probe. It
// validates that enabling write-back requires both a configured git connection and a
// usable status mapping.
type PlanningConnectorService struct {
	connectors port.PlanningConnectorRepository
	projects   port.ProjectRepository
	resolver   port.GitCredentialResolver
	sinks      port.PlanningSinkFactory
}

// NewPlanningConnectorService wires the connector service.
func NewPlanningConnectorService(
	connectors port.PlanningConnectorRepository,
	projects port.ProjectRepository,
	resolver port.GitCredentialResolver,
	sinks port.PlanningSinkFactory,
) *PlanningConnectorService {
	return &PlanningConnectorService{connectors: connectors, projects: projects, resolver: resolver, sinks: sinks}
}

// LoadProject fetches the project for owner-or-admin authorization (404 if absent).
func (s *PlanningConnectorService) LoadProject(ctx context.Context, projectID uuid.UUID) (*model.Project, error) {
	return s.projects.GetByID(ctx, projectID)
}

// GetConnector returns the persisted connector or a not-found DomainError (the GET
// handler maps that to 404, matching the contract).
func (s *PlanningConnectorService) GetConnector(ctx context.Context, projectID uuid.UUID) (*model.PlanningConnector, error) {
	return s.connectors.Get(ctx, projectID)
}

// getConnectorOrNil returns the persisted connector, or nil when absent (no error).
func (s *PlanningConnectorService) getConnectorOrNil(ctx context.Context, projectID uuid.UUID) (*model.PlanningConnector, error) {
	conn, err := s.connectors.Get(ctx, projectID)
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// SetConnectorInput is the validated PUT payload (the handler applies enum defaults).
type SetConnectorInput struct {
	Source           string
	ProjectURL       *string
	StatusField      string
	DoneOptions      []string
	EpicIssueType    string
	StatusMapping    model.PlanningStatusMapping
	WritebackEnabled bool
	PostRunComment   bool
}

// SetConnector validates then upserts the connector. Enabling write-back requires a
// configured git connection (422 NO_GIT_CONNECTION) and a usable mapping (422
// INVALID_MAPPING).
func (s *PlanningConnectorService) SetConnector(ctx context.Context, projectID uuid.UUID, in SetConnectorInput) (*model.PlanningConnector, error) {
	source := in.Source
	if source == "" {
		source = string(port.SourceGitHub)
	}
	statusField := in.StatusField
	if statusField == "" {
		statusField = defaultStatusField
	}
	epicType := in.EpicIssueType
	if epicType == "" {
		epicType = "Epic"
	}

	if in.WritebackEnabled {
		tok, err := s.resolver.TokenForProject(ctx, projectID)
		if err != nil {
			return nil, err
		}
		if tok.Value == "" {
			return nil, errors.NewInvalidState(CodePlanningConnectorNoGitConnection,
				"write-back requires a configured git connection; connect this project to GitHub first")
		}
		if !in.StatusMapping.HasAnyTarget() {
			return nil, errors.NewInvalidState(CodePlanningConnectorInvalidMapping,
				"write-back requires a status mapping with at least one tracker option target")
		}
	}

	return s.connectors.Upsert(ctx, &model.PlanningConnector{
		ProjectID:        projectID,
		Source:           source,
		ProjectURL:       in.ProjectURL,
		StatusField:      statusField,
		DoneOptions:      in.DoneOptions,
		EpicIssueType:    epicType,
		StatusMapping:    in.StatusMapping,
		WritebackEnabled: in.WritebackEnabled,
		PostRunComment:   in.PostRunComment,
	})
}

// StatusOptions live-probes the tracker for the status single-select field + options.
// project_url / status_field come from the query overrides, else the persisted
// connector (else "Status"). A reachable-but-unresolvable field is a 422.
func (s *PlanningConnectorService) StatusOptions(ctx context.Context, projectID uuid.UUID, projectURLOverride, statusFieldOverride *string) (port.PlanningStatusOptions, error) {
	projectURL := ""
	statusField := ""
	if conn, err := s.getConnectorOrNil(ctx, projectID); err != nil {
		return port.PlanningStatusOptions{}, err
	} else if conn != nil {
		if conn.ProjectURL != nil {
			projectURL = *conn.ProjectURL
		}
		statusField = conn.StatusField
	}
	if projectURLOverride != nil && *projectURLOverride != "" {
		projectURL = *projectURLOverride
	}
	if statusFieldOverride != nil && *statusFieldOverride != "" {
		statusField = *statusFieldOverride
	}
	if statusField == "" {
		statusField = defaultStatusField
	}
	if projectURL == "" {
		return port.PlanningStatusOptions{}, errors.NewInvalidState(CodePlanningStatusFieldNotFound,
			"no board URL configured; save a connector or pass project_url")
	}

	sink, err := s.sinks.Sink(ctx, projectID)
	if err != nil {
		return port.PlanningStatusOptions{}, errors.NewInvalidState(CodePlanningConnectorNoGitConnection, err.Error())
	}
	opts, err := sink.StatusOptions(ctx, projectURL, statusField)
	if err != nil {
		return port.PlanningStatusOptions{}, errors.NewInvalidState(CodePlanningStatusFieldNotFound, err.Error())
	}
	return opts, nil
}
