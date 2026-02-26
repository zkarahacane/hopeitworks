package service

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockAPIKeyRepo is a hand-written mock implementing port.APIKeyRepository.
type mockAPIKeyRepo struct {
	mu   sync.Mutex
	keys map[uuid.UUID]*model.UserAPIKey
}

func newMockAPIKeyRepo() *mockAPIKeyRepo {
	return &mockAPIKeyRepo{keys: make(map[uuid.UUID]*model.UserAPIKey)}
}

func (m *mockAPIKeyRepo) Create(_ context.Context, key *model.UserAPIKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check unique constraint: user_id + provider + key_name
	for _, existing := range m.keys {
		if existing.UserID == key.UserID && existing.Provider == key.Provider && existing.KeyName == key.KeyName {
			return apperrors.NewConflict("api_key", key.Provider+"/"+key.KeyName)
		}
	}

	m.keys[key.ID] = key
	return nil
}

func (m *mockAPIKeyRepo) GetByID(_ context.Context, id uuid.UUID) (*model.UserAPIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, ok := m.keys[id]
	if !ok {
		return nil, apperrors.NewNotFound("api_key", id)
	}
	return key, nil
}

func (m *mockAPIKeyRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*model.UserAPIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*model.UserAPIKey
	for _, key := range m.keys {
		if key.UserID == userID {
			result = append(result, key)
		}
	}
	return result, nil
}

func (m *mockAPIKeyRepo) GetByUserAndProvider(_ context.Context, userID uuid.UUID, provider string) (*model.UserAPIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range m.keys {
		if key.UserID == userID && key.Provider == provider {
			return key, nil
		}
	}
	return nil, apperrors.NewNotFound("api_key", provider)
}

func (m *mockAPIKeyRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.keys, id)
	return nil
}

func TestAPIKeyService_CreateAndList(t *testing.T) {
	repo := newMockAPIKeyRepo()
	svc := NewAPIKeyService(repo, "test-master-key")
	ctx := context.Background()
	userID := uuid.New()

	// Create a key
	key, err := svc.CreateKey(ctx, userID, "claude", "default", "sk-ant-api03-abc123xyz789")
	if err != nil {
		t.Fatalf("CreateKey failed: %v", err)
	}

	if key.ID == uuid.Nil {
		t.Fatal("key ID should not be nil")
	}
	if key.Provider != "claude" {
		t.Fatalf("provider: got %q, want %q", key.Provider, "claude")
	}
	if key.KeyName != "default" {
		t.Fatalf("key_name: got %q, want %q", key.KeyName, "default")
	}

	// Hint should be last 4 chars with "..." prefix
	if key.KeyHint != "...z789" {
		t.Fatalf("key_hint: got %q, want %q", key.KeyHint, "...z789")
	}

	// EncryptedKey must not contain the raw key
	if strings.Contains(string(key.EncryptedKey), "sk-ant-api03-abc123xyz789") {
		t.Fatal("encrypted key should not contain raw key")
	}

	// List keys
	keys, err := svc.ListKeys(ctx, userID)
	if err != nil {
		t.Fatalf("ListKeys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].KeyHint != "...z789" {
		t.Fatalf("listed key hint: got %q, want %q", keys[0].KeyHint, "...z789")
	}
}

func TestAPIKeyService_CreateAndDecrypt(t *testing.T) {
	repo := newMockAPIKeyRepo()
	svc := NewAPIKeyService(repo, "test-master-key")
	ctx := context.Background()
	userID := uuid.New()

	rawKey := "sk-ant-api03-secret-key-value"
	key, err := svc.CreateKey(ctx, userID, "claude", "main", rawKey)
	if err != nil {
		t.Fatalf("CreateKey failed: %v", err)
	}

	// Decrypt by ID
	decrypted, err := svc.DecryptKey(ctx, key.ID)
	if err != nil {
		t.Fatalf("DecryptKey failed: %v", err)
	}
	if decrypted != rawKey {
		t.Fatalf("DecryptKey: got %q, want %q", decrypted, rawKey)
	}

	// Decrypt by user+provider
	decrypted2, err := svc.DecryptKeyForUserProvider(ctx, userID, "claude")
	if err != nil {
		t.Fatalf("DecryptKeyForUserProvider failed: %v", err)
	}
	if decrypted2 != rawKey {
		t.Fatalf("DecryptKeyForUserProvider: got %q, want %q", decrypted2, rawKey)
	}
}

func TestAPIKeyService_DeleteKey(t *testing.T) {
	repo := newMockAPIKeyRepo()
	svc := NewAPIKeyService(repo, "test-master-key")
	ctx := context.Background()
	userID := uuid.New()

	key, err := svc.CreateKey(ctx, userID, "opencode", "default", "oc-key-12345678")
	if err != nil {
		t.Fatalf("CreateKey failed: %v", err)
	}

	// Delete the key
	if err := svc.DeleteKey(ctx, key.ID); err != nil {
		t.Fatalf("DeleteKey failed: %v", err)
	}

	// List should return empty
	keys, err := svc.ListKeys(ctx, userID)
	if err != nil {
		t.Fatalf("ListKeys after delete failed: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys after delete, got %d", len(keys))
	}
}

func TestAPIKeyService_CreateValidation(t *testing.T) {
	repo := newMockAPIKeyRepo()
	svc := NewAPIKeyService(repo, "test-master-key")
	ctx := context.Background()
	userID := uuid.New()

	tests := []struct {
		name     string
		provider string
		keyName  string
		rawKey   string
	}{
		{"empty provider", "", "default", "some-key"},
		{"empty key_name", "claude", "", "some-key"},
		{"empty api_key", "claude", "default", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateKey(ctx, userID, tt.provider, tt.keyName, tt.rawKey)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestGenerateHint(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sk-ant-api03-abc123xyz789", "...z789"},
		{"abcd", "...abcd"},
		{"ab", "...ab"},
		{"a", "...a"},
		{"", "..."},
	}
	for _, tt := range tests {
		got := generateHint(tt.input)
		if got != tt.expected {
			t.Errorf("generateHint(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
