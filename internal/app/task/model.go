package task

import "github.com/google/uuid"

type VideoTask struct {
	VideoID    uuid.UUID `json:"video_id"`
	UserID     int64     `json:"user_id"`
	VideoTitle string    `json:"video_title"`
}

type DBUpload struct {
	VideoID    uuid.UUID `json:"video_id"`
	UserID     int64     `json:"user_id"`
	VideoTitle string    `json:"video_title"`
	URL        string    `json:"video_master_playlist_url"`
}
