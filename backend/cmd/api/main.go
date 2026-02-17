package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	dockeradapter "github.com/zakari/hopeitworks/backend/internal/adapter/docker"
	hbadapter "github.com/zakari/hopeitworks/backend/internal/adapter/handlebars"
	pgadapter "github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	"github.com/zakari/hopeitworks/backend/internal/api/handler"
	authmw "github.com/zakari/hopeitworks/backend/internal/api/middleware"
	internalconfig "github.com/zakari/hopeitworks/backend/internal/config"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	pkglog "github.com/zakari/hopeitworks/backend/pkg/log"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// Load configuration
	cfg, err := internalconfig.Load("config.yaml")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Initialize structured logger
	logger := pkglog.New(cfg.Log.Level)
	logger.Info("config loaded")

	// Connect to database
	pool, err := pgadapter.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()
	logger.Info("database connected")

	// Build event bus (dedicated connection for LISTEN/NOTIFY)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User, cfg.Database.Password,
		cfg.Database.Host, cfg.Database.Port,
		cfg.Database.Name, cfg.Database.SSLMode,
	)
	eventBus, err := pgadapter.NewEventBus(ctx, dsn, logger)
	if err != nil {
		return fmt.Errorf("creating event bus: %w", err)
	}
	defer func() { _ = eventBus.Close() }()
	logger.Info("event bus connected")

	// Build dependency graph
	queries := pgadapter.New(pool)

	// Auth service and middleware
	userRepo := pgadapter.NewUserRepository(pool)
	jwtSecret := getEnvOrDefault("JWT_SECRET", "dev-secret-key-change-in-production")
	jwtExpiration := 24 * time.Hour
	authService := service.NewAuthService(userRepo, jwtSecret, jwtExpiration)

	// Project repository (shared)
	projectRepo := pgadapter.NewProjectRepo(queries)

	// Pipeline config service
	pipelineConfigRepo := pgadapter.NewPipelineConfigRepo(queries)
	pipelineConfigService := service.NewPipelineConfigService(pipelineConfigRepo)
	pipelineConfigHandler := handler.NewPipelineConfigHandler(pipelineConfigService)

	// Project user service
	projectUserRepo := pgadapter.NewProjectUserRepo(queries)
	projectUserService := service.NewProjectUserService(projectUserRepo, projectRepo, userRepo)

	// Project service (with pipeline config seeding on creation)
	projectService := service.NewProjectService(projectRepo)
	projectService.SetPipelineConfigService(pipelineConfigService)
	projectHandler := handler.NewProjectHandler(projectService, projectUserService)

	// Epic service
	epicRepo := pgadapter.NewEpicRepo(queries)
	epicService := service.NewEpicService(epicRepo)
	epicHandler := handler.NewEpicHandler(epicService)

	// Story service
	storyRepo := pgadapter.NewStoryRepo(queries)
	storyService := service.NewStoryService(storyRepo)
	storyHandler := handler.NewStoryHandler(storyService)

	// Prompt template service
	promptTemplateRepo := pgadapter.NewPromptTemplateRepo(queries)
	promptTemplateService := service.NewPromptTemplateService(promptTemplateRepo)
	promptTemplateHandler := handler.NewPromptTemplateHandler(promptTemplateService)

	// Template rendering service (Handlebars engine for prompt templates)
	handlebarsRenderer := hbadapter.NewRenderer()
	_ = service.NewTemplateService(promptTemplateRepo, handlebarsRenderer, logger)

	// Auth handler
	authHandler := handler.NewAuthHandler(authService, userRepo, false)

	// User service
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	// Run service
	runRepo := pgadapter.NewRunRepo(queries)
	runService := service.NewRunService(runRepo, projectRepo)
	runHandler := handler.NewRunHandler(runService)

	// Container manager (Docker adapter)
	containerMgr, err := dockeradapter.NewDockerContainerManager(cfg.Docker.Host, logger)
	if err != nil {
		logger.Warn("docker container manager unavailable, timeout enforcer and orphan cleaner disabled", "error", err)
	}

	// Orphan cleanup and timeout enforcement (requires Docker)
	appCtx, appCancel := context.WithCancel(ctx)
	defer appCancel()
	if containerMgr != nil {
		orphanCleaner := service.NewOrphanCleaner(containerMgr, runRepo, logger)
		if err := orphanCleaner.CleanupOrphans(appCtx); err != nil {
			logger.Error("orphan cleanup failed on startup", "error", err)
		}

		timeoutEnforcer := service.NewTimeoutEnforcer(
			containerMgr, runRepo, projectRepo, logger,
			30*time.Minute, // default container timeout
			30*time.Second, // check interval
		)
		go func() {
			if err := timeoutEnforcer.Start(appCtx); err != nil && err != context.Canceled {
				logger.Error("timeout enforcer failed", "error", err)
			}
		}()
	}

	server := handler.NewServer(authHandler, projectHandler, userHandler, epicHandler, storyHandler, promptTemplateHandler, runHandler, pipelineConfigHandler)

	// Project user handler
	projectUserHandler := handler.NewProjectUserHandler(projectUserService)

	// Build router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)

	// Auth middleware skips public paths
	r.Use(authmw.Auth(authService))

	// Health check (skipped by auth middleware)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler.HandlerFromMuxWithBaseURL(server, r, "/api/v1")

	// Mount project_users routes (manually registered, not in OpenAPI spec yet)
	r.Route("/api/v1/projects/{id}/users", func(r chi.Router) {
		r.Use(authmw.RequireProjectAccess(projectUserRepo))
		r.Get("/", projectUserHandler.ListMembers)
		r.Post("/", projectUserHandler.AddUser)
		r.Delete("/{user_id}", projectUserHandler.RemoveUser)
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Graceful shutdown
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGTERM, syscall.SIGINT)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdownCh:
		logger.Info("shutting down gracefully", "signal", sig.String())

		// Stop background services (timeout enforcer)
		appCancel()

		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
		}

		if closeErr := eventBus.Close(); closeErr != nil {
			logger.Error("failed to close event bus", "error", closeErr)
		}
		pool.Close()
		logger.Info("server stopped")
	}

	return nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
