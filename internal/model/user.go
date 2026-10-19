// Package model — доменные сущности. Они не знают ни про HTTP, ни про JSON,
// ни про базу данных и не зависят от других пакетов проекта.
package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
