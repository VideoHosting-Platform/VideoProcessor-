package processor

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type VideoEvent struct {
	VideoID    uuid.UUID `json:"video_id"`
	UserID     int64     `json:"user_id"`
	VideoTitle string    `json:"video_title"`
}

type Processor struct {
	mc *minio.Client
}

func (p *Processor) Process(event VideoEvent, done chan<- struct{}) {
	fmt.Println("Received a message", event)
	done <- struct{}{}
}
