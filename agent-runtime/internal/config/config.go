// Package config parses environment variables into a typed configuration struct
// for the agent runtime binary.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all configuration for the agent runtime.
type Config struct {
	Prompt          string // PROMPT — the task prompt
	RepoURL         string // REPO_URL — git clone URL
	BranchName      string // BRANCH_NAME — branch to checkout/create
	CallbackURL     string // CALLBACK_URL — base URL for HTTP callbacks (e.g. http://api:8080)
	AuthToken       string // AUTH_TOKEN — bearer token for callback auth
	RunID           string // RUN_ID
	StepID          string // STEP_ID
	Provider        string // PROVIDER — "claude" or "opencode"
	Model           string // MODEL — e.g. "claude-sonnet-4-6", "gpt-4o"
	APIKey          string // API_KEY — the decrypted API key for the LLM provider
	GitToken        string // GIT_TOKEN — for git clone auth
	GitProvider     string // GIT_PROVIDER — "github" or "gitea"
	StoryKey        string // STORY_KEY — e.g. "S-03"
	ClaudeMDContent string // CLAUDE_MD_CONTENT — optional, written to .claude/CLAUDE.md in workspace
}

// requiredEnvVars lists the environment variables that must be set.
var requiredEnvVars = []string{
	"PROMPT",
	"REPO_URL",
	"BRANCH_NAME",
	"CALLBACK_URL",
	"AUTH_TOKEN",
	"RUN_ID",
	"STEP_ID",
	"PROVIDER",
	"MODEL",
	"API_KEY",
	"GIT_TOKEN",
}

// Load reads environment variables and returns a validated Config.
// Required: PROMPT, REPO_URL, BRANCH_NAME, CALLBACK_URL, AUTH_TOKEN, RUN_ID, STEP_ID, PROVIDER, MODEL, API_KEY, GIT_TOKEN.
// Optional: CLAUDE_MD_CONTENT, STORY_KEY, GIT_PROVIDER (default "github").
func Load() (*Config, error) {
	var missing []string
	for _, key := range requiredEnvVars {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	providerVal := os.Getenv("PROVIDER")
	if providerVal != "claude" && providerVal != "opencode" {
		return nil, fmt.Errorf("invalid PROVIDER %q: must be \"claude\" or \"opencode\"", providerVal)
	}

	gitProvider := os.Getenv("GIT_PROVIDER")
	if gitProvider == "" {
		gitProvider = "github"
	}

	return &Config{
		Prompt:          os.Getenv("PROMPT"),
		RepoURL:         os.Getenv("REPO_URL"),
		BranchName:      os.Getenv("BRANCH_NAME"),
		CallbackURL:     os.Getenv("CALLBACK_URL"),
		AuthToken:       os.Getenv("AUTH_TOKEN"),
		RunID:           os.Getenv("RUN_ID"),
		StepID:          os.Getenv("STEP_ID"),
		Provider:        providerVal,
		Model:           os.Getenv("MODEL"),
		APIKey:          os.Getenv("API_KEY"),
		GitToken:        os.Getenv("GIT_TOKEN"),
		GitProvider:     gitProvider,
		StoryKey:        os.Getenv("STORY_KEY"),
		ClaudeMDContent: os.Getenv("CLAUDE_MD_CONTENT"),
	}, nil
}
