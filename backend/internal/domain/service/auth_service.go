package service

import (
	"context"
	"errors"
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
	jwtSecret     []byte
	jwtExpiration time.Duration
}

func NewAuthService(repo port.UserRepository, jwtSecret string, jwtExpiration time.Duration) *AuthService {
	return &AuthService{
		repo:          repo,
		jwtSecret:     []byte(jwtSecret),
		jwtExpiration: jwtExpiration,
	}
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

func (s *AuthService) generateToken(userID uuid.UUID, role model.Role) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
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
