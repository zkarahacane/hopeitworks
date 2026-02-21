package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrValidation         = errors.New("validation error")
	ErrTokenRevoked       = errors.New("token has been revoked")
	ErrResetTokenExpired  = errors.New("reset token expired")
	ErrResetTokenInvalid  = errors.New("reset token invalid or already used")
)

// Claims represents the JWT claims payload.
type Claims struct {
	UserID uuid.UUID  `json:"user_id"`
	Role   model.Role `json:"role"`
	jwt.RegisteredClaims
}

// AuthService handles user authentication logic.
type AuthService struct {
	repo          port.UserRepository
	blacklistRepo port.TokenBlacklistRepository
	tokenRepo     port.PasswordResetTokenRepository
	emailSender   port.EmailSender
	frontendURL   string
	jwtSecret     []byte
	jwtExpiration time.Duration
}

// NewAuthService creates a new AuthService with all dependencies.
// blacklistRepo may be nil for backwards compatibility (environments without token revocation).
func NewAuthService(
	repo port.UserRepository,
	tokenRepo port.PasswordResetTokenRepository,
	emailSender port.EmailSender,
	frontendURL string,
	jwtSecret string,
	jwtExpiration time.Duration,
) *AuthService {
	return &AuthService{
		repo:          repo,
		tokenRepo:     tokenRepo,
		emailSender:   emailSender,
		frontendURL:   frontendURL,
		jwtSecret:     []byte(jwtSecret),
		jwtExpiration: jwtExpiration,
	}
}

// SetBlacklistRepo injects the token blacklist repository (optional).
func (s *AuthService) SetBlacklistRepo(repo port.TokenBlacklistRepository) {
	s.blacklistRepo = repo
}

// Register creates a new user and returns the user with a JWT token.
func (s *AuthService) Register(ctx context.Context, email, password, name string) (*model.User, string, error) {
	if email == "" || password == "" || name == "" {
		return nil, "", ErrValidation
	}
	if len(password) < 8 {
		return nil, "", ErrValidation
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	user, err := s.repo.Create(ctx, &model.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Role:         model.RoleUser,
	})
	if err != nil {
		// Check for unique constraint violation (email already exists).
		// pgx returns a *pgconn.PgError with code 23505 for unique violations.
		// We check the error string to avoid importing pgx in the domain layer.
		if isDuplicateKeyError(err) {
			return nil, "", ErrEmailAlreadyExists
		}
		return nil, "", err
	}

	token, err := s.generateToken(user.ID, user.Role)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Login authenticates a user and returns the user with a JWT token.
func (s *AuthService) Login(ctx context.Context, email, password string) (*model.User, string, error) {
	if email == "" || password == "" {
		return nil, "", ErrValidation
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.generateToken(user.ID, user.Role)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// ValidateToken parses and validates a JWT token string.
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// JWTExpiration returns the configured JWT expiration duration.
func (s *AuthService) JWTExpiration() time.Duration {
	return s.jwtExpiration
}

// Logout invalidates the given JWT token string by adding its JTI to the blacklist.
func (s *AuthService) Logout(ctx context.Context, tokenString string) error {
	if s.blacklistRepo == nil {
		return nil // blacklist not configured, skip revocation
	}
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil //nolint:nilerr // intentional: expired/invalid tokens don't need revocation
	}
	jti := claims.ID
	if jti == "" {
		return nil // legacy token without JTI — skip
	}
	expiresAt := claims.ExpiresAt.Time
	return s.blacklistRepo.Revoke(ctx, jti, expiresAt)
}

// PurgeExpiredTokens removes expired entries from the token blacklist.
func (s *AuthService) PurgeExpiredTokens(ctx context.Context) error {
	if s.blacklistRepo == nil {
		return nil
	}
	return s.blacklistRepo.DeleteExpired(ctx)
}

// ForgotPassword generates a reset token and sends an email if the address is registered.
// Always returns nil to prevent email enumeration.
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	if email == "" {
		return ErrValidation
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil //nolint:nilerr // intentional: return nil to prevent email enumeration
	}

	rawToken, err := generateSecureToken()
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	if _, err := s.tokenRepo.Create(ctx, user.ID, rawToken, expiresAt); err != nil {
		return err
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, rawToken)
	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       user.Email,
		Subject:  "Reset your HopeItWorks password",
		HTMLBody: buildResetEmailHTML(user.Name, resetLink),
	})
}

// ResetPassword validates the token and updates the user's password.
func (s *AuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if rawToken == "" || newPassword == "" {
		return ErrValidation
	}
	if len(newPassword) < 8 {
		return ErrValidation
	}

	prt, err := s.tokenRepo.GetByToken(ctx, rawToken)
	if err != nil {
		return ErrResetTokenInvalid
	}
	if prt.IsUsed() {
		return ErrResetTokenInvalid
	}
	if prt.IsExpired() {
		return ErrResetTokenExpired
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user, err := s.repo.GetByID(ctx, prt.UserID)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	if _, err := s.repo.Update(ctx, user); err != nil {
		return err
	}

	return s.tokenRepo.MarkUsed(ctx, prt.ID)
}

func (s *AuthService) generateToken(userID uuid.UUID, role model.Role) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// generateSecureToken returns a 32-byte URL-safe base64-encoded random token.
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// buildResetEmailHTML returns a minimal HTML email body with the reset link.
func buildResetEmailHTML(name, resetLink string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: sans-serif; padding: 24px;">
  <h2>Password Reset Request</h2>
  <p>Hi %s,</p>
  <p>We received a request to reset your HopeItWorks password.
     Click the button below to set a new password. This link expires in <strong>1 hour</strong>.</p>
  <p><a href="%s" style="background:#4F46E5;color:#fff;padding:12px 24px;border-radius:6px;text-decoration:none;">
    Reset my password
  </a></p>
  <p>If you did not request a password reset, you can ignore this email.</p>
</body>
</html>`, name, resetLink)
}

// sqlStateError is an interface for errors that expose a SQL state code.
// This avoids importing pgx/pgconn directly in the domain layer.
type sqlStateError interface {
	SQLState() string
}

// isDuplicateKeyError checks if the error is a PostgreSQL unique violation (23505).
func isDuplicateKeyError(err error) bool {
	var pgErr sqlStateError
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
