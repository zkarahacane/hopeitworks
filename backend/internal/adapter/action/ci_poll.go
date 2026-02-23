package action

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// CIPollConfig holds tuneable parameters for CI polling.
type CIPollConfig struct {
	// DefaultPollInterval is how often to check CI status (default: 30s).
	DefaultPollInterval time.Duration
	// DefaultTimeout is the maximum time to wait for CI to pass (default: 15min).
	DefaultTimeout time.Duration
}

// CIPollAction implements model.Action for polling CI status via GitProvider.
type CIPollAction struct {
	gitProviderFactory port.GitProviderFactory
	eventPub           port.EventPublisher
	config             CIPollConfig
	logger             *slog.Logger
}

// NewCIPollAction creates a new CIPollAction.
func NewCIPollAction(
	gitProviderFactory port.GitProviderFactory,
	eventPub port.EventPublisher,
	config CIPollConfig,
	logger *slog.Logger,
) *CIPollAction {
	return &CIPollAction{
		gitProviderFactory: gitProviderFactory,
		eventPub:           eventPub,
		config:             config,
		logger:             logger,
	}
}

// Name returns the action identifier.
func (a *CIPollAction) Name() string { return "ci_poll" }

// Execute polls CI status for a merged PR until it passes, fails, or times out.
// It reads pr_url from runCtx.Metadata and optionally poll_interval_seconds and timeout_seconds.
func (a *CIPollAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
	prURL, _ := runCtx.Metadata["pr_url"].(string)
	if prURL == "" {
		return fmt.Errorf("CI_POLL_MISSING_PR_URL: missing required metadata key pr_url")
	}

	gitProvider, err := a.gitProviderFactory.ForProjectID(ctx, runCtx.ProjectID)
	if err != nil {
		return fmt.Errorf("resolve git provider: %w", err)
	}

	workDir, _ := runCtx.Metadata["work_dir"].(string)

	pollInterval := a.config.DefaultPollInterval
	if secs, ok := runCtx.Metadata["poll_interval_seconds"].(float64); ok && secs > 0 {
		pollInterval = time.Duration(secs) * time.Second
	}
	timeout := a.config.DefaultTimeout
	if secs, ok := runCtx.Metadata["timeout_seconds"].(float64); ok && secs > 0 {
		timeout = time.Duration(secs) * time.Second
	}

	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pollCtx.Done():
			if ctx.Err() != nil {
				// Parent context was cancelled — propagate it.
				return ctx.Err()
			}
			return fmt.Errorf("CI_POLL_TIMEOUT: timed out after %v waiting for CI on %s", timeout, prURL)
		case <-ticker.C:
			status, err := gitProvider.GetCIStatus(pollCtx, workDir)
			if err != nil {
				a.logger.Warn("ci_poll: GetCIStatus error", "error", err, "pr_url", prURL)
				a.publishPollingEvent(ctx, runCtx, prURL, "error")
				continue
			}
			switch status {
			case "pass":
				a.publishPollingEvent(ctx, runCtx, prURL, status)
				return nil
			case "fail":
				a.publishPollingEvent(ctx, runCtx, prURL, status)
				return fmt.Errorf("CI_POLL_FAILED: CI checks failed for PR %s", prURL)
			default:
				// "pending", "no_checks" — keep polling.
				a.publishPollingEvent(ctx, runCtx, prURL, status)
			}
		}
	}
}

// publishPollingEvent publishes a ci_poll.checking event to the event system.
// Errors are logged as warnings and do not interrupt polling.
func (a *CIPollAction) publishPollingEvent(ctx context.Context, runCtx *model.RunContext, prURL, status string) {
	payload, err := json.Marshal(map[string]string{
		"pr_url": prURL,
		"status": status,
	})
	if err != nil {
		a.logger.Warn("ci_poll: failed to marshal polling event payload", "error", err)
		return
	}

	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  runCtx.ProjectID,
		EntityType: "ci_poll",
		EntityID:   runCtx.RunStep.ID,
		Action:     "checking",
		Payload:    payload,
	}

	if err := a.eventPub.Publish(ctx, event); err != nil {
		a.logger.Warn("ci_poll: failed to publish polling event", "error", err)
	}
}
