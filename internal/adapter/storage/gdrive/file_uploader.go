package gdrive

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"path"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github.com/dreamers-be/internal/domain/storage"
)

var _ storage.FileUploader = (*FileUploader)(nil)

// FileUploader uploads files to Google Drive and returns public view URLs.
type FileUploader struct {
	service    *drive.Service
	folderID   string
	maxSizeMB  int64
}

// Config holds Google Drive uploader configuration.
type Config struct {
	CredentialsJSON []byte // Service account JSON key
	FolderID        string // Optional: parent folder ID
	MaxSizeMB       int64  // Max file size in MB (default 2)
}

// NewFileUploader creates a new Google Drive file uploader.
func NewFileUploader(ctx context.Context, cfg Config) (*FileUploader, error) {
	if len(cfg.CredentialsJSON) == 0 {
		return nil, fmt.Errorf("credentials JSON is required")
	}
	if cfg.MaxSizeMB <= 0 {
		cfg.MaxSizeMB = 2
	}

	// Use drive.DriveFileScope for drive file access
	opts := []option.ClientOption{
		option.WithCredentialsJSON(cfg.CredentialsJSON),
		option.WithScopes(drive.DriveFileScope),
	}
	svc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}

	return &FileUploader{
		service:   svc,
		folderID:  cfg.FolderID,
		maxSizeMB: cfg.MaxSizeMB * 1024 * 1024,
	}, nil
}

// Upload uploads data to Google Drive and returns a public view URL.
func (u *FileUploader) Upload(ctx context.Context, filename string, data []byte, contentType string) (string, error) {
	if int64(len(data)) > u.maxSizeMB {
		return "", fmt.Errorf("file exceeds max size of %d MB", u.maxSizeMB/1024/1024)
	}

	// Infer content type from extension if not provided
	if contentType == "" {
		contentType = mime.TypeByExtension(path.Ext(filename))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	file := &drive.File{
		Name: sanitizeFilename(filename),
	}
	if u.folderID != "" {
		file.Parents = []string{u.folderID}
	}

	fileMeta, err := u.service.Files.Create(file).
		Context(ctx).
		Media(bytes.NewReader(data)).
		Do()
	if err != nil {
		return "", fmt.Errorf("upload to drive: %w", err)
	}

	// Share with "anyone with the link"
	_, err = u.service.Permissions.Create(fileMeta.Id, &drive.Permission{
		Type: "anyone",
		Role: "reader",
	}).Context(ctx).Do()
	if err != nil {
		_ = u.service.Files.Delete(fileMeta.Id).Context(ctx).Do() // cleanup on failure
		return "", fmt.Errorf("share file: %w", err)
	}

	// Return web view link
	return fmt.Sprintf("https://drive.google.com/file/d/%s/view?usp=sharing", fileMeta.Id), nil
}

func sanitizeFilename(name string) string {
	// Remove path components and control chars
	name = path.Base(name)
	var b strings.Builder
	for _, r := range name {
		if r > 0 && r < 32 {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
