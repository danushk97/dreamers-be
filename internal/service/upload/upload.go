package uploadservice

import (
	"context"
	"fmt"
	"io"

	"github.com/dreamers-be/internal/domain/storage"
)

// UploadService handles file upload business rules (size limits).
type UploadService struct {
	uploader storage.FileUploader
	maxMB    int64
}

// NewUploadService returns a new upload service.
func NewUploadService(uploader storage.FileUploader, maxMB int64) *UploadService {
	if maxMB <= 0 {
		maxMB = 2
	}
	return &UploadService{uploader: uploader, maxMB: maxMB}
}

// Upload reads from r, uploads to storage, and returns the object key.
// folder must be already normalized by the server layer.
func (uc *UploadService) Upload(ctx context.Context, filename string, r io.Reader, contentType string, folder string) (string, error) {
	maxBytes := uc.maxMB * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return "", fmt.Errorf("file exceeds %d MB limit", uc.maxMB)
	}
	return uc.uploader.Upload(ctx, filename, data, contentType, folder)
}

