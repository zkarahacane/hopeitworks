package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
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
	if user.PasswordHash != "" {
		existing.PasswordHash = user.PasswordHash
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

func (m *mockUserRepository) UpdatePasswordHash(_ context.Context, id uuid.UUID, hash string) error {
	u, ok := m.users[id.String()]
	if !ok {
		return errors.New("no rows")
	}
	u.PasswordHash = hash
	return nil
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

// mockPasswordResetTokenRepo is a test double for port.PasswordResetTokenRepository.
type mockPasswordResetTokenRepo struct {
	tokens       map[string]*model.PasswordResetToken
	createFn     func(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*model.PasswordResetToken, error)
	getByTokenFn func(ctx context.Context, token string) (*model.PasswordResetToken, error)
	markUsedFn   func(ctx context.Context, id uuid.UUID) error
}

func newMockTokenRepo() *mockPasswordResetTokenRepo {
	return &mockPasswordResetTokenRepo{
		tokens: make(map[string]*model.PasswordResetToken),
	}
}

func (m *mockPasswordResetTokenRepo) Create(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*model.PasswordResetToken, error) {
	if m.createFn != nil {
		return m.createFn(ctx, userID, token, expiresAt)
	}
	prt := &model.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	m.tokens[token] = prt
	return prt, nil
}

func (m *mockPasswordResetTokenRepo) GetByToken(ctx context.Context, token string) (*model.PasswordResetToken, error) {
	if m.getByTokenFn != nil {
		return m.getByTokenFn(ctx, token)
	}
	prt, ok := m.tokens[token]
	if !ok {
		return nil, errors.New("not found")
	}
	return prt, nil
}

func (m *mockPasswordResetTokenRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	if m.markUsedFn != nil {
		return m.markUsedFn(ctx, id)
	}
	for _, prt := range m.tokens {
		if prt.ID == id {
			now := time.Now()
			prt.UsedAt = &now
			return nil
		}
	}
	return errors.New("not found")
}

// mockEmailSender is a test double for port.EmailSender.
type mockEmailSender struct {
	sendFn   func(ctx context.Context, msg port.EmailMessage) error
	lastMsg  *port.EmailMessage
	sendCall int
}

func newMockEmailSender() *mockEmailSender {
	return &mockEmailSender{}
}

func (m *mockEmailSender) Send(ctx context.Context, msg port.EmailMessage) error {
	m.sendCall++
	m.lastMsg = &msg
	if m.sendFn != nil {
		return m.sendFn(ctx, msg)
	}
	return nil
}

// newTestAuthService creates an AuthService with all mock dependencies.
func newTestAuthService(repo *mockUserRepository, tokenRepo *mockPasswordResetTokenRepo, emailSender *mockEmailSender) *AuthService {
	return NewAuthService(repo, tokenRepo, emailSender, "http://localhost:5173", "test-secret", 24*time.Hour)
}

func TestRegister_Success(t *testing.T) {
	repo := newMockRepo()
	svc := newTestAuthService(repo, newMockTokenRepo(), newMockEmailSender())

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
	svc := newTestAuthService(repo, newMockTokenRepo(), newMockEmailSender())

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
	svc := newTestAuthService(repo, newMockTokenRepo(), newMockEmailSender())

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
	svc := newTestAuthService(repo, newMockTokenRepo(), newMockEmailSender())

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
	svc := newTestAuthService(repo, newMockTokenRepo(), newMockEmailSender())

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
	svc := newTestAuthService(repo, newMockTokenRepo(), newMockEmailSender())

	_, _, err := svc.Login(context.Background(), "nonexistent@example.com", "secureP@ss1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateToken_Success(t *testing.T) {
	repo := newMockRepo()
	svc := newTestAuthService(repo, newMockTokenRepo(), newMockEmailSender())

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
	svc := newTestAuthService(newMockRepo(), newMockTokenRepo(), newMockEmailSender())

	_, err := svc.ValidateToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	repo := newMockRepo()
	tokenRepo := newMockTokenRepo()
	emailSender := newMockEmailSender()
	svc1 := NewAuthService(repo, tokenRepo, emailSender, "http://localhost:5173", "secret-1", 24*time.Hour)
	svc2 := NewAuthService(repo, tokenRepo, emailSender, "http://localhost:5173", "secret-2", 24*time.Hour)

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
	svc := NewAuthService(repo, newMockTokenRepo(), newMockEmailSender(), "http://localhost:5173", "test-secret", -1*time.Hour)

	_, token, err := svc.Register(context.Background(), "test@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, err = svc.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

// ── ForgotPassword tests ────────────────────────────────────────────────

func TestForgotPassword_ValidEmail(t *testing.T) {
	repo := newMockRepo()
	tokenRepo := newMockTokenRepo()
	emailSender := newMockEmailSender()
	svc := newTestAuthService(repo, tokenRepo, emailSender)

	// Register a user first
	_, _, err := svc.Register(context.Background(), "user@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	err = svc.ForgotPassword(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify token was created
	if len(tokenRepo.tokens) != 1 {
		t.Errorf("expected 1 token, got %d", len(tokenRepo.tokens))
	}

	// Verify email was sent
	if emailSender.sendCall != 1 {
		t.Errorf("expected 1 email send call, got %d", emailSender.sendCall)
	}
	if emailSender.lastMsg == nil {
		t.Fatal("expected email message, got nil")
	}
	if emailSender.lastMsg.To != "user@example.com" {
		t.Errorf("expected email to user@example.com, got %s", emailSender.lastMsg.To)
	}
}

func TestForgotPassword_UnknownEmail(t *testing.T) {
	svc := newTestAuthService(newMockRepo(), newMockTokenRepo(), newMockEmailSender())

	err := svc.ForgotPassword(context.Background(), "unknown@example.com")
	if err != nil {
		t.Fatalf("expected nil error (no enumeration), got %v", err)
	}
}

func TestForgotPassword_EmptyEmail(t *testing.T) {
	svc := newTestAuthService(newMockRepo(), newMockTokenRepo(), newMockEmailSender())

	err := svc.ForgotPassword(context.Background(), "")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

// ── ResetPassword tests ─────────────────────────────────────────────────

func TestResetPassword_ValidToken(t *testing.T) {
	repo := newMockRepo()
	tokenRepo := newMockTokenRepo()
	emailSender := newMockEmailSender()
	svc := newTestAuthService(repo, tokenRepo, emailSender)

	// Register a user and create a token
	_, _, err := svc.Register(context.Background(), "user@example.com", "secureP@ss1", "Test User")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	err = svc.ForgotPassword(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("forgot password failed: %v", err)
	}

	// Get the created token
	var tokenStr string
	for k := range tokenRepo.tokens {
		tokenStr = k
		break
	}

	err = svc.ResetPassword(context.Background(), tokenStr, "newSecureP@ss1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the password was updated (can log in with new password)
	_, _, err = svc.Login(context.Background(), "user@example.com", "newSecureP@ss1")
	if err != nil {
		t.Errorf("expected login with new password to succeed, got %v", err)
	}

	// Verify old password no longer works
	_, _, err = svc.Login(context.Background(), "user@example.com", "secureP@ss1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials with old password, got %v", err)
	}

	// Verify token is marked as used
	prt := tokenRepo.tokens[tokenStr]
	if prt.UsedAt == nil {
		t.Error("expected token to be marked as used")
	}
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	svc := newTestAuthService(newMockRepo(), tokenRepo, newMockEmailSender())

	// Create an expired token directly
	expiredToken := &model.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	tokenRepo.tokens["expired-token"] = expiredToken

	err := svc.ResetPassword(context.Background(), "expired-token", "newSecureP@ss1")
	if !errors.Is(err, ErrResetTokenExpired) {
		t.Errorf("expected ErrResetTokenExpired, got %v", err)
	}
}

func TestResetPassword_UsedToken(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	svc := newTestAuthService(newMockRepo(), tokenRepo, newMockEmailSender())

	// Create a used token directly
	usedAt := time.Now().Add(-30 * time.Minute)
	usedToken := &model.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Token:     "used-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		UsedAt:    &usedAt,
		CreatedAt: time.Now().Add(-30 * time.Minute),
	}
	tokenRepo.tokens["used-token"] = usedToken

	err := svc.ResetPassword(context.Background(), "used-token", "newSecureP@ss1")
	if !errors.Is(err, ErrResetTokenInvalid) {
		t.Errorf("expected ErrResetTokenInvalid, got %v", err)
	}
}

func TestResetPassword_TokenNotFound(t *testing.T) {
	svc := newTestAuthService(newMockRepo(), newMockTokenRepo(), newMockEmailSender())

	err := svc.ResetPassword(context.Background(), "nonexistent-token", "newSecureP@ss1")
	if !errors.Is(err, ErrResetTokenInvalid) {
		t.Errorf("expected ErrResetTokenInvalid, got %v", err)
	}
}

func TestResetPassword_WeakPassword(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	svc := newTestAuthService(newMockRepo(), tokenRepo, newMockEmailSender())

	tokenRepo.tokens["valid-token"] = &model.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Token:     "valid-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := svc.ResetPassword(context.Background(), "valid-token", "short")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestResetPassword_EmptyFields(t *testing.T) {
	svc := newTestAuthService(newMockRepo(), newMockTokenRepo(), newMockEmailSender())

	tests := []struct {
		name     string
		token    string
		password string
	}{
		{"empty token", "", "newSecureP@ss1"},
		{"empty password", "some-token", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ResetPassword(context.Background(), tt.token, tt.password)
			if !errors.Is(err, ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}
