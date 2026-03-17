package server

import (
	"net/http"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/gin-gonic/gin"
)

// ReadonlyMiddleware returns a Gin middleware that blocks mutating requests
// when the application is configured for readonly mode.
// Blocked requests receive HTTP 403 Forbidden with the READONLY error code.
func ReadonlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.GetConfig().Readonly {
			AbortWithError(c, http.StatusForbidden, ErrCodeReadonly, "write operations are disabled in readonly mode")
			return
		}
		c.Next()
	}
}
