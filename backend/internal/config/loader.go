package config

import (
	"fmt"
	"os"
	"strconv"

	pkgconfig "github.com/zakari/hopeitworks/backend/pkg/config"
	"gopkg.in/yaml.v3"
)

// Load reads a YAML config file and applies environment variable overrides.
func Load(path string) (*pkgconfig.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg pkgconfig.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	applyEnvOverrides(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return &cfg, nil
}

func applyEnvOverrides(cfg *pkgconfig.Config) {
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		}
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		cfg.Database.SSLMode = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("DOCKER_HOST"); v != "" {
		cfg.Docker.Host = v
	}
	if v := os.Getenv("DOCKER_AGENT_NETWORK"); v != "" {
		cfg.Docker.AgentNetwork = v
	}
}

func validate(cfg *pkgconfig.Config) error {
	if cfg.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	if cfg.Database.Name == "" {
		return fmt.Errorf("database.name is required")
	}
	if cfg.Database.User == "" {
		return fmt.Errorf("database.user is required")
	}
	if cfg.Database.Password == "" {
		return fmt.Errorf("database.password is required")
	}
	return nil
}
