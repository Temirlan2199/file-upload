// Package service — бизнес-логика. Сервис не знает про HTTP и SQL.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Temirlan2199/file-upload/internal/model"
	"github.com/Temirlan2199/file-upload/internal/repository"
)

var (
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid token")
)

// UserRepository описывает, что сервису нужно от хранилища пользователей.
// Сервис зависит от интерфейса, а не от конкретной структуры — поэтому
// в тестах вместо настоящей базы можно будет подставить заглушку.
type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type AuthService struct {
	users     UserRepository
	jwtSecret []byte
	jwtTTL    time.Duration
}

func NewAuthService(users UserRepository, jwtSecret string, jwtTTL time.Duration) *AuthService {
	return &AuthService{users: users, jwtSecret: []byte(jwtSecret), jwtTTL: jwtTTL}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*model.User, error) {
	// Пароль никогда не храним в открытом виде — только bcrypt-хэш.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := s.users.Create(ctx, email, string(hash))
	if errors.Is(err, repository.ErrConflict) {
		return nil, ErrEmailTaken
	}
	return user, err
}

// Login проверяет email и пароль и возвращает JWT-токен.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	// В токен кладём id пользователя (sub) и срок жизни (exp).
	// Токен подписан секретом — подделать его без секрета нельзя.
	claims := jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

// ParseToken проверяет подпись и срок жизни токена и возвращает id пользователя.
func (s *AuthService) ParseToken(token string) (uuid.UUID, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) {
		return s.jwtSecret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	return uuid.Parse(claims.Subject)
}

func (s *AuthService) GetUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.users.GetByID(ctx, id)
}
