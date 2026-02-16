//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/api"
	internalconfig "github.com/zakari/hopeitworks/backend/internal/config"
	pkgconfig "github.com/zakari/hopeitworks/backend/pkg/config"
	pkglog "github.com/zakari/hopeitworks/backend/pkg/log"
)

// App holds the wired application dependencies.
type App struct {
	Config *pkgconfig.Config
	Logger *slog.Logger
	Pool   *pgxpool.Pool
	Router chi.Router
}

// InitializeApp wires all application dependencies together.
func InitializeApp(ctx context.Context, configPath string) (*App, error) {
	wire.Build(
		internalconfig.Load,
		provideLogLevel,
		pkglog.New,
		provideDatabaseConfig,
		postgres.NewPool,
		api.NewRouter,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}

// provideLogLevel extracts the log level string from config.
func provideLogLevel(cfg *pkgconfig.Config) string {
	return cfg.Log.Level
}

// provideDatabaseConfig extracts database config from the main config.
func provideDatabaseConfig(cfg *pkgconfig.Config) pkgconfig.DatabaseConfig {
	return cfg.Database
}
