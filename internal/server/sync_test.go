package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configWithBrokenJournalPath points the config at a non-existent journal file
// so that ledger CLI validation fails immediately.  It restores the original
// config when the test ends.  This helper is shared by the sync-rollback
// regression tests to avoid duplicating setup/teardown logic.
func configWithBrokenJournalPath(t *testing.T) {
	t.Helper()
	orig := config.GetConfig()
	yaml := "journal_path: /nonexistent/path/main.ledger\ndb_path: paisa.db"
	require.NoError(t, config.LoadConfig([]byte(yaml), ""))
	t.Cleanup(func() {
		_ = config.LoadConfig(
			[]byte(fmt.Sprintf("journal_path: %s\ndb_path: %s", orig.JournalPath, orig.DBPath)),
			"",
		)
	})
}

// TestSync_SuccessResponseShape verifies that a Sync call with no stages
// requested returns a success response that includes posting_count and
// price_count so operators always have the diagnostic fields available.
func TestSync_SuccessResponseShape(t *testing.T) {
	db := openTestDB(t)

	// No sync flags set – all stages are skipped; function must return success.
	result := Sync(db, SyncRequest{})

	raw, err := json.Marshal(result)
	require.NoError(t, err)

	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &top))

	assert.Contains(t, top, "success", "response must include 'success' field")
	assert.Contains(t, top, "posting_count", "response must include 'posting_count' diagnostic field")
	assert.Contains(t, top, "price_count", "response must include 'price_count' diagnostic field")
	assert.NotContains(t, top, "failed_stage", "success response must not include 'failed_stage'")

	var success bool
	require.NoError(t, json.Unmarshal(top["success"], &success))
	assert.True(t, success)

	var postingCount int
	require.NoError(t, json.Unmarshal(top["posting_count"], &postingCount))
	assert.Equal(t, 0, postingCount, "posting_count must be zero when journal sync was not requested")

	var priceCount int
	require.NoError(t, json.Unmarshal(top["price_count"], &priceCount))
	assert.Equal(t, 0, priceCount, "price_count must be zero when journal sync was not requested")
}

// TestSync_JournalFailureResponseShape verifies that when the journal stage is
// requested but the ledger CLI is unavailable or the journal file is missing,
// the Sync function returns a structured failure response that:
//   - sets success=false
//   - includes a non-empty failed_stage so operators know which step broke
//   - includes a non-empty message with the underlying error
//
// This is the server-level regression guard for the sync-rollback scenario:
// a failed journal parse must never silently succeed.
func TestSync_JournalFailureResponseShape(t *testing.T) {
	db := openTestDB(t)
	// Point config at a journal file that does not exist on disk.  The ledger
	// CLI (or its absence) will fail during validation, exercising the failure
	// branch of Sync without needing a real ledger binary.
	configWithBrokenJournalPath(t)

	result := Sync(db, SyncRequest{Journal: true})

	raw, err := json.Marshal(result)
	require.NoError(t, err)

	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &top))

	assert.Contains(t, top, "success", "failure response must include 'success' field")

	var success bool
	require.NoError(t, json.Unmarshal(top["success"], &success))
	assert.False(t, success, "success must be false when journal sync fails")

	assert.Contains(t, top, "failed_stage", "failure response must include 'failed_stage'")
	var failedStage string
	require.NoError(t, json.Unmarshal(top["failed_stage"], &failedStage))
	assert.NotEmpty(t, failedStage, "failed_stage must identify which stage broke")

	assert.Contains(t, top, "message", "failure response must include 'message' with error details")
	var message string
	require.NoError(t, json.Unmarshal(top["message"], &message))
	assert.NotEmpty(t, message, "message must contain the underlying error text")
}

// TestIntegration_SyncFailure_HTTPResponseShape verifies end-to-end that the
// POST /api/sync HTTP endpoint returns HTTP 200 with a JSON body containing
// success=false, failed_stage, and message when journal sync fails.  HTTP 200
// is correct here: the transport succeeded; only the sync operation itself
// failed, and the caller must inspect the body to detect that.
func TestIntegration_SyncFailure_HTTPResponseShape(t *testing.T) {
	// Use a non-existent journal path so the ledger CLI fails immediately.
	configWithBrokenJournalPath(t)

	db := openTestDB(t)
	router := Build(db, false)

	req := httptest.NewRequest(http.MethodPost, "/api/sync",
		strings.NewReader(`{"journal":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Transport must succeed (200).
	assert.Equal(t, http.StatusOK, rec.Code,
		"sync endpoint must return 200 even when sync fails (failure is in the body)")

	var top map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&top))

	var success bool
	require.NoError(t, json.Unmarshal(top["success"], &success))
	assert.False(t, success, "body.success must be false when the journal stage fails")

	rawStage, hasStage := top["failed_stage"]
	assert.True(t, hasStage, "body must contain 'failed_stage'")
	var failedStage string
	require.NoError(t, json.Unmarshal(rawStage, &failedStage))
	assert.NotEmpty(t, failedStage, "failed_stage must name the broken stage")

	rawMsg, hasMsg := top["message"]
	assert.True(t, hasMsg, "body must contain 'message'")
	var message string
	require.NoError(t, json.Unmarshal(rawMsg, &message))
	assert.NotEmpty(t, message, "message must carry the underlying error text")
}
