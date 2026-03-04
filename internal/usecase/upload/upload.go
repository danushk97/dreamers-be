package upload

import (
	"context"
	"fmt"
	"io"

	"github.com/dreamers-be/internal/domain/storage"
)

// UploadUseCase handles file uploads.
type UploadUseCase struct {
	uploader storage.FileUploader
	maxMB    int64
}

// NewUploadUseCase returns a new upload use case.
func NewUploadUseCase(uploader storage.FileUploader, maxMB int64) *UploadUseCase {
	if maxMB <= 0 {
		maxMB = 2
	}
	return &UploadUseCase{uploader: uploader, maxMB: maxMB}
}

// Upload reads from r, uploads to storage, and returns the object key.
// folder: "profile_photo", "aadhar", or empty for "uploads".
func (uc *UploadUseCase) Upload(ctx context.Context, filename string, r io.Reader, contentType string, folder string) (string, error) {
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
