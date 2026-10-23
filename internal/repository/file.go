package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Temirlan2199/file-upload/internal/model"
)

type FileRepository struct {
	db *pgxpool.Pool
}

func NewFileRepository(db *pgxpool.Pool) *FileRepository {
	return &FileRepository{db: db}
}

// Create сохраняет метаданные файла. ID заполняет сервис, created_at — база.
func (r *FileRepository) Create(ctx context.Context, f *model.File) error {
	query := `
		INSERT INTO files (id, owner_id, name, size, content_type)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at`

	// Scan записывает created_at из базы прямо в переданную структуру.
	return r.db.QueryRow(ctx, query, f.ID, f.OwnerID, f.Name, f.Size, f.ContentType).
		Scan(&f.CreatedAt)
}

// GetByID ищет файл по id. Владельца здесь не проверяем —
// это делает сервис (для публичных ссылок проверка не нужна).
func (r *FileRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.File, error) {
	query := `
		SELECT id, owner_id, name, size, content_type, created_at
		FROM files
		WHERE id = $1`

	var f model.File
	err := r.db.QueryRow(ctx, query, id).
		Scan(&f.ID, &f.OwnerID, &f.Name, &f.Size, &f.ContentType, &f.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ListByOwner возвращает файлы пользователя постранично, новые сверху.
// limit — сколько записей вернуть, offset — сколько пропустить с начала.
func (r *FileRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]model.File, error) {
	query := `
		SELECT id, owner_id, name, size, content_type, created_at
		FROM files
		WHERE owner_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	// Query (в отличие от QueryRow) возвращает несколько строк.
	rows, err := r.db.Query(ctx, query, ownerID, limit, offset)
	if err != nil {
		return nil, err
	}
	// rows обязательно закрываем — иначе соединение не вернётся в пул.
	defer rows.Close()

	// Пустой срез, а не nil: тогда в JSON будет [], а не null.
	files := []model.File{}
	for rows.Next() {
		var f model.File
		if err := rows.Scan(&f.ID, &f.OwnerID, &f.Name, &f.Size, &f.ContentType, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	// rows.Err() — ошибка, которая могла случиться во время перебора строк.
	return files, rows.Err()
}

// Delete удаляет файл владельца. Если такого файла нет — ErrNotFound.
func (r *FileRepository) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	query := `DELETE FROM files WHERE id = $1 AND owner_id = $2`

	// Exec — для запросов, которые не возвращают строки.
	tag, err := r.db.Exec(ctx, query, id, ownerID)
	if err != nil {
		return err
	}
	// RowsAffected — сколько строк удалено. 0 — значит, удалять было нечего.
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
