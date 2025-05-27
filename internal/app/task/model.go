package task

import "github.com/google/uuid"

type VideoTask struct {
	VideoID    uuid.UUID `json:"video_id"`
	UserID     int64     `json:"user_id"`
	VideoTitle string    `json:"video_title"`
}
