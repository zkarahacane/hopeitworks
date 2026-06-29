package main

import (
	"context"
	"encoding/json"
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
	memoryadapter "github.com/zakari/hopeitworks/backend/internal/adapter/memory"
	microsandboxadapter "github.com/zakari/hopeitworks/backend/internal/adapter/microsandbox"
	planningadapter "github.com/zakari/hopeitworks/backend/internal/adapter/planning"
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
	pkgconfig "github.com/zakari/hopeitworks/backend/pkg/config"
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

	// Story repository (shared by story service and the project handler, which
	// uses it to enrich each listed project with its story_count via the same
	// CountByProject query that backs the project detail's stories total).
	storyRepo := pgadapter.NewStoryRepo(queries)

	projectHandler := handler.NewProjectHandler(projectService, projectUserService, circuitBreakerService, storyRepo)

	// Run repository (shared by run service, pipeline executor, story/epic handlers,
	// and other components).
	runRepo := pgadapter.NewRunRepo(queries)

	// Epic service
	epicRepo := pgadapter.NewEpicRepo(queries)
	epicService := service.NewEpicService(epicRepo)

	// Encryption master key — shared by user API keys, credentials, and git PATs
	// (derived once via crypto.DeriveKey inside each service).
	encryptionKey := getEnvOrDefault("ENCRYPTION_KEY", cfg.Security.EncryptionKey)

	// Git connection (Phase 1): encrypted PAT per project behind the credential seam.
	// The service is BOTH the management API (Status/Set/Test/Clear) AND the resolver
	// that both factories consume; it self-heals advisory status on real 401/403 (C1).
	// The resolved PAT is consumed only by server-side git adapters — never injected
	// into an agent container (A4 invariant).
	gitConnRepo := pgadapter.NewGitConnectionRepository(queries)
	gitConnValidator := gitadapter.NewGitHubConnectionValidator(getEnvOrDefault("GITHUB_API_BASE_URL", ""), logger)
	gitConnSvc := service.NewGitConnectionService(gitConnRepo, projectRepo, gitConnValidator, encryptionKey, eventRepo, logger)

	// Planning import (one-way connector): a source factory resolves the adapter
	// per kind (markdown live; github_projects resolves the PAT via the credential
	// seam), and the service owns every upsert decision. Backs both POST
	// /planning/import and the legacy /stories/import shim below.
	planningFactory := planningadapter.NewFactory(projectRepo, gitConnSvc, logger)
	planningImportService := service.NewPlanningImportService(storyRepo, epicRepo, planningFactory)

	// Planning write-back (one-way outbound: hopeitworks -> tracker). The connector
	// service persists the per-project config (status field + mapping + toggles) and
	// serves the live status-options probe; the write-back service is the River worker
	// side that pushes a single status. Both reuse planningFactory as the sink factory
	// (token via the same credential seam) and gitConnSvc as the resolver.
	planningConnectorRepo := pgadapter.NewPlanningConnectorRepository(queries)
	planningWriteBackRepo := pgadapter.NewPlanningWriteBackRepository(queries)
	planningConnectorService := service.NewPlanningConnectorService(planningConnectorRepo, projectRepo, gitConnSvc, planningFactory)
	planningWriteBackService := service.NewPlanningWriteBackService(
		planningConnectorRepo, storyRepo, planningWriteBackRepo, planningFactory,
		getEnvOrDefault("PUBLIC_BASE_URL", ""), logger,
	)
	planningConnectorHandler := handler.NewPlanningConnectorHandler(planningConnectorService)

	// Story service
	storyService := service.NewStoryService(storyRepo)
	storyHandler := handler.NewStoryHandler(storyService, runRepo, planningImportService)

	// Scheduler service (DAG computation, pure domain service)
	schedulerService := service.NewSchedulerService()
	epicHandler := handler.NewEpicHandler(epicService, schedulerService, storyRepo, runRepo)

	// Agent repository and service (first-class Agent entity, replaces prompt_templates)
	agentRepo := pgadapter.NewAgentRepo(queries)
	agentService := service.NewAgentService(agentRepo)
	agentHandler := handler.NewAgentHandler(agentService)

	// Stack catalogue (P2a): read-only catalogue of pinned runtime images. RunService
	// uses the repo to resolve an agent's effective launch image when it references a
	// stack; the service + handler back the read API for the agent editor.
	stackRepo := pgadapter.NewStackRepo(queries)
	stackService := service.NewStackService(stackRepo)
	stackHandler := handler.NewStackHandler(stackService)

	// Seed the stack catalogue from versioned config (source of truth). Idempotent
	// UPSERT on every boot, so image digests can change without a migration. An empty
	// catalogue is a no-op fallback to the migration-inlined seed (000034); a seed
	// error is logged but does not abort boot (the read path still works on existing rows).
	if err := service.SeedStacks(ctx, stackRepo, stackCatalogueFromConfig(cfg.Stacks, logger), logger); err != nil {
		logger.Error("failed to seed stack catalogue from config", "error", err)
	}

	// Environment persistence (P2c1): one execution composition per project (stacks +
	// sidecar services + config source + commands). Wired into the agent_run path below
	// (P2c2c): when a project has an Environment with sidecar services, they are brought
	// up per-run and their connection strings injected into the agent container.
	environmentRepo := pgadapter.NewEnvironmentRepo(queries)
	environmentService := service.NewEnvironmentService(environmentRepo)
	environmentHandler := handler.NewEnvironmentHandler(environmentService)

	// Git connection handler (Phase 1): owner-or-admin gated PAT management.
	gitConnectionHandler := handler.NewGitConnectionHandler(gitConnSvc)

	// Template renderer (Handlebars engine for prompt templates)
	handlebarsRenderer := hbadapter.NewRenderer()

	// Auth handler
	authHandler := handler.NewAuthHandler(authService, userRepo, false)

	// User service
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)
	profileHandler := handler.NewProfileHandler(userService)

	// API Key service (encrypted API key storage for users). encryptionKey is the
	// shared master key resolved above.
	apiKeyRepo := pgadapter.NewAPIKeyRepository(queries)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo, encryptionKey)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeySvc)

	// Capabilities + credentials + fetch-at-startup bundle (P1 runtime/capabilities rework).
	// Credentials are encrypted at rest with the same AES-256 key as user API keys. The
	// bundle endpoint is served on the internal container-token channel (see below).
	capabilityRepo := pgadapter.NewCapabilityRepository(queries)
	credentialRepo := pgadapter.NewCredentialRepository(queries)
	credentialSvc := service.NewCredentialService(credentialRepo, encryptionKey)
	bundleSvc := service.NewBundleService(capabilityRepo, credentialSvc, agentRepo, logger)
	agentBundleHandler := handler.NewAgentBundleHandler(bundleSvc, logger)

	// Application-wide context for background services
	appCtx, appCancel := context.WithCancel(ctx)
	defer appCancel()

	// Background cleanup of expired revoked tokens
	startTokenCleanup(appCtx, authService, logger)

	// Cost service
	costRepo := pgadapter.NewCostRepo(queries)
	costService := service.NewCostService(costRepo, projectRepo, storyRepo, runRepo, logger)
	costHandler := handler.NewCostHandler(costService)

	// Container manager (Docker adapter)
	containerMgr, err := dockeradapter.NewDockerContainerManager(cfg.Docker.Host, logger)
	if err != nil {
		logger.Warn("docker container manager unavailable, timeout enforcer and orphan cleaner disabled", "error", err)
	}

	// Substrate selection. selectSubstrate returns the AgentRuntime adapter for the
	// configured SUBSTRATE: the alternative adapter (microsandbox) when selected,
	// otherwise the Docker adapter (docker.Runtime) by default — so the live
	// agent_run flow ALWAYS dispatches THROUGH port.AgentRuntime; Docker is an
	// adapter behind the port, not a special path. It needs the Docker container
	// manager (sidecars, logs), so it is only built when one is available; with no
	// container manager the agent_run/runService wiring below is disabled anyway.
	var agentRuntime port.AgentRuntime
	if containerMgr != nil {
		agentRuntime = selectSubstrate(cfg.Substrate.Kind, stackRepo, containerMgr, cfg.Docker.AgentNetwork, cfg.Docker.IsolateRuns, logger)
	} else {
		logger.Warn("substrate selection skipped: no container manager available, agent runs disabled", "substrate", cfg.Substrate.Kind)
	}

	// HITL repository (created early because hitl_gate action needs it)
	hitlRepo := pgadapter.NewHITLRepo(queries)

	// Action registry
	actionReg := service.NewActionRegistry()

	// Git provider factory (resolves GitProvider per project: github or gitea)
	cmdRunner := pkgexec.NewRealCommandRunner()
	gitProviderFactory := gitadapter.NewGitProviderFactory(projectRepo, gitConnSvc, cmdRunner, logger)

	// CI Poll action (no Docker required)
	ciPollCfg := actionadapter.CIPollConfig{
		DefaultPollInterval: 30 * time.Second,
		DefaultTimeout:      15 * time.Minute,
	}
	ciPollAction := actionadapter.NewCIPollAction(gitProviderFactory, eventRepo, ciPollCfg, logger)
	actionReg.Register(ciPollAction)
	logger.Info("ci_poll action registered")

	// HITL Gate action (no Docker required)
	hitlGateAction := actionadapter.NewHITLGateAction(hitlRepo, runRepo, gitProviderFactory, eventRepo, storyRepo, logger)
	actionReg.Register(hitlGateAction)
	logger.Info("hitl_gate action registered")

	// Git Branch action (no Docker required)
	gitBranchAction := actionadapter.NewGitBranchAction(gitProviderFactory, storyRepo, projectRepo, logger)
	actionReg.Register(gitBranchAction)
	logger.Info("git_branch action registered")

	// Git PR action (no Docker required)
	gitPRAction := actionadapter.NewGitPRAction(gitProviderFactory, storyRepo, projectRepo, logger)
	actionReg.Register(gitPRAction)
	logger.Info("git_pr action registered")

	// Human action (no Docker required)
	humanAction := actionadapter.NewHumanAction(hitlRepo, runRepo, storyRepo, eventRepo, logger)
	actionReg.Register(humanAction)
	logger.Info("human action registered")

	// Notification action (no Docker required)
	notificationAction := actionadapter.NewNotificationAction(eventRepo, storyRepo, logger)
	actionReg.Register(notificationAction)
	logger.Info("notification action registered")

	// Cost tracking (for agent_run action)
	costSvc := costService

	// In-memory stores for agent container callback mode
	containerTokenStore := memoryadapter.NewContainerTokenStore(appCtx)
	callbackStatusStore := memoryadapter.NewCallbackStatusStore()

	// Sidecar manager: brings up an Environment's services on a per-run isolated
	// network over the same Docker ContainerManager. Hoisted here (instead of the
	// agent_run block below) so the periodic sidecar GC can also reference it.
	var sidecarMgr port.SidecarManager

	// Agent run action (requires Docker)
	if containerMgr != nil {
		logStreamer, logErr := dockeradapter.NewDockerLogStreamerFromHost(cfg.Docker.Host, logger)
		if logErr != nil {
			logger.Warn("log streamer unavailable, agent_run action disabled", "error", logErr)
		} else {
			agentCfg := actionadapter.AgentConfig{
				DefaultMemory: 4294967296, // 4GB
				DefaultCPUs:   2.0,
				NetworkName:   cfg.Docker.AgentNetwork,
				IsolateRuns:   cfg.Docker.IsolateRuns,
				LogTailLines:  50,
			}
			sidecarMgr = dockeradapter.NewDockerSidecarManagerWithIsolation(
				containerMgr, cfg.Docker.IsolateRuns, cfg.Docker.APIContainerName, logger)
			if cfg.Docker.IsolateRuns {
				logger.Info("East-West run isolation enabled: agents are single-homed on their per-run network; API attached per-run",
					"api_container", cfg.Docker.APIContainerName)
			}

			agentRunAction := actionadapter.NewAgentRunAction(
				containerMgr, logStreamer, eventRepo,
				storyRepo, projectRepo, runRepo,
				environmentRepo, sidecarMgr, stackRepo,
				handlebarsRenderer, costSvc, agentCfg, logger,
				apiKeySvc, containerTokenStore, callbackStatusStore,
				cfg.Docker.CallbackBaseURL,
				actionadapter.WithAgentRuntime(agentRuntime),
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
				runRepo, agentRunAction, logger,
			)
			actionReg.Register(incrementalRetryAction)
			logger.Info("incremental_retry action registered")
		}
	}

	// Pipeline executor: wired with the real action registry and event publisher
	pipelineExecutor := service.NewPipelineExecutor(runRepo, storyRepo, actionReg, eventRepo, logger)
	pipelineExecutor.SetCircuitBreaker(circuitBreakerService)
	// HITL repo enables gate-transition enforcement at stage boundaries (INC 3).
	pipelineExecutor.SetHITLRepo(hitlRepo)

	// River job queue for async pipeline execution
	workers := river.NewWorkers()
	river.AddWorker(workers, riveradapter.NewExecuteRunWorker(pipelineExecutor))
	// Async tracker status write-back worker (one-way outbound). Registered before the
	// client is created (River requires all workers up front).
	river.AddWorker(workers, riveradapter.NewWriteBackWorker(planningWriteBackService))

	jobQueue, err := riveradapter.NewJobQueue(pool, workers)
	if err != nil {
		logger.Warn("river job queue unavailable, run launching disabled", "error", err)
	}
	if jobQueue != nil {
		// The executor enqueues a write-back on each running/done/failed transition.
		pipelineExecutor.SetWriteBackEnqueuer(jobQueue)
		go func() {
			if startErr := jobQueue.Client().Start(appCtx); startErr != nil && startErr != context.Canceled {
				logger.Error("river client failed", "error", startErr)
			}
		}()
	}

	runService := service.NewRunService(runRepo, projectRepo, storyRepo, pipelineConfigRepo, jobQueue, eventRepo)
	if containerMgr != nil {
		runService.SetContainerManager(containerMgr)
		runService.SetAgentRuntime(agentRuntime)
	}
	runService.SetAgentRepo(agentRepo)
	runService.SetStackRepo(stackRepo)
	runHandler := handler.NewRunHandler(runService)

	// Orphan cleanup and timeout enforcement (requires Docker)
	if containerMgr != nil {
		orphanCleaner := service.NewOrphanCleaner(containerMgr, runRepo, logger)
		if err := orphanCleaner.CleanupOrphans(appCtx); err != nil {
			logger.Error("orphan cleanup failed on startup", "error", err)
		}

		// Reconcile DB run statuses against live containers: runs left `running`
		// by a previous crash (container gone) are marked failed so they leave
		// "Active runs". Runs once at boot, then on every watchdog tick below.
		orphanReconciler := service.NewOrphanReconciler(
			containerMgr, runRepo, logger, service.DefaultOrphanGraceWindow,
		)
		if err := orphanReconciler.ReconcileOrphanedRuns(appCtx); err != nil {
			logger.Error("orphan run reconciliation failed on startup", "error", err)
		}

		timeoutEnforcer := service.NewTimeoutEnforcer(
			containerMgr, runRepo, projectRepo, logger,
			30*time.Minute, // default container timeout
			30*time.Second, // check interval
			orphanReconciler,
		)
		go func() {
			if err := timeoutEnforcer.Start(appCtx); err != nil && err != context.Canceled {
				logger.Error("timeout enforcer failed", "error", err)
			}
		}()
	}

	// Periodic sidecar GC: best-effort safety net that reaps orphan per-run
	// networks (and their sidecars) leaked when the process dies abruptly between
	// Launch and the agent_run defer Cleanup (SIGKILL/OOM). The normal teardown is
	// still the defer; this only catches crashes. Wide interval/window so it can
	// never touch a run that is still starting up (GC already skips networks with a
	// running sidecar; the window is a second margin against the create-then-start
	// race). Only started when the sidecar manager is wired (Docker available).
	if sidecarMgr != nil {
		sidecarGC := service.NewSidecarGC(
			sidecarMgr, logger,
			service.DefaultSidecarGCInterval,
			service.DefaultSidecarGCWindow,
		)
		go func() {
			if err := sidecarGC.Start(appCtx); err != nil && err != context.Canceled {
				logger.Error("sidecar gc failed", "error", err)
			}
		}()
	}

	// HITL service and handler
	hitlService := service.NewHITLService(hitlRepo, runRepo, jobQueue, eventRepo, logger)
	hitlHandler := handler.NewHITLHandler(hitlService)

	// Guard watchdog (INC 4a): out-of-band ticker that scans running steps and
	// evaluates the board-side probes (log_silence/wallclock/cost_batch). On a
	// halt-gate breach it raises a probe_halt HITL via the HITL service. Runs
	// independently of the executor (no Docker dependency) so it always supervises.
	watchdogRepo := pgadapter.NewWatchdogRepo(queries)
	watchdog := service.NewWatchdog(
		watchdogRepo, pipelineConfigRepo, costRepo, runRepo, hitlService, logger,
		30*time.Second, // scan interval
	)
	go func() {
		if err := watchdog.Start(appCtx); err != nil && err != context.Canceled {
			logger.Error("guard watchdog failed", "error", err)
		}
	}()

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

	// Planning import handler (POST /projects/{projectId}/planning/import).
	planningHandler := handler.NewPlanningHandler(planningImportService)

	server := handler.NewServer(authHandler, projectHandler, userHandler, profileHandler, epicHandler, storyHandler, agentHandler, stackHandler, runHandler, pipelineConfigHandler, hitlHandler, costHandler, notificationHandler, epicRunHandler, environmentHandler, apiKeyHandler, planningHandler, planningConnectorHandler, gitConnectionHandler)

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

	// Mount internal callback routes for agent containers (container token auth, NOT JWT).
	// These routes are excluded from JWT auth via the isPublicPath check in auth middleware.
	agentCallbackHandler := handler.NewAgentCallbackHandler(eventRepo, costSvc, callbackStatusStore, runRepo)
	r.Route("/internal/agent/callback", func(r chi.Router) {
		r.Use(authmw.InternalAuth(containerTokenStore))
		r.Post("/runs/{runId}/steps/{stepId}/logs", agentCallbackHandler.HandleLogs)
		r.Post("/runs/{runId}/steps/{stepId}/cost", agentCallbackHandler.HandleCost)
		r.Post("/runs/{runId}/steps/{stepId}/status", agentCallbackHandler.HandleStatus)
		// Fetch-at-startup capability bundle; the agent is resolved from the token.
		r.Get("/bundle", agentBundleHandler.HandleBundle)
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

// stackCatalogueFromConfig converts the YAML stack catalogue into domain stacks,
// marshalling each toolchain map into the jsonb bytes the model carries. Entries
// with an empty key or image_ref are skipped (logged); a toolchain that fails to
// marshal is skipped rather than aborting boot. Returns nil for an empty catalogue.
func stackCatalogueFromConfig(entries []pkgconfig.StackConfig, logger *slog.Logger) []model.Stack {
	if len(entries) == 0 {
		return nil
	}
	out := make([]model.Stack, 0, len(entries))
	for _, e := range entries {
		if e.Key == "" || e.ImageRef == "" {
			logger.Warn("skipping stack catalogue entry with empty key or image_ref", "key", e.Key, "image_ref", e.ImageRef)
			continue
		}
		toolchain := []byte("{}")
		if len(e.Toolchain) > 0 {
			b, err := json.Marshal(e.Toolchain)
			if err != nil {
				logger.Warn("skipping stack catalogue entry with invalid toolchain", "key", e.Key, "error", err)
				continue
			}
			toolchain = b
		}
		out = append(out, model.Stack{Key: e.Key, ImageRef: e.ImageRef, Toolchain: toolchain})
	}
	return out
}

// selectSubstrate logs the configured execution substrate and constructs its
// AgentRuntime adapter. As of Stage 2 of the substrate-abstraction migration the
// returned runtime IS wired into the live agent_run flow: the caller injects it
// via WithAgentRuntime and Execute dispatches through port.AgentRuntime. Returns
// the alternative adapter (microsandbox) when SUBSTRATE selects one, otherwise the
// Docker adapter (docker.Runtime) by default — so Docker is an adapter behind the
// port, not a special path. The caller invokes it only when containerMgr is
// non-nil (guaranteed by the concrete *ContainerManager nil check at the call
// site, which avoids the typed-nil-interface trap).
//
// The microsandbox adapter's live half (Launch/Wait/Stop) is real only in a
// binary built with `-tags microsandbox` on a KVM/HVF host (P3b); the default
// build returns microsandbox.ErrNotBuilt, so selecting microsandbox without that
// build fails the run clearly rather than silently falling back to Docker.
// enabled=true lets the tagged build launch microVMs; the fallback build ignores it.
func selectSubstrate(kind string, stacks port.StackRepository, containerMgr port.ContainerManager, networkName string, isolateRuns bool, logger *slog.Logger) port.AgentRuntime {
	switch kind {
	case pkgconfig.SubstrateMicrosandbox:
		logger.Info("substrate selected", "substrate", pkgconfig.SubstrateMicrosandbox)
		rt := microsandboxadapter.NewRuntime(true, stacks, logger)
		logger.Warn("microsandbox runtime is wired into live agent_run; live microVM ops require a binary built with -tags microsandbox on a KVM host, else Launch returns ErrNotBuilt and the run fails clearly",
			"substrate", pkgconfig.SubstrateMicrosandbox)
		return rt
	default:
		logger.Info("substrate selected", "substrate", pkgconfig.SubstrateDocker)
		// isolateRuns makes the Docker adapter single-home the agent on its per-run
		// network (East-West isolation) instead of dual-homing on the shared network.
		return dockeradapter.NewRuntimeWithIsolation(containerMgr, networkName, isolateRuns, logger)
	}
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
