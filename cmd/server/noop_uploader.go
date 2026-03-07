package main

import (
	"context"
	"fmt"

	"github.com/dreamers-be/internal/domain/storage"
)

// Ensure noopUploader implements storage.FileUploader.
var _ storage.FileUploader = (*noopUploader)(nil)

// noopUploader returns keys for development when S3 is not configured (no actual upload).
type noopUploader struct{}

func (n *noopUploader) Upload(ctx context.Context, filename string, data []byte, contentType string, folder string) (string, error) {
	if folder == "" {
		folder = "uploads"
	}
	return fmt.Sprintf("%s/%s", folder, filename), nil
}
