// Package broker — работа с Kafka: названия топиков, формат событий, отправка.
//
// Kafka — брокер сообщений. Один сервис (producer) пишет событие в топик,
// другой (consumer) читает его и обрабатывает — когда сможет.
// Сервисы не вызывают друг друга напрямую и не ждут друг друга.
package broker

import "github.com/google/uuid"

// Топики — «каналы» в Kafka, по одному на тип события.
const (
	TopicFileUploaded = "file.uploaded" // файл загружен — нужно его обработать
	TopicFileDeleted  = "file.deleted"  // файл удалён — нужно убрать объект из MinIO
)

// FileEvent — содержимое сообщения (в Kafka уходит как JSON).
// Тип события определяется топиком, поэтому внутри достаточно id файла.
type FileEvent struct {
	FileID uuid.UUID `json:"file_id"`
}
