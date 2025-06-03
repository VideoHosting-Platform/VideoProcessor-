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

const (
	MastePLName            = "master.m3u8"
	VariantPlaylistPattern = "stream_%v.m3u8"   // Шаблон для плейлистов HLS
	SegmentPattern         = "segment_%v_%d.ts" // Шаблон для сегментов HLS

)
