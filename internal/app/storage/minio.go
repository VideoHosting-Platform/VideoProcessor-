package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
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

func newMinioClient(cfg MinioConfig) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.Secure,
	})
	if err != nil {
		return nil, fmt.Errorf("creating MinIO client: %w", err)
	}
	return client, nil
}

func NewMinioStorage(cfg MinioConfig) (*MinioStorage, error) {
	client, err := newMinioClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("initializing MinIO storage: %w", err)
	}
	return &MinioStorage{
		client: client,
		bucket: cfg.BucketName,
	}, nil
}

func (ms *MinioStorage) Upload(pathUpload string) (io.WriteCloser, error) {
	ctx := context.Background()

	bucket, objectName, err := ms.parsePath(pathUpload)

	if err != nil {
		return nil, fmt.Errorf("upload to Minio failed (parsing path):: %w", err)
	}

	if err := ms.createBucketIfNotExists(ctx, bucket); err != nil {
		return nil, fmt.Errorf("upload to Minio failed (bucket check):: %w", err)
	}

	// Загружаем файл
	pr, pw := io.Pipe()
	go func() {
		defer pr.Close()
		slog.Info("Начало загрузки видео в MinIO", "bucket", bucket, "objectName", objectName)
		_, err = ms.client.PutObject(ctx, bucket, objectName, pr, -1, minio.PutObjectOptions{
			ContentType: "video/mp4",
		})
		if err != nil {
			// TODO: проверить ошибку на EOF?
			slog.Error("Ошибка загрузки видео в MinIO", "bucket", bucket, "objectName", objectName, "error", err)
			return
		}
		slog.Info("Видео успешно загружено в MinIO", "bucket", bucket, "objectName", objectName)
	}()

	return pw, nil
}

func (ms *MinioStorage) Download(pathDownload string) (io.Reader, error) {
	ctx := context.Background()

	bucket, objectName, err := ms.parsePath(pathDownload)
	if err != nil {
		return nil, fmt.Errorf("download from Minio failed (parsing path): %w", err)
	}

	obj, err := ms.client.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("download from Minio failed (getting object): %w", err)
	}

	return obj, nil
}

func (ms *MinioStorage) GetPresignedURL(pathDownload string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)

	bucket, objectName, err := ms.parsePath(pathDownload)
	if err != nil {
		return "", fmt.Errorf("get presigned URL from Minio failed (parsing path): %w", err)
	}

	presignedURL, err := ms.client.PresignedGetObject(context.Background(), bucket, objectName, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to get object (presigned URL): %w", err)
	}
	return presignedURL.String(), nil
}

func (ms *MinioStorage) parsePath(path string) (string, string, error) {
	arr := strings.Split(path, "/")
	if len(arr) < 2 {
		return "", "", fmt.Errorf("error path video in minio: %s", path)
	}
	bucket := arr[0]
	objectName := strings.Join(arr[1:], "/")
	return bucket, objectName, nil
}

func (ms *MinioStorage) createBucketIfNotExists(ctx context.Context, bucket string) error {
	exists, err := ms.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("bucket authentication error: %w", err)
	}
	if !exists {
		if err = ms.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("bucket create error: %w", err)
		}
	}
	return nil
}
