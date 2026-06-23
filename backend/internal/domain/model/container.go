package model

import "time"

// ContainerOpts specifies configuration for creating an agent container.
type ContainerOpts struct {
	// Image is the Docker image to use (e.g., "hopeitworks/agent:latest").
	Image string

	// Env is a list of environment variables in KEY=VALUE format.
	Env []string

	// NetworkName is the Docker network to attach the container to.
	NetworkName string

	// Labels are key-value pairs for container metadata.
	// Standard labels: managed_by, run_id, step_id.
	Labels map[string]string

	// Memory is the memory limit in bytes (0 = unlimited).
	Memory int64

	// CPUs is the CPU limit as a float (0 = unlimited, 1.0 = 1 CPU).
	CPUs float64

	// ExtraNetworks are additional Docker networks to attach the container to,
	// in addition to NetworkName. Optional and nil-safe: empty/nil means the
	// current behaviour is preserved exactly (only NetworkName is used).
	ExtraNetworks []string

	// Aliases maps a network name to a DNS alias for the container on that
	// network. Optional and nil-safe: an entry is only applied if its network
	// appears in ExtraNetworks. Empty/nil means no aliases are set.
	Aliases map[string]string

	// Healthcheck, when non-nil, configures a Docker HEALTHCHECK on the
	// container so readiness can be polled via ContainerInspect. Optional and
	// nil-safe: nil means no healthcheck is configured (current behaviour).
	Healthcheck *ContainerHealthcheck
}

// ContainerHealthcheck describes a Docker HEALTHCHECK probe for a container.
// All durations are optional; zero values let Docker apply its own defaults.
type ContainerHealthcheck struct {
	// Test is the healthcheck command, e.g. {"CMD-SHELL", "pg_isready"} or
	// {"CMD", "redis-cli", "ping"}.
	Test []string

	// Interval is the time between two consecutive checks.
	Interval time.Duration

	// Timeout is the maximum time a single check may take before it is treated
	// as failed.
	Timeout time.Duration

	// Retries is the number of consecutive failures before the container is
	// considered unhealthy.
	Retries int

	// StartPeriod is the grace period during which a failing check does not
	// count towards the retry budget.
	StartPeriod time.Duration
}

// NetworkInfo represents metadata about a managed Docker network. It is the
// canonical shape shared by the ContainerManager and SidecarManager ports so
// the type is not duplicated across the port boundary.
type NetworkInfo struct {
	ID        string
	Name      string
	Labels    map[string]string
	CreatedAt time.Time
}
