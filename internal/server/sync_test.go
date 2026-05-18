package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	result, details := Sync(db, SyncRequest{}, nil)

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

	// When no sync stages are requested there should be no diagnostic details.
	assert.Empty(t, details, "no details expected when no sync stages were requested")
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

	result, _ := Sync(db, SyncRequest{Journal: true}, nil)

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

// TestIntegration_SyncAsync_Returns202WithJobID verifies end-to-end that the
// POST /api/sync HTTP endpoint returns HTTP 202 Accepted with a non-empty
// job_id.  The sync work is performed asynchronously in the background; the
// caller polls the job status via GET /api/jobs/:id rather than waiting for the
// response body to carry the outcome.
func TestIntegration_SyncAsync_Returns202WithJobID(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	router := Build(db, false)

	req := httptest.NewRequest(http.MethodPost, "/api/sync",
		strings.NewReader(`{"journal":false}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code,
		"async sync endpoint must return 202 Accepted")

	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	rawJobID, ok := body["job_id"]
	assert.True(t, ok, "response must contain 'job_id'")
	var jobID string
	require.NoError(t, json.Unmarshal(rawJobID, &jobID))
	assert.NotEmpty(t, jobID, "job_id must be non-empty")
}

// TestIntegration_SyncAsync_ForcePricesMetadata verifies that the optional
// force_prices request flag is accepted by POST /api/sync and preserved in the
// submitted job metadata for later inspection by the UI.
func TestIntegration_SyncAsync_ForcePricesMetadata(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	router := Build(db, false)

	req := httptest.NewRequest(http.MethodPost, "/api/sync",
		strings.NewReader(`{"prices":true,"force_prices":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)

	var syncBody map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&syncBody))
	var jobID string
	require.NoError(t, json.Unmarshal(syncBody["job_id"], &jobID))
	require.NotEmpty(t, jobID)

	jobReq := httptest.NewRequest(http.MethodGet, "/api/jobs/"+jobID, nil)
	jobRec := httptest.NewRecorder()
	router.ServeHTTP(jobRec, jobReq)

	require.Equal(t, http.StatusOK, jobRec.Code)

	var job struct {
		Metadata map[string]any `json:"metadata"`
	}
	require.NoError(t, json.NewDecoder(jobRec.Body).Decode(&job))
	require.NotNil(t, job.Metadata)
	assert.Equal(t, true, job.Metadata["force_prices"])
}

// TestIntegration_GetJob_ReturnsJobStatus verifies that GET /api/jobs/:id
// returns the job status for a job submitted via POST /api/sync.
func TestIntegration_GetJob_ReturnsJobStatus(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	router := Build(db, false)

	// Submit a sync job.
	req := httptest.NewRequest(http.MethodPost, "/api/sync",
		strings.NewReader(`{"journal":false}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusAccepted, rec.Code)

	var syncBody map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&syncBody))
	var jobID string
	require.NoError(t, json.Unmarshal(syncBody["job_id"], &jobID))
	require.NotEmpty(t, jobID)

	// Poll the job status endpoint.
	jobReq := httptest.NewRequest(http.MethodGet, "/api/jobs/"+jobID, nil)
	jobRec := httptest.NewRecorder()
	router.ServeHTTP(jobRec, jobReq)

	assert.Equal(t, http.StatusOK, jobRec.Code, "GET /api/jobs/:id must return 200")

	var job map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(jobRec.Body).Decode(&job))
	assert.Contains(t, job, "id", "job response must include 'id'")
	assert.Contains(t, job, "status", "job response must include 'status'")
}

// TestIntegration_GetJob_NotFound verifies that GET /api/jobs/:id returns 404
// for an unknown job ID.
func TestIntegration_GetJob_NotFound(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	router := Build(db, false)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs/does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code, "unknown job ID must return 404")
}

func TestIntegration_JobsStream_EmitsUpdates(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	router := Build(db, false)

	syncReq := httptest.NewRequest(http.MethodPost, "/api/sync", strings.NewReader(`{"journal":false}`))
	syncReq.Header.Set("Content-Type", "application/json")
	syncRec := httptest.NewRecorder()
	router.ServeHTTP(syncRec, syncReq)
	require.Equal(t, http.StatusAccepted, syncRec.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(syncRec.Body).Decode(&body))
	var jobID string
	require.NoError(t, json.Unmarshal(body["job_id"], &jobID))
	require.NotEmpty(t, jobID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streamReq := httptest.NewRequest(http.MethodGet, "/api/jobs/stream", nil).WithContext(ctx)
	streamRec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(streamRec, streamReq)
		close(done)
	}()

	assert.Eventually(t, func() bool {
		return strings.Contains(streamRec.Body.String(), `"id":"`+jobID+`"`)
	}, 2*time.Second, 10*time.Millisecond, "stream must emit at least one event for submitted job")

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stream handler did not stop after context cancellation")
	}

	assert.Equal(t, http.StatusOK, streamRec.Code)
	assert.Contains(t, streamRec.Header().Get("Content-Type"), "text/event-stream")
}

// TestIntegration_SyncAsync_FailureStillReturns202 verifies that even when the
// sync job would fail (e.g. journal not found), the HTTP layer still returns 202
// immediately.  The failure is observable asynchronously via the job status.
func TestIntegration_SyncAsync_FailureStillReturns202(t *testing.T) {
	configWithBrokenJournalPath(t)

	db := openTestDB(t)
	router := Build(db, false)

	req := httptest.NewRequest(http.MethodPost, "/api/sync",
		strings.NewReader(`{"journal":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code,
		"async sync endpoint must return 202 even when the sync job will fail")

	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	rawJobID, ok := body["job_id"]
	assert.True(t, ok, "response must contain 'job_id'")
	var jobID string
	require.NoError(t, json.Unmarshal(rawJobID, &jobID))
	assert.NotEmpty(t, jobID, "job_id must be non-empty")
}

// TestSync_DetailsReturnedOnSuccess verifies that Sync returns a valid
// details slice (nil is acceptable when there are no per-step failures) even
// on a fully successful run, and that no stages being requested produces no
// details.
func TestSync_DetailsReturnedOnSuccess(t *testing.T) {
	db := openTestDB(t)

	_, details := Sync(db, SyncRequest{}, nil)

	// When no sync stages are requested and there are no investment postings,
	// WarmXIRRCache is not called, so details is nil (same as empty).
	// Nil and empty slices are semantically identical for "no details".
	assert.Empty(t, details, "no details expected when no sync stages were requested")
}
