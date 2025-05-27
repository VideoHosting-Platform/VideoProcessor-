package main

import (
	"log"

	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/queue"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/services"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/storage"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/task"
)

func main() {
	// 1 TODO Load configuration

	// 2 TODO Initialize logger

	// 3 Initialize queue connection
	rabbit, err := queue.NewRabbitMQConsumer()
	if err != nil {
		log.Fatal(err)
	}

	// 4 Initialize video storage connection
	minioStorage := storage.NewMinioStorage()

	// 5 Initialize processor
	process := task.NewVideoProcess()

	// 6 Run queue consumer
	vs := services.NewVideoService(minioStorage, process)
	rabbit.Consume(vs)

	// 7 TODO Metrics and helth endpoints

}
