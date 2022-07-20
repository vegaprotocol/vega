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

package validators

import (
	"bytes"
	"context"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/broker/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestValidatorsCheckpoint(t *testing.T) {
	topology := &Topology{}
	topology.validators = map[string]*valState{}

	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID:              "id1",
			VegaPubKey:      "pubkey1",
			VegaPubKeyIndex: 1,
			TmPubKey:        "tmpubkey1",
			EthereumAddress: "eth1",
			FromEpoch:       1,
		},
		blockAdded:                   2,
		status:                       ValidatorStatusTendermint,
		statusChangeBlock:            1,
		lastBlockWithPositiveRanking: 1,
		heartbeatTracker: &validatorHeartbeatTracker{
			expectedNextHash:      "",
			expectedNexthashSince: time.Now(),
			blockIndex:            0,
			blockSigs:             [10]bool{true, false, false, false, false, false, false, false, false, false},
		},
		numberOfEthereumEventsForwarded: 4,
		validatorPower:                  100,
	}
	topology.validators["node1"] = &valState{
		data: ValidatorData{
			ID:              "id2",
			VegaPubKey:      "pubkey2",
			VegaPubKeyIndex: 0,
			TmPubKey:        "tmpubkey2",
			EthereumAddress: "eth2",
			FromEpoch:       2,
		},
		blockAdded:                   1,
		status:                       ValidatorStatusTendermint,
		statusChangeBlock:            2,
		lastBlockWithPositiveRanking: 1,
		heartbeatTracker: &validatorHeartbeatTracker{
			expectedNextHash:      "abcde",
			expectedNexthashSince: time.Now(),
			blockIndex:            0,
			blockSigs:             [10]bool{false, false, false, false, false, false, false, false, false, false},
		},
		numberOfEthereumEventsForwarded: 3,
		validatorPower:                  120,
	}

	topology.pendingPubKeyRotations = make(pendingKeyRotationMapping)
	hash1, err := topology.Checkpoint()
	require.NoError(t, err)

	topology2 := &Topology{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	broker := bmocks.NewMockBroker(ctrl)
	topology2.broker = broker
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	topology2.Load(context.Background(), hash1)
	hash2, err := topology2.Checkpoint()
	require.NoError(t, err)

	require.True(t, bytes.Equal(hash1, hash2))
}
