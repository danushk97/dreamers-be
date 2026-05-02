package gin

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BasicAuthCredentials holds admin credentials for Basic auth (player list, mutating auction APIs).
// TODO: move to config and proper auth
var BasicAuthCredentials = map[string]string{
	"admin": "johnny@123",
}

func checkBasicAuth(authHeader string, accounts map[string]string) (ok bool, reason string) {
	if authHeader == "" {
		return false, "missing Authorization header"
	}
	const prefix = "Basic "
	if !strings.HasPrefix(authHeader, prefix) {
		return false, "invalid Authorization format"
	}
	decoded, err := base64.StdEncoding.DecodeString(authHeader[len(prefix):])
	if err != nil {
		return false, "invalid credentials"
	}
	pair := strings.SplitN(string(decoded), ":", 2)
	if len(pair) != 2 {
		return false, "invalid credentials"
	}
	username, password := pair[0], pair[1]
	if expected, found := accounts[username]; !found || expected != password {
		return false, "invalid username or password"
	}
	return true, ""
}

// ValidBasicAuthCredentials reports whether the request's Authorization header matches accounts
// (same rules as BasicAuth middleware). Does not write a response.
func ValidBasicAuthCredentials(c *gin.Context, accounts map[string]string) bool {
	ok, _ := checkBasicAuth(c.GetHeader("Authorization"), accounts)
	return ok
}

// BasicAuth returns middleware that validates Basic auth. Uses Error response on failure.
func BasicAuth(accounts map[string]string) func(*gin.Context) {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		ok, reason := checkBasicAuth(auth, accounts)
		if ok {
			c.Next()
			return
		}
		log.Printf("BasicAuth: %s", reason)
		Error(c, http.StatusUnauthorized, "Unauthorized", reason)
		c.Abort()
	}
}
