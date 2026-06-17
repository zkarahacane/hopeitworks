package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ClaudeProvider wraps the Claude Code CLI.
type ClaudeProvider struct {
	apiKey string
}

// claudeContentBlock is one block of an assistant message's content array.
type claudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// claudeStreamLine represents a single line of Claude Code's --output-format
// stream-json output. Assistant text lives at message.content[].text; the final
// "result" event carries total_cost_usd and usage at the TOP level of the event.
type claudeStreamLine struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Message *struct {
		Content []claudeContentBlock `json:"content"`
	} `json:"message"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	Usage        *struct {
		InputTokens  int64 `json:"input_tokens"`
		OutputTokens int64 `json:"output_tokens"`
	} `json:"usage"`
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
	// Route the credential to the right auth env based on its type:
	//   - OAuth tokens (sk-ant-oat...) authenticate Claude Code against a
	//     subscription via CLAUDE_CODE_OAUTH_TOKEN (no API billing).
	//   - API keys (sk-ant-api... or any other) use the billed API via ANTHROPIC_API_KEY.
	authEnv := "ANTHROPIC_API_KEY=" + c.apiKey
	if strings.HasPrefix(c.apiKey, "sk-ant-oat") {
		authEnv = "CLAUDE_CODE_OAUTH_TOKEN=" + c.apiKey
	}
	cmd.Env = append(cmd.Environ(), authEnv)

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
				if parsed.Message != nil {
					var sb strings.Builder
					for _, b := range parsed.Message.Content {
						if b.Type == "text" && b.Text != "" {
							if sb.Len() > 0 {
								sb.WriteByte('\n')
							}
							sb.WriteString(b.Text)
						}
					}
					if sb.Len() > 0 {
						events <- Event{Type: "log", Message: sb.String()}
					}
				}

			case "result":
				var inputTokens, outputTokens int64
				if parsed.Usage != nil {
					inputTokens = parsed.Usage.InputTokens
					outputTokens = parsed.Usage.OutputTokens
				}
				events <- Event{
					Type:         "cost",
					InputTokens:  inputTokens,
					OutputTokens: outputTokens,
					Model:        model,
					CostUSD:      parsed.TotalCostUSD,
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
