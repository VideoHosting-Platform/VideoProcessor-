package storage

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
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

func UploadVideo(client *minio.Client, bucket, objectName, filePath string) {
	ctx := context.Background()

	// Проверяем, есть ли бакет
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		log.Fatalln(err)
	}
	if !exists {
		if err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			log.Fatalln(err)
		}
	}

	// Загружаем файл
	info, err := client.FPutObject(ctx, bucket, objectName, filePath, minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Успешно загружено: %s (%.2f MB)\n", info.Key, float64(info.Size)/(1024*1024))
}

func DownloadVideo(client *minio.Client, bucket, objectName, destPath string) {
	ctx := context.Background()
	err := client.FGetObject(ctx, bucket, objectName, destPath, minio.GetObjectOptions{})
	if err != nil {
		log.Fatalln("Ошибка скачивания:", err)
	}
	log.Printf("Успешно скачано: %s\n", destPath)
}
