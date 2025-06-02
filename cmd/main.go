package main

import (
	"log"

	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/config"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/queue"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/services"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/storage"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/task"
	"github.com/joho/godotenv"
)

func main() {

	// Загружаем переменные среды из файла .env
	// Если файл не найден, используем переменные среды системы
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем переменные среды системы")
	}

	// 1 Load configuration
	cfg := config.MustLoadConfig()

	// 2 TODO Initialize logger

	// 3 Initialize queue connection
	rabbit, err := queue.NewRabbitMQConsumer(cfg.RabbitMQ)
	if err != nil {
		log.Fatal(err)
	}

	// 4 Initialize video storage connection
	minioStorage := storage.NewMinioStorage(cfg.MinIO)

	// 5 Initialize processor
	process := task.NewVideoProcess()

	// 6 Run queue consumer
	vs := services.NewVideoService(minioStorage, process)
	rabbit.Run(vs)

	// 7 TODO Metrics and helth endpoints

}
