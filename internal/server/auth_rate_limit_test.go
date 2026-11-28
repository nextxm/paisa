package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/session"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenAuthMiddleware_RateLimitScopedByClientIP(t *testing.T) {
	db := openTestDB(t)
	t.Cleanup(func() {
		_ = config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db\n"), "")
	})

	yaml := "journal_path: main.ledger\ndb_path: paisa.db\nuser_accounts:\n  - username: alice\n    password: sha256:" + strings.Repeat("0", 64) + "\n"
	require.NoError(t, config.LoadConfig([]byte(yaml), ""))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TokenAuthMiddleware(db))
	r.GET("/api/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	// Exhaust rate limit from IP 1.2.3.4
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
	reqBlocked := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	reqBlocked.RemoteAddr = "1.2.3.4:1234"
	wBlocked := httptest.NewRecorder()
	r.ServeHTTP(wBlocked, reqBlocked)
	assert.Equal(t, http.StatusTooManyRequests, wBlocked.Code)

	// Different IP with valid session must not be locked out
	s, err := session.Create(db, "alice")
	require.NoError(t, err)
	t.Cleanup(func() { invalidateSessionCache(s.Token) })

	reqOther := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	reqOther.RemoteAddr = "5.6.7.8:9999"
	reqOther.Header.Set("X-Auth", s.Token)
	wOther := httptest.NewRecorder()
	r.ServeHTTP(wOther, reqOther)
	assert.Equal(t, http.StatusOK, wOther.Code)

	// Invalidating session cache on logout removes cached validity
	invalidateSessionCache(s.Token)
	require.NoError(t, session.DeleteByToken(db, s.Token))
	wAfterLogout := httptest.NewRecorder()
	r.ServeHTTP(wAfterLogout, reqOther)
	assert.Equal(t, http.StatusUnauthorized, wAfterLogout.Code)
}
