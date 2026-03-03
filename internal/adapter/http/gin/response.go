package gin

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ErrorDetail follows RFC 7807 Problem Details for HTTP APIs.
type ErrorDetail struct {
	Type   string `json:"type,omitempty"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// ErrorResponse is the standard error response shape.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// Error writes a standardized error response.
func Error(c *gin.Context, status int, title, detail string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Type:   "https://api.dreamers.be/errors/" + strconv.Itoa(status),
			Title:  title,
			Status: status,
			Detail: detail,
		},
	})
}
