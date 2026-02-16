package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// mockUserRepository is a test double for port.UserRepository.
type mockUserRepository struct {
	users    map[string]*model.User
	byEmail  map[string]*model.User
	createFn func(ctx context.Context, user *model.User) (*model.User, error)
}

func newMockRepo() *mockUserRepository {
	return &mockUserRepository{
		users:   make(map[string]*model.User),
		byEmail: make(map[string]*model.User),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	if _, exists := m.byEmail[user.Email]; exists {
		return nil, &pgDuplicateKeyError{}
	}
	user.ID = uuid.New()
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	m.users[user.ID.String()] = user
	m.byEmail[user.Email] = user
	return user, nil
}

func (m *mockUserRepository) GetByEmail(_ context.Context, email string) (*model.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, errors.New("no rows")
	}
	return u, nil
}

func (m *mockUserRepository) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	u, ok := m.users[id.String()]
	if !ok {
		return nil, errors.New("no rows")
	}
	return u, nil
}

func (m *mockUserRepository) List(_ context.Context, limit, offset int32) ([]*model.User, error) {
	var result []*model.User
	for _, u := range m.users {
		result = append(result, u)
	}
	return result, nil
}

func (m *mockUserRepository) Count(_ context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

func (m *mockUserRepository) Update(_ context.Context, user *model.User) (*model.User, error) {
	existing, ok := m.users[user.ID.String()]
	if !ok {
		return nil, errors.New("no rows")
	}
	if user.Name != "" {
		existing.Name = user.Name
	}
	if user.Email != "" {
		existing.Email = user.Email
	}
	existing.UpdatedAt = time.Now()
	return existing, nil
}

func (m *mockUserRepository) Delete(_ context.Context, id uuid.UUID) error {
	u, ok := m.users[id.String()]
	if !ok {
		return errors.New("no rows")
	}
	delete(m.byEmail, u.Email)
	delete(m.users, id.String())
	return nil
}

// pgDuplicateKeyError simulates a PostgreSQL unique constraint violation.
type pgDuplicateKeyError struct{}

func (e *pgDuplicateKeyError) Error() string    { return "duplicate key" }
func (e *pgDuplicateKeyError) SQLState() string { return "23505" }

func TestRegister_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	user, token, err := svc.Register(context.Background(), "test@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}
	if user.Name != "Test User" {
		t.Errorf("expected name Test User, got %s", user.Name)
	}
	if user.Role != model.RoleUser {
		t.Errorf("expected role user, got %s", user.Role)
	}
	if token == "" {
		t.Error("expected token, got empty string")
	}

	// Verify password is hashed
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("secureP@ss1")); err != nil {
		t.Error("password hash does not match")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newMockRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	_, _, err := svc.Register(context.Background(), "test@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	_, _, err = svc.Register(context.Background(), "test@example.com", "secureP@ss1", "Another User")
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestRegister_ValidationError(t *testing.T) {
	repo := newMockRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	tests := []struct {
		name     string
		email    string
		password string
		userName string
	}{
		{"empty email", "", "secureP@ss1", "Test"},
		{"empty password", "test@example.com", "", "Test"},
		{"empty name", "test@example.com", "secureP@ss1", ""},
		{"short password", "test@example.com", "short", "Test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := svc.Register(context.Background(), tt.email, tt.password, tt.userName)
			if !errors.Is(err, ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

func TestLogin_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	_, _, err := svc.Register(context.Background(), "test@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	user, token, err := svc.Login(context.Background(), "test@example.com", "secureP@ss1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}
	if token == "" {
		t.Error("expected token, got empty string")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	_, _, err := svc.Register(context.Background(), "test@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, _, err = svc.Login(context.Background(), "test@example.com", "wrongpassword")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_NonexistentUser(t *testing.T) {
	repo := newMockRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	_, _, err := svc.Login(context.Background(), "nonexistent@example.com", "secureP@ss1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateToken_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	_, token, err := svc.Register(context.Background(), "test@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claims.Role != model.RoleUser {
		t.Errorf("expected role user, got %s", claims.Role)
	}
	if claims.UserID == uuid.Nil {
		t.Error("expected non-nil user ID")
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	repo := newMockRepo()
	svc := NewAuthService(repo, "test-secret", 24*time.Hour)

	_, err := svc.ValidateToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	repo := newMockRepo()
	svc1 := NewAuthService(repo, "secret-1", 24*time.Hour)
	svc2 := NewAuthService(repo, "secret-2", 24*time.Hour)

	_, token, err := svc1.Register(context.Background(), "test@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, err = svc2.ValidateToken(token)
	if err == nil {
		t.Error("expected error for token signed with different secret")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	repo := newMockRepo()
	// Use negative expiration to create immediately expired tokens
	svc := NewAuthService(repo, "test-secret", -1*time.Hour)

	_, token, err := svc.Register(context.Background(), "test@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, err = svc.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}
