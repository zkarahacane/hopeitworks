package model

import (
	"time"

	"github.com/google/uuid"
)

// NOTE: This file is P0 scaffolding. The `stacks` table, multi-arch pinned
// images and the agent StackRef foreign key land in P2. The free-form
// `agents.image` column is kept until then; nothing here is persisted or wired yet.

// Stack keys for the catalogued runtime images.
const (
	StackKeyGo     = "go"
	StackKeyNode   = "node"
	StackKeyPython = "python"
	StackKeyGoNode = "go-node"
)

// Stack is a catalogued, digest-pinned, multi-arch image carrying a toolchain,
// the runtime CLI (claude/opencode) and the per-language LSP. Agents will
// reference a stack by key instead of a free-form image string.
type Stack struct {
	ID        uuid.UUID `json:"id"`
	Key       string    `json:"key"`       // StackKey*
	ImageRef  string    `json:"image_ref"` // digest-pinned, multi-arch
	Toolchain []byte    `json:"toolchain"` // jsonb: toolchain + CLI + LSP versions
	CreatedAt time.Time `json:"created_at"`
}
