package model

import "github.com/google/uuid"

// ShareLink — то, на что указывает публичная ссылка: какой файл и чей.
// Хранится в Redis под ключом "share:<token>" и сам исчезает по истечении TTL.
type ShareLink struct {
	FileID  uuid.UUID
	OwnerID uuid.UUID
}
