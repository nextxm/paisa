package server

// Integration tests for the readonly policy and error envelope contract.
//
// These tests build the full router via Build() and verify end-to-end behaviour
// so that CI fails whenever either contract is broken.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openTestDB returns an in-memory SQLite database suitable for integration tests.
// The session table is migrated so that auth tests also work correctly.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	require.NoError(t, err, "failed to open in-memory SQLite database")
	require.NoError(t, db.AutoMigrate(&session.Session{}), "failed to migrate session table")
	return db
}

// loadTestConfig sets up a minimal config with the given readonly flag and
// restores the previous config when the test ends.
func loadTestConfig(t *testing.T, readonly bool) {
	t.Helper()
	orig := config.GetConfig()

	readonlyStr := "false"
	if readonly {
		readonlyStr = "true"
	}
	yaml := "journal_path: main.ledger\ndb_path: paisa.db\nreadonly: " + readonlyStr
	require.NoError(t, config.LoadConfig([]byte(yaml), ""), "loadTestConfig: LoadConfig failed")

	t.Cleanup(func() {
		_ = config.LoadConfig([]byte("journal_path: "+orig.JournalPath+"\ndb_path: "+orig.DBPath), "")
	})
}

// writeEndpoints lists every mutating endpoint that must be protected by the
// readonly policy.  Each entry is (method, path, body).
var writeEndpoints = []struct {
	method string
	path   string
	body   string
}{
	{http.MethodPost, "/api/config", `{"journal_path":"main.ledger"}`},
	{http.MethodPost, "/api/init", ``},
	{http.MethodPost, "/api/sync", `{"journal":false}`},
	{http.MethodPost, "/api/price/delete", ``},
	{http.MethodPost, "/api/price/providers/delete/test", ``},
	{http.MethodPost, "/api/editor/save", `{"name":"main.ledger","content":""}`},
	{http.MethodPost, "/api/sheets/save", `{"name":"main.paisa","content":""}`},
	{http.MethodPost, "/api/templates/upsert", `{"name":"t","content":""}`},
	{http.MethodPost, "/api/templates/delete", `{"name":"t"}`},
}

// ---------------------------------------------------------------------------
// Readonly policy integration tests
// ---------------------------------------------------------------------------

// TestIntegration_ReadonlyPolicy_AllWriteEndpointsReturn403 verifies that every
// mutating endpoint returns HTTP 403 Forbidden (not 200 or 404) when the server
// is configured with readonly: true.
func TestIntegration_ReadonlyPolicy_AllWriteEndpointsReturn403(t *testing.T) {
	loadTestConfig(t, true)
	db := openTestDB(t)
	router := Build(db, false)

	for _, ep := range writeEndpoints {
		ep := ep // capture
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var bodyReader *strings.Reader
			if ep.body != "" {
				bodyReader = strings.NewReader(ep.body)
			} else {
				bodyReader = strings.NewReader("")
			}

			req := httptest.NewRequest(ep.method, ep.path, bodyReader)
			if ep.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusForbidden, rec.Code,
				"endpoint %s %s must return 403 in readonly mode", ep.method, ep.path)
		})
	}
}

// TestIntegration_ReadonlyPolicy_ErrorEnvelopeShape verifies that every write
// endpoint returns the canonical error envelope when blocked by readonly mode.
func TestIntegration_ReadonlyPolicy_ErrorEnvelopeShape(t *testing.T) {
	loadTestConfig(t, true)
	db := openTestDB(t)
	router := Build(db, false)

	for _, ep := range writeEndpoints {
		ep := ep // capture
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var bodyReader *strings.Reader
			if ep.body != "" {
				bodyReader = strings.NewReader(ep.body)
			} else {
				bodyReader = strings.NewReader("")
			}

			req := httptest.NewRequest(ep.method, ep.path, bodyReader)
			if ep.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusForbidden, rec.Code)

			// Top-level must contain exactly one key: "error".
			var top map[string]json.RawMessage
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&top),
				"response body must be valid JSON")
			assert.Len(t, top, 1, "error envelope must have exactly one top-level key")
			rawErr, ok := top["error"]
			assert.True(t, ok, "top-level key must be \"error\"")

			// The nested error object must have "code" and "message".
			var detail ErrorDetail
			require.NoError(t, json.Unmarshal(rawErr, &detail),
				"\"error\" value must unmarshal into ErrorDetail")
			assert.Equal(t, ErrCodeReadonly, detail.Code,
				"error code must be READONLY")
			assert.NotEmpty(t, detail.Message,
				"error message must not be empty")
		})
	}
}

// TestIntegration_ReadonlyPolicy_ReadEndpointsUnaffected verifies that read-only
// (GET) endpoints continue to work normally when readonly mode is enabled.
func TestIntegration_ReadonlyPolicy_ReadEndpointsUnaffected(t *testing.T) {
	loadTestConfig(t, true)
	db := openTestDB(t)
	router := Build(db, false)

	readEndpoints := []string{
		"/api/ping",
		"/api/config",
	}

	for _, path := range readEndpoints {
		path := path
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code,
				"GET %s must not be blocked by readonly mode", path)
		})
	}
}

// ---------------------------------------------------------------------------
// Error envelope contract integration tests
// ---------------------------------------------------------------------------

// TestIntegration_ErrorEnvelope_InvalidJSONOnWriteEndpoints verifies that
// malformed request bodies on write endpoints produce a 400 INVALID_REQUEST
// error with the canonical envelope shape, not a 403 READONLY, and that the
// server is running in non-readonly mode.
func TestIntegration_ErrorEnvelope_InvalidJSONOnWriteEndpoints(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	router := Build(db, false)

	// These endpoints use BindJSONOrError and must return 400 on bad input.
	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/sync"},
		{http.MethodPost, "/api/editor/save"},
		{http.MethodPost, "/api/sheets/save"},
		{http.MethodPost, "/api/templates/upsert"},
		{http.MethodPost, "/api/templates/delete"},
	}

	for _, ep := range endpoints {
		ep := ep
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(`{not json}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code,
				"endpoint %s %s must return 400 on malformed JSON", ep.method, ep.path)

			var top map[string]json.RawMessage
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&top))
			assert.Len(t, top, 1)
			_, hasError := top["error"]
			assert.True(t, hasError, "response must contain top-level \"error\" key")

			rawErr := top["error"]
			var detail ErrorDetail
			require.NoError(t, json.Unmarshal(rawErr, &detail))
			assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
			assert.NotEmpty(t, detail.Message)
		})
	}
}

// TestIntegration_ErrorEnvelope_ContentTypeJSON verifies that error responses
// always carry the application/json content type.
func TestIntegration_ErrorEnvelope_ContentTypeJSON(t *testing.T) {
	loadTestConfig(t, true)
	db := openTestDB(t)
	router := Build(db, false)

	// Pick one representative write endpoint.
	req := httptest.NewRequest(http.MethodPost, "/api/sync", strings.NewReader(`{"journal":false}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	ct := rec.Header().Get("Content-Type")
	assert.Contains(t, ct, "application/json",
		"error responses must have Content-Type: application/json")
}

// TestIntegration_ErrorEnvelope_CodeIsString verifies that the "code" field is
// always a JSON string (not a number or null), matching the ErrorCode type.
func TestIntegration_ErrorEnvelope_CodeIsString(t *testing.T) {
	loadTestConfig(t, true)
	db := openTestDB(t)
	router := Build(db, false)

	req := httptest.NewRequest(http.MethodPost, "/api/sync", strings.NewReader(`{"journal":false}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)

	var top map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&top))

	var errObj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(top["error"], &errObj))

	codeRaw, ok := errObj["code"]
	assert.True(t, ok, "\"code\" field must be present in error object")

	// A JSON string starts with '"'.
	assert.Equal(t, byte('"'), codeRaw[0], "error.code must be a JSON string")
}
