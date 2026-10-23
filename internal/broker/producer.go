package broker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Producer отправляет события о файлах в Kafka.
type Producer struct {
	writer *kafka.Writer
}

func NewProducer(addr string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr: kafka.TCP(addr),
			// Топик не указываем здесь — он задаётся в каждом сообщении.
			// Сами топики создаёт CreateTopics при старте.

			// Ждём подтверждения, что Kafka действительно сохранила сообщение.
			RequiredAcks: kafka.RequireAll,

			// По умолчанию Writer копит сообщения в пачку до 1 секунды,
			// и каждая отправка ждала бы эту секунду. Нам важнее скорость ответа.
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

func (p *Producer) PublishFileUploaded(ctx context.Context, fileID uuid.UUID) error {
	return p.publish(ctx, TopicFileUploaded, fileID)
}

func (p *Producer) PublishFileDeleted(ctx context.Context, fileID uuid.UUID) error {
	return p.publish(ctx, TopicFileDeleted, fileID)
}

func (p *Producer) publish(ctx context.Context, topic string, fileID uuid.UUID) error {
	value, err := json.Marshal(FileEvent{FileID: fileID})
	if err != nil {
		return err
	}

	// Key — id файла. Сообщения с одинаковым ключом попадают в одну партицию,
	// а внутри партиции Kafka сохраняет порядок. Значит, события одного файла
	// воркер получит в том порядке, в котором они были отправлены.
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(fileID.String()),
		Value: value,
	})
}

// Close отправляет то, что осталось в буфере, и закрывает соединения.
func (p *Producer) Close() error {
	return p.writer.Close()
}
