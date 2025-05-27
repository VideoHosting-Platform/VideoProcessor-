package storage

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
	client *minio.Client
}

func NewMinioClient() *minio.Client {
	client, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("admin", "password", ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalln("Ошибка инициализации MinIO:", err)
	}
	return client
}

func NewMinioStorage() *MinioStorage {
	client := NewMinioClient()
	return &MinioStorage{
		client: client,
	}
}

func (ms *MinioStorage) Upload(pathLocal, pathUpload string) error {
	ctx := context.Background()

	arr := strings.Split(pathUpload, "/")

	if len(arr) != 3 {
		log.Println("неправильный формат пути видео")
		return fmt.Errorf("error path video in minio")
	}

	bucket, objectName := arr[0], arr[1]+"/"+arr[2]

	// Проверяем, есть ли бакет
	exists, err := ms.client.BucketExists(ctx, bucket)
	if err != nil {
		log.Fatalln(err)
	}
	if !exists {
		if err = ms.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			log.Fatalln(err)
		}
	}

	// Загружаем файл
	info, err := ms.client.FPutObject(ctx, bucket, objectName, pathLocal, minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Успешно загружено: %s (%.2f MB)\n", info.Key, float64(info.Size)/(1024*1024))

	return nil
}

func (ms *MinioStorage) Download(pathDownload, pathLocal string) error {
	ctx := context.Background()

	arr := strings.Split(pathDownload, "/")
	if len(arr) != 2 {
		log.Println("не правильный формат пути видео")
		return fmt.Errorf("error path video in minio")
	}

	bucket, objectName := arr[0], arr[1]

	err := ms.client.FGetObject(ctx, bucket, objectName, pathLocal, minio.GetObjectOptions{})
	if err != nil {
		log.Println("Ошибка скачивания:", err)
		return err
	}
	log.Printf("Успешно скачано: %s\n", pathLocal)

	return nil
}
