package gin

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dreamers-be/internal/domain/storage"
	"github.com/dreamers-be/internal/usecase/upload"
)

// UploadHandler handles file upload endpoints.
type UploadHandler struct {
	uc        *upload.UploadUseCase
	presigner storage.Presigner // optional, for S3 presigned URLs
}

// NewUploadHandler returns a new upload handler.
func NewUploadHandler(uc *upload.UploadUseCase, presigner storage.Presigner) *UploadHandler {
	return &UploadHandler{uc: uc, presigner: presigner}
}

// Upload accepts a multipart file, uploads to S3, and returns key + presigned URL.
// POST /api/v1/upload
// Form: file (required), type (optional): "profile_photo" | "aadhar"
// Response: {"key": "profile_photo/...", "url": "https://...presigned..."}
func (h *UploadHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		Error(c, http.StatusBadRequest, "Bad Request", "file is required")
		return
	}
	defer file.Close()

	filename := header.Filename
	if filename == "" {
		filename = "upload"
	}

	// Accept type from multiple form fields (type, folder, uploadType) for client flexibility
	folder := c.PostForm("type")
	if folder == "" {
		folder = c.PostForm("folder")
	}
	if folder == "" {
		folder = c.PostForm("uploadType")
	}
	folder = strings.TrimSpace(strings.ToLower(folder))
	// Normalize common client values: profilePhoto, profile-photo -> profile_photo; aadharCard, aadhar-card -> aadhar
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

	key, err := h.uc.Upload(c.Request.Context(), filename, file, header.Header.Get("Content-Type"), folder)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred")
		return
	}

	resp := gin.H{"key": key}

	if h.presigner != nil && (strings.HasPrefix(key, "profile_photo/") || strings.HasPrefix(key, "aadhar/") || strings.HasPrefix(key, "uploads/")) {
		url, err := h.presigner.Presign(c.Request.Context(), key, 1*time.Hour)
		if err == nil {
			resp["url"] = url
		} else {
			resp["url"] = key
		}
	} else {
		resp["url"] = key
	}

	c.JSON(http.StatusOK, resp)
}
