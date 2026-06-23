package config

import "time"

// Config holds the complete application configuration.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Docker    DockerConfig    `yaml:"docker"`
	Substrate SubstrateConfig `yaml:"substrate"`
	Log       LogConfig       `yaml:"logging"`
	SMTP      SMTPConfig      `yaml:"smtp"`
	Security  SecurityConfig  `yaml:"security"`
	Stacks    []StackConfig   `yaml:"stacks"`
}

// Substrate kinds: the execution substrate that realises an agent run. "docker"
// is the live default; "microsandbox" selects the microVM substrate, which is a
// scaffold in P3a (selecting it does not change live execution — see main.go).
const (
	SubstrateDocker       = "docker"
	SubstrateMicrosandbox = "microsandbox"
)

// SubstrateConfig selects which execution substrate the platform uses. The
// substrate adapter (microsandbox/K8s later) implements port.AgentRuntime; the
// Docker substrate remains the default and the only one wired into the live
// agent_run flow until later P3 sub-phases.
type SubstrateConfig struct {
	// Kind is the substrate selector: "docker" (default) or "microsandbox".
	// Override via the SUBSTRATE env var.
	Kind string `yaml:"kind"`
}

// StackConfig is one entry of the stack catalogue: a catalogued runtime image
// (key -> image_ref + toolchain). This versioned config is the source of truth
// for the catalogue; it is re-applied as an idempotent UPSERT on every API boot,
// so pinning a stack to a new digest means editing this config, not a migration.
// The migration-inlined seed (000034) remains a bootstrap fallback only.
type StackConfig struct {
	Key       string         `yaml:"key"`
	ImageRef  string         `yaml:"image_ref"`
	Toolchain map[string]any `yaml:"toolchain"`
}

// SMTPConfig holds outbound email relay settings.
type SMTPConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	From        string `yaml:"from"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	FrontendURL string `yaml:"frontend_url"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Name            string `yaml:"name"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	SSLMode         string `yaml:"sslmode"`
	MaxConns        int32  `yaml:"max_conns"`
	MinConns        int32  `yaml:"min_conns"`
	MaxConnLifetime string `yaml:"max_conn_lifetime"`
	// AutoMigrate controls whether pending database migrations are applied
	// automatically on startup. Defaults to true.
	AutoMigrate *bool `yaml:"auto_migrate"`
}

// DockerConfig holds Docker connection settings.
type DockerConfig struct {
	// Host is the Docker API endpoint (e.g., "tcp://socket-proxy:2375").
	Host string `yaml:"host"`
	// AgentNetwork is the Docker network for agent containers.
	AgentNetwork string `yaml:"agent_network"`
	// CallbackBaseURL is the base URL agent containers use to call back to the API
	// (e.g., "http://api:8080"). Set via CALLBACK_BASE_URL env var.
	CallbackBaseURL string `yaml:"callback_base_url"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string `yaml:"level"`
}

// SecurityConfig holds security-related settings.
type SecurityConfig struct {
	// EncryptionKey is the master key used to derive AES-256 encryption keys
	// for storing sensitive data (e.g., user API keys). Override via ENCRYPTION_KEY env var.
	EncryptionKey string `yaml:"encryption_key"`
}
