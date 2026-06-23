package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// ContainerTokenStore is an in-memory implementation of port.ContainerTokenStore.
// It uses a sync.Map for concurrent-safe token storage with TTL-based expiry.
type ContainerTokenStore struct {
	tokens sync.Map // map[string]*model.ContainerToken
}

// NewContainerTokenStore creates a new in-memory container token store and starts
// a background goroutine that cleans up expired tokens every 60 seconds.
// The cleanup goroutine stops when the context is cancelled.
func NewContainerTokenStore(ctx context.Context) *ContainerTokenStore {
	s := &ContainerTokenStore{}

	go s.cleanupLoop(ctx)

	return s
}

// Create generates and stores a new token for a run step bound to an agent.
// role is the step's pipeline role (e.g. "dev", "review") used for RBAC capability filtering;
// an empty role means the step is not role-scoped (only universal capabilities are granted).
// Returns the token string.
func (s *ContainerTokenStore) Create(_ context.Context, runID, stepID, agentID uuid.UUID, role string, ttl time.Duration) (string, error) {
	token := uuid.New().String()
	ct := &model.ContainerToken{
		Token:     token,
		RunID:     runID,
		StepID:    stepID,
		AgentID:   agentID,
		Role:      role,
		ExpiresAt: time.Now().Add(ttl),
	}
	s.tokens.Store(token, ct)
	return token, nil
}

// Validate checks if a token is valid and returns the associated container token.
// Returns an error if the token is invalid or expired.
func (s *ContainerTokenStore) Validate(_ context.Context, token string) (*model.ContainerToken, error) {
	val, ok := s.tokens.Load(token)
	if !ok {
		return nil, fmt.Errorf("token not found")
	}

	ct := val.(*model.ContainerToken)
	if time.Now().After(ct.ExpiresAt) {
		s.tokens.Delete(token)
		return nil, fmt.Errorf("token expired")
	}

	return ct, nil
}

// Revoke invalidates a token by removing it from the store.
func (s *ContainerTokenStore) Revoke(_ context.Context, token string) error {
	s.tokens.Delete(token)
	return nil
}

// cleanupLoop periodically removes expired tokens from the store.
func (s *ContainerTokenStore) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			s.tokens.Range(func(key, value any) bool {
				ct := value.(*model.ContainerToken)
				if now.After(ct.ExpiresAt) {
					s.tokens.Delete(key)
				}
				return true
			})
		}
	}
}
