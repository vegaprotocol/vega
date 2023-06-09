package oracles_test

import (
	"bytes"
	"context"
	"math/big"
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

var contractCallKey = (&types.PayloadEthContractCallEvent{}).Key()

func TestEthereumOracleVerifierSnapshotEmpty(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()

	assert.Equal(t, 1, len(eov.Keys()))

	s, _, err := eov.GetState(contractCallKey)
	require.Nil(t, err)
	require.NotNil(t, s)
}

func TestEthereumOracleVerifierWithPendingQueryResults(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()

	eov.ethCallEngine.EXPECT().CallContract(gomock.Any(), "testspec", big.NewInt(1)).Return(result, nil)

	eov.ts.EXPECT().GetTimeNow().Times(1)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check()
			return nil
		})

	s1, _, err := eov.GetState(contractCallKey)
	require.Nil(t, err)
	require.NotNil(t, s1)

	callEvent := types.EthContractCallEvent{
		BlockHeight: 1,
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

	// Restore
	restoredVerifier := getTestEthereumOracleVerifier(t)
	defer restoredVerifier.ctrl.Finish()
	restoredVerifier.witness.EXPECT().RestoreResource(gomock.Any(), gomock.Any()).Times(1)

	_, err = restoredVerifier.LoadState(context.Background(), types.PayloadFromProto(snap))
	require.Nil(t, err)
	// Check its there by adding it again and checking for duplication error
	require.ErrorIs(t, oracles.ErrDuplicatedEthereumCallEvent, restoredVerifier.ProcessEthereumContractCallResult(callEvent))
}
