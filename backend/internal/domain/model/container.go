package model

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
}
