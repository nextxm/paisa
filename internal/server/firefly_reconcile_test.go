package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestFireflyReconcileHandler_Forbidden(t *testing.T) {
	db := openTestDB(t)
	config.LoadConfig([]byte(`
journal_path: /tmp/test.journal
db_path: /tmp/test.db
labs:
  firefly_reconcile: false
`), "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FireflyReconcileHandler(db)(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
