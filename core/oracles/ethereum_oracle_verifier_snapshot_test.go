package oracles_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	contractCallKey = (&types.PayloadEthContractCallEvent{}).Key()
	lastEthBlockKey = (&types.PayloadEthOracleLastBlock{}).Key()
)

func TestEthereumOracleVerifierSnapshotEmpty(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()

	assert.Equal(t, 2, len(eov.Keys()))

	state, _, err := eov.GetState(contractCallKey)
	require.Nil(t, err)
	require.NotNil(t, state)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	slbstate, _, err := eov.GetState(lastEthBlockKey)
	require.Nil(t, err)

	slbsnap := &snapshot.Payload{}
	err = proto.Unmarshal(slbstate, slbsnap)
	require.Nil(t, err)

	// Restore
	restoredVerifier := getTestEthereumOracleVerifier(t)
	defer restoredVerifier.ctrl.Finish()

	_, err = restoredVerifier.LoadState(context.Background(), types.PayloadFromProto(snap))
	require.Nil(t, err)
	_, err = restoredVerifier.LoadState(context.Background(), types.PayloadFromProto(slbsnap))
	require.Nil(t, err)

	// As the verifier has no state, the call engine should not have its last block set.
	restoredVerifier.OnStateLoaded(context.Background())
}

func TestEthereumOracleVerifierWithPendingQueryResults(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()

	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(5)).Return(result, nil)
	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethConfirmations.EXPECT().Check(uint64(5)).Return(nil)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	s1, _, err := eov.GetState(contractCallKey)
	require.Nil(t, err)
	require.NotNil(t, s1)

	slb1, _, err := eov.GetState(lastEthBlockKey)
	require.Nil(t, err)
	require.NotNil(t, slb1)

	callEvent := types.EthContractCallEvent{
		BlockHeight: 5,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}

	err = eov.ProcessEthereumContractCallResult(callEvent)
	assert.NoError(t, err)
	assert.NoError(t, checkResult)

	s2, _, err := eov.GetState(contractCallKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))

	state, _, err := eov.GetState(contractCallKey)
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(state, snap)
	require.Nil(t, err)

	slb2, _, err := eov.GetState(lastEthBlockKey)
	require.Nil(t, err)
	require.False(t, bytes.Equal(slb1, slb2))

	slbstate, _, err := eov.GetState(lastEthBlockKey)
	require.Nil(t, err)

	slbsnap := &snapshot.Payload{}
	err = proto.Unmarshal(slbstate, slbsnap)
	require.Nil(t, err)

	// Restore
	restoredVerifier := getTestEthereumOracleVerifier(t)
	defer restoredVerifier.ctrl.Finish()
	restoredVerifier.witness.EXPECT().RestoreResource(gomock.Any(), gomock.Any()).Times(1)

	_, err = restoredVerifier.LoadState(context.Background(), types.PayloadFromProto(snap))
	require.Nil(t, err)
	_, err = restoredVerifier.LoadState(context.Background(), types.PayloadFromProto(slbsnap))
	require.Nil(t, err)

	// After the state of the verifier is loaded it should inform the call engine of the last processed block
	// and this should match the restored values.
	restoredVerifier.ethCallEngine.EXPECT().UpdatePreviousEthBlock(uint64(5), uint64(100))
	restoredVerifier.OnStateLoaded(context.Background())

	// Check its there by adding it again and checking for duplication error
	require.ErrorIs(t, oracles.ErrDuplicatedEthereumCallEvent, restoredVerifier.ProcessEthereumContractCallResult(callEvent))
}
