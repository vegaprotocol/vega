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
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	contractCallKey = (&types.PayloadEthContractCallEvent{}).Key()
	lastEthBlockKey = (&types.PayloadEthOracleLastBlock{}).Key()
	miscKey         = (&types.PayloadEthVerifierMisc{}).Key()
)

func TestEthereumOracleVerifierSnapshotEmpty(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()

	assert.Equal(t, 3, len(eov.Keys()))

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

	restoredVerifier.ethCallEngine.EXPECT().Start()

	// As the verifier has no state, the call engine should not have its last block set.
	restoredVerifier.OnStateLoaded(context.Background())
}

func TestEthereumOracleVerifierWithPendingQueryResults(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()
	eov.ethCallEngine.EXPECT().GetEthTime(gomock.Any(), uint64(5)).Return(uint64(100), nil)
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(5)).Return(result, nil)
	eov.ethCallEngine.EXPECT().GetInitialTriggerTime("testspec").Return(uint64(90), nil)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations("testspec").Return(uint64(5), nil).Times(2)

	eov.ts.EXPECT().GetTimeNow().AnyTimes()
	eov.ethConfirmations.EXPECT().CheckRequiredConfirmations(uint64(5), uint64(5)).Return(nil)

	var checkResult error
	eov.witness.EXPECT().StartCheckWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time, _ int64) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	s1, _, err := eov.GetState(contractCallKey)
	require.Nil(t, err)
	require.NotNil(t, s1)

	slb1, _, err := eov.GetState(lastEthBlockKey)
	require.Nil(t, err)
	require.NotNil(t, slb1)

	misc1, _, err := eov.GetState(miscKey)
	require.Nil(t, err)
	require.NotNil(t, misc1)

	callEvent := ethcall.ContractCallEvent{
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

	misc2, _, err := eov.GetState(miscKey)
	require.Nil(t, err)
	assert.NotEqual(t, misc1, misc2)

	miscState := &snapshot.Payload{}
	err = proto.Unmarshal(misc2, miscState)
	require.Nil(t, err)

	// Restore
	restoredVerifier := getTestEthereumOracleVerifier(t)
	defer restoredVerifier.ctrl.Finish()

	restoredVerifier.ts.EXPECT().GetTimeNow().AnyTimes()
	restoredVerifier.witness.EXPECT().RestoreResource(gomock.Any(), gomock.Any()).Times(1)

	_, err = restoredVerifier.LoadState(context.Background(), types.PayloadFromProto(snap))
	require.Nil(t, err)
	_, err = restoredVerifier.LoadState(context.Background(), types.PayloadFromProto(slbsnap))
	require.Nil(t, err)
	_, err = restoredVerifier.LoadState(context.Background(), types.PayloadFromProto(miscState))
	require.Nil(t, err)

	// After the state of the verifier is loaded it should start the call engine at the restored height
	restoredVerifier.ethCallEngine.EXPECT().StartAtHeight(uint64(5), uint64(100))
	restoredVerifier.OnStateLoaded(context.Background())

	// Check its there by adding it again and checking for duplication error
	require.ErrorIs(t, errors.ErrDuplicatedEthereumCallEvent, restoredVerifier.ProcessEthereumContractCallResult(callEvent))
}

func TestEthereumVerifierPatchBlock(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	patchBlock := uint64(5)

	callEvent := ethcall.ContractCallEvent{
		BlockHeight: patchBlock,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}

	err, checkResult := sendEthereumEvent(t, eov, callEvent)
	assert.NoError(t, err)
	assert.NoError(t, checkResult)

	// now we want to restore as if we are doing an upgrade
	ctx := vgcontext.WithSnapshotInfo(context.Background(), "v0.74.9", true)

	lb, _, err := eov.GetState(lastEthBlockKey)
	require.Nil(t, err)
	require.NotNil(t, lb)

	state := &snapshot.Payload{}
	err = proto.Unmarshal(lb, state)
	require.Nil(t, err)

	restoredVerifier := getTestEthereumOracleVerifier(t)
	defer restoredVerifier.ctrl.Finish()

	restoredVerifier.ts.EXPECT().GetTimeNow().AnyTimes()
	restoredVerifier.ethCallEngine.EXPECT().StartAtHeight(gomock.Any(), gomock.Any()).Times(1)
	_, err = restoredVerifier.LoadState(ctx, types.PayloadFromProto(state))
	require.NoError(t, err)
	restoredVerifier.OnStateLoaded(ctx)

	// now send in an event with an block height before
	oldEvent := ethcall.ContractCallEvent{
		BlockHeight: patchBlock - 1,
		BlockTime:   50,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}

	err = restoredVerifier.ProcessEthereumContractCallResult(oldEvent)
	assert.ErrorIs(t, err, errors.ErrEthereumCallEventTooOld)

	// send in a new later event so that last block updates
	callEvent = ethcall.ContractCallEvent{
		BlockHeight: patchBlock + 5,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}
	err, checkResult = sendEthereumEvent(t, eov, callEvent)
	assert.NoError(t, err)
	assert.NoError(t, checkResult)

	// restore from the snapshot not at upgrade height
	ctx = context.Background()
	lb, _, err = restoredVerifier.GetState(lastEthBlockKey)
	require.Nil(t, err)
	require.NotNil(t, lb)

	state = &snapshot.Payload{}
	err = proto.Unmarshal(lb, state)
	require.Nil(t, err)

	m, _, err := restoredVerifier.GetState(miscKey)
	require.Nil(t, err)
	require.NotNil(t, m)

	miscState := &snapshot.Payload{}
	err = proto.Unmarshal(m, miscState)
	require.Nil(t, err)

	restoredVerifier = getTestEthereumOracleVerifier(t)
	defer restoredVerifier.ctrl.Finish()

	restoredVerifier.ts.EXPECT().GetTimeNow().AnyTimes()
	restoredVerifier.ethCallEngine.EXPECT().StartAtHeight(gomock.Any(), gomock.Any()).Times(1)
	restoredVerifier.LoadState(ctx, types.PayloadFromProto(state))
	restoredVerifier.LoadState(ctx, types.PayloadFromProto(miscState))
	restoredVerifier.OnStateLoaded(ctx)

	// check that the patch block hasn't updated, old event is still old
	err = restoredVerifier.ProcessEthereumContractCallResult(oldEvent)
	assert.ErrorIs(t, err, errors.ErrEthereumCallEventTooOld)

	// event at block after patch-block but before last block is allowed
	callEvent = ethcall.ContractCallEvent{
		BlockHeight: patchBlock + 2,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}
	err, checkResult = sendEthereumEvent(t, eov, callEvent)
	assert.NoError(t, err)
	assert.NoError(t, checkResult)
}

func TestEthereumVerifierRejectTooOld(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	now := time.Now()

	patchBlock := uint64(5)
	callEvent := ethcall.ContractCallEvent{
		BlockHeight: patchBlock,
		BlockTime:   uint64(now.Unix()),
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}

	err, checkResult := sendEthereumEvent(t, eov, callEvent)
	assert.NoError(t, err)
	assert.NoError(t, checkResult)

	// send it in again and check its rejected as a dupe
	eov.ts.EXPECT().GetTimeNow().Times(1).Return(now)
	err = eov.ProcessEthereumContractCallResult(callEvent)
	assert.ErrorIs(t, err, errors.ErrDuplicatedEthereumCallEvent)

	// let time pass more than a week
	now = now.Add(24 * 7 * time.Hour)
	eov.onTick(context.Background(), now)

	// now send in the event again
	eov.ts.EXPECT().GetTimeNow().Times(1).Return(now)
	err = eov.ProcessEthereumContractCallResult(callEvent)
	assert.ErrorIs(t, err, errors.ErrEthereumCallEventTooOld)
}

func sendEthereumEvent(t *testing.T, eov *verifierTest, callEvent ethcall.ContractCallEvent) (error, error) {
	t.Helper()
	result := okResult()
	eov.ethCallEngine.EXPECT().GetEthTime(gomock.Any(), callEvent.BlockHeight).Return(callEvent.BlockTime, nil)
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", callEvent.BlockHeight).Return(result, nil)
	eov.ethCallEngine.EXPECT().GetInitialTriggerTime("testspec").Return(uint64(90), nil)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations("testspec").Return(uint64(5), nil).Times(2)

	eov.ts.EXPECT().GetTimeNow().Times(2)
	eov.ethConfirmations.EXPECT().CheckRequiredConfirmations(callEvent.BlockHeight, uint64(5)).Return(nil)

	var checkResult error
	eov.witness.EXPECT().StartCheckWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time, _ int64) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	err := eov.ProcessEthereumContractCallResult(callEvent)

	return err, checkResult
}
