package docker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// Ensure DockerSidecarManager implements port.SidecarManager at compile time.
var _ port.SidecarManager = (*DockerSidecarManager)(nil)

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
)

// serviceProfile holds the per-type knowledge needed to probe and (later, in
// P2c2c) build connection strings for a sidecar. It is keyed by a detected
// service type derived from the image name.
type serviceProfile struct {
	// healthTest is the Docker HEALTHCHECK command for this service type. Empty
	// means "no sensible healthcheck" -> fall back to running + grace delay.
	healthTest []string
	// port is the service's default listen port, factored here for reuse by
	// P2c2c connection-string injection.
	port int
}

// serviceProfiles maps a detected service type to its probe/port profile. It is
// intentionally small and additive; unknown types fall back to a running check.
var serviceProfiles = map[string]serviceProfile{
	"postgres": {healthTest: []string{"CMD-SHELL", "pg_isready -U postgres || pg_isready"}, port: 5432},
	"redis":    {healthTest: []string{"CMD", "redis-cli", "ping"}, port: 6379},
	"mysql":    {healthTest: []string{"CMD-SHELL", "mysqladmin ping -h 127.0.0.1 --silent"}, port: 3306},
	"mariadb":  {healthTest: []string{"CMD-SHELL", "mysqladmin ping -h 127.0.0.1 --silent"}, port: 3306},
	"mongo":    {healthTest: []string{"CMD-SHELL", "mongosh --eval 'db.runCommand({ping:1})' || mongo --eval 'db.runCommand({ping:1})'"}, port: 27017},
}

// servicePort returns the default listen port for a detected service type, or
// 0 when unknown. Factored here for reuse by P2c2c connection-string injection.
func servicePort(svcType string) int {
	return serviceProfiles[svcType].port
}

// detectServiceType maps an image reference to a known service type, or "" when
// unknown. It matches on the repository segment of the image (ignoring registry
// host and tag), e.g. "docker.io/library/postgres:16" -> "postgres".
func detectServiceType(image string) string {
	ref := image
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		ref = ref[i+1:]
	}
	if i := strings.IndexAny(ref, ":@"); i >= 0 {
		ref = ref[:i]
	}
	ref = strings.ToLower(ref)
	if _, ok := serviceProfiles[ref]; ok {
		return ref
	}
	return ""
}

// DockerSidecarManager orchestrates Environment sidecars over the injected
// ContainerManager. It does not own a Docker client of its own.
type DockerSidecarManager struct {
	containers port.ContainerManager
	logger     *slog.Logger

	// Tunables (defaulted in the constructor) kept as fields for testability.
	readinessTimeout  time.Duration
	readinessInterval time.Duration
	runningGrace      time.Duration

	// now is injectable so GC windowing is deterministic in tests.
	now func() time.Time
}

// NewDockerSidecarManager builds a SidecarManager backed by the given
// ContainerManager. The ContainerManager is reused for both networks and
// containers — no second Docker client is created.
func NewDockerSidecarManager(containers port.ContainerManager, logger *slog.Logger) *DockerSidecarManager {
	return &DockerSidecarManager{
		containers:        containers,
		logger:            logger,
		readinessTimeout:  defaultReadinessTimeout,
		readinessInterval: defaultReadinessInterval,
		runningGrace:      defaultRunningGrace,
		now:               time.Now,
	}
}

// networkName returns the per-run network name for a run id (full UUID).
func networkName(runID uuid.UUID) string {
	return sidecarNetworkPrefix + runID.String()
}

// Launch brings up the Environment's services on a fresh per-run network. It is
// nil-safe (nil/empty env -> empty context, no side effects) and rolls back
// atomically on any error.
func (s *DockerSidecarManager) Launch(ctx context.Context, runID uuid.UUID, env *model.Environment) (*port.SidecarContext, error) {
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
func (s *DockerSidecarManager) launchService(ctx context.Context, sc *port.SidecarContext, runID uuid.UUID, svc model.EnvironmentService) error {
	svcType := detectServiceType(svc.Image)

	opts := model.ContainerOpts{
		Image:         svc.Image,
		Env:           envMapToSlice(svc.Env),
		ExtraNetworks: []string{sc.NetworkName},
		Aliases:       map[string]string{sc.NetworkName: svc.Name},
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

	if err := s.waitReady(ctx, id, svcType); err != nil {
		return fmt.Errorf("sidecar %s not ready: %w", svc.Name, err)
	}

	// DNS hostname on the run network is the service name (set as alias above).
	sc.ServiceAddrs[svc.Name] = svc.Name
	return nil
}

// waitReady blocks until the sidecar is ready or the readiness timeout elapses.
// Services with a real healthcheck are polled via InspectHealth until "healthy";
// services without one fall back to a running check plus a short grace delay.
func (s *DockerSidecarManager) waitReady(ctx context.Context, containerID, svcType string) error {
	hasHealthcheck := svcType != "" && len(serviceProfiles[svcType].healthTest) > 0

	deadline := s.now().Add(s.readinessTimeout)
	for {
		status, err := s.containers.InspectHealth(ctx, containerID)
		if err != nil {
			return err
		}

		if hasHealthcheck {
			switch status {
			case model.HealthHealthy:
				return nil
			case model.HealthUnhealthy:
				return fmt.Errorf("container %s reported unhealthy", containerID)
			}
		} else {
			// No sensible healthcheck for this type: a running container plus a
			// short grace delay is the best signal we have. Documented limit:
			// readiness is approximate for unknown service types.
			if status == model.HealthRunning {
				s.sleep(ctx, s.runningGrace)
				return nil
			}
			if status == model.HealthNotRunning {
				return fmt.Errorf("container %s is not running", containerID)
			}
		}

		if !s.now().Before(deadline) {
			return fmt.Errorf("readiness timeout after %s (last status %q)", s.readinessTimeout, status)
		}
		if err := s.sleep(ctx, s.readinessInterval); err != nil {
			return err
		}
	}
}

// sleep waits for d, honouring context cancellation.
func (s *DockerSidecarManager) sleep(ctx context.Context, d time.Duration) error {
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
// container, then remove the network. Best-effort, log-only, never panics.
func (s *DockerSidecarManager) rollback(ctx context.Context, sc *port.SidecarContext) {
	s.teardownContainers(ctx, sc)
	if sc.NetworkName != "" {
		if err := s.containers.RemoveNetwork(ctx, sc.NetworkName); err != nil {
			s.logger.Warn("rollback: remove network failed",
				slog.String("network", sc.NetworkName),
				slog.String("error", err.Error()),
			)
		}
	}
}

// teardownContainers stops and removes all sidecar containers in sc.
func (s *DockerSidecarManager) teardownContainers(ctx context.Context, sc *port.SidecarContext) {
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
func (s *DockerSidecarManager) Stop(ctx context.Context, sc *port.SidecarContext) error {
	if sc == nil || len(sc.ContainerIDs) == 0 {
		return nil
	}
	for name, id := range sc.ContainerIDs {
		if err := s.containers.Stop(ctx, id); err != nil {
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
// Best-effort, idempotent, defer-safe.
func (s *DockerSidecarManager) Cleanup(ctx context.Context, sc *port.SidecarContext) error {
	if sc == nil || (len(sc.ContainerIDs) == 0 && sc.NetworkName == "") {
		return nil
	}
	s.teardownContainers(ctx, sc)
	if sc.NetworkName != "" {
		if err := s.containers.RemoveNetwork(ctx, sc.NetworkName); err != nil {
			s.logger.Warn("cleanup: remove network failed",
				slog.String("network", sc.NetworkName),
				slog.String("error", err.Error()),
			)
		}
	}
	return nil
}

// ListOrphanNetworks lists managed run networks with no managed sidecar
// container still attached. A network is matched to its containers by the
// shared run_id label, avoiding a per-network container lookup.
func (s *DockerSidecarManager) ListOrphanNetworks(ctx context.Context) ([]model.NetworkInfo, error) {
	managed := map[string]string{labelManagedBy: managedByLabel}

	networks, err := s.containers.ListNetworks(ctx, managed)
	if err != nil {
		return nil, err
	}
	containers, err := s.containers.ListContainers(ctx, managed)
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
func (s *DockerSidecarManager) GC(ctx context.Context, olderThan time.Duration) error {
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
	profile, ok := serviceProfiles[svcType]
	if !ok || len(profile.healthTest) == 0 {
		return nil
	}
	return &model.ContainerHealthcheck{
		Test:        profile.healthTest,
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
