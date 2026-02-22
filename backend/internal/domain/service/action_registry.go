package service

import (
	"sync"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure InMemoryActionRegistry implements port.ActionRegistry at compile time.
var _ port.ActionRegistry = (*InMemoryActionRegistry)(nil)

// InMemoryActionRegistry is a thread-safe in-memory implementation of port.ActionRegistry.
type InMemoryActionRegistry struct {
	mu      sync.RWMutex
	actions map[string]model.Action
}

// NewActionRegistry creates a new InMemoryActionRegistry.
func NewActionRegistry() *InMemoryActionRegistry {
	return &InMemoryActionRegistry{
		actions: make(map[string]model.Action),
	}
}

// Register registers an action by its name.
// If an action with the same name exists, it is overwritten.
func (r *InMemoryActionRegistry) Register(action model.Action) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.actions[action.Name()] = action
}

// RegisterAlias registers an existing action under an alias name.
// The action is stored under the alias key; its own Name() is not affected.
func (r *InMemoryActionRegistry) RegisterAlias(alias string, action model.Action) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.actions[alias] = action
}

// Get retrieves an action by name.
// Returns ACTION_NOT_FOUND error if action is not registered.
func (r *InMemoryActionRegistry) Get(name string) (model.Action, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	action, ok := r.actions[name]
	if !ok {
		return nil, errors.NewNotFound("action", name)
	}
	return action, nil
}
