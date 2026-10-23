package broker

import (
	"errors"
	"net"
	"strconv"

	"github.com/segmentio/kafka-go"
)

// CreateTopics создаёт топики, если их ещё нет. Вызывается при старте API и воркера.
//
// Почему не полагаться на автосоздание при первой записи: первая запись
// в несуществующий топик может не пройти, а consumer, запущенный раньше
// появления топика, на него не подпишется.
func CreateTopics(addr string) error {
	// Подключаемся к любому брокеру и узнаём, кто контроллер кластера —
	// создавать топики может только он.
	conn, err := kafka.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	ctrlConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer ctrlConn.Close()

	// У нас один брокер, поэтому одна партиция и одна копия (replication factor).
	err = ctrlConn.CreateTopics(
		kafka.TopicConfig{Topic: TopicFileUploaded, NumPartitions: 1, ReplicationFactor: 1},
		kafka.TopicConfig{Topic: TopicFileDeleted, NumPartitions: 1, ReplicationFactor: 1},
	)
	// Топики уже есть (например, после перезапуска) — это не ошибка.
	if errors.Is(err, kafka.TopicAlreadyExists) {
		return nil
	}
	return err
}
