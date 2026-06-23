package memory

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerTokenStore_PersistsRole(t *testing.T) {
	ctx := context.Background()
	s := NewContainerTokenStore(ctx)
	token, err := s.Create(ctx, uuid.New(), uuid.New(), uuid.New(), "review", time.Hour)
	require.NoError(t, err)
	ct, err := s.Validate(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, "review", ct.Role)
}
