// Package callback provides an HTTP client for sending agent runtime events
// back to the hopeitworks API server.
package callback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// maxRetries is the number of retry attempts for retryable HTTP errors.
const maxRetries = 3

// Client sends callback events to the API server.
type Client struct {
	baseURL   string
	authToken string
	runID     string
	stepID    string
	client    *http.Client
}

// New creates a new callback Client.
func New(baseURL, authToken, runID, stepID string) *Client {
	return &Client{
		baseURL:   baseURL,
		authToken: authToken,
		runID:     runID,
		stepID:    stepID,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// logPayload is the request body for log callbacks.
// The API log callback handler (LogCallbackRequest) expects a "lines" array,
// so a single message is wrapped as a one-element slice.
type logPayload struct {
	Lines []string `json:"lines"`
}

// costPayload is the request body for cost callbacks.
type costPayload struct {
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	Model        string  `json:"model"`
	CostUSD      float64 `json:"cost_usd"`
}

// statusPayload is the request body for status callbacks.
type statusPayload struct {
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error"`
}

// Bundle is the fetch-at-startup capability bundle returned by the API. It mirrors the
// backend's model.RuntimeBundle JSON contract. An agent with no capabilities receives a
// zero-value bundle (IsEmpty), in which case the runtime materialises nothing and behaves
// exactly as it did before the capabilities layer existed.
type Bundle struct {
	SystemPrompt string            `json:"system_prompt"`
	Skills       []BundleSkill     `json:"skills"`
	MCP          BundleMCP         `json:"mcp"`
	ToolPolicy   *BundleToolPolicy `json:"tool_policy"`
	Credentials  map[string]string `json:"credentials"`
}

// BundleSkill is a skill rendered as files keyed by relative path (e.g. SKILL.md).
type BundleSkill struct {
	Name  string            `json:"name"`
	Files map[string]string `json:"files"`
}

// BundleMCP is the .mcp.json projection: server name -> opaque connection entry. The
// entries are passed through verbatim so the runtime can write a valid .mcp.json without
// re-modelling every transport detail.
type BundleMCP struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

// BundleToolPolicy is an allow/deny tool list.
type BundleToolPolicy struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// IsEmpty reports whether the bundle carries nothing to materialise.
func (b *Bundle) IsEmpty() bool {
	if b == nil {
		return true
	}
	return b.SystemPrompt == "" &&
		len(b.Skills) == 0 &&
		len(b.MCP.MCPServers) == 0 &&
		b.ToolPolicy == nil &&
		len(b.Credentials) == 0
}

// FetchBundle GETs the agent's capability bundle from the API at startup, authenticated
// by the same container token as the callbacks. The agent is resolved server-side from
// the token. A 404 (older API without the endpoint) is treated as "no bundle" and returns
// (nil, nil) so the runtime stays back-compatible. The caller treats any error as an empty
// bundle and proceeds — capability provisioning must never block the run.
func (c *Client) FetchBundle(ctx context.Context) (*Bundle, error) {
	url := fmt.Sprintf("%s/internal/agent/callback/bundle", c.baseURL)

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create bundle request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.authToken)

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("bundle request failed: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return nil, nil // older API without the bundle endpoint: behave as no-bundle
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var bundle Bundle
			decErr := json.NewDecoder(resp.Body).Decode(&bundle)
			resp.Body.Close()
			if decErr != nil {
				return nil, fmt.Errorf("decode bundle: %w", decErr)
			}
			return &bundle, nil
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// Non-retryable client error (4xx except 429)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return nil, fmt.Errorf("bundle returned non-retryable status %d", resp.StatusCode)
		}
		lastErr = fmt.Errorf("bundle returned retryable status %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("bundle fetch failed after %d attempts: %w", maxRetries, lastErr)
}

// SendLog posts a log event to the API server.
func (c *Client) SendLog(ctx context.Context, message string) error {
	url := fmt.Sprintf("%s/internal/agent/callback/runs/%s/steps/%s/logs", c.baseURL, c.runID, c.stepID)
	return c.postJSON(ctx, url, logPayload{Lines: []string{message}})
}

// SendCost posts a cost event to the API server.
func (c *Client) SendCost(ctx context.Context, inputTokens, outputTokens int64, model string, costUSD float64) error {
	url := fmt.Sprintf("%s/internal/agent/callback/runs/%s/steps/%s/cost", c.baseURL, c.runID, c.stepID)
	return c.postJSON(ctx, url, costPayload{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Model:        model,
		CostUSD:      costUSD,
	})
}

// SendStatus posts completion status to the API server.
func (c *Client) SendStatus(ctx context.Context, exitCode int, errMsg string) error {
	url := fmt.Sprintf("%s/internal/agent/callback/runs/%s/steps/%s/status", c.baseURL, c.runID, c.stepID)
	return c.postJSON(ctx, url, statusPayload{ExitCode: exitCode, Error: errMsg})
}

// postJSON marshals the payload and sends it as a POST request with retries.
func (c *Client) postJSON(ctx context.Context, url string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal callback payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create callback request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.authToken)

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("callback request failed: %w", err)
			continue
		}
		// Drain and close body to allow connection reuse
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// Success
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		// Non-retryable client error (4xx except 429)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return fmt.Errorf("callback returned non-retryable status %d for %s", resp.StatusCode, url)
		}

		// Retryable error (5xx or 429)
		lastErr = fmt.Errorf("callback returned retryable status %d for %s", resp.StatusCode, url)
	}

	return fmt.Errorf("callback failed after %d attempts: %w", maxRetries, lastErr)
}
