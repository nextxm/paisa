package server

import (
	"net/http"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/gin-gonic/gin"
)

type providerHTTPDebugRequest struct {
	Enabled bool `json:"enabled"`
}

func UpdateProviderHTTPDebug() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body providerHTTPDebugRequest
		if !BindJSONOrError(c, &body) {
			return
		}

		cfg := config.GetConfig()
		cfg.ProviderDebugHTTP = body.Enabled
		if err := config.SaveConfigObject(cfg); err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "enabled": body.Enabled})
	}
}
