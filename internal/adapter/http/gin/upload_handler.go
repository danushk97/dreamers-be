package gin

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dreamers-be/internal/domain/storage"
	uploadsrv "github.com/dreamers-be/internal/server/upload"
)

// UploadHandler handles file upload endpoints.
type UploadHandler struct {
	server    *uploadsrv.UploadServer
	presigner storage.Presigner // optional, for S3 presigned URLs
}

// NewUploadHandler returns a new upload handler.
func NewUploadHandler(server *uploadsrv.UploadServer, presigner storage.Presigner) *UploadHandler {
	return &UploadHandler{server: server, presigner: presigner}
}

// Upload accepts a multipart file, uploads to S3, and returns key + presigned URL.
// POST /api/v1/upload
// Form: file (required), type (optional): "profile_photo" | "aadhar"
// Response: {"key": "profile_photo/...", "url": "https://...presigned..."}
func (h *UploadHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		log.Printf("Upload: file required but missing")
		Error(c, http.StatusBadRequest, "Bad Request", "file is required")
		return
	}
	defer file.Close()

	filename := header.Filename
	if filename == "" {
		filename = "upload"
	}

	rawType := c.PostForm("type")
	rawFolder := c.PostForm("folder")
	rawUploadType := c.PostForm("uploadType")

	key, err := h.server.Upload(
		c.Request.Context(),
		filename,
		file,
		header.Header.Get("Content-Type"),
		rawType,
		rawFolder,
		rawUploadType,
	)
	if err != nil {
		log.Printf("Upload error filename=%s type=%s folder=%s uploadType=%s: %v", filename, rawType, rawFolder, rawUploadType, err)
		Error(c, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred")
		return
	}

	resp := gin.H{"key": key}

	if h.presigner != nil && (strings.HasPrefix(key, "profile_photo/") || strings.HasPrefix(key, "aadhar/") || strings.HasPrefix(key, "uploads/")) {
		url, err := h.presigner.Presign(c.Request.Context(), key, 1*time.Hour)
		if err == nil {
			resp["url"] = url
		} else {
			log.Printf("Upload: presign failed for key=%s: %v", key, err)
			resp["url"] = key
		}
	} else {
		resp["url"] = key
	}

	log.Printf("Upload success filename=%s key=%s", filename, key)
	c.JSON(http.StatusOK, resp)
}
