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

package validators

import (
	"bytes"
	"context"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
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
