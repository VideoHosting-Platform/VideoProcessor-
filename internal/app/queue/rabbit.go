package queue

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/task"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQConsumerConfig struct {
	User     string `env:"RABBITMQ_USER"`
	Password string `env:"RABBITMQ_PASSWORD"`
	Host     string `env:"RABBITMQ_HOST"`
	Port     string `env:"RABBITMQ_PORT"`

	ConsumerName string `env:"RABBITMQ_CONSUMER_NAME"`
	ProducerName string `env:"RABBITMQ_PRODUCER_NAME"`
}

type TaskHandler interface {
	Execute(t task.VideoTask) (string, error)
}

type RabbitConsumer struct {
	Conn         *amqp.Connection
	consumerName string // name
	producerName string
}

func NewRabbitMQConsumer(cfg RabbitMQConsumerConfig) (*RabbitConsumer, error) {
	dsn := fmt.Sprintf(
		"amqp://%s:%s@%s:%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
	)

	conn, err := amqp.Dial(dsn)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	return &RabbitConsumer{Conn: conn,
		consumerName: cfg.ConsumerName,
		producerName: cfg.ProducerName,
	}, nil
}

func (r *RabbitConsumer) newConsumeChan(tag string) (<-chan amqp.Delivery, error) {
	ch, err := r.Conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed create channel (Consumer) %w", err)
	}

	q, err := ch.QueueDeclare(
		tag,
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	if err != nil {
		return nil, fmt.Errorf("queue deckareted failed: %w", err)
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
		return nil, fmt.Errorf("failed get queued dilivery (Consumer) %w", err)
	}

	return msgs, nil

}

func (r *RabbitConsumer) publish(name string, body []byte) error {
	ch, err := r.Conn.Channel()
	if err != nil {
		return fmt.Errorf("failed create channel (Producer) %w", err)
	}

	q, err := ch.QueueDeclare(
		name,
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	if err != nil {
		return fmt.Errorf("queue deckareted error (Producer): %w", err)
	}

	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish failed (Producer): %w", err)
	}

	return nil

}

func (r *RabbitConsumer) Run(handler TaskHandler) error {

	consumChan, err := r.newConsumeChan(r.consumerName)

	if err != nil {
		return fmt.Errorf("failed to create consume channel (Run): %w", err)
	}

	for msg := range consumChan {
		var vt task.VideoTask
		err := json.Unmarshal(msg.Body, &vt)
		if err != nil {
			slog.Error("error unmarshal message in consume", "error", err, "body", string(msg.Body))
			msg.Nack(false, false) // Отменяем сообщение, если не удалось разобрать
			continue
		}
		slog.Info("message received", "queue", r.consumerName, "body", vt)

		url, err := handler.Execute(vt)
		if err != nil {
			slog.Error("error execute task", "error", err, "task", vt)
			msg.Nack(false, false) // Отменяем сообщение, если обработка не удалась
			continue
		}

		post := task.DBUpload{
			VideoID:    vt.VideoID,
			UserID:     vt.UserID,
			VideoTitle: vt.VideoTitle,
			URL:        url,
		}

		body, err := json.Marshal(post)
		if err != nil {
			slog.Error("error marshal post", "error", err, "post", post)
			msg.Nack(false, false) // Отменяем сообщение, если не удалось сериализовать
			continue
		}

		err = r.publish(r.producerName, body)
		if err != nil {
			slog.Error("error publish message", "error", err, "body", string(body))
			msg.Nack(false, false) // Отменяем сообщение, если публикация не удалась
			continue
		}
		slog.Info("message published", "body", "queue", r.producerName, string(body))

		msg.Ack(false)
	}

	return nil
}
