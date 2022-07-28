// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities_test

import (
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
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
	}

	vegaTime := entities.NanosToPostgresTimestamp(now.UnixNano())

	reward, err := entities.RewardFromProto(pbReward, vegaTime)
	require.NoError(t, err)
	assert.Equal(t, "a0b1", reward.PartyID.String())
	assert.Equal(t, "c2d3", reward.AssetID.String())
	assert.Equal(t, int64(42), reward.EpochID)
	assert.Equal(t, entities.NanosToPostgresTimestamp(timestamp.UnixNano()), reward.Timestamp)
	assert.InDelta(t, 3.14, reward.PercentOfTotal, 0.001)
	fmt.Printf("%v - %v\n", now, reward.VegaTime)
	assert.True(t, vegaTime.Equal(reward.VegaTime))

}
