package queue

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitConsumer struct {
	msgs <-chan amqp.Delivery
}

func NewRabbitMQ() (*RabbitConsumer, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()

	if err != nil {
		return nil, err
	}

	q, err := ch.QueueDeclare(
		"test_queue", // name
		false,        // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
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

func (r *RabbitConsumer) Consume() {
	for msg := range r.msgs {
		process(msg)
		msg.Ack(false)
	}
}

func process(msg amqp.Delivery) {
	fmt.Printf("Received a message: %s", msg.Body)
}
