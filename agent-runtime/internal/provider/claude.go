package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// ClaudeProvider wraps the Claude Code CLI.
type ClaudeProvider struct {
	apiKey string
}

// claudeStreamLine represents a single line of Claude Code's stream-json output.
type claudeStreamLine struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Message string `json:"message"`
	Result  *struct {
		TotalCostUSD float64 `json:"total_cost_usd"`
		Usage        *struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	} `json:"result"`
}

// Run executes the Claude Code CLI in workDir with the given prompt.
// It streams NDJSON output and emits events for logs, cost, and the final result.
func (c *ClaudeProvider) Run(ctx context.Context, workDir string, prompt string, model string) (<-chan Event, error) {
	cmd := exec.CommandContext(ctx,
		"claude",
		"--print",
		"--output-format", "stream-json",
		"--model", model,
		"--dangerously-skip-permissions",
		"--verbose",
		prompt,
	)
	cmd.Dir = workDir
	cmd.Env = append(cmd.Environ(), "ANTHROPIC_API_KEY="+c.apiKey)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude CLI: %w", err)
	}

	events := make(chan Event, 64)

	go func() {
		defer close(events)

		scanner := bufio.NewScanner(stdout)
		// Allow up to 1MB per line for large JSON payloads
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var parsed claudeStreamLine
			if err := json.Unmarshal(line, &parsed); err != nil {
				// Non-JSON line, emit as log
				events <- Event{Type: "log", Message: string(line)}
				continue
			}

			switch parsed.Type {
			case "assistant":
				msg := parsed.Content
				if msg == "" {
					msg = parsed.Message
				}
				if msg != "" {
					events <- Event{Type: "log", Message: msg}
				}

			case "result":
				if parsed.Result != nil {
					var inputTokens, outputTokens int64
					if parsed.Result.Usage != nil {
						inputTokens = parsed.Result.Usage.InputTokens
						outputTokens = parsed.Result.Usage.OutputTokens
					}
					events <- Event{
						Type:         "cost",
						InputTokens:  inputTokens,
						OutputTokens: outputTokens,
						Model:        model,
						CostUSD:      parsed.Result.TotalCostUSD,
					}
				}
			}
		}

		exitCode := 0
		errMsg := ""
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
			errMsg = err.Error()
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
