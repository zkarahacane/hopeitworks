package main

import (
	"context"
	"fmt"
	"log/slog"
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
	gitadapter "github.com/zakari/hopeitworks/backend/internal/adapter/git"
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
	"github.com/zakari/hopeitworks/backend/migrations"
	pkgexec "github.com/zakari/hopeitworks/backend/pkg/exec"
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

	// Build DSN (shared by auto-migrate and event bus)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User, cfg.Database.Password,
		cfg.Database.Host, cfg.Database.Port,
		cfg.Database.Name, cfg.Database.SSLMode,
	)

	// Auto-migrate: apply pending migrations before any service initialization
	if cfg.Database.AutoMigrate != nil && *cfg.Database.AutoMigrate {
		logger.Info("running database migrations")
		if err := pgadapter.RunMigrations(migrations.FS, dsn, logger); err != nil {
			return fmt.Errorf("running migrations: %w", err)
		}
	} else {
		logger.Info("auto-migration disabled, skipping")
	}

	// Build dependency graph
	queries := pgadapter.New(pool)

	// Event publisher (for persisting events to DB) — created early because
	// EventBus needs it for enriching NOTIFY payloads with full event data.
	eventRepo := pgadapter.NewEventRepo(queries)

	// Build event bus (dedicated connection for LISTEN/NOTIFY).
	// The eventRepo is passed so the bus can enrich NOTIFY payloads with
	// full event data (including payload) from the database, since pg_notify
	// has an 8KB limit and the trigger only sends metadata.
	eventBus, err := pgadapter.NewEventBus(ctx, dsn, eventRepo, logger)
	if err != nil {
		return fmt.Errorf("creating event bus: %w", err)
	}
	defer func() { _ = eventBus.Close() }()
	logger.Info("event bus connected")

	// Auth service and middleware
	userRepo := pgadapter.NewUserRepository(pool)
	blacklistRepo := pgadapter.NewTokenBlacklistRepo(pool)
	passwordResetTokenRepo := pgadapter.NewPasswordResetTokenRepository(pool)
	emailSender := smtpadapter.NewEmailSender(cfg.SMTP)
	jwtSecret := getEnvOrDefault("JWT_SECRET", "dev-secret-key-change-in-production")
	jwtExpiration := 24 * time.Hour
	authService := service.NewAuthService(userRepo, passwordResetTokenRepo, emailSender, cfg.SMTP.FrontendURL, jwtSecret, jwtExpiration)
	authService.SetBlacklistRepo(blacklistRepo)

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

	// Background cleanup of expired revoked tokens
	startTokenCleanup(appCtx, authService, logger)

	// Run repository (shared by run service, pipeline executor, and other components)
	runRepo := pgadapter.NewRunRepo(queries)

	// Cost service
	costRepo := pgadapter.NewCostRepo(queries)
	costService := service.NewCostService(costRepo, projectRepo, storyRepo, runRepo, logger)
	costHandler := handler.NewCostHandler(costService)

	// Container manager (Docker adapter)
	containerMgr, err := dockeradapter.NewDockerContainerManager(cfg.Docker.Host, logger)
	if err != nil {
		logger.Warn("docker container manager unavailable, timeout enforcer and orphan cleaner disabled", "error", err)
	}

	// HITL repository (created early because hitl_gate action needs it)
	hitlRepo := pgadapter.NewHITLRepo(queries)

	// Action registry
	actionReg := service.NewActionRegistry()

	// Git provider (needed by ci_poll and hitl_gate actions)
	cmdRunner := pkgexec.NewRealCommandRunner()
	gitProvider := gitadapter.NewGhCliAdapter(cmdRunner, logger)

	// CI Poll action (no Docker required)
	ciPollCfg := actionadapter.CIPollConfig{
		DefaultPollInterval: 30 * time.Second,
		DefaultTimeout:      15 * time.Minute,
	}
	ciPollAction := actionadapter.NewCIPollAction(gitProvider, eventRepo, ciPollCfg, logger)
	actionReg.Register(ciPollAction)
	logger.Info("ci_poll action registered")

	// HITL Gate action (no Docker required)
	hitlGateAction := actionadapter.NewHITLGateAction(hitlRepo, runRepo, gitProvider, eventRepo, storyRepo, logger)
	actionReg.Register(hitlGateAction)
	logger.Info("hitl_gate action registered")

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

			// Register action_type aliases so pipeline configs using
			// implement/review/merge resolve to AgentRunAction
			for _, alias := range []string{"implement", "review", "merge"} {
				actionReg.RegisterAlias(alias, agentRunAction)
			}
			logger.Info("action aliases registered", "aliases", []string{"implement", "review", "merge"})

			// Incremental retry action (delegates to agent_run)
			incrementalRetryAction := actionadapter.NewIncrementalRetryAction(
				runRepo, templateSvc, agentRunAction, logger,
			)
			actionReg.Register(incrementalRetryAction)
			logger.Info("incremental_retry action registered")
		}
	}

	// Pipeline executor: wired with the real action registry and event publisher
	pipelineExecutor := service.NewPipelineExecutor(runRepo, storyRepo, actionReg, eventRepo, logger)
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
	if containerMgr != nil {
		runService.SetContainerManager(containerMgr)
	}
	runHandler := handler.NewRunHandler(runService)

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
	hitlService := service.NewHITLService(hitlRepo, runRepo, eventRepo, logger)
	hitlHandler := handler.NewHITLHandler(hitlService)

	// Notifier adapters
	notifiers := map[string]port.Notifier{
		model.ChannelTypeDiscord: discordadapter.NewNotifier(),
		model.ChannelTypeWebhook: webhookadapter.NewNotifier(),
	}

	// Notification configs
	notificationConfigRepo := pgadapter.NewNotificationConfigRepository(queries)
	notificationConfigService := service.NewNotificationConfigService(notificationConfigRepo, notifiers)
	notificationHandler := handler.NewNotificationHandler(notificationConfigService)

	// Notification dispatcher (background goroutine)
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
	r.Use(authmw.Auth(authService, blacklistRepo))

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

// startTokenCleanup launches a goroutine that periodically purges expired revoked tokens.
func startTokenCleanup(ctx context.Context, authService *service.AuthService, logger *slog.Logger) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := authService.PurgeExpiredTokens(ctx); err != nil {
					logger.Warn("failed to purge expired revoked tokens", "error", err)
				} else {
					logger.Debug("purged expired revoked tokens")
				}
			}
		}
	}()
}
