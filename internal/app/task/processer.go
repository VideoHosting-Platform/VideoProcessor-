package task

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Processer interface {
	Process(t VideoTask, inputFile string, outputDir string) error
}

type VideoProcess struct {
}

func NewVideoProcess() *VideoProcess {
	return &VideoProcess{}
}

func (vh *VideoProcess) Process(t VideoTask, inputFile string, outputDir string) error {
	fmt.Printf("Получена таска:\n\tvideiID: %s\n\tUserID: %d\n\tTitle: %s\n", t.VideoID, t.UserID, t.VideoTitle)
	// if err := os.MkdirAll("videos", 0755); err != nil {
	// 	log.Fatalf("Не удалось создать папку videos: %v", err)
	// }

	hlsSegmentFilename := filepath.Join(outputDir, "stream_%v_seg%03d.ts")
	// masterPlaylistName := filepath.Join(outputDir, "master.m3u8") // ffmpeg сам по идее найдет директорию для мастера
	variantPlaylistPattern := filepath.Join(outputDir, "stream_%v.m3u8")

	cmd := exec.Command("ffmpeg",
		"-i", inputFile,
		"-filter_complex", "[0:v]split=3[v0][v1][v2];[v0]scale=640:360[v0out];[v1]scale=1280:720[v1out];[v2]scale=1920:1080[v2out]",
		"-map", "[v0out]", "-c:v:0", "libx264", "-b:v:0", "800k",
		"-map", "[v1out]", "-c:v:1", "libx264", "-b:v:1", "1500k",
		"-map", "[v2out]", "-c:v:2", "libx264", "-b:v:2", "3000k",
		"-f", "hls",
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", hlsSegmentFilename,
		"-master_pl_name", "master.m3u8",
		"-var_stream_map", "v:0,name:360p v:1,name:720p v:2,name:1080p",
		variantPlaylistPattern,
	)
	// Направляем вывод FFmpeg в консоль
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Запуск FFmpeg с командой:")
	fmt.Println(cmd.String())

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ошибка выполнения FFmpeg для видео %s: %w", t.VideoID, err)
	}

	fmt.Println("Конвертация успешно завершена!")

	return nil
}
