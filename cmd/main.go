package main

import (
	"log/slog"
	"os"

	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/config"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/queue"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/services"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/storage"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/task"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/logger"
	"github.com/joho/godotenv"
)

func main() {

	// Загружаем переменные среды из файла .env
	// Если файл не найден, используем переменные среды системы
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found, using system environment variables", "error", err)
	}

	// 1 Load configuration
	cfg := config.MustLoadConfig()

	// 2 TODO Initialize logger
	logger.Init(logger.Env(cfg.Environment))

	// 3 Initialize queue connection
	rabbit, err := queue.NewRabbitMQConsumer(cfg.RabbitMQ)
	if err != nil {
		slog.Error("Failed to initialize RabbitMQ consumer", "error", err)
		os.Exit(1)
	}

	// 4 Initialize video storage connection
	minioStorage, err := storage.NewMinioStorage(cfg.MinIO)
	if err != nil {
		slog.Error("Failed to initialize MinIO storage", "error", err)
		os.Exit(1)
	}

	// 5 Initialize processor
	process := task.NewVideoProcess()

	// 6 Run queue consumer
	vs := services.NewVideoService(minioStorage, process)
	rabbit.Run(vs)

	// 7 TODO Metrics and helth endpoints

}
