package provider

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
)

// OpenCodeProvider wraps the OpenCode CLI.
type OpenCodeProvider struct {
	apiKey string
}

// pricing maps model names to per-million-token costs (input, output).
var pricing = map[string]struct{ InputPerMTok, OutputPerMTok float64 }{
	"gpt-4o":            {2.5, 10.0},
	"gpt-4o-mini":       {0.15, 0.6},
	"gemini-2.0-flash":  {0.1, 0.4},
	"gemini-2.5-pro":    {1.25, 10.0},
	"deepseek-chat":     {0.14, 0.28},
	"claude-opus-4-6":   {15.0, 75.0},
	"claude-sonnet-4-6": {3.0, 15.0},
}

// openCodeResult represents the JSON output from opencode run.
type openCodeResult struct {
	Content string `json:"content"`
	Usage   *struct {
		InputTokens  int64 `json:"input_tokens"`
		OutputTokens int64 `json:"output_tokens"`
	} `json:"usage"`
}

// Run executes the OpenCode CLI in workDir with the given prompt.
// It parses the JSON output and emits log, cost, and result events.
func (o *OpenCodeProvider) Run(ctx context.Context, workDir string, prompt string, model string, opts RunOptions) (<-chan Event, error) {
	cmd := exec.CommandContext(ctx,
		"opencode", "run",
		"--format", "json",
		"--model", model,
		prompt,
	)
	cmd.Dir = workDir

	envKey := apiKeyEnvVar(model)
	cmd.Env = append(cmd.Environ(), envKey+"="+o.apiKey)
	// Resolved secrets are injected into the harness child env only. opencode reads MCP
	// servers from the materialised .mcp.json in workDir and expands ${KEY} from the env.
	// The system-prompt / tool-policy flags have no stable opencode equivalent yet, so
	// those capabilities are warn+skipped here (per the capability × runtime matrix);
	// MCP-over-env still works through ExtraEnv.
	cmd.Env = append(cmd.Env, opts.ExtraEnv...)

	events := make(chan Event, 16)

	go func() {
		defer close(events)

		output, err := cmd.Output()

		// Parse output even if command failed (might have partial results)
		if len(output) > 0 {
			var result openCodeResult
			if jsonErr := json.Unmarshal(output, &result); jsonErr == nil {
				if result.Content != "" {
					events <- Event{Type: "log", Message: result.Content}
				}

				if result.Usage != nil {
					costUSD := computeCost(model, result.Usage.InputTokens, result.Usage.OutputTokens)
					events <- Event{
						Type:         "cost",
						InputTokens:  result.Usage.InputTokens,
						OutputTokens: result.Usage.OutputTokens,
						Model:        model,
						CostUSD:      costUSD,
					}
				}
			} else {
				// Non-JSON output, emit as log
				events <- Event{Type: "log", Message: string(output)}
			}
		}

		exitCode := 0
		errMsg := ""
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
				errMsg = string(exitErr.Stderr)
			} else {
				exitCode = 1
				errMsg = err.Error()
			}
		}

		events <- Event{
			Type:     "result",
			Message:  errMsg,
			ExitCode: exitCode,
			Model:    model,
		}
	}()

	return events, nil
}

// apiKeyEnvVar returns the environment variable name for the LLM API key
// based on the model prefix.
func apiKeyEnvVar(model string) string {
	switch {
	case strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1-"):
		return "OPENAI_API_KEY"
	case strings.HasPrefix(model, "gemini-"):
		return "GOOGLE_API_KEY"
	case strings.HasPrefix(model, "deepseek-"):
		return "DEEPSEEK_API_KEY"
	case strings.HasPrefix(model, "claude-"):
		return "ANTHROPIC_API_KEY"
	default:
		return "OPENAI_API_KEY"
	}
}

// computeCost calculates the USD cost based on token usage and the pricing table.
func computeCost(model string, inputTokens, outputTokens int64) float64 {
	p, ok := pricing[model]
	if !ok {
		return 0
	}
	inputCost := float64(inputTokens) / 1_000_000 * p.InputPerMTok
	outputCost := float64(outputTokens) / 1_000_000 * p.OutputPerMTok
	return inputCost + outputCost
}
