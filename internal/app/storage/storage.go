package storage

import "io"

type StorageProvider interface {
	Download(path string) (io.Reader, error)
	Upload(path string, writer io.Reader) error
}
