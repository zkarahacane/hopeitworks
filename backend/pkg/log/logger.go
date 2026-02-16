package log

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type contextKey struct{}

// sensitiveKeys defines attribute keys whose values should be redacted.
var sensitiveKeys = map[string]bool{
	"password":      true,
	"token":         true,
	"secret":        true,
	"api_key":       true,
	"authorization": true,
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

// ScrubHandler wraps a slog.Handler to redact sensitive attribute values.
type ScrubHandler struct {
	slog.Handler
}

func (h *ScrubHandler) Handle(ctx context.Context, r slog.Record) error {
	var scrubbed []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		if sensitiveKeys[strings.ToLower(a.Key)] {
			scrubbed = append(scrubbed, slog.String(a.Key, "[REDACTED]"))
		} else {
			scrubbed = append(scrubbed, a)
		}
		return true
	})

	// Build a new record with scrubbed attrs.
	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	newRecord.AddAttrs(scrubbed...)
	return h.Handler.Handle(ctx, newRecord)
}

func (h *ScrubHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	scrubbed := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		if sensitiveKeys[strings.ToLower(a.Key)] {
			scrubbed[i] = slog.String(a.Key, "[REDACTED]")
		} else {
			scrubbed[i] = a
		}
	}
	return &ScrubHandler{Handler: h.Handler.WithAttrs(scrubbed)}
}

func (h *ScrubHandler) WithGroup(name string) slog.Handler {
	return &ScrubHandler{Handler: h.Handler.WithGroup(name)}
}
