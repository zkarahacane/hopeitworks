package model

import (
	"time"

	"github.com/google/uuid"
)

// ChannelTypeDiscord is the channel type constant for Discord webhook notifications.
const ChannelTypeDiscord = "discord"

// ChannelTypeWebhook is the channel type constant for generic webhook notifications.
const ChannelTypeWebhook = "webhook"

// NotificationConfig represents a notification destination for a project.
type NotificationConfig struct {
	ID           uuid.UUID
	ProjectID    uuid.UUID
	ChannelType  string            // "discord" or "webhook"
	Config       map[string]string // e.g., {"url": "https://discord.com/api/webhooks/..."}
	EventsFilter []string          // e.g., ["run.completed", "run.failed"]
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
