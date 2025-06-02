package config

import (
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/queue"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/storage"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	MinIO    storage.MinioConfig          `envDefault:""`
	RabbitMQ queue.RabbitMQConsumerConfig `envDefault:""`
}

func MustLoadConfig() Config {
	// parse
	var cfg Config

	err := env.Parse(&cfg)
	if err != nil {
		panic("failed to load config: " + err.Error())
	}
	return cfg
}
