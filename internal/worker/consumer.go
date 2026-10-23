// Package worker — чтение событий из Kafka. Это «транспортный» слой,
// как handler для HTTP: принять сообщение, разобрать, вызвать сервис.
package worker

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Temirlan2199/file-upload/internal/broker"
)

// Processor — методы сервиса обработки, которые нужны воркеру.
type Processor interface {
	ProcessUploaded(ctx context.Context, id uuid.UUID) error
	DeleteObject(ctx context.Context, id uuid.UUID) error
}

type Consumer struct {
	reader *kafka.Reader
	proc   Processor
	log    *slog.Logger
}

func NewConsumer(addr string, proc Processor, log *slog.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{addr},
		// GroupID — имя группы потребителей (consumer group). Kafka запоминает,
		// до какого сообщения группа дочитала (offset), поэтому после перезапуска
		// воркер продолжит с того же места. Если запустить несколько воркеров
		// с одним GroupID, Kafka распределит сообщения между ними.
		GroupID: "file-worker",
		// Читаем сразу оба топика.
		GroupTopics: []string{broker.TopicFileUploaded, broker.TopicFileDeleted},
	})
	return &Consumer{reader: reader, proc: proc, log: log}
}

// Run читает сообщения в бесконечном цикле, пока не отменят ctx (Ctrl+C).
func (c *Consumer) Run(ctx context.Context) error {
	for {
		// FetchMessage ждёт следующее сообщение (блокирует, пока его нет).
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // нас остановили — штатный выход
			}
			return err
		}

		c.handle(ctx, msg)

		// Commit — сообщаем Kafka: «это сообщение обработано, дальше не отдавай».
		// Коммитим ПОСЛЕ обработки: если воркер упадёт посередине, после
		// перезапуска он получит это сообщение снова (at-least-once — «хотя бы раз»).
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
	}
}

// handle разбирает одно сообщение и вызывает нужный метод сервиса.
func (c *Consumer) handle(ctx context.Context, msg kafka.Message) {
	var event broker.FileEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		// Битое сообщение не исправится от повторов — пропускаем его.
		c.log.Error("bad message", "topic", msg.Topic, "error", err)
		return
	}

	var err error
	switch msg.Topic {
	case broker.TopicFileUploaded:
		err = c.proc.ProcessUploaded(ctx, event.FileID)
	case broker.TopicFileDeleted:
		err = c.proc.DeleteObject(ctx, event.FileID)
	}

	// Для простоты ошибки только логируем, без повторных попыток.
	if err != nil {
		c.log.Error("process event", "topic", msg.Topic, "file_id", event.FileID, "error", err)
		return
	}
	c.log.Info("event processed", "topic", msg.Topic, "file_id", event.FileID)
}

// Close закрывает соединение с Kafka и выходит из consumer group.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
