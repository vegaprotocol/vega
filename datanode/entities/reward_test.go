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

// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewardFromProto(t *testing.T) {
	now := time.Now()
	timestamp := now.Add(-1 * time.Second)

	pbReward := eventspb.RewardPayoutEvent{
		Party:                "a0b1",
		Asset:                "c2d3",
		EpochSeq:             "42",
		Amount:               "123456789",
		PercentOfTotalReward: "3.14",
		Timestamp:            timestamp.UnixNano(),
		LockedUntilEpoch:     "44",
	}

	vegaTime := entities.NanosToPostgresTimestamp(now.UnixNano())

	reward, err := entities.RewardFromProto(pbReward, generateTxHash(), vegaTime, 1)
	require.NoError(t, err)
	assert.Equal(t, "a0b1", reward.PartyID.String())
	assert.Equal(t, "c2d3", reward.AssetID.String())
	assert.Equal(t, int64(42), reward.EpochID)
	assert.Equal(t, entities.NanosToPostgresTimestamp(timestamp.UnixNano()), reward.Timestamp)
	assert.InDelta(t, 3.14, reward.PercentOfTotal, 0.001)
	assert.True(t, vegaTime.Equal(reward.VegaTime))
	assert.Equal(t, uint64(1), reward.SeqNum)
}
