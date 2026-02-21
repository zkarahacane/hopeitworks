package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Notifier implements port.Notifier for generic HTTP webhook delivery.
type Notifier struct {
	client *http.Client
}

// Ensure Notifier implements port.Notifier at compile time.
var _ port.Notifier = (*Notifier)(nil)

// NewNotifier creates a new webhook Notifier.
func NewNotifier() *Notifier {
	return &Notifier{client: &http.Client{}}
}

// Send posts the full event payload as JSON to the configured webhook URL.
func (n *Notifier) Send(ctx context.Context, event model.Event, config map[string]string) error {
	url, ok := config["url"]
	if !ok || url == "" {
		return &apperrors.DomainError{
			Category: apperrors.CategoryValidation,
			Code:     "WEBHOOK_URL_MISSING",
			Message:  "webhook notifier: config missing required 'url' field",
		}
	}

	body, err := json.Marshal(event)
	if err != nil {
		return apperrors.NewInternal("webhook notifier: failed to marshal event", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return apperrors.NewInternal("webhook notifier: failed to build request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return apperrors.NewInternal("webhook notifier: HTTP request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apperrors.NewInternal(
			fmt.Sprintf("webhook notifier: unexpected status %d", resp.StatusCode),
			nil,
		)
	}

	return nil
}
