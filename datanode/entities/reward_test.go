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
	gameID := "Test"
	pbReward := eventspb.RewardPayoutEvent{
		Party:                "a0b1",
		EpochSeq:             "42",
		Asset:                "c2d3",
		Amount:               "123456789",
		PercentOfTotalReward: "3.14",
		Timestamp:            timestamp.UnixNano(),
		RewardType:           "Some Type",
		GameId:               &gameID,
		LockedUntilEpoch:     "44",
		QuantumAmount:        "292929",
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
