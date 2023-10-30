// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
