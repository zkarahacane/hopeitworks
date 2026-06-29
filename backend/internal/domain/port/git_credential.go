package port

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// GitToken is a resolved, ready-to-use credential. Short-lived in memory; it MUST
// never be logged or persisted. It implements slog.LogValuer + Stringer so an
// accidental %v/%s or structured-log call redacts even when the attribute key is
// not in the ScrubHandler list.
//
// INVARIANT (security A4): the resolved token is consumed ONLY by server-side git
// adapters. It is NEVER injected into an agent container (env/prompt/transcript).
// The agent_run path deliberately does NOT depend on GitCredentialResolver.
type GitToken struct {
	Value     string
	ExpiresAt *time.Time // non-nil when the expiry is known (fine-grained PATs)
}

// LogValue redacts the token in any structured log.
func (GitToken) LogValue() slog.Value { return slog.StringValue("[REDACTED]") }

// String redacts the token under %v/%s/fmt.Stringer.
func (GitToken) String() string { return "[REDACTED]" }

// GitCredentialResolver is THE single seam both git.DefaultGitProviderFactory and
// planning.Factory use to obtain a usable token for a project, regardless of how it
// is stored. With no git_connections row it falls back to the legacy env path so
// existing projects keep working. It also self-heals the advisory connection status
// from real operation errors (C1): 401 -> invalid, 403 -> insufficient_scope, while
// transient failures (429/5xx) leave the status untouched.
type GitCredentialResolver interface {
	// TokenForProject returns the active token for a project, or a zero GitToken
	// (Value == "") when nothing is configured.
	TokenForProject(ctx context.Context, projectID uuid.UUID) (GitToken, error)

	// ReconcileFromOperationError inspects an error returned by a REAL git/planning
	// operation and, when it is a definitive auth failure, flips the stored
	// connection status to match reality. Best-effort and non-fatal: it never
	// returns an error and is a no-op when no connection row exists.
	ReconcileFromOperationError(ctx context.Context, projectID uuid.UUID, opErr error)
}
