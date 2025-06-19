package main

// test1
import (
	"log"
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
		log.Println("No .env file found, using system environment variables", "error", err)
	}

	// 1 Load configuration
	cfg := config.MustLoadConfig()

	log.Println("Configuration loaded successfully", "environment", cfg.Environment)

	// 2 TODO Initialize logger
	logger.Init(logger.Env(cfg.Environment))

	slog.Info("Logger initialized", "environment", cfg.Environment)

	// 3 Initialize queue connection
	rabbit, err := queue.NewRabbitMQConsumer(cfg.RabbitMQ)
	if err != nil {
		slog.Error("Failed to initialize RabbitMQ consumer", "error", err)
		os.Exit(1)
	}

	slog.Info("RabbitMQ consumer initialized", "consumerName", cfg.RabbitMQ.ConsumerName, "producerName", cfg.RabbitMQ.ProducerName)

	// 4 Initialize video storage connection
	minioStorage, err := storage.NewMinioStorage(cfg.MinIO)
	if err != nil {
		slog.Error("Failed to initialize MinIO storage", "error", err)
		os.Exit(1)
	}

	slog.Info("MinIO storage initialized", "endpoint", cfg.MinIO.Host, "bucket", cfg.MinIO.Port, "bucketName", cfg.MinIO.BucketName)

	// 5 Initialize processor
	process := task.NewVideoProcess()

	// 6 Run queue consumer
	vs := services.NewVideoService(minioStorage, process)

	slog.Info("Video service initialized and ready to run")
	if err := rabbit.Run(vs); err != nil {
		slog.Error("Failed to run service", "error", err)
	}

	// 7 TODO Metrics and helth endpoints

}
