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
func (vs *VideoService) Execute(vt task.VideoTask) (string, string, error) {
	processID := uuid.New().String()
	logger := slog.Default().With(
		"component", "VideoService",
		"method", "Execute",
		"videoID", vt.VideoID,
		"processID", processID,
	)

	logger.Info("Starting video processing task")

	taskTempDir, err := os.MkdirTemp("", "video-process-")

	if err != nil {
		return "", processID, fmt.Errorf("failed to create temp dir for task %s: %w", vt.VideoID, err)
	}

	logger.Debug("Created temporary directory for task", "tempDir", taskTempDir)

	defer os.RemoveAll(taskTempDir)
	defer logger.Debug("Temporary directory removed", "tempDir", taskTempDir)

	localOutputPath := filepath.Join(taskTempDir, "processed_output") // processed dir

	if err := os.MkdirAll(localOutputPath, 0755); err != nil {
		return "", processID, fmt.Errorf("failed to create output temp subdir for task %s: %w", vt.VideoID, err)
	}

	// Загрузка

	downloadPath := filepath.Join(BUCKET_NAME, vt.VideoID.String())
	url, err := vs.storage.GetPresignedURL(downloadPath, EXPIRY_TIME)
	if err != nil {
		return "", processID, fmt.Errorf("failed to get presigned URL for %s: %w", downloadPath, err)
	}

	logger.Info("Presigned URL for download", "download_path", downloadPath)

	//Обработка
	err = vs.Process(vt, url, localOutputPath)
	if err != nil {
		return "", processID, fmt.Errorf("failed to process video %s: %w", vt.VideoID, err)
	}

	//Выгрузка

	uploadPrefix := fmt.Sprintf("%s/%s", BUCKET_NAME, processID)

	// 2) Рекурсивно ходим по локальной папке LOCAL_DIR
	logger.Info("Uploading processed files", "localOutputPath", localOutputPath, "uploadPrefix", uploadPrefix)
	err = vs.uploadAllFilesInDir(localOutputPath, uploadPrefix, logger)
	if err != nil {
		return "", processID, fmt.Errorf("failed to upload files from %s to %s: %w", localOutputPath, uploadPrefix, err)
	}

	logger.Info("All files uploaded successfully", "uploadPrefix", uploadPrefix)

	// Возвращаем не presigned url, а url вида /{bucketName}/{videoID}/{masterPlaylistName}
	url = "/" + uploadPrefix + "/" + task.MastePLName
	return url, processID, nil

}

func (vs *VideoService) uploadAllFilesInDir(sourceFolder string, remoteFolderPrefix string, logger *slog.Logger) error {
	logger = logger.With(
		"method", "uploadAllFilesInDir",
		"sourceFolder", sourceFolder,
		"remoteFolderPrefix", remoteFolderPrefix,
	)
	err := filepath.Walk(sourceFolder, func(path string, info os.FileInfo, err error) error {
		logger.Debug("uploading file", "path", path, "info", info)
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
		logger.Debug("uploading file to storage", "objectPath", objectPath)

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

		logger.Debug("File uploaded successfully", "objectPath", objectPath)

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
