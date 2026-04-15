package storage

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Local struct {
	Root      string
	PublicURL string
}

func NewLocal(root, publicURL string) *Local {
	return &Local{Root: root, PublicURL: strings.TrimRight(publicURL, "/")}
}

func (l *Local) absPath(key string) string {
	return filepath.Join(l.Root, filepath.FromSlash(key))
}

func (l *Local) Put(key string, data io.Reader, size int64, contentType string) error {
	p := l.absPath(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, data)
	return err
}

func (l *Local) Get(key string) (io.ReadCloser, int64, error) {
	f, err := os.Open(l.absPath(key))
	if err != nil {
		return nil, 0, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, err
	}
	return f, info.Size(), nil
}

func (l *Local) Delete(key string) error {
	err := os.Remove(l.absPath(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (l *Local) Exists(key string) bool {
	_, err := os.Stat(l.absPath(key))
	return err == nil
}

func (l *Local) URL(key string) string {
	return l.PublicURL + "/i/" + key
}
