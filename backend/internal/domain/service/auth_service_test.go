package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func (m *mockUserRepository) List(_ context.Context, _, _ int32) ([]*model.User, error) {
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
	svc := NewAuthService(repo, nil, "test-secret", 24*time.Hour)

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
	svc := NewAuthService(repo, nil, "test-secret", 24*time.Hour)

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
	svc := NewAuthService(repo, nil, "test-secret", 24*time.Hour)

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
	svc := NewAuthService(repo, nil, "test-secret", 24*time.Hour)

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
	svc := NewAuthService(repo, nil, "test-secret", 24*time.Hour)

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
	svc := NewAuthService(repo, nil, "test-secret", 24*time.Hour)

	_, _, err := svc.Login(context.Background(), "nonexistent@example.com", "secureP@ss1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateToken_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewAuthService(repo, nil, "test-secret", 24*time.Hour)

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
	svc := NewAuthService(repo, nil, "test-secret", 24*time.Hour)

	_, err := svc.ValidateToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	repo := newMockRepo()
	svc1 := NewAuthService(repo, nil, "secret-1", 24*time.Hour)
	svc2 := NewAuthService(repo, nil, "secret-2", 24*time.Hour)

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
	svc := NewAuthService(repo, nil, "test-secret", -1*time.Hour)

	_, token, err := svc.Register(context.Background(), "test@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, err = svc.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

// mockBlacklistRepo is a test double for port.TokenBlacklistRepository.
type mockBlacklistRepo struct {
	revoked   map[string]bool
	revokeFn  func(ctx context.Context, jti string, expiresAt time.Time) error
	revokedFn func(ctx context.Context, jti string) (bool, error)
}

func newMockBlacklistRepo() *mockBlacklistRepo {
	return &mockBlacklistRepo{revoked: make(map[string]bool)}
}

func (m *mockBlacklistRepo) Revoke(ctx context.Context, jti string, expiresAt time.Time) error {
	if m.revokeFn != nil {
		return m.revokeFn(ctx, jti, expiresAt)
	}
	m.revoked[jti] = true
	return nil
}

func (m *mockBlacklistRepo) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if m.revokedFn != nil {
		return m.revokedFn(ctx, jti)
	}
	return m.revoked[jti], nil
}

func (m *mockBlacklistRepo) DeleteExpired(_ context.Context) error {
	return nil
}

func TestGenerateToken_HasJTI(t *testing.T) {
	repo := newMockRepo()
	svc := NewAuthService(repo, nil, "test-secret", 24*time.Hour)

	_, token, err := svc.Register(context.Background(), "test@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}

	if claims.ID == "" {
		t.Error("expected non-empty JTI (claims.ID)")
	}
}

func TestAuthService_Logout_RevokesToken(t *testing.T) {
	repo := newMockRepo()
	blacklist := newMockBlacklistRepo()
	svc := NewAuthService(repo, blacklist, "test-secret", 24*time.Hour)

	_, token, err := svc.Register(context.Background(), "test@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Parse claims to get the JTI
	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}

	err = svc.Logout(context.Background(), token)
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	if !blacklist.revoked[claims.ID] {
		t.Error("expected token JTI to be revoked in blacklist")
	}
}

func TestAuthService_Logout_InvalidToken_Noop(t *testing.T) {
	repo := newMockRepo()
	blacklist := newMockBlacklistRepo()
	svc := NewAuthService(repo, blacklist, "test-secret", 24*time.Hour)

	// Logout with invalid token should not error and should not call Revoke
	err := svc.Logout(context.Background(), "invalid-token")
	if err != nil {
		t.Fatalf("expected no error for invalid token, got %v", err)
	}

	if len(blacklist.revoked) != 0 {
		t.Error("expected no revocations for invalid token")
	}
}

func TestAuthService_Logout_EmptyJTI_Noop(t *testing.T) {
	repo := newMockRepo()
	blacklist := newMockBlacklistRepo()
	// Create a service that generates tokens WITHOUT JTI for this test.
	// We simulate a legacy token by creating one without JTI.
	svc := NewAuthService(repo, blacklist, "test-secret", 24*time.Hour)

	// Generate a token without JTI manually
	legacyClaims := &Claims{
		UserID: uuid.New(),
		Role:   "user",
	}
	legacyClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(24 * time.Hour))
	legacyClaims.IssuedAt = jwt.NewNumericDate(time.Now())
	// ID intentionally left empty

	legacyToken := jwt.NewWithClaims(jwt.SigningMethodHS256, legacyClaims)
	tokenStr, err := legacyToken.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	err = svc.Logout(context.Background(), tokenStr)
	if err != nil {
		t.Fatalf("expected no error for empty JTI, got %v", err)
	}

	if len(blacklist.revoked) != 0 {
		t.Error("expected no revocations for token without JTI")
	}
}
