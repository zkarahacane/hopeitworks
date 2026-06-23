package model

// Container bookkeeping labels shared across the run path and the Docker
// substrate so the same key/value is used where a label is STAMPED (the action
// that launches ephemeral command containers) and where it is MATCHED (the GC
// reaper that finds orphans). Centralising them in the domain avoids a literal
// drifting between the two packages.
const (
	// LabelManagedBy / LabelManagedByValue mark every container/network the
	// platform owns, so GC can scope its sweeps to platform-managed objects.
	LabelManagedBy      = "managed_by"
	LabelManagedByValue = "hopeitworks"

	// LabelRunID ties a container/network to its run for attribution and GC.
	LabelRunID = "run_id"

	// LabelRole distinguishes the kind of container within a run. RoleEnvCommand
	// marks the ephemeral one-shot containers that run an Environment setup
	// command (build/migrate/seed/test); the GC reaper matches on it.
	LabelRole      = "role"
	RoleEnvCommand = "env_command"

	// LabelCommandKey records which Environment command key (build/migrate/…) an
	// ephemeral command container ran, for diagnostics.
	LabelCommandKey = "command_key"
)
