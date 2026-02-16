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
	"github.com/zakari/hopeitworks/backend/internal/adapter/postgres"
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
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()
	logger.Info("database connected")

	// Build dependency graph
	queries := pgadapter.New(pool)

	// Auth service and middleware
	userRepo := pgadapter.NewUserRepository(pool)
	jwtSecret := getEnvOrDefault("JWT_SECRET", "dev-secret-key-change-in-production")
	jwtExpiration := 24 * time.Hour
	authService := service.NewAuthService(userRepo, jwtSecret, jwtExpiration)

	// Project repository (shared)
	projectRepo := pgadapter.NewProjectRepo(queries)

	// Project user service
	projectUserRepo := pgadapter.NewProjectUserRepo(queries)
	projectUserService := service.NewProjectUserService(projectUserRepo, projectRepo, userRepo)

	// Project service
	projectService := service.NewProjectService(projectRepo)
	projectHandler := handler.NewProjectHandler(projectService, projectUserService)

	// User service
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	server := handler.NewServer(projectHandler, userHandler)

	// Project user handler
	projectUserHandler := handler.NewProjectUserHandler(projectUserService)

	// Build router
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(authmw.Auth(authService))

	handler.HandlerFromMuxWithBaseURL(server, r, "/api/v1")

	// Mount project_users routes (manually registered, not in OpenAPI spec yet)
	r.Route("/api/v1/projects/{id}/users", func(r chi.Router) {
		r.Use(authmw.RequireProjectAccess(projectUserRepo))
		r.Get("/", projectUserHandler.ListMembers)
		r.Post("/", projectUserHandler.AddUser)
		r.Delete("/{user_id}", projectUserHandler.RemoveUser)
	})

	// Health check
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
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

		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
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
