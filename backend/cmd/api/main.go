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
	"github.com/riverqueue/river"
	actionadapter "github.com/zakari/hopeitworks/backend/internal/adapter/action"
	discordadapter "github.com/zakari/hopeitworks/backend/internal/adapter/discord"
	dockeradapter "github.com/zakari/hopeitworks/backend/internal/adapter/docker"
	hbadapter "github.com/zakari/hopeitworks/backend/internal/adapter/handlebars"
	pgadapter "github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
	riveradapter "github.com/zakari/hopeitworks/backend/internal/adapter/river"
	smtpadapter "github.com/zakari/hopeitworks/backend/internal/adapter/smtp"
	webhookadapter "github.com/zakari/hopeitworks/backend/internal/adapter/webhook"
	"github.com/zakari/hopeitworks/backend/internal/api/handler"
	authmw "github.com/zakari/hopeitworks/backend/internal/api/middleware"
	internalconfig "github.com/zakari/hopeitworks/backend/internal/config"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
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
	passwordResetTokenRepo := pgadapter.NewPasswordResetTokenRepository(pool)
	emailSender := smtpadapter.NewEmailSender(cfg.SMTP)
	jwtSecret := getEnvOrDefault("JWT_SECRET", "dev-secret-key-change-in-production")
	jwtExpiration := 24 * time.Hour
	authService := service.NewAuthService(userRepo, passwordResetTokenRepo, emailSender, cfg.SMTP.FrontendURL, jwtSecret, jwtExpiration)

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

	// Event publisher (for persisting events to DB) — needed by circuit breaker
	eventRepo := pgadapter.NewEventRepo(queries)

	// Circuit breaker service
	circuitBreakerService := service.NewCircuitBreakerService(projectRepo, eventRepo, logger)

	projectHandler := handler.NewProjectHandler(projectService, projectUserService, circuitBreakerService)

	// Epic service
	epicRepo := pgadapter.NewEpicRepo(queries)
	epicService := service.NewEpicService(epicRepo)

	// Story service
	storyRepo := pgadapter.NewStoryRepo(queries)
	storyService := service.NewStoryService(storyRepo)
	storyHandler := handler.NewStoryHandler(storyService)

	// Scheduler service (DAG computation, pure domain service)
	schedulerService := service.NewSchedulerService()
	epicHandler := handler.NewEpicHandler(epicService, schedulerService, storyRepo)

	// Prompt template service
	promptTemplateRepo := pgadapter.NewPromptTemplateRepo(queries)
	promptTemplateService := service.NewPromptTemplateService(promptTemplateRepo)
	promptTemplateHandler := handler.NewPromptTemplateHandler(promptTemplateService)

	// Template rendering service (Handlebars engine for prompt templates)
	handlebarsRenderer := hbadapter.NewRenderer()
	templateSvc := service.NewTemplateService(promptTemplateRepo, handlebarsRenderer, logger)

	// Auth handler
	authHandler := handler.NewAuthHandler(authService, userRepo, false)

	// User service
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)
	profileHandler := handler.NewProfileHandler(userService)

	// Application-wide context for background services
	appCtx, appCancel := context.WithCancel(ctx)
	defer appCancel()

	// Run service and job queue
	runRepo := pgadapter.NewRunRepo(queries)

	// Pipeline executor (will be used by River workers)
	// NOTE: event publisher and action registry wiring deferred to later story
	pipelineExecutor := service.NewPipelineExecutor(runRepo, nil, nil, logger)
	pipelineExecutor.SetCircuitBreaker(circuitBreakerService)

	// River job queue for async pipeline execution
	workers := river.NewWorkers()
	river.AddWorker(workers, riveradapter.NewExecuteRunWorker(pipelineExecutor))

	jobQueue, err := riveradapter.NewJobQueue(pool, workers)
	if err != nil {
		logger.Warn("river job queue unavailable, run launching disabled", "error", err)
	}
	if jobQueue != nil {
		go func() {
			if startErr := jobQueue.Client().Start(appCtx); startErr != nil && startErr != context.Canceled {
				logger.Error("river client failed", "error", startErr)
			}
		}()
	}

	runService := service.NewRunService(runRepo, projectRepo, storyRepo, pipelineConfigRepo, jobQueue, eventRepo)
	runHandler := handler.NewRunHandler(runService)

	// Cost service
	costRepo := pgadapter.NewCostRepo(queries)
	costService := service.NewCostService(costRepo, projectRepo, storyRepo, runRepo, logger)
	costHandler := handler.NewCostHandler(costService)

	// Container manager (Docker adapter)
	containerMgr, err := dockeradapter.NewDockerContainerManager(cfg.Docker.Host, logger)
	if err != nil {
		logger.Warn("docker container manager unavailable, timeout enforcer and orphan cleaner disabled", "error", err)
	}

	// Action registry
	actionReg := service.NewActionRegistry()

	// Cost tracking (for agent_run action)
	costSvc := costService

	// Agent run action (requires Docker)
	if containerMgr != nil {
		logStreamer, logErr := dockeradapter.NewDockerLogStreamerFromHost(cfg.Docker.Host, logger)
		if logErr != nil {
			logger.Warn("log streamer unavailable, agent_run action disabled", "error", logErr)
		} else {
			agentCfg := actionadapter.AgentConfig{
				DefaultImage:  getEnvOrDefault("AGENT_IMAGE", "hopeitworks/agent:latest"),
				DefaultMemory: 4294967296, // 4GB
				DefaultCPUs:   2.0,
				NetworkName:   cfg.Docker.AgentNetwork,
				LogTailLines:  50,
				ClaudeMDPath:  getEnvOrDefault("CLAUDE_MD_PATH", "agent/claude-md"),
			}
			agentRunAction := actionadapter.NewAgentRunAction(
				containerMgr, logStreamer, eventRepo,
				storyRepo, projectRepo, runRepo,
				templateSvc, costSvc, agentCfg, logger,
			)
			actionReg.Register(agentRunAction)
			logger.Info("agent_run action registered")

			// Incremental retry action (delegates to agent_run)
			incrementalRetryAction := actionadapter.NewIncrementalRetryAction(
				runRepo, templateSvc, agentRunAction, logger,
			)
			actionReg.Register(incrementalRetryAction)
			logger.Info("incremental_retry action registered")
		}
	}

	// Orphan cleanup and timeout enforcement (requires Docker)
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

	// HITL service and handler
	hitlRepo := pgadapter.NewHITLRepo(queries)
	hitlService := service.NewHITLService(hitlRepo, runRepo, eventRepo, logger)
	hitlHandler := handler.NewHITLHandler(hitlService)

	// Notification configs
	notificationConfigRepo := pgadapter.NewNotificationConfigRepository(queries)
	notificationConfigService := service.NewNotificationConfigService(notificationConfigRepo)
	notificationHandler := handler.NewNotificationHandler(notificationConfigService)

	// Notification dispatcher (background goroutine)
	notifiers := map[string]port.Notifier{
		model.ChannelTypeDiscord: discordadapter.NewNotifier(),
		model.ChannelTypeWebhook: webhookadapter.NewNotifier(),
	}
	notificationDispatcher := service.NewNotificationDispatcher(eventBus, notificationConfigRepo, projectRepo, notifiers)
	notificationDispatcher.Start(appCtx)
	logger.Info("notification dispatcher started")

	// SSE handler for real-time event streaming
	sseHandler := handler.NewSSEHandler(eventBus, eventRepo, projectUserRepo, logger)

	// Epic run orchestration
	epicRunRepo := pgadapter.NewEpicRunRepo(queries)
	parallelGroupExecutor := service.NewParallelGroupExecutor(epicRunRepo, runService, pipelineExecutor, eventRepo, logger)
	epicRunService := service.NewEpicRunService(epicRunRepo, storyRepo, epicRepo, schedulerService, parallelGroupExecutor, eventRepo, logger)
	epicRunHandler := handler.NewEpicRunHandler(epicRunService)

	server := handler.NewServer(authHandler, projectHandler, userHandler, profileHandler, epicHandler, storyHandler, promptTemplateHandler, runHandler, pipelineConfigHandler, hitlHandler, costHandler, notificationHandler, epicRunHandler)

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

	// SSE endpoint for real-time event streaming
	r.Get("/api/v1/events/stream", sseHandler.ServeHTTP)

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
