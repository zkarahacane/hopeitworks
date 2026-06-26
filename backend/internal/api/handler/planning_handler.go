package handler

import (
	"net/http"
	"strings"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// PlanningHandler implements the one-way planning import endpoint
// (POST /projects/{projectId}/planning/import). It validates + maps HTTP to the
// source-discriminated ImportConfig and delegates every decision to the service.
type PlanningHandler struct {
	svc *service.PlanningImportService
}

// NewPlanningHandler creates a new PlanningHandler.
func NewPlanningHandler(svc *service.PlanningImportService) *PlanningHandler {
	return &PlanningHandler{svc: svc}
}

// ImportPlanning handles POST /projects/{projectId}/planning/import (admin only).
// 400 on a malformed/empty source config; 422 (SOURCE_ERROR) when the source is
// reachable-but-unusable; otherwise 200 with the import summary (or dry-run plan).
func (h *PlanningHandler) ImportPlanning(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath) {
	if !requireAdmin(w, r) {
		return
	}

	var req PlanningImportRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	cfg, err := buildImportConfig(req)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	summary, err := h.svc.Import(r.Context(), projectID, cfg)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPIPlanningResult(summary))
}

// buildImportConfig validates the request and builds the typed, discriminated
// ImportConfig. Empty markdown content / empty github project_url => 400.
func buildImportConfig(req PlanningImportRequest) (port.ImportConfig, error) {
	dryRun := req.DryRun != nil && *req.DryRun
	switch req.Source {
	case PlanningImportRequestSourceMarkdown:
		if req.Markdown == nil || strings.TrimSpace(req.Markdown.Content) == "" {
			return port.ImportConfig{}, errors.NewValidation("markdown.content", "must not be empty")
		}
		return port.ImportConfig{
			Source:   port.SourceMarkdown,
			DryRun:   dryRun,
			Markdown: &port.MarkdownConfig{Content: req.Markdown.Content},
		}, nil
	case PlanningImportRequestSourceGithubProjects:
		if req.GithubProjects == nil || strings.TrimSpace(req.GithubProjects.ProjectUrl) == "" {
			return port.ImportConfig{}, errors.NewValidation("github_projects.project_url", "must not be empty")
		}
		gh := &port.GitHubProjectsConfig{
			ProjectURL:    req.GithubProjects.ProjectUrl,
			StatusField:   strFromPtr(req.GithubProjects.StatusField),
			EpicIssueType: strFromPtr(req.GithubProjects.EpicIssueType),
		}
		if req.GithubProjects.DoneOptions != nil {
			gh.DoneOptions = *req.GithubProjects.DoneOptions
		}
		return port.ImportConfig{
			Source:         port.SourceGitHub,
			DryRun:         dryRun,
			GitHubProjects: gh,
		}, nil
	default:
		return port.ImportConfig{}, errors.NewValidation("source", "must be one of: markdown, github_projects")
	}
}

// toAPIPlanningResult maps the domain ImportSummary to the API result type.
func toAPIPlanningResult(s *port.ImportSummary) PlanningImportResult {
	res := PlanningImportResult{
		Source:         PlanningImportResultSource(s.Source),
		DryRun:         s.DryRun,
		EpicsCreated:   s.EpicsCreated,
		EpicsUpdated:   s.EpicsUpdated,
		StoriesCreated: s.StoriesCreated,
		StoriesUpdated: s.StoriesUpdated,
		Skipped:        s.Skipped,
		Locked:         s.Locked,
		Failed:         s.Failed,
		Errors:         make([]PlanningImportError, len(s.Errors)),
		Warnings:       make([]PlanningImportWarning, len(s.Warnings)),
		Items:          make([]PlanningImportItem, len(s.Items)),
	}
	res.SourceUrl = ptrIfNonEmptyStr(s.SourceURL)

	for i, e := range s.Errors {
		res.Errors[i] = PlanningImportError{
			Code:       e.Code,
			Message:    e.Message,
			Key:        ptrIfNonEmptyStr(e.Key),
			ExternalId: ptrIfNonEmptyStr(e.ExternalID),
		}
	}
	for i, wn := range s.Warnings {
		res.Warnings[i] = PlanningImportWarning{
			Code:    wn.Code,
			Message: wn.Message,
			Key:     ptrIfNonEmptyStr(wn.Key),
		}
	}
	for i, it := range s.Items {
		res.Items[i] = PlanningImportItem{
			Key:          it.Key,
			Kind:         PlanningImportItemKind(it.Kind),
			Action:       PlanningImportItemAction(it.Action),
			SourceUrl:    ptrIfNonEmptyStr(it.SourceURL),
			MappedStatus: ptrIfNonEmptyStr(it.MappedStatus),
			Reason:       ptrIfNonEmptyStr(it.Reason),
		}
	}
	return res
}

func strFromPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ptrIfNonEmptyStr returns a pointer to s, or nil when empty (for nullable JSON).
func ptrIfNonEmptyStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
