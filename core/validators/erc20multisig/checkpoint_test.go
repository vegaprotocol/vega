// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package erc20multisig_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultisigTopologyCheckpoint(t *testing.T) {
	top := getTestTopology(t)
	defer top.ctrl.Finish()

	top.OnTick(context.Background(), time.Unix(10, 0))
	// first set the threshold and 1 validator

	// Let's create threshold
	// first assert we have no threshold
	assert.Equal(t, uint32(0), top.GetThreshold())

	thresholdEvent1 := types.SignerThresholdSetEvent{
		Threshold:   666,
		BlockNumber: 10,
		LogIndex:    11,
		TxHash:      "0xacbde",
		ID:          "someid",
		Nonce:       "123",
		BlockTime:   123456789,
	}

	var cb func(interface{}, bool)
	var res validators.Resource
	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		cb = f
		res = r
		return nil
	})

	assert.NoError(t, top.ProcessThresholdEvent(&thresholdEvent1))

	// now we can call the callback
	cb(res, true)

	// now we can update the time
	top.broker.EXPECT().Send(gomock.Any()).Times(1)
	top.OnTick(context.Background(), time.Unix(11, 0))
	assert.Equal(t, top.GetThreshold(), uint32(666))

	// now the signer

	// first assert we have no signers
	assert.Len(t, top.GetSigners(), 0)

	signerEvent1 := types.SignerEvent{
		BlockNumber: 150,
		LogIndex:    11,
		TxHash:      "0xacbde",
		ID:          "someid",
		Address:     "0xe82EfC4187705655C9b484dFFA25f240e8A6B0BA",
		Nonce:       "123",
		BlockTime:   123456789,
		Kind:        types.SignerEventKindAdded,
	}

	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		cb = f
		res = r
		return nil
	})

	assert.NoError(t, top.ProcessSignerEvent(&signerEvent1))

	// now we can call the callback
	cb(res, true)

	// now we can update the time
	top.broker.EXPECT().Send(gomock.Any()).Times(1)
	top.OnTick(context.Background(), time.Unix(12, 0))

	t.Run("ensure the signer list is updated", func(t *testing.T) {
		signers := top.GetSigners()
		assert.Len(t, signers, 1)
		assert.Equal(t, "0xe82EfC4187705655C9b484dFFA25f240e8A6B0BA", signers[0])
	})

	t.Run("check if our party IsSigner", func(t *testing.T) {
		assert.True(t, top.IsSigner("0xe82EfC4187705655C9b484dFFA25f240e8A6B0BA"))
	})

	t.Run("check excess signers", func(t *testing.T) {
		okAddresses := []string{"0xe82EfC4187705655C9b484dFFA25f240e8A6B0BA"}
		koAddresses := []string{}

		assert.True(t, top.ExcessSigners(koAddresses))
		assert.False(t, top.ExcessSigners(okAddresses))
	})

	// now we will add some pending ones

	thresholdEvent2 := types.SignerThresholdSetEvent{
		Threshold:   500,
		BlockNumber: 150,
		LogIndex:    1,
		TxHash:      "0xacbde2",
		ID:          "someidthreshold2",
		Nonce:       "1234",
		BlockTime:   133456790,
	}

	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		return nil
	})

	assert.NoError(t, top.ProcessThresholdEvent(&thresholdEvent2))

	signerEvent2 := types.SignerEvent{
		BlockNumber: 101,
		LogIndex:    19,
		TxHash:      "0xacbde3",
		ID:          "someid3",
		Address:     "0xa587765281c2514E899ecFFa9626b6254582a3bA",
		Nonce:       "1239",
		BlockTime:   133456789,
		Kind:        types.SignerEventKindAdded,
	}

	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		return nil
	})

	assert.NoError(t, top.ProcessSignerEvent(&signerEvent2))

	// now we can make a checkpoint and load it.
	// here we expect the following:
	// threshold set to 666
	// 1 validator
	// block set to the most recent pending

	cp, err := top.Checkpoint()
	assert.NoError(t, err)
	assert.True(t, len(cp) > 0)

	top2 := getTestTopology(t)

	top2.broker.EXPECT().Send(gomock.Any()).Times(2)
	top2.ethEventSource.EXPECT().UpdateMultisigControlStartingBlock(gomock.Any()).Do(
		func(block uint64) {
			// ensure we restart at the right block
			assert.Equal(t, int(block), 101)
		},
	)

	require.NoError(t, top2.Load(context.Background(), cp))

	// no assert state is restored correctly
	assert.Equal(t, int(top2.GetThreshold()), 666)
	signers := top2.GetSigners()
	assert.Len(t, signers, 1)
	assert.Equal(t, signers[0], "0xe82EfC4187705655C9b484dFFA25f240e8A6B0BA")
}
