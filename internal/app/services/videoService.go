package services

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/storage"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/task"
	"github.com/google/uuid"
)

// Временные константы
const (
	BUCKET_NAME = "videos"
	EXPIRY_TIME = 1 * time.Hour // Время жизни presigned URL
)

type VideoService struct {
	storage storage.StorageStreamProvider
	task.Processer
}

func NewVideoService(st storage.StorageStreamProvider, p task.Processer) *VideoService {
	return &VideoService{storage: st, Processer: p}

}

// Сервис выполняет 3 функции, загрузки, обработки, выгрузки видео.
func (vs *VideoService) Execute(vt task.VideoTask) (string, error) {

	taskTempDir, err := os.MkdirTemp("", "video-process-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir for task %s: %w", vt.VideoID, err)
	}

	defer os.RemoveAll(taskTempDir)

	localOutputPath := filepath.Join(taskTempDir, "processed_output") // processed dir

	if err := os.MkdirAll(localOutputPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create output temp subdir for task %s: %w", vt.VideoID, err)
	}

	// Загрузка

	dowloadPath := filepath.Join(BUCKET_NAME, vt.VideoID.String())
	url, err := vs.storage.GetPresignedURL(dowloadPath, EXPIRY_TIME)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL for %s: %w", dowloadPath, err)
	}

	slog.Info("Downloading video", "videoID", vt.VideoID, "url", url)
	//Обработка
	err = vs.Process(vt, url, localOutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to process video %s: %w", vt.VideoID, err)
	}

	//Выгрузка
	// 1) Генерируем уникальный ID для этого процесса
	processID := uuid.New().String()
	uploadPrefix := fmt.Sprintf("%s/%s", BUCKET_NAME, processID)

	// 2) Рекурсивно ходим по локальной папке LOCAL_DIR
	err = vs.uploadAllFilesInDir(localOutputPath, uploadPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to upload files from %s to %s: %w", localOutputPath, uploadPrefix, err)
	}

	// Если нужно возвращать URL, то можно сделать presigned URL для папки
	url, err = vs.storage.GetPresignedURL(uploadPrefix+"/"+task.MastePLName, EXPIRY_TIME)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL(in Execute) for %s: %w", uploadPrefix, err)
	}
	return url, nil

}

func (vs *VideoService) uploadAllFilesInDir(sourceFolder string, remoteFolderPrefix string) error {
	err := filepath.Walk(sourceFolder, func(path string, info os.FileInfo, err error) error {
		slog.Info("uploading file", "path", path, "info", info)
		if err != nil {
			return err
		}
		// Пропускаем папки
		if info.IsDir() {
			return nil
		}
		// Относительный путь внутри localDir
		relPath, err := filepath.Rel(sourceFolder, path)
		if err != nil {
			return err
		}
		// Собираем путь в бакете: "<bucket>/<processID>/<relPath>"
		objectPath := filepath.ToSlash(filepath.Join(remoteFolderPrefix, relPath))

		writer, err := vs.storage.Upload(objectPath)
		if err != nil {
			return fmt.Errorf("upload %s failed: %w", path, err)
		}
		defer writer.Close()
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open file %s failed: %w", path, err)
		}
		defer file.Close()
		if _, err := io.Copy(writer, file); err != nil {
			return fmt.Errorf("copy file %s to storage failed: %w", path, err)
		}
		fmt.Printf("+ uploaded %s → %s\n", path, objectPath)
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
