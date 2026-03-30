package uploadserver

import (
	"context"
	"io"
	"strings"

	"github.com/dreamers-be/internal/domain/storage"
	uploadsvc "github.com/dreamers-be/internal/service/upload"
)

// UploadServer is a thin application layer that normalizes upload inputs before calling the service.
type UploadServer struct {
	uc *uploadsvc.UploadService
}

func NewUploadServer(uc *uploadsvc.UploadService) *UploadServer {
	return &UploadServer{uc: uc}
}

// Upload accepts already-parsed multipart file data (reader), and normalizes folder selection.
func (s *UploadServer) Upload(
	ctx context.Context,
	filename string,
	r io.Reader,
	contentType string,
	rawType string,
	rawFolder string,
	rawUploadType string,
) (string, error) {
	// Accept type from multiple form fields (type, folder, uploadType) for client flexibility.
	folder := rawType
	if folder == "" {
		folder = rawFolder
	}
	if folder == "" {
		folder = rawUploadType
	}

	folder = strings.TrimSpace(strings.ToLower(folder))
	switch folder {
	case "profile_photo", "profilephoto", "profile-photo":
		folder = storage.FolderProfilePhoto
	case "aadhar", "aadharcard", "aadhar-card", "aadhar_card":
		folder = storage.FolderAadhar
	default:
		if folder != storage.FolderProfilePhoto && folder != storage.FolderAadhar {
			folder = "uploads"
		}
	}

	return s.uc.Upload(ctx, filename, r, contentType, folder)
}

