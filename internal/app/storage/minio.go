package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioConfig struct {
	Endpoint   string `env:"MINIO_ENDPOINT,required"`
	BucketName string `env:"MINIO_BUCKET_NAME" envDefault:"my-bucket"`
	AccessKey  string `env:"MINIO_ACCESS_KEY,required"`
	SecretKey  string `env:"MINIO_SECRET_KEY,required"`
	Secure     bool   `env:"MINIO_SECURE" envDefault:"false"`
}

type MinioStorage struct {
	client *minio.Client
	bucket string
}

func newMinioClient(cfg MinioConfig) *minio.Client {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalln("Ошибка инициализации MinIO:", err)
	}
	return client
}

func NewMinioStorage(cfg MinioConfig) *MinioStorage {
	client := newMinioClient(cfg)
	return &MinioStorage{
		client: client,
		bucket: cfg.BucketName,
	}
}

func (ms *MinioStorage) Upload(pathUpload string) (io.WriteCloser, error) {
	ctx := context.Background()

	bucket, objectName, err := ms.parsePath(pathUpload)

	if err != nil {
		log.Println("не правильный формат пути видео")
		return nil, fmt.Errorf("error path video in minio: %w", err)
	}
	// Проверяем, есть ли бакет
	if err := ms.createBucketIfNotExists(ctx, bucket); err != nil {
		log.Println("Ошибка создания бакета:", err)
		return nil, fmt.Errorf("ошибка создания бакета: %w", err)
	}

	// Загружаем файл
	pr, pw := io.Pipe()
	go func() {
		defer pr.Close()
		log.Println("Начало загрузки видео:", bucket, objectName)
		_, err = ms.client.PutObject(ctx, bucket, objectName, pr, -1, minio.PutObjectOptions{
			ContentType: "video/mp4",
		})
		if err != nil {
			// TODO: проверить ошибку на EOF?
			log.Println("Ошибка загрузки видео:", err)
			return
		}
		log.Println("Загрузка видео завершена:", bucket, objectName)
	}()

	return pw, nil
}

func (ms *MinioStorage) Download(pathDownload string) (io.Reader, error) {
	ctx := context.Background()

	arr := strings.Split(pathDownload, "/")
	if len(arr) != 2 {
		log.Println("не правильный формат пути видео")
		return nil, fmt.Errorf("error path video in minio")
	}

	bucket, objectName := arr[0], arr[1]
	obj, err := ms.client.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	// err := ms.client.FGetObject(ctx, bucket, objectName, pathLocal, minio.GetObjectOptions{})
	if err != nil {
		log.Println("Ошибка скачивания:", err)
		return nil, err
	}
	// log.Printf("Успешно скачано: %s\n", pathLocal)

	return obj, nil
}

func (ms *MinioStorage) GetPresignedURL(pathDownload string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	arr := strings.Split(pathDownload, "/")
	if len(arr) != 2 {
		log.Println("не правильный формат пути видео")
		return "", fmt.Errorf("error path video in minio")
	}

	bucket, objectName := arr[0], arr[1]
	//reqParams.Set("response-content-disposition", "attachment; filename=\""+objectName+"\"") // Опционально

	presignedURL, err := ms.client.PresignedGetObject(context.Background(), bucket, objectName, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL: %w", err)
	}
	return presignedURL.String(), nil
}

func (ms *MinioStorage) parsePath(path string) (string, string, error) {
	arr := strings.Split(path, "/")
	if len(arr) < 2 {
		return "", "", fmt.Errorf("неправильный формат пути видео")
	}
	bucket := arr[0]
	objectName := strings.Join(arr[1:], "/")
	return bucket, objectName, nil
}

func (ms *MinioStorage) createBucketIfNotExists(ctx context.Context, bucket string) error {
	exists, err := ms.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("ошибка проверки существования бакета: %w", err)
	}
	if !exists {
		if err = ms.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("ошибка создания бакета: %w", err)
		}
	}
	return nil
}
