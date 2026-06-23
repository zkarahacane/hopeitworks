package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Default scheduling for the sidecar GC. These are deliberately large so the GC
// can NEVER reap a run that is still starting up.
//
// Safety reasoning: a network is only a GC candidate once it has no RUNNING
// managed sidecar attached (SidecarManager.ListOrphanNetworks cross-references
// running containers). The window below is a *second* safety margin against the
// narrow race where a run network is created but its containers have not started
// yet — it must stay comfortably larger than any plausible startup latency.
const (
	// DefaultSidecarGCInterval is how often orphan run networks are swept.
	DefaultSidecarGCInterval = 30 * time.Minute
	// DefaultSidecarGCWindow is the minimum age an orphan network must reach
	// before it can be removed. Far larger than any run startup, so an actively
	// starting run is never reaped during the create-network/start-container gap.
	DefaultSidecarGCWindow = 1 * time.Hour
)

// SidecarGC periodically garbage-collects orphan per-run sidecar networks (and
// their sidecars) left behind when the API process dies abruptly between Launch
// and the agent_run defer Cleanup (e.g. SIGKILL/OOM). It is a best-effort safety
// net on top of that defer, never the primary teardown path.
type SidecarGC struct {
	sidecarMgr port.SidecarManager
	logger     *slog.Logger
	interval   time.Duration
	window     time.Duration
}

// NewSidecarGC creates a SidecarGC. A non-positive interval or window falls back
// to the safe defaults so a misconfiguration can never produce a tight,
// run-reaping sweep.
func NewSidecarGC(
	sidecarMgr port.SidecarManager,
	logger *slog.Logger,
	interval time.Duration,
	window time.Duration,
) *SidecarGC {
	if interval <= 0 {
		interval = DefaultSidecarGCInterval
	}
	if window <= 0 {
		window = DefaultSidecarGCWindow
	}
	return &SidecarGC{
		sidecarMgr: sidecarMgr,
		logger:     logger,
		interval:   interval,
		window:     window,
	}
}

// Start runs the GC sweep at the configured interval until the context is
// cancelled. Each sweep is best-effort: errors are logged at Warn and never
// abort the loop. Start blocks until the context is cancelled, mirroring the
// other background services started from main.go.
func (g *SidecarGC) Start(ctx context.Context) error {
	g.logger.Info("sidecar gc started",
		"interval", g.interval,
		"window", g.window,
	)

	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			g.logger.Info("sidecar gc stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := g.sidecarMgr.GC(ctx, g.window); err != nil {
				g.logger.Warn("sidecar gc sweep failed", "error", err)
			}
		}
	}
}
