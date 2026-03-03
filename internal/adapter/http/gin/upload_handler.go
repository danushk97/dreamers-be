package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dreamers-be/internal/usecase/upload"
)

// UploadHandler handles file upload endpoints.
type UploadHandler struct {
	uc *upload.UploadUseCase
}

// NewUploadHandler returns a new upload handler.
func NewUploadHandler(uc *upload.UploadUseCase) *UploadHandler {
	return &UploadHandler{uc: uc}
}

// Upload accepts a multipart file and returns its public URL.
// POST /api/v1/upload
// Form: file (required)
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

	url, err := h.uc.Upload(c.Request.Context(), filename, file, header.Header.Get("Content-Type"))
	if err != nil {
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}
