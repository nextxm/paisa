package server

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
