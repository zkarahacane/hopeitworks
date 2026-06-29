package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// PlanningWriteBackService pushes an internal story status to the external tracker
// (one-way, hopeitworks -> tracker). It is the worker side of the River job: it loads
// the persisted connector, resolves the target option from the status mapping, calls
// the sink, and records the outcome on the story (writeback_status) + an audit row.
//
// Every no-op path (no connector / write-back off / wrong source / no mapping target)
// resolves writeback_status to "disabled" and returns nil — a write-back is never a
// hard failure of the run.
type PlanningWriteBackService struct {
	connectors port.PlanningConnectorRepository
	stories    port.StoryRepository
	audit      port.PlanningWriteBackRepository
	sinks      port.PlanningSinkFactory
	baseURL    string // public base URL for run links in comments ("" => id-only)
	logger     *slog.Logger
}

// NewPlanningWriteBackService wires the write-back service.
func NewPlanningWriteBackService(
	connectors port.PlanningConnectorRepository,
	stories port.StoryRepository,
	audit port.PlanningWriteBackRepository,
	sinks port.PlanningSinkFactory,
	baseURL string,
	logger *slog.Logger,
) *PlanningWriteBackService {
	if logger == nil {
		logger = slog.Default()
	}
	return &PlanningWriteBackService{
		connectors: connectors,
		stories:    stories,
		audit:      audit,
		sinks:      sinks,
		baseURL:    strings.TrimRight(baseURL, "/"),
		logger:     logger,
	}
}

// SyncStatus pushes internalStatus for one story to its tracker. runID is optional
// (nil for a non-run transition). It is best-effort: it returns an error only on an
// infrastructure failure worth retrying by River (e.g. the remote call failed); the
// no-op and "story missing" cases return nil.
func (s *PlanningWriteBackService) SyncStatus(ctx context.Context, storyID uuid.UUID, runID *uuid.UUID, internalStatus string) error {
	story, err := s.stories.GetByID(ctx, storyID)
	if err != nil {
		if isNotFound(err) {
			s.logger.Warn("write-back: story not found, skipping", "story_id", storyID)
			return nil
		}
		return err
	}

	conn, err := s.connectors.Get(ctx, story.ProjectID)
	if isNotFound(err) {
		s.markDisabled(ctx, storyID)
		return nil
	}
	if err != nil {
		return err
	}

	// No-op guards: write-back off, or either side is not a github_projects board.
	if !conn.WritebackEnabled ||
		conn.Source != string(port.SourceGitHub) ||
		story.Source != string(port.SourceGitHub) {
		s.markDisabled(ctx, storyID)
		return nil
	}

	optionID := conn.StatusMapping.OptionFor(internalStatus)
	if optionID == "" {
		// No mapping target for this transition: nothing to push (not an error).
		s.markDisabled(ctx, storyID)
		return nil
	}

	itemID := derefStr(story.ExternalItemID)
	projectURL := derefStr(conn.ProjectURL)
	if itemID == "" || projectURL == "" {
		msg := "missing tracker item id or board URL (re-import the board to capture item ids)"
		s.recordFailure(ctx, story, runID, internalStatus, "MISSING_TARGET", msg)
		return nil
	}

	sink, err := s.sinks.Sink(ctx, story.ProjectID)
	if err != nil {
		s.recordFailure(ctx, story, runID, internalStatus, classifyWriteBackError(err), err.Error())
		return nil
	}

	req := port.WriteBackRequest{
		ProjectURL:      projectURL,
		ItemID:          itemID,
		ContentNodeID:   derefStr(story.ExternalID),
		StatusFieldName: conn.StatusField,
		OptionID:        optionID,
	}
	if conn.PostRunComment {
		req.Comment = s.buildComment(story, runID, internalStatus)
	}

	res, err := sink.WriteBack(ctx, req)
	if err != nil {
		code := classifyWriteBackError(err)
		if isTransientWriteBackCode(code) {
			// Transient (rate limit / 5xx): leave writeback_status as pending and
			// return the error so River retries with backoff. The reconciling sink has
			// already left the connection status untouched for transient signals.
			s.logger.Warn("write-back transient failure, will retry", "story_id", story.ID, "code", code, "error", err)
			return err
		}
		s.recordFailure(ctx, story, runID, internalStatus, code, err.Error())
		return nil
	}

	s.recordSuccess(ctx, story, runID, internalStatus, res.RemoteStatus)
	return nil
}

// markDisabled sets writeback_status=disabled for a no-op (best-effort).
func (s *PlanningWriteBackService) markDisabled(ctx context.Context, storyID uuid.UUID) {
	if err := s.stories.SetWritebackStatus(ctx, storyID, string(model.WritebackDisabled)); err != nil {
		s.logger.Warn("write-back: failed to set disabled status", "story_id", storyID, "error", err)
	}
}

func (s *PlanningWriteBackService) recordSuccess(ctx context.Context, story *model.Story, runID *uuid.UUID, internalStatus, remoteStatus string) {
	if err := s.stories.SetWritebackStatus(ctx, story.ID, string(model.WritebackSynced)); err != nil {
		s.logger.Warn("write-back: failed to set synced status", "story_id", story.ID, "error", err)
	}
	s.writeAudit(ctx, story, runID, internalStatus, &remoteStatus, true, nil, nil)
	s.logger.Info("write-back synced", "story_id", story.ID, "internal_status", internalStatus, "remote_status", remoteStatus)
}

func (s *PlanningWriteBackService) recordFailure(ctx context.Context, story *model.Story, runID *uuid.UUID, internalStatus, code, msg string) {
	if err := s.stories.SetWritebackStatus(ctx, story.ID, string(model.WritebackFailed)); err != nil {
		s.logger.Warn("write-back: failed to set failed status", "story_id", story.ID, "error", err)
	}
	s.writeAudit(ctx, story, runID, internalStatus, nil, false, &code, &msg)
	s.logger.Warn("write-back failed", "story_id", story.ID, "internal_status", internalStatus, "code", code, "error", msg)
}

func (s *PlanningWriteBackService) writeAudit(ctx context.Context, story *model.Story, runID *uuid.UUID, internalStatus string, remoteStatus *string, success bool, code, msg *string) {
	source := story.Source
	wb := &model.PlanningWriteBack{
		ProjectID:      story.ProjectID,
		StoryID:        story.ID,
		RunID:          runID,
		Source:         &source,
		ExternalID:     story.ExternalItemID,
		InternalStatus: &internalStatus,
		RemoteStatus:   remoteStatus,
		Success:        success,
		ErrorCode:      code,
		ErrorMessage:   truncatePtr(msg, 1000),
	}
	if _, err := s.audit.Create(ctx, wb); err != nil {
		s.logger.Warn("write-back: failed to record audit row", "story_id", story.ID, "error", err)
	}
}

// buildComment formats the tracker comment posted on a transition. It links the run
// when a public base URL is configured, else falls back to the run id only.
func (s *PlanningWriteBackService) buildComment(story *model.Story, runID *uuid.UUID, internalStatus string) string {
	if runID == nil {
		return fmt.Sprintf("hopeitworks: story **%s** moved to **%s**.", story.Key, internalStatus)
	}
	if s.baseURL != "" {
		link := fmt.Sprintf("%s/projects/%s/runs/%s", s.baseURL, story.ProjectID, runID)
		return fmt.Sprintf("hopeitworks: story **%s** moved to **%s** by run [%s](%s).", story.Key, internalStatus, runID, link)
	}
	return fmt.Sprintf("hopeitworks: story **%s** moved to **%s** by run %s.", story.Key, internalStatus, runID)
}

// classifyWriteBackError derives a short, stable audit code from a remote error.
func classifyWriteBackError(err error) string {
	msg := strings.ToLower(err.Error())
	switch {
	case hasHTTPStatus(msg, "401") || strings.Contains(msg, "bad credentials") || strings.Contains(msg, "unauthorized"):
		return "UNAUTHORIZED"
	case hasHTTPStatus(msg, "403") || strings.Contains(msg, "forbidden"):
		return "FORBIDDEN"
	case hasHTTPStatus(msg, "429") || strings.Contains(msg, "rate limit"):
		return "RATE_LIMITED"
	case hasHTTPStatus(msg, "500") || hasHTTPStatus(msg, "502") || hasHTTPStatus(msg, "503") || hasHTTPStatus(msg, "504"):
		return "SERVER_ERROR"
	default:
		return "WRITE_BACK_ERROR"
	}
}

// isTransientWriteBackCode reports whether a failure code is worth a River retry.
func isTransientWriteBackCode(code string) bool {
	return code == "RATE_LIMITED" || code == "SERVER_ERROR"
}

// truncatePtr caps a string pointer to n runes (audit columns are bounded).
func truncatePtr(s *string, n int) *string {
	if s == nil {
		return nil
	}
	v := *s
	if len(v) > n {
		v = v[:n]
	}
	return &v
}
