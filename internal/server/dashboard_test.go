package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ananthakumaran/paisa/internal/model/dashboard_snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBuildDashboardSnapshotPayload_MatchesLiveDashboardJSON(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)

	livePayload, err := json.Marshal(GetDashboard(db))
	require.NoError(t, err)

	snapshotPayload, err := buildDashboardSnapshotPayload(db)
	require.NoError(t, err)

	assert.JSONEq(t, string(livePayload), string(snapshotPayload))
}

func TestDashboardRoute_ReadsSnapshotWhenPresent(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	router := Build(db, false)

	payload := []byte(`{"snapshot":true}`)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return dashboard_snapshot.Replace(tx, payload)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, string(payload), rec.Body.String())
}

func TestDashboardRoute_FallsBackWhenSnapshotPayloadInvalid(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	router := Build(db, false)

	require.NoError(t, db.Create(&dashboard_snapshot.DashboardSnapshot{
		Name:          dashboard_snapshot.SnapshotName,
		SchemaVersion: dashboard_snapshot.SchemaVersion,
		Payload:       []byte(`{"broken"`),
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, json.Valid(rec.Body.Bytes()))

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body, "checkingBalances")
	assert.Contains(t, body, "networth")
	assert.Contains(t, body, "expenses")
	assert.Contains(t, body, "cashFlows")
	assert.Contains(t, body, "transactionSequences")
	assert.Contains(t, body, "transactions")
	assert.Contains(t, body, "budget")
	assert.Contains(t, body, "goalSummaries")
}
