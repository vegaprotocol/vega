// Copyright (c) 2023 Gobalsky Labs Limited
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

package ethverifier_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	errors "code.vegaprotocol.io/vega/core/datasource/errors"
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
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
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	eov := getTestEthereumOracleVerifier(ctx, t)
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
	restoredVerifier := getTestEthereumOracleVerifier(ctx, t)
	defer restoredVerifier.ctrl.Finish()

	_, err = restoredVerifier.LoadState(context.Background(), types.PayloadFromProto(snap))
	require.Nil(t, err)
	_, err = restoredVerifier.LoadState(context.Background(), types.PayloadFromProto(slbsnap))
	require.Nil(t, err)

	restoredVerifier.ethCallEngine.EXPECT().Start()

	// As the verifier has no state, the call engine should not have its last block set.
	restoredVerifier.OnStateLoaded(context.Background())
}

func TestEthereumOracleVerifierWithPendingQueryResults(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	eov := getTestEthereumOracleVerifier(ctx, t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()

	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(5)).Return(result, nil)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations("testspec").Return(uint64(5), nil)

	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethConfirmations.EXPECT().CheckRequiredConfirmations(uint64(5), uint64(5)).Return(nil)

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

	callEvent := ethcall.ContractCallEvent{
		BlockHeight: 5,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}

	eov.ethContractCallEventChan <- callEvent
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
	restoredVerifier := getTestEthereumOracleVerifier(ctx, t)
	defer restoredVerifier.ctrl.Finish()
	restoredVerifier.witness.EXPECT().RestoreResource(gomock.Any(), gomock.Any()).Times(1)

	_, err = restoredVerifier.LoadState(context.Background(), types.PayloadFromProto(snap))
	require.Nil(t, err)
	_, err = restoredVerifier.LoadState(context.Background(), types.PayloadFromProto(slbsnap))
	require.Nil(t, err)

	// After the state of the verifier is loaded it should start the call engine at the restored height
	restoredVerifier.ethCallEngine.EXPECT().StartAtHeight(uint64(5), uint64(100))
	restoredVerifier.OnStateLoaded(context.Background())

	// Check its there by adding it again and checking for duplication error
	require.ErrorIs(t, errors.ErrDuplicatedEthereumCallEvent, restoredVerifier.ProcessEthereumContractCallResult(callEvent))
}
