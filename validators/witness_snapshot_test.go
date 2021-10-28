package validators_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshot(t *testing.T) {
	for i := 0; i < 100; i++ {
		erc := getTestWitness(t)
		defer erc.ctrl.Finish()
		defer erc.Stop()

		key := (&types.PayloadWitness{}).Key()

		state1, err := erc.Witness.GetState(key)
		require.Nil(t, err)

		erc.top.EXPECT().IsValidator().AnyTimes().Return(true)
		res := testRes{"resource-id-1", func() error {
			return nil
		}}
		checkUntil := erc.startTime.Add(700 * time.Second)
		cb := func(interface{}, bool) {}

		err = erc.StartCheck(res, cb, checkUntil)
		assert.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		// take a snapshot after the resource has been added
		state2, err := erc.Witness.GetState(key)
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
		erc2.top.EXPECT().SelfNodeID().AnyTimes().Return("1234")

		_, err = erc2.LoadState(context.Background(), payload)
		require.Nil(t, err)
		erc2.RestoreResource(res, cb)

		// expect the hash and state have been restored successfully
		state3, err := erc2.GetState(key)
		require.Nil(t, err)
		require.True(t, bytes.Equal(state2, state3))

		// add a vote
		erc2.top.EXPECT().IsValidatorNode(gomock.Any()).Times(1).Return(true)
		err = erc2.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: res.id, PubKey: []byte("1234")})

		assert.NoError(t, err)

		// expect the hash/state to have changed
		state4, err := erc2.GetState(key)
		require.Nil(t, err)
		require.False(t, bytes.Equal(state4, state3))

		// restore from the state with vote
		proto.Unmarshal(state4, &pl)
		payload = types.PayloadFromProto(&pl)

		erc3 := getTestWitness(t)
		defer erc3.ctrl.Finish()
		defer erc3.Stop()
		erc3.top.EXPECT().IsValidator().AnyTimes().Return(true)
		erc3.top.EXPECT().SelfNodeID().AnyTimes().Return("1234")

		_, err = erc3.LoadState(context.Background(), payload)
		require.Nil(t, err)
		erc3.RestoreResource(res, cb)

		state5, err := erc3.GetState(key)
		require.Nil(t, err)
		require.True(t, bytes.Equal(state5, state4))
	}
}
