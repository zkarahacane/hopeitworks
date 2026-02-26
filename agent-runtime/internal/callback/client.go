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
type logPayload struct {
	Message string `json:"message"`
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

// SendLog posts a log event to the API server.
func (c *Client) SendLog(ctx context.Context, message string) error {
	url := fmt.Sprintf("%s/internal/agent/callback/runs/%s/steps/%s/logs", c.baseURL, c.runID, c.stepID)
	return c.postJSON(ctx, url, logPayload{Message: message})
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
