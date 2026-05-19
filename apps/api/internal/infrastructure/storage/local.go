package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/novudesk/novudesk/internal/domain/storage"
)

type LocalProvider struct {
	basePath string
	baseURL  string
}

func NewLocalProvider(basePath, baseURL string) (*LocalProvider, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("create local storage dir: %w", err)
	}
	return &LocalProvider{basePath: basePath, baseURL: baseURL}, nil
}

func (p *LocalProvider) Upload(_ context.Context, key string, r io.Reader, _ storage.UploadOptions) error {
	fullPath := filepath.Join(p.basePath, key)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (p *LocalProvider) Download(_ context.Context, key string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(p.basePath, key))
}

func (p *LocalProvider) Delete(_ context.Context, key string) error {
	return os.Remove(filepath.Join(p.basePath, key))
}

func (p *LocalProvider) PublicURL(key string) string {
	return fmt.Sprintf("%s/files/%s", p.baseURL, key)
}
