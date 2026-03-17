package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configWithReadonly loads a minimal config with the given readonly value.
// It restores the original config when the test ends.
func configWithReadonly(t *testing.T, readonly bool) {
	t.Helper()
	orig := config.GetConfig()
	readonlyStr := "false"
	if readonly {
		readonlyStr = "true"
	}
	yaml := "journal_path: main.ledger\ndb_path: paisa.db\nreadonly: " + readonlyStr
	require.NoError(t, config.LoadConfig([]byte(yaml), ""))
	t.Cleanup(func() {
		_ = config.LoadConfig([]byte("journal_path: "+orig.JournalPath+"\ndb_path: "+orig.DBPath), "")
	})
}

func TestReadonlyMiddleware_AllowsWhenDisabled(t *testing.T) {
	configWithReadonly(t, false)

	router := gin.New()
	router.POST("/test", ReadonlyMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestReadonlyMiddleware_BlocksWhenEnabled(t *testing.T) {
	configWithReadonly(t, true)

	router := gin.New()
	router.POST("/test", ReadonlyMiddleware(), func(c *gin.Context) {
		t.Error("handler must not be reached in readonly mode")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeReadonly, detail.Code)
	assert.NotEmpty(t, detail.Message)
}

func TestReadonlyMiddleware_ErrorShape(t *testing.T) {
	configWithReadonly(t, true)

	router := gin.New()
	router.POST("/test", ReadonlyMiddleware(), func(c *gin.Context) {})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Top-level must have exactly one key: "error".
	var top map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&top))
	assert.Len(t, top, 1)
	_, hasError := top["error"]
	assert.True(t, hasError)
}

func TestErrCodeReadonly_Value(t *testing.T) {
	assert.Equal(t, ErrorCode("READONLY"), ErrCodeReadonly)
}
