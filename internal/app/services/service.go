package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/storage"
	"github.com/VideoHosting-Platform/VideoProcessor/internal/app/task"
	"github.com/google/uuid"
)

// Временные константы
const (
	BUCKET_NAME = "videos"
)

type VideoService struct {
	storage storage.StorageProvider
	task.Processer
}

func NewVideoService(st storage.StorageProvider, p task.Processer) *VideoService {
	return &VideoService{storage: st, Processer: p}

}

// Сервис выполняет 3 функции, загрузки, обработки, выгрузки.
func (vs *VideoService) Execute(vt task.VideoTask) error {

	// localInputPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s_input.mp4", vt.VideoID.String()))

	taskTempDir, err := os.MkdirTemp("", "video-process-")
	if err != nil {
		// TODO: Log error
		return fmt.Errorf("failed to create temp dir for task %s: %w", vt.VideoID, err)
	}

	defer os.RemoveAll(taskTempDir)

	localInputPath := filepath.Join(taskTempDir, "input.mp4")         // dowloaded dir
	localOutputPath := filepath.Join(taskTempDir, "processed_output") // processed dir

	if err := os.MkdirAll(localOutputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output temp subdir for task %s: %w", vt.VideoID, err)
	}

	// Загрузка

	// Предполагается, что в MinIO лежит videoID.mp4
	// dowloadPath := filepath.Join(BUCKET_NAME, vt.VideoID.String()+".mp4")
	dowloadPath := filepath.Join(BUCKET_NAME, vt.VideoID.String()+"")
	fmt.Println("path ", dowloadPath)
	err = vs.storage.Download(dowloadPath, localInputPath)
	if err != nil {
		fmt.Printf("error execute %v\n", err)
		return err
	}

	//Обработка
	err = vs.Process(vt, localInputPath, localOutputPath)
	if err != nil {
		fmt.Printf("error process %v\n", err)
		return err
	}

	//Выгрузка
	// 1) Генерируем уникальный ID для этого процесса
	processID := uuid.New().String()
	// uploadPrefix = "my-bucket/<processID>"
	uploadPrefix := fmt.Sprintf("%s/%s", BUCKET_NAME, processID)

	// 2) Рекурсивно ходим по локальной папке LOCAL_DIR
	err = vs.uploadAllFilesInDir(localOutputPath, uploadPrefix)
	if err != nil {
		return err
	}

	return nil

}

func (vs *VideoService) uploadAllFilesInDir(sourceFolder string, remoteFolderPrefix string) error {
	err := filepath.Walk(sourceFolder, func(path string, info os.FileInfo, err error) error {
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

		// Выгрузка видео в minio
		if err := vs.storage.Upload(path, objectPath); err != nil {
			return fmt.Errorf("upload %s failed: %w", path, err)
		}
		fmt.Printf("+ uploaded %s → %s\n", path, objectPath)
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
