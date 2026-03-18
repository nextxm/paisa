package server

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/session"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// loginRequest is the expected JSON body for POST /api/auth/login.
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// loginResponse is returned on a successful login.
type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Username  string    `json:"username"`
}

// Logout returns a handler for POST /api/auth/logout.
// It deletes the session identified by the X-Auth token (best-effort) so that
// the token can no longer be used even before its natural expiry.
func Logout(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("X-Auth")
		if token != "" {
			if err := session.DeleteByToken(db, token); err != nil {
				log.Warnf("logout: failed to delete session token: %v", err)
			}
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

// Login returns a handler for POST /api/auth/login.
// It validates the supplied credentials against the configured user accounts
// and, on success, creates a session in the database and returns the session
// token together with its expiry time.
func Login(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if !BindJSONOrError(c, &req) {
			return
		}

		hashed := utils.Sha256(req.Password)
		authenticated := false
		for _, ua := range config.GetConfig().UserAccounts {
			if subtle.ConstantTimeCompare([]byte(ua.Username), []byte(req.Username)) == 1 &&
				subtle.ConstantTimeCompare([]byte(ua.Password), []byte("sha256:"+hashed)) == 1 {
				authenticated = true
				break
			}
		}

		if !authenticated {
			AbortWithError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "Invalid username or password")
			return
		}

		s, err := session.Create(db, req.Username)
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, "Failed to create session")
			return
		}

		c.JSON(http.StatusOK, loginResponse{
			Token:     s.Token,
			ExpiresAt: s.ExpiresAt,
			Username:  s.Username,
		})
	}
}
