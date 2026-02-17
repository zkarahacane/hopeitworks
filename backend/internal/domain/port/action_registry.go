package port

import "github.com/zakari/hopeitworks/backend/internal/domain/model"

// ActionRegistry manages registration and lookup of pipeline step actions.
type ActionRegistry interface {
	// Register registers an action by its name.
	// If an action with the same name exists, it is overwritten.
	Register(action model.Action)

	// Get retrieves an action by name.
	// Returns ACTION_NOT_FOUND error if action is not registered.
	Get(name string) (model.Action, error)
}
