package snapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractIntervalFromViewDefinition(t *testing.T) {
	viewDefinition := ` SELECT balances.account_id,
	time_bucket('01:00:00'::interval, balances.vega_time) AS bucket,
		last(balances.balance, balances.vega_time) AS balance,
		last(balances.tx_hash, balances.vega_time) AS tx_hash,
		last(balances.vega_time, balances.vega_time) AS vega_time
	FROM balances
	GROUP BY balances.account_id, (time_bucket('01:00:00'::interval, balances.vega_time));`

	interval, err := extractIntervalFromViewDefinition(viewDefinition)
	require.NoError(t, err)
	assert.Equal(t, "01:00:00", interval)

	viewDefinition = ` SELECT balances.account_id,
	time_bucket('1 day'::interval, balances.vega_time) AS bucket,
		last(balances.balance, balances.vega_time) AS balance,
		last(balances.tx_hash, balances.vega_time) AS tx_hash,
		last(balances.vega_time, balances.vega_time) AS vega_time
	FROM balances
	GROUP BY balances.account_id, (time_bucket('1 day'::interval, balances.vega_time));`

	interval, err = extractIntervalFromViewDefinition(viewDefinition)
	require.NoError(t, err)
	assert.Equal(t, "1 day", interval)
}
