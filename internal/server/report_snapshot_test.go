package server

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/report_snapshot"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportSnapshots_NetworthAndGainLifecycle(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)

	d := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	seedTransactions(t, db, []posting.Posting{
		{TransactionID: "t1", Date: d, Account: "Assets:Equity:NIFTY", Commodity: "INR", Amount: decimal.NewFromInt(1000), Quantity: decimal.NewFromInt(1000)},
		{TransactionID: "t1", Date: d, Account: "Income:Salary", Commodity: "INR", Amount: decimal.NewFromInt(-1000), Quantity: decimal.NewFromInt(-1000)},
	})

	require.NoError(t, markAllSnapshotsDirty(db))
	assert.True(t, isSnapshotDirty(db, snapshotKindNetworth))
	assert.True(t, isSnapshotDirty(db, snapshotKindGain))

	nwPayload, ok := getNetworthSnapshotPayload(db)
	require.True(t, ok)
	assert.NotEmpty(t, nwPayload)
	assert.False(t, isSnapshotDirty(db, snapshotKindNetworth))

	gainPayload, ok := getGainSnapshotPayload(db)
	require.True(t, ok)
	assert.NotEmpty(t, gainPayload)
	assert.False(t, isSnapshotDirty(db, snapshotKindGain))

	snap, err := report_snapshot.Get(db, networthSnapshotName)
	require.NoError(t, err)
	assert.Equal(t, networthSnapshotSchemaVersion, snap.SchemaVersion)
}

func TestReportSnapshots_PopulatesOnMissWithoutPriorSync(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)

	d := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	seedTransactions(t, db, []posting.Posting{
		{TransactionID: "t1", Date: d, Account: "Assets:Equity:NIFTY", Commodity: "INR", Amount: decimal.NewFromInt(2500), Quantity: decimal.NewFromInt(2500)},
		{TransactionID: "t1", Date: d, Account: "Income:Salary", Commodity: "INR", Amount: decimal.NewFromInt(-2500), Quantity: decimal.NewFromInt(-2500)},
	})

	// Neither dirty marker nor snapshot row exists yet.
	nwPayload, ok := getNetworthSnapshotPayload(db)
	require.True(t, ok)
	assert.NotEmpty(t, nwPayload)

	gainPayload, ok := getGainSnapshotPayload(db)
	require.True(t, ok)
	assert.NotEmpty(t, gainPayload)

	_, err := report_snapshot.Get(db, networthSnapshotName)
	require.NoError(t, err)
	_, err = report_snapshot.Get(db, gainSnapshotName)
	require.NoError(t, err)
}
