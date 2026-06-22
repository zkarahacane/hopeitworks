// Package runner orchestrates the agent runtime lifecycle:
// git clone, CLAUDE.md setup, LLM provider execution, and result callbacks.
package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zakari/hopeitworks/agent-runtime/internal/callback"
	"github.com/zakari/hopeitworks/agent-runtime/internal/config"
	"github.com/zakari/hopeitworks/agent-runtime/internal/git"
	"github.com/zakari/hopeitworks/agent-runtime/internal/provider"
)

// Runner orchestrates the full agent runtime lifecycle.
type Runner struct {
	cfg      *config.Config
	provider provider.Provider
	callback *callback.Client
}

// New creates a new Runner with the given configuration, provider, and callback client.
func New(cfg *config.Config, prov provider.Provider, cb *callback.Client) *Runner {
	return &Runner{
		cfg:      cfg,
		provider: prov,
		callback: cb,
	}
}

// Run executes the full agent runtime pipeline:
// 1. Clone the repository
// 2. Write CLAUDE.md if content is provided
// 3. Execute the LLM provider
// 4. Forward events to the API server via callbacks
// 5. Report final status
func (r *Runner) Run(ctx context.Context) error {
	workDir := "/workspace/repo"

	// 1. Clone repo
	if err := git.Clone(ctx, r.cfg.RepoURL, r.cfg.BranchName, r.cfg.GitToken, r.cfg.GitProvider, workDir); err != nil {
		_ = r.callback.SendStatus(ctx, 1, fmt.Sprintf("git clone failed: %v", err))
		return err
	}

	// 2. Write CLAUDE.md if provided
	if r.cfg.ClaudeMDContent != "" {
		claudeDir := filepath.Join(workDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			_ = r.callback.SendStatus(ctx, 1, fmt.Sprintf("create .claude dir failed: %v", err))
			return err
		}
		if err := os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte(r.cfg.ClaudeMDContent), 0644); err != nil {
			_ = r.callback.SendStatus(ctx, 1, fmt.Sprintf("write CLAUDE.md failed: %v", err))
			return err
		}
	}

	// 2b. Provision capabilities (skills / MCP / tool policy / secrets) from the bundle.
	// Fail-soft and a no-op for capability-less agents — see provisionCapabilities.
	runOpts := r.provisionCapabilities(ctx, workDir)

	// 3. Run provider
	events, err := r.provider.Run(ctx, workDir, r.cfg.Prompt, r.cfg.Model, runOpts)
	if err != nil {
		_ = r.callback.SendStatus(ctx, 1, fmt.Sprintf("provider start failed: %v", err))
		return err
	}

	// 4. Process events and forward to callbacks
	var lastResult provider.Event
	for event := range events {
		switch event.Type {
		case "log":
			_ = r.callback.SendLog(ctx, event.Message)
		case "cost":
			_ = r.callback.SendCost(ctx, event.InputTokens, event.OutputTokens, event.Model, event.CostUSD)
		case "result":
			lastResult = event
		}
	}

	// 4b. Commit and push the agent's changes when the provider succeeded.
	// No-op when the working tree is clean (e.g. review-only roles). This is the
	// step that turns claude's file edits into a real commit on the run's branch.
	if lastResult.ExitCode == 0 {
		msg := "feat: agent implementation"
		if r.cfg.StoryKey != "" {
			msg = fmt.Sprintf("feat(%s): agent implementation", r.cfg.StoryKey)
		}
		if err := git.CommitAndPush(ctx, workDir, r.cfg.BranchName, msg); err != nil {
			_ = r.callback.SendLog(ctx, fmt.Sprintf("commit/push failed: %v", err))
			_ = r.callback.SendStatus(ctx, 1, fmt.Sprintf("commit/push failed: %v", err))
			return err
		}
	}

	// 5. Send final status
	errMsg := ""
	if lastResult.ExitCode != 0 {
		errMsg = lastResult.Message
	}
	if err := r.callback.SendStatus(ctx, lastResult.ExitCode, errMsg); err != nil {
		return fmt.Errorf("send final status: %w", err)
	}

	return nil
}
