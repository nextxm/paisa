package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformanceTelemetryHeaders_OnTargetEndpoints(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	router := Build(db, false)

	endpoints := []string{
		"/api/config",
		"/api/dashboard",
		"/api/networth/projection",
	}

	for _, endpoint := range endpoints {
		endpoint := endpoint
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			assertNonNegativeIntHeader(t, rec, perfHeaderLatencyMS)
			assertNonNegativeIntHeader(t, rec, perfHeaderSQLCount)
			assertNonNegativeFloatHeader(t, rec, perfHeaderSQLTimeMS)
		})
	}
}

func assertNonNegativeIntHeader(t *testing.T, rec *httptest.ResponseRecorder, header string) {
	t.Helper()
	value := rec.Header().Get(header)
	require.NotEmpty(t, value, "missing %s header", header)
	parsed, err := strconv.ParseInt(value, 10, 64)
	require.NoError(t, err, "header %s must be an integer", header)
	assert.GreaterOrEqual(t, parsed, int64(0), "header %s must be non-negative", header)
}

func assertNonNegativeFloatHeader(t *testing.T, rec *httptest.ResponseRecorder, header string) {
	t.Helper()
	value := rec.Header().Get(header)
	require.NotEmpty(t, value, "missing %s header", header)
	parsed, err := strconv.ParseFloat(value, 64)
	require.NoError(t, err, "header %s must be numeric", header)
	assert.GreaterOrEqual(t, parsed, 0.0, "header %s must be non-negative", header)
}
