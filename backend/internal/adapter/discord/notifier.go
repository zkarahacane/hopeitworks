package discord

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

// color constants for Discord embed severity.
const (
	colorGreen  = 0x2ECC71
	colorRed    = 0xE74C3C
	colorYellow = 0xF1C40F
	colorGrey   = 0x95A5A6
)

// eventColors maps event names to Discord embed colors.
var eventColors = map[string]int{
	"run.completed":     colorGreen,
	"run.failed":        colorRed,
	"hitl_gate.pending": colorYellow,
}

// discordPayload is the JSON body sent to the Discord webhook API.
type discordPayload struct {
	Embeds []discordEmbed `json:"embeds"`
}

// discordEmbed represents a single Discord embed object.
type discordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

// Notifier implements port.Notifier for Discord webhook delivery.
type Notifier struct {
	client *http.Client
}

// Ensure Notifier implements port.Notifier at compile time.
var _ port.Notifier = (*Notifier)(nil)

// NewNotifier creates a new Discord Notifier.
func NewNotifier() *Notifier {
	return &Notifier{client: &http.Client{}}
}

// Send posts a Discord embed notification for the given event to the configured webhook URL.
func (n *Notifier) Send(ctx context.Context, event model.Event, config map[string]string) error {
	url, ok := config["url"]
	if !ok || url == "" {
		return &apperrors.DomainError{
			Category: apperrors.CategoryValidation,
			Code:     "DISCORD_URL_MISSING",
			Message:  "discord notifier: config missing required 'url' field",
		}
	}

	eventName := event.EventName()
	color, ok := eventColors[eventName]
	if !ok {
		color = colorGrey
	}

	payload := discordPayload{
		Embeds: []discordEmbed{
			{
				Title:       eventName,
				Description: fmt.Sprintf("Project: %s | Entity: %s (%s)", event.ProjectID, event.EntityType, event.EntityID),
				Color:       color,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return apperrors.NewInternal("discord notifier: failed to marshal payload", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return apperrors.NewInternal("discord notifier: failed to build request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return apperrors.NewInternal("discord notifier: HTTP request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apperrors.NewInternal(
			fmt.Sprintf("discord notifier: unexpected status %d", resp.StatusCode),
			nil,
		)
	}

	return nil
}
