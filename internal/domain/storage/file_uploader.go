package storage

import (
	"context"
	"time"
)

// Folder names for S3 uploads.
const (
	FolderProfilePhoto = "profile_photo"
	FolderAadhar       = "aadhar"
)

// FileUploader uploads files and returns a key (S3 object key) or URL for non-S3 backends.
// folder: "profile_photo" or "aadhar"; empty defaults to "uploads".
type FileUploader interface {
	Upload(ctx context.Context, filename string, data []byte, contentType string, folder string) (key string, err error)
}

// Presigner generates presigned URLs for reading objects by key.
// S3 implementations use this for private objects.
type Presigner interface {
	Presign(ctx context.Context, key string, expiry time.Duration) (url string, err error)
}
