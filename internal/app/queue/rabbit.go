package queue

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/task"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQConsumerConfig struct {
	RabbitMQURL  string `env:"RABBITMQ_URL"`
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

func failOnError(err error, msg string) {
	if err != nil {
		// TODO log
		log.Fatalf("%s %s", msg, err)
	}
}

func NewRabbitMQConsumer(cfg RabbitMQConsumerConfig) (*RabbitConsumer, error) {
	conn, err := amqp.Dial(cfg.RabbitMQURL)

	if err != nil {
		return nil, err
	}
	// ch, err := conn.Channel()

	// if err != nil {
	// 	return nil, err
	// }

	// q, err := ch.QueueDeclare(
	// 	"video_processing", // name
	// 	false,              // durable
	// 	false,              // delete when unused
	// 	false,              // exclusive
	// 	false,              // no-wait
	// 	nil,                // arguments
	// )

	// if err != nil {
	// 	return nil, err
	// }

	// msgs, err := ch.Consume(
	// 	q.Name,
	// 	"",
	// 	false,
	// 	false,
	// 	false,
	// 	false,
	// 	nil,
	// )

	// if err != nil {
	// 	return nil, err
	// }

	return &RabbitConsumer{Conn: conn,
		consumerName: cfg.ConsumerName,
		producerName: cfg.ProducerName,
	}, nil
}

func (r *RabbitConsumer) newConsumeChan(tag string) (<-chan amqp.Delivery, error) {
	ch, err := r.Conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("create chan for consene %v", err)
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
		return nil, fmt.Errorf("queue deckareted error: %v", err)
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
		return nil, fmt.Errorf("msgs get error: %v", err)
	}

	return msgs, nil

}

func (r *RabbitConsumer) publish(tag string, body []byte) error {
	ch, err := r.Conn.Channel()
	if err != nil {
		return fmt.Errorf("create chan for consene %v", err)
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
		return fmt.Errorf("queue deckareted error: %v", err)
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
		return fmt.Errorf("msgs get error: %v", err)
	}

	return nil

}

func (r *RabbitConsumer) Run(handler TaskHandler) error {

	consumChan, err := r.newConsumeChan(r.consumerName)

	if err != nil {
		return err
	}

	for msg := range consumChan {
		var vt task.VideoTask
		err := json.Unmarshal(msg.Body, &vt)
		if err != nil {
			log.Printf("error ummarshal in consume: %v\n", err)
		}
		url, err := handler.Execute(vt)
		if err != nil {
			log.Printf("error handler in consume: %v\n", err)
		}
		// TODO: при ошибке не потвержать, а так же обработку лучше сделать.
		msg.Ack(false)

		post := task.DBUpload{
			VideoID:    vt.VideoID,
			UserID:     vt.UserID,
			VideoTitle: vt.VideoTitle,
			URL:        url,
		}

		body, err := json.Marshal(post)
		if err != nil {
			log.Printf("error marshal post %s", err)
		}

		err = r.publish(r.producerName, body)
		if err != nil {
			log.Printf("error publish post %s", err)
		}
	}

	return nil
}
