// Package service — бизнес-логика. Сервис ничего не знает про HTTP (gin)
// и про конкретную базу: зависимости приходят через интерфейсы.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Temirlan2199/file-upload/internal/model"
	"github.com/Temirlan2199/file-upload/internal/repository"
)

// Бизнес-ошибки. Хендлер по ним выбирает HTTP-статус.
var (
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrPasswordTooLong    = errors.New("password is too long")
	ErrInvalidToken       = errors.New("invalid token")
	ErrUserNotFound       = errors.New("user not found")
)

// dummyHash — bcrypt-хэш, с которым сравниваем пароль, когда пользователь
// не найден (см. Login). Считается один раз при старте программы.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

// UserRepository — то, что сервису нужно от хранилища пользователей.
// Интерфейс объявлен здесь, у потребителя, а не в repository: сервис сам
// говорит, что ему нужно («accept interfaces, return structs»).
// В тестах подставим вместо него мок.
type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type AuthService struct {
	users     UserRepository
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewAuthService(users UserRepository, jwtSecret string, tokenTTL time.Duration) *AuthService {
	return &AuthService{
		users:     users,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*model.User, error) {
	email = normalizeEmail(email)

	// bcrypt сам генерирует соль и хранит её внутри хэша.
	// DefaultCost (10) — разумный баланс скорости и стойкости к перебору.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// bcrypt принимает максимум 72 байта (кириллица — 2 байта на символ).
		if errors.Is(err, bcrypt.ErrPasswordTooLong) {
			return nil, ErrPasswordTooLong
		}
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, email, string(hash))
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// Login проверяет email+пароль и выдаёт JWT access-токен.
func (s *AuthService) Login(ctx context.Context, email, password string) (token string, expiresAt time.Time, err error) {
	user, err := s.users.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		// Не сообщаем, что именно не так — email или пароль,
		// иначе по ответам можно перебором узнать, какие email зарегистрированы.
		if errors.Is(err, repository.ErrNotFound) {
			// По этой же причине тратим то же время, что и на проверку
			// настоящего пароля: иначе ответ «нет такого email» (~0.5ms)
			// отличался бы от «неверный пароль» (~50ms) — timing attack.
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return "", time.Time{}, ErrInvalidCredentials
		}
		return "", time.Time{}, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", time.Time{}, ErrInvalidCredentials
	}

	now := time.Now()
	expiresAt = now.Add(s.tokenTTL)

	// JWT = header.payload.signature. Payload (claims) читается кем угодно
	// (это base64, не шифрование!), поэтому кладём туда только id пользователя.
	// Подпись HMAC-SHA256 гарантирует, что токен не подделан.
	claims := jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return token, expiresAt, nil
}

// ParseToken проверяет подпись и срок жизни токена и возвращает id пользователя.
func (s *AuthService) ParseToken(tokenString string) (uuid.UUID, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(tokenString, &claims,
		func(*jwt.Token) (any, error) { return s.jwtSecret, nil },
		// Явно разрешаем только HS256: защита от атаки с подменой алгоритма
		// (например, "alg": "none").
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	return userID, nil
}

func (s *AuthService) GetUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// normalizeEmail приводит email к единому виду, чтобы User@Mail.com
// и user@mail.com считались одним пользователем.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
