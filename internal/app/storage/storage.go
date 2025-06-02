package storage

import (
	"io"
	"time"
)

type StorageProvider interface {
	Download(pathDownload, pathLocal string) error
	Upload(pathLocal, pathUpload string) error
}

// StorageStreamProvider is an interface for storage providers
// that support streaming uploads and downloads.
type StorageStreamProvider interface {
	Download(pathDownload string) (io.Reader, error)
	Upload(pathUpload string) (io.WriteCloser, error)
	GetPresignedURL(pathDownload string, expiry time.Duration) (string, error)
}
