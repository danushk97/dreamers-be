package main

import (
	"context"
	"fmt"

	"github.com/dreamers-be/internal/domain/storage"
)

// Ensure noopUploader implements storage.FileUploader.
var _ storage.FileUploader = (*noopUploader)(nil)

// noopUploader returns placeholder URLs for development when GDrive is not configured.
type noopUploader struct{}

func (n *noopUploader) Upload(ctx context.Context, filename string, data []byte, contentType string) (string, error) {
	return fmt.Sprintf("https://example.com/placeholder/%s", filename), nil
}
