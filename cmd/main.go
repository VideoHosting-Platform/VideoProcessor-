package main

import (
	"log"

	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/queue"
)

func main() {
	// TODO Load configuration
	// TODO Initialize logger

	// TODO Initialize video storage connection
	// client := storage.NewMinioClient()

	// storage.UploadVideo(client, "test1", "video1", "./video.mp4")
	// storage.DownloadVideo(client, "test1", "video1", "./qwe.mp4")

	// TODO Initialize queue connection
	rabbit, err := queue.NewRabbitMQ()
	if err != nil {
		log.Fatal(err)
	}

	// TODO Run queue consumer
	rabbit.Consume()

	// TODO Metrics and helth endpoints

}
