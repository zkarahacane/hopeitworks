package config

import "time"

// Config holds the complete application configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Docker   DockerConfig   `yaml:"docker"`
	Log      LogConfig      `yaml:"logging"`
	SMTP     SMTPConfig     `yaml:"smtp"`
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
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string `yaml:"level"`
}
