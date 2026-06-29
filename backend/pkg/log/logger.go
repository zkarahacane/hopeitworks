package log

import (
	"context"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

type contextKey struct{}

// sensitiveKeys defines attribute keys whose values are always redacted (exact match).
var sensitiveKeys = map[string]bool{
	"password":      true,
	"token":         true,
	"secret":        true,
	"api_key":       true,
	"authorization": true,
}

// safeIdentifierKeys are *_key attribute names that are entity identifiers, NOT
// secrets, and must stay readable for log traceability (they bypass the substring
// "key" rule below). The regex value scrubbing still applies to their values.
var safeIdentifierKeys = map[string]bool{
	"story_key":   true,
	"stack_key":   true,
	"command_key": true,
}

// Token/credential value patterns (A2). These are applied to the message AND to every
// string attribute value so a token never survives in a log even under a non-sensitive
// key or interpolated into a message/error string.
var (
	reGitHubToken    = regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}`)
	reFineGrainedPAT = regexp.MustCompile(`github_pat_[A-Za-z0-9_]+`)
	reURLCredentials = regexp.MustCompile(`(https?://)[^@\s/]+(?::[^@\s/]*)?@`)
)

const redacted = "[REDACTED]"

// Scrub removes credential material from a string: GitHub classic/OAuth/app/refresh
// tokens, fine-grained PATs, and userinfo (user:pass@) embedded in URLs. It is safe to
// call on log messages and on error text before wrapping/returning them.
func Scrub(s string) string {
	if s == "" {
		return s
	}
	s = reFineGrainedPAT.ReplaceAllString(s, redacted)
	s = reGitHubToken.ReplaceAllString(s, redacted)
	s = reURLCredentials.ReplaceAllString(s, "$1"+redacted+"@")
	return s
}

// isSensitiveKey reports whether an attribute key's value must be fully redacted.
// Exact sensitive keys win; identifier *_key names are explicitly spared; otherwise a
// substring match on token/secret/password/authorization/key triggers redaction.
func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	if sensitiveKeys[k] {
		return true
	}
	if safeIdentifierKeys[k] {
		return false
	}
	return strings.Contains(k, "token") ||
		strings.Contains(k, "secret") ||
		strings.Contains(k, "password") ||
		strings.Contains(k, "authorization") ||
		strings.Contains(k, "key")
}

// New creates a new *slog.Logger with JSON output on stdout.
// The level parameter accepts: debug, info, warn, error.
func New(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})

	return slog.New(&ScrubHandler{Handler: jsonHandler})
}

// WithLogger stores a logger in the context.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext retrieves the logger from the context, or returns the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// ScrubHandler wraps a slog.Handler to redact sensitive attribute values (by key
// name) and to scrub credential patterns from the message and from every string
// value (recursively through groups).
type ScrubHandler struct {
	slog.Handler
}

func (h *ScrubHandler) Handle(ctx context.Context, r slog.Record) error {
	scrubbed := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		scrubbed = append(scrubbed, scrubAttr(a))
		return true
	})

	newRecord := slog.NewRecord(r.Time, r.Level, Scrub(r.Message), r.PC)
	newRecord.AddAttrs(scrubbed...)
	return h.Handler.Handle(ctx, newRecord)
}

func (h *ScrubHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	scrubbed := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		scrubbed[i] = scrubAttr(a)
	}
	return &ScrubHandler{Handler: h.Handler.WithAttrs(scrubbed)}
}

func (h *ScrubHandler) WithGroup(name string) slog.Handler {
	return &ScrubHandler{Handler: h.Handler.WithGroup(name)}
}

// scrubAttr redacts a sensitive-keyed attribute entirely, recurses into groups, and
// scrubs credential patterns out of string values. Non-string scalars pass through.
func scrubAttr(a slog.Attr) slog.Attr {
	if isSensitiveKey(a.Key) {
		return slog.String(a.Key, redacted)
	}
	v := a.Value.Resolve()
	switch v.Kind() {
	case slog.KindGroup:
		group := v.Group()
		out := make([]slog.Attr, len(group))
		for i, ga := range group {
			out[i] = scrubAttr(ga)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(out...)}
	case slog.KindString:
		return slog.String(a.Key, Scrub(v.String()))
	default:
		return slog.Attr{Key: a.Key, Value: v}
	}
}
