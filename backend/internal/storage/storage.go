package storage

import "io"

type Storage interface {
	Put(key string, data io.Reader, size int64, contentType string) error
	Get(key string) (io.ReadCloser, int64, error)
	Delete(key string) error
	Exists(key string) bool
	URL(key string) string
}
