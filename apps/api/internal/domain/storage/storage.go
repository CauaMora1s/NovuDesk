package storage

import (
	"context"
	"io"
)

// Provider abstracts file storage. Swap implementations without touching application code.
type Provider interface {
	Upload(ctx context.Context, key string, r io.Reader, opts UploadOptions) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	PublicURL(key string) string
}

type UploadOptions struct {
	ContentType string
	SizeBytes   int64
}

// AllowedMIMETypes is the server-side allowlist for uploaded files.
var AllowedMIMETypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"application/pdf": true,
	"text/plain":      true,
	"text/csv":        true,
	"application/zip": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":      true,
}

const MaxFileSizeBytes = 25 * 1024 * 1024 // 25 MB
