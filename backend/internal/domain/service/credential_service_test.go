package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentialService_RoundTrip(t *testing.T) {
	repo := &mockCredentialRepo{}
	svc := NewCredentialService(repo, "master-key")

	cred, err := svc.CreateGlobal(context.Background(), "API_TOKEN", "plaintext-value")
	require.NoError(t, err)
	// Stored ciphertext must not equal the plaintext.
	assert.NotEqual(t, "plaintext-value", string(cred.EncryptedValue))
	assert.NotEmpty(t, cred.EncryptedValue)

	got, err := svc.Resolve(context.Background(), "API_TOKEN", nil)
	require.NoError(t, err)
	assert.Equal(t, "plaintext-value", got)
}

// Project-scoped credentials shadow global ones of the same name; resolution falls back to
// global when no project credential exists.
func TestCredentialService_ScopeResolution(t *testing.T) {
	repo := &mockCredentialRepo{}
	svc := NewCredentialService(repo, "master-key")
	projectID := uuid.New()

	_, err := svc.CreateGlobal(context.Background(), "TOKEN", "global-value")
	require.NoError(t, err)
	_, err = svc.CreateProject(context.Background(), "TOKEN", projectID, "project-value")
	require.NoError(t, err)

	// With a project scope, the project credential wins.
	got, err := svc.Resolve(context.Background(), "TOKEN", &projectID)
	require.NoError(t, err)
	assert.Equal(t, "project-value", got)

	// A different project with no override falls back to the global credential.
	other := uuid.New()
	got, err = svc.Resolve(context.Background(), "TOKEN", &other)
	require.NoError(t, err)
	assert.Equal(t, "global-value", got)

	// No scope at all resolves the global credential.
	got, err = svc.Resolve(context.Background(), "TOKEN", nil)
	require.NoError(t, err)
	assert.Equal(t, "global-value", got)
}

func TestCredentialService_ResolveNotFound(t *testing.T) {
	svc := NewCredentialService(&mockCredentialRepo{}, "master-key")
	_, err := svc.Resolve(context.Background(), "MISSING", nil)
	require.Error(t, err)
	assert.True(t, isNotFound(err))
}
