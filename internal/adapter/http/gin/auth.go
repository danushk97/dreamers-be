package gin

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BasicAuthCredentials holds hardcoded credentials for list players.
// TODO: move to config and proper auth
var BasicAuthCredentials = map[string]string{
	"admin": "dreamers-secret",
}

// BasicAuth returns middleware that validates Basic auth. Uses Error response on failure.
func BasicAuth(accounts map[string]string) func(*gin.Context) {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			Error(c, http.StatusUnauthorized, "Unauthorized", "missing Authorization header")
			c.Abort()
			return
		}

		const prefix = "Basic "
		if !strings.HasPrefix(auth, prefix) {
			Error(c, http.StatusUnauthorized, "Unauthorized", "invalid Authorization format")
			c.Abort()
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
		if err != nil {
			Error(c, http.StatusUnauthorized, "Unauthorized", "invalid credentials")
			c.Abort()
			return
		}

		pair := strings.SplitN(string(decoded), ":", 2)
		if len(pair) != 2 {
			Error(c, http.StatusUnauthorized, "Unauthorized", "invalid credentials")
			c.Abort()
			return
		}

		username, password := pair[0], pair[1]
		if expected, ok := accounts[username]; !ok || expected != password {
			Error(c, http.StatusUnauthorized, "Unauthorized", "invalid username or password")
			c.Abort()
			return
		}

		c.Next()
	}
}
