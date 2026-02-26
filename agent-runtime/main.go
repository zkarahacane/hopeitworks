// Package main is the entry point for the agent runtime binary.
// It reads configuration from environment variables, initializes the LLM provider
// and callback client, then runs the full agent pipeline.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/zakari/hopeitworks/agent-runtime/internal/callback"
	"github.com/zakari/hopeitworks/agent-runtime/internal/config"
	"github.com/zakari/hopeitworks/agent-runtime/internal/provider"
	"github.com/zakari/hopeitworks/agent-runtime/internal/runner"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	prov, err := provider.New(cfg.Provider, cfg.APIKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "provider error: %v\n", err)
		os.Exit(1)
	}

	cb := callback.New(cfg.CallbackURL, cfg.AuthToken, cfg.RunID, cfg.StepID)

	r := runner.New(cfg, prov, cb)
	if err := r.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "run error: %v\n", err)
		os.Exit(1)
	}
}
