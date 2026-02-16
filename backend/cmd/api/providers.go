package main

import (
	"github.com/google/wire"

	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/api"
	internalconfig "github.com/zakari/hopeitworks/backend/internal/config"
	pkglog "github.com/zakari/hopeitworks/backend/pkg/log"
)

// ConfigSet provides config loading.
var ConfigSet = wire.NewSet(internalconfig.Load)

// LogSet provides structured logger creation.
var LogSet = wire.NewSet(pkglog.New)

// PostgresSet provides pgx pool creation.
var PostgresSet = wire.NewSet(postgres.NewPool)

// RouterSet provides chi router creation.
var RouterSet = wire.NewSet(api.NewRouter)
