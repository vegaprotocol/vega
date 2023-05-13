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

package validators_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshot(t *testing.T) {
	erc := getTestWitness(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	key := (&types.PayloadWitness{}).Key()

	state1, _, err := erc.Witness.GetState(key)
	require.Nil(t, err)

	erc.top.EXPECT().IsValidator().AnyTimes().Return(true)

	ctx, cancel := context.WithCancel(context.Background())
	res := testRes{"resource-id-1", func() error {
		cancel()
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)

	cb := func(interface{}, bool) {}
	err = erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	// wait until we've done a check
	<-ctx.Done()

	// take a snapshot after the resource has been added
	state2, _, err := erc.Witness.GetState(key)
	require.Nil(t, err)

	// verify it has changed from before the resource
	require.False(t, bytes.Equal(state1, state2))

	var pl snapshot.Payload
	proto.Unmarshal(state2, &pl)
	payload := types.PayloadFromProto(&pl)

	// reload the state
	erc2 := getTestWitness(t)
	defer erc2.ctrl.Finish()
	defer erc2.Stop()
	erc2.top.EXPECT().IsValidator().AnyTimes().Return(true)
	erc2.top.EXPECT().SelfVegaPubKey().AnyTimes().Return("1234")

	_, err = erc2.LoadState(context.Background(), payload)
	require.Nil(t, err)
	erc2.RestoreResource(res, cb)

	// expect the hash and state have been restored successfully
	state3, _, err := erc2.GetState(key)
	require.Nil(t, err)
	require.True(t, bytes.Equal(state2, state3))

	// add a vote
	pubkey := newPublicKey("1234")
	erc2.top.EXPECT().IsValidatorVegaPubKey(pubkey.Hex()).Times(1).Return(true)
	erc2.top.EXPECT().IsTendermintValidator(pubkey.Hex()).AnyTimes().Return(true)
	err = erc2.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: res.id}, pubkey)

	assert.NoError(t, err)

	// expect the hash/state to have changed
	state4, _, err := erc2.GetState(key)
	require.Nil(t, err)
	require.False(t, bytes.Equal(state4, state3))

	// restore from the state with vote
	proto.Unmarshal(state4, &pl)
	payload = types.PayloadFromProto(&pl)

	erc3 := getTestWitness(t)
	defer erc3.ctrl.Finish()
	defer erc3.Stop()
	erc3.top.EXPECT().IsValidator().AnyTimes().Return(true)
	erc3.top.EXPECT().SelfVegaPubKey().AnyTimes().Return("1234")

	_, err = erc3.LoadState(context.Background(), payload)
	require.Nil(t, err)
	erc3.RestoreResource(res, cb)

	state5, _, err := erc3.GetState(key)
	require.Nil(t, err)
	require.True(t, bytes.Equal(state5, state4))
}
