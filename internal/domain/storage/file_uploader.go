package storage

import "context"

// FileUploader uploads files and returns a public URL.
// Dependency Inversion: higher layers depend on this abstraction.
type FileUploader interface {
	Upload(ctx context.Context, filename string, data []byte, contentType string) (url string, err error)
}
