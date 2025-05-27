package storage

type StorageProvider interface {
	Download(pathDownload, pathLocal string) error
	Upload(pathLocal, pathUpload string) error
}
