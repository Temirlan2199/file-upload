package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Temirlan2199/file-upload/internal/model"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create вставляет пользователя. id и created_at генерирует база,
// RETURNING сразу возвращает их без второго запроса.
func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (*model.User, error) {
	const query = `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at`

	// Параметры $1, $2 передаются отдельно от текста запроса —
	// это и есть защита от SQL-инъекций. Никогда не склеивай SQL через fmt.Sprintf.
	var u model.User
	err := r.db.QueryRow(ctx, query, email, passwordHash).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	const query = `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE email = $1`

	return r.getOne(ctx, query, email)
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	const query = `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE id = $1`

	return r.getOne(ctx, query, id)
}

func (r *UserRepository) getOne(ctx context.Context, query string, arg any) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx, query, arg).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		// pgx.ErrNoRows — «строк нет». Переводим в нашу ошибку,
		// чтобы сервис не зависел от pgx.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select user: %w", err)
	}
	return &u, nil
}
