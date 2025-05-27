package queue

import (
	"encoding/json"
	"log"

	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/task"
	amqp "github.com/rabbitmq/amqp091-go"
)

type TaskHandler interface {
	Execute(t task.VideoTask) error
}

type RabbitConsumer struct {
	msgs <-chan amqp.Delivery
}

func NewRabbitMQConsumer() (*RabbitConsumer, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()

	if err != nil {
		return nil, err
	}

	q, err := ch.QueueDeclare(
		"video_processing", // name
		false,              // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)

	if err != nil {
		return nil, err
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return nil, err
	}

	return &RabbitConsumer{
		msgs: msgs,
	}, nil
}

func (r *RabbitConsumer) Consume(handler TaskHandler) {
	for msg := range r.msgs {
		var vt task.VideoTask
		err := json.Unmarshal(msg.Body, &vt)
		if err != nil {
			log.Printf("error ummarshal in consume: %v\n", err)
		}
		err = handler.Execute(vt)
		if err != nil {
			log.Printf("error handler in consume: %v\n", err)
		}
		// TODO: при ошибке не потвержать, а так же обработку лучше сделать.
		msg.Ack(false)
	}
}
