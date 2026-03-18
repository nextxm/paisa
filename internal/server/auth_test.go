package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/session"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configWithSingleAccount temporarily loads a minimal config that contains one
// user account for username with the given plain-text password.  It restores
// the original config when the test ends.
func configWithSingleAccount(t *testing.T, username, plainPassword string) {
	t.Helper()
	orig := config.GetConfig()
	hashed := "sha256:" + utils.Sha256(plainPassword)
	yaml := fmt.Sprintf(
		"journal_path: main.ledger\ndb_path: paisa.db\nuser_accounts:\n  - username: %s\n    password: %s\n",
		username, hashed,
	)
	require.NoError(t, config.LoadConfig([]byte(yaml), ""))
	t.Cleanup(func() {
		_ = config.LoadConfig(
			[]byte(fmt.Sprintf("journal_path: %s\ndb_path: %s", orig.JournalPath, orig.DBPath)),
			"",
		)
	})
}

// --- Login handler tests ---

func TestLogin_MissingFields(t *testing.T) {
	db := openTestDB(t)
	configWithSingleAccount(t, "alice", "secret")

	router := gin.New()
	router.POST("/api/auth/login", Login(db))

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

func TestLogin_WrongPassword(t *testing.T) {
	db := openTestDB(t)
	configWithSingleAccount(t, "alice", "secret")

	router := gin.New()
	router.POST("/api/auth/login", Login(db))

	body := strings.NewReader(`{"username":"alice","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeUnauthorized, detail.Code)
}

func TestLogin_UnknownUser(t *testing.T) {
	db := openTestDB(t)
	configWithSingleAccount(t, "alice", "secret")

	router := gin.New()
	router.POST("/api/auth/login", Login(db))

	body := strings.NewReader(`{"username":"bob","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogin_Success_IssuesTokenAndExpiry(t *testing.T) {
	db := openTestDB(t)
	configWithSingleAccount(t, "alice", "secret")

	router := gin.New()
	router.POST("/api/auth/login", Login(db))

	body := strings.NewReader(`{"username":"alice","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	before := time.Now()
	router.ServeHTTP(rec, req)
	after := time.Now()

	require.Equal(t, http.StatusOK, rec.Code)

	var resp loginResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

	assert.NotEmpty(t, resp.Token, "token must not be empty")
	assert.Equal(t, "alice", resp.Username)
	// ExpiresAt should be roughly 24 h from now.
	assert.True(t, resp.ExpiresAt.After(before.Add(23*time.Hour)), "expiry should be ~24 h from now")
	assert.True(t, resp.ExpiresAt.Before(after.Add(25*time.Hour)), "expiry should be ~24 h from now")
}

func TestLogin_Success_PersistsSession(t *testing.T) {
	db := openTestDB(t)
	configWithSingleAccount(t, "alice", "secret")

	router := gin.New()
	router.POST("/api/auth/login", Login(db))

	body := strings.NewReader(`{"username":"alice","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp loginResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

	// The token must be findable in the database.
	s, err := session.FindByToken(db, resp.Token)
	require.NoError(t, err)
	assert.Equal(t, "alice", s.Username)
	assert.True(t, s.ExpiresAt.After(time.Now()))
}

// --- TokenAuthMiddleware session-token acceptance tests ---

func TestTokenAuthMiddleware_SessionToken_Accepted(t *testing.T) {
	db := openTestDB(t)
	configWithSingleAccount(t, "alice", "secret")

	// Create a session directly.
	s, err := session.Create(db, "alice")
	require.NoError(t, err)

	router := gin.New()
	router.Use(TokenAuthMiddleware(db))
	router.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("X-Auth", s.Token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTokenAuthMiddleware_InvalidSessionToken_Rejected(t *testing.T) {
	db := openTestDB(t)
	configWithSingleAccount(t, "alice", "secret")

	router := gin.New()
	router.Use(TokenAuthMiddleware(db))
	router.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("X-Auth", "00000000-0000-0000-0000-000000000000")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestTokenAuthMiddleware_LoginRoute_Bypasses(t *testing.T) {
	db := openTestDB(t)
	configWithSingleAccount(t, "alice", "secret")

	router := gin.New()
	router.Use(TokenAuthMiddleware(db))
	router.POST("/api/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	// Deliberately no X-Auth header.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// configWithSingleAccountAndLegacyAuth is like configWithSingleAccount but also
// sets allow_legacy_auth: true so that the legacy username:password X-Auth path
// is enabled.
func configWithSingleAccountAndLegacyAuth(t *testing.T, username, plainPassword string) {
	t.Helper()
	orig := config.GetConfig()
	hashed := "sha256:" + utils.Sha256(plainPassword)
	yaml := fmt.Sprintf(
		"journal_path: main.ledger\ndb_path: paisa.db\nallow_legacy_auth: true\nuser_accounts:\n  - username: %s\n    password: %s\n",
		username, hashed,
	)
	require.NoError(t, config.LoadConfig([]byte(yaml), ""))
	t.Cleanup(func() {
		_ = config.LoadConfig(
			[]byte(fmt.Sprintf("journal_path: %s\ndb_path: %s", orig.JournalPath, orig.DBPath)),
			"",
		)
	})
}

// --- Legacy auth flag tests ---

// TestTokenAuthMiddleware_LegacyAuth_AcceptedWhenEnabled verifies that a valid
// legacy username:password credential in X-Auth is accepted when AllowLegacyAuth
// is true.
func TestTokenAuthMiddleware_LegacyAuth_AcceptedWhenEnabled(t *testing.T) {
	db := openTestDB(t)
	configWithSingleAccountAndLegacyAuth(t, "alice", "secret")

	router := gin.New()
	router.Use(TokenAuthMiddleware(db))
	router.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("X-Auth", "alice:secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestTokenAuthMiddleware_LegacyAuth_RejectedWhenDisabled verifies that a
// legacy username:password credential in X-Auth is rejected when AllowLegacyAuth
// is false (the default).
func TestTokenAuthMiddleware_LegacyAuth_RejectedWhenDisabled(t *testing.T) {
	db := openTestDB(t)
	// AllowLegacyAuth defaults to false when not set in YAML.
	configWithSingleAccount(t, "alice", "secret")

	router := gin.New()
	router.Use(TokenAuthMiddleware(db))
	router.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("X-Auth", "alice:secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestTokenAuthMiddleware_ExpiredSession_Rejected verifies that the middleware
// rejects a session token whose ExpiresAt timestamp is in the past, even
// though the token exists in the database.
func TestTokenAuthMiddleware_ExpiredSession_Rejected(t *testing.T) {
	db := openTestDB(t)
	configWithSingleAccount(t, "alice", "secret")

	// Insert an already-expired session directly – session.Create always sets a
	// future expiry, so we bypass it here to simulate a stale record.
	expired := &session.Session{
		Token:     "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Username:  "alice",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, db.Create(expired).Error)

	router := gin.New()
	router.Use(TokenAuthMiddleware(db))
	router.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("X-Auth", expired.Token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeUnauthorized, detail.Code)
}

// TestLogout_InvalidatesSession verifies that after POST /api/auth/logout the
// session token is immediately invalidated: subsequent requests with the same
// token must be rejected with 401 Unauthorized.
func TestLogout_InvalidatesSession(t *testing.T) {
	db := openTestDB(t)
	configWithSingleAccount(t, "alice", "secret")

	router := gin.New()
	router.Use(TokenAuthMiddleware(db))
	router.POST("/api/auth/login", Login(db))
	router.POST("/api/auth/logout", Logout(db))
	router.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// 1. Obtain a session token via login.
	body := strings.NewReader(`{"username":"alice","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var loginResp loginResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&loginResp))
	token := loginResp.Token
	require.NotEmpty(t, token)

	// 2. Confirm the token grants access before logout.
	req = httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("X-Auth", token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "token must be valid before logout")

	// 3. Log out – this deletes the session from the database.
	req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("X-Auth", token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// 4. The same token must now be rejected immediately.
	req = httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("X-Auth", token)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code,
		"token must be invalidated immediately after logout")
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeUnauthorized, detail.Code)
}

// TestTokenAuthMiddleware_SessionToken_WorksRegardlessOfLegacyFlag verifies
// that a valid session token is accepted regardless of the AllowLegacyAuth flag.
func TestTokenAuthMiddleware_SessionToken_WorksRegardlessOfLegacyFlag(t *testing.T) {
	for _, legacyEnabled := range []bool{false, true} {
		legacyEnabled := legacyEnabled
		t.Run(fmt.Sprintf("legacy_auth=%v", legacyEnabled), func(t *testing.T) {
			db := openTestDB(t)
			if legacyEnabled {
				configWithSingleAccountAndLegacyAuth(t, "alice", "secret")
			} else {
				configWithSingleAccount(t, "alice", "secret")
			}

			s, err := session.Create(db, "alice")
			require.NoError(t, err)

			router := gin.New()
			router.Use(TokenAuthMiddleware(db))
			router.GET("/api/ping", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
			req.Header.Set("X-Auth", s.Token)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}
