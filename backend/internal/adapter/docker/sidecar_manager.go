package docker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Ensure SidecarManager implements port.SidecarManager at compile time.
var _ port.SidecarManager = (*SidecarManager)(nil)

const (
	// sidecarNetworkPrefix is the name prefix for per-run sidecar networks. The
	// full run UUID is appended (never truncated) to avoid collisions.
	sidecarNetworkPrefix = "hopeitworks-run-"

	// labelManagedBy / labelRunID / labelSidecar are the bookkeeping labels put
	// on every sidecar network and container so GC can find and reap them.
	labelManagedBy = "managed_by"
	labelRunID     = "run_id"
	labelSidecar   = "sidecar"

	// defaultReadinessTimeout bounds how long Launch waits for a sidecar to
	// become ready before failing and rolling back. Never infinite.
	defaultReadinessTimeout = 30 * time.Second
	// defaultReadinessInterval is the poll period while waiting for readiness.
	defaultReadinessInterval = 2 * time.Second
	// defaultRunningGrace is the short settle delay used as a fallback for
	// services that declare no sensible healthcheck.
	defaultRunningGrace = 1 * time.Second
	// defaultTeardownTimeout bounds rollback/Stop/Cleanup. Teardown runs on a
	// context detached from the caller's (which may already be cancelled), so it
	// needs its own deadline to avoid hanging forever against the real daemon.
	defaultTeardownTimeout = 30 * time.Second
)

// serviceHealthTests is the Docker HEALTHCHECK command for each known service
// type. It is the Docker-specific facet of a service profile; the type detection
// and default ports live in the domain (model.DetectServiceType / model.ServicePort)
// so the mapping is not duplicated between this readiness probe and the run-path
// connection-string injection (action.buildConnStrings). An empty/absent entry
// means "no sensible healthcheck" -> fall back to running + grace delay.
//
// TODO: probe creds (redis AUTH / mariadb-admin). These probes assume no auth: a
// redis configured with requirepass returns NOAUTH (false unhealthy), and
// mysqladmin is deprecated on recent MariaDB (use mariadb-admin). The real fix
// injects the service credentials into the probe.
var serviceHealthTests = map[string][]string{
	model.ServiceTypePostgres: {"CMD-SHELL", "pg_isready -U postgres || pg_isready"},
	model.ServiceTypeRedis:    {"CMD", "redis-cli", "ping"},
	model.ServiceTypeMySQL:    {"CMD-SHELL", "mysqladmin ping -h 127.0.0.1 --silent"},
	model.ServiceTypeMariaDB:  {"CMD-SHELL", "mysqladmin ping -h 127.0.0.1 --silent"},
	model.ServiceTypeMongo:    {"CMD-SHELL", "mongosh --eval 'db.runCommand({ping:1})' || mongo --eval 'db.runCommand({ping:1})'"},
}

// detectServiceType maps an image reference to a known service type, or "" when
// unknown. Thin wrapper over the domain's single source of truth so detection is
// not duplicated across adapters.
func detectServiceType(image string) string {
	return model.DetectServiceType(image)
}

// SidecarManager orchestrates Environment sidecars over the injected
// ContainerManager. It does not own a Docker client of its own.
type SidecarManager struct {
	containers port.ContainerManager
	logger     *slog.Logger

	// Tunables (defaulted in the constructor) kept as fields for testability.
	readinessTimeout  time.Duration
	readinessInterval time.Duration
	runningGrace      time.Duration
	teardownTimeout   time.Duration

	// now is injectable so GC windowing is deterministic in tests.
	now func() time.Time
}

// NewDockerSidecarManager builds a SidecarManager backed by the given
// ContainerManager. The ContainerManager is reused for both networks and
// containers — no second Docker client is created.
func NewDockerSidecarManager(containers port.ContainerManager, logger *slog.Logger) *SidecarManager {
	return &SidecarManager{
		containers:        containers,
		logger:            logger,
		readinessTimeout:  defaultReadinessTimeout,
		readinessInterval: defaultReadinessInterval,
		runningGrace:      defaultRunningGrace,
		teardownTimeout:   defaultTeardownTimeout,
		now:               time.Now,
	}
}

// teardownContext derives a context for teardown that is immune to the caller's
// cancellation (Launch's ctx may already be cancelled when rollback runs) but
// still bounded by its own timeout so it cannot hang forever.
func (s *SidecarManager) teardownContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), s.teardownTimeout)
}

// networkName returns the per-run network name for a run id (full UUID).
func networkName(runID uuid.UUID) string {
	return sidecarNetworkPrefix + runID.String()
}

// Launch brings up the Environment's services on a fresh per-run network. It is
// nil-safe (nil/empty env -> empty context, no side effects) and rolls back
// atomically on any error.
func (s *SidecarManager) Launch(ctx context.Context, runID uuid.UUID, env *model.Environment) (*port.SidecarContext, error) {
	sc := &port.SidecarContext{
		RunID:        runID,
		ContainerIDs: map[string]string{},
		ServiceAddrs: map[string]string{},
	}

	// Nil-safe: nothing to do.
	if env == nil || len(env.Services) == 0 {
		return sc, nil
	}

	netName := networkName(runID)
	labels := map[string]string{
		labelManagedBy: managedByLabel,
		labelRunID:     runID.String(),
	}

	if _, err := s.containers.CreateNetwork(ctx, netName, labels); err != nil {
		return nil, fmt.Errorf("creating run network %s: %w", netName, err)
	}
	sc.NetworkName = netName

	for _, svc := range env.Services {
		if err := s.launchService(ctx, sc, runID, svc); err != nil {
			// Rollback atomically: tear down everything started so far.
			s.rollback(ctx, sc)
			return nil, err
		}
	}

	s.logger.Info("sidecars launched",
		slog.String("run_id", runID.String()),
		slog.String("network", netName),
		slog.Int("services", len(sc.ContainerIDs)),
	)
	return sc, nil
}

// launchService creates, starts and waits-ready a single sidecar, recording it
// into sc on success.
func (s *SidecarManager) launchService(ctx context.Context, sc *port.SidecarContext, runID uuid.UUID, svc model.EnvironmentService) error {
	svcType := detectServiceType(svc.Image)

	// Attach the sidecar to the run network DIRECTLY at creation (NetworkName,
	// not ExtraNetworks) so it never lands on the default bridge first — that
	// would break per-run isolation. The alias makes it reachable by svc.Name on
	// the run network (Create applies Aliases on the primary endpoint too).
	opts := model.ContainerOpts{
		Image:       svc.Image,
		Env:         envMapToSlice(svc.Env),
		NetworkName: sc.NetworkName,
		Aliases:     map[string]string{sc.NetworkName: svc.Name},
		Labels: map[string]string{
			labelManagedBy: managedByLabel,
			labelRunID:     runID.String(),
			labelSidecar:   svc.Name,
		},
		Healthcheck: healthcheckFor(svcType, s.readinessInterval),
	}

	id, err := s.containers.Create(ctx, opts)
	if err != nil {
		return fmt.Errorf("creating sidecar %s: %w", svc.Name, err)
	}
	// Record immediately so rollback can reach it even if Start/readiness fails.
	sc.ContainerIDs[svc.Name] = id

	if err := s.containers.Start(ctx, id); err != nil {
		return fmt.Errorf("starting sidecar %s: %w", svc.Name, err)
	}

	if err := s.waitReady(ctx, id); err != nil {
		return fmt.Errorf("sidecar %s not ready: %w", svc.Name, err)
	}

	// DNS hostname on the run network is the service name (set as alias above).
	sc.ServiceAddrs[svc.Name] = svc.Name
	return nil
}

// waitReady blocks until the sidecar is ready or the readiness timeout elapses.
// Readiness is deduced from the status actually returned by InspectHealth, not
// only from whether we configured a profile healthcheck: a custom image may bake
// its own HEALTHCHECK, so "healthy" must count as ready even for unknown types,
// and "unhealthy" must count as failure. Containers with no healthcheck at all
// fall back to a running check plus a short grace delay.
func (s *SidecarManager) waitReady(ctx context.Context, containerID string) error {
	deadline := s.now().Add(s.readinessTimeout)
	for {
		status, err := s.containers.InspectHealth(ctx, containerID)
		if err != nil {
			return err
		}

		switch status {
		case model.HealthHealthy:
			// A passing healthcheck (configured by us OR baked in the image) is
			// the strongest readiness signal.
			return nil
		case model.HealthUnhealthy:
			return fmt.Errorf("container %s reported unhealthy", containerID)
		case model.HealthRunning:
			// No healthcheck reported by Docker: a running container plus a short
			// grace delay is the best signal we have. Documented limit: readiness
			// is approximate when neither we nor the image declare a healthcheck.
			if err := s.sleep(ctx, s.runningGrace); err != nil {
				return err
			}
			return nil
		case model.HealthNotRunning:
			return fmt.Errorf("container %s is not running", containerID)
		}
		// model.HealthStarting (healthcheck in its start period): keep polling.

		if !s.now().Before(deadline) {
			return fmt.Errorf("readiness timeout after %s (last status %q)", s.readinessTimeout, status)
		}
		if err := s.sleep(ctx, s.readinessInterval); err != nil {
			return err
		}
	}
}

// sleep waits for d, honouring context cancellation.
func (s *SidecarManager) sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// rollback tears down a partially-launched run: stop+remove every started
// container, then remove the network. Best-effort, log-only, never panics. Runs
// on a context detached from the caller's so it still works if Launch's ctx was
// cancelled (otherwise every teardown call would fail with context.Canceled).
func (s *SidecarManager) rollback(ctx context.Context, sc *port.SidecarContext) {
	tctx, cancel := s.teardownContext(ctx)
	defer cancel()

	s.teardownContainers(tctx, sc)
	if sc.NetworkName != "" {
		if err := s.containers.RemoveNetwork(tctx, sc.NetworkName); err != nil {
			s.logger.Warn("rollback: remove network failed",
				slog.String("network", sc.NetworkName),
				slog.String("error", err.Error()),
			)
		}
	}
}

// teardownContainers stops and removes all sidecar containers in sc.
func (s *SidecarManager) teardownContainers(ctx context.Context, sc *port.SidecarContext) {
	for name, id := range sc.ContainerIDs {
		if err := s.containers.Stop(ctx, id); err != nil {
			s.logger.Warn("teardown: stop sidecar failed",
				slog.String("sidecar", name),
				slog.String("container_id", id),
				slog.String("error", err.Error()),
			)
		}
		if err := s.containers.Remove(ctx, id); err != nil {
			s.logger.Warn("teardown: remove sidecar failed",
				slog.String("sidecar", name),
				slog.String("container_id", id),
				slog.String("error", err.Error()),
			)
		}
	}
}

// Stop stops the sidecar containers in sc. Best-effort, idempotent, defer-safe.
// Runs on a detached, bounded context so it still works when called from a defer
// after the run's context was cancelled.
func (s *SidecarManager) Stop(ctx context.Context, sc *port.SidecarContext) error {
	if sc == nil || len(sc.ContainerIDs) == 0 {
		return nil
	}
	tctx, cancel := s.teardownContext(ctx)
	defer cancel()

	for name, id := range sc.ContainerIDs {
		if err := s.containers.Stop(tctx, id); err != nil {
			s.logger.Warn("stop sidecar failed",
				slog.String("sidecar", name),
				slog.String("container_id", id),
				slog.String("error", err.Error()),
			)
		}
	}
	return nil
}

// Cleanup stops+removes the sidecar containers and removes the run network.
// Best-effort, idempotent, defer-safe. Runs on a detached, bounded context so it
// still works when called from a defer after the run's context was cancelled.
func (s *SidecarManager) Cleanup(ctx context.Context, sc *port.SidecarContext) error {
	if sc == nil || (len(sc.ContainerIDs) == 0 && sc.NetworkName == "") {
		return nil
	}
	tctx, cancel := s.teardownContext(ctx)
	defer cancel()

	s.teardownContainers(tctx, sc)
	if sc.NetworkName != "" {
		if err := s.containers.RemoveNetwork(tctx, sc.NetworkName); err != nil {
			s.logger.Warn("cleanup: remove network failed",
				slog.String("network", sc.NetworkName),
				slog.String("error", err.Error()),
			)
		}
	}
	return nil
}

// ListOrphanNetworks lists managed run networks with no RUNNING managed sidecar
// container still attached. A network is matched to its containers by the shared
// run_id label, avoiding a per-network container lookup. Only running containers
// keep a network alive — exited-but-not-yet-removed sidecars must not block GC.
func (s *SidecarManager) ListOrphanNetworks(ctx context.Context) ([]model.NetworkInfo, error) {
	managed := map[string]string{labelManagedBy: managedByLabel}

	networks, err := s.containers.ListNetworks(ctx, managed)
	if err != nil {
		return nil, err
	}
	containers, err := s.containers.ListRunningContainers(ctx, managed)
	if err != nil {
		return nil, err
	}

	liveRunIDs := map[string]bool{}
	for _, c := range containers {
		if rid := c.Labels[labelRunID]; rid != "" {
			liveRunIDs[rid] = true
		}
	}

	orphans := make([]model.NetworkInfo, 0)
	for _, n := range networks {
		rid := n.Labels[labelRunID]
		if rid == "" {
			// Not a per-run sidecar network; leave it alone.
			continue
		}
		if !liveRunIDs[rid] {
			orphans = append(orphans, n)
		}
	}
	return orphans, nil
}

// GC removes orphan run networks older than olderThan. Best-effort, log-only.
func (s *SidecarManager) GC(ctx context.Context, olderThan time.Duration) error {
	orphans, err := s.ListOrphanNetworks(ctx)
	if err != nil {
		return err
	}

	cutoff := s.now().Add(-olderThan)
	removed := 0
	for _, n := range orphans {
		if n.CreatedAt.After(cutoff) {
			// Too recent: a run may still be starting up. Keep it.
			continue
		}
		if err := s.containers.RemoveNetwork(ctx, n.ID); err != nil {
			s.logger.Warn("gc: remove orphan network failed",
				slog.String("network", n.Name),
				slog.String("network_id", n.ID),
				slog.String("error", err.Error()),
			)
			continue
		}
		removed++
	}

	if removed > 0 {
		s.logger.Info("gc removed orphan networks", slog.Int("count", removed))
	}
	return nil
}

// healthcheckFor returns the HEALTHCHECK config for a service type, or nil when
// no sensible healthcheck exists (readiness then falls back to running+grace).
func healthcheckFor(svcType string, interval time.Duration) *model.ContainerHealthcheck {
	test, ok := serviceHealthTests[svcType]
	if !ok || len(test) == 0 {
		return nil
	}
	return &model.ContainerHealthcheck{
		Test:        test,
		Interval:    interval,
		Timeout:     3 * time.Second,
		Retries:     3,
		StartPeriod: 1 * time.Second,
	}
}

// envMapToSlice converts a service env map to the KEY=value slice the Docker
// adapter expects. A nil map yields a nil slice.
func envMapToSlice(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}
