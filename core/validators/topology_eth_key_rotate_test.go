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
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/core/validators/mocks"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	types1 "github.com/tendermint/tendermint/proto/tendermint/types"
)

func TestTopologyEthereumKeyRotate(t *testing.T) {
	t.Run("rotate ethereum key - success", testRotateEthereumKeySuccess)
	t.Run("rotate ethereum key - fails when node does not exists", testRotateEthereumKeyFailsOnNonExistingNode)
	t.Run("rotate ethereum key - fails when target block height is less then current block height", testRotateEthereumKeyFailsWhenTargetBlockHeightIsLessThenCurrentBlockHeight)
	t.Run("rotate ethereum key - fails when current address does not match", testRotateEthereumKeyFailsWhenCurrentAddressDoesNotMatch)
	t.Run("ethereum key rotation begin block - success", testEthereumKeyRotationBeginBlock)
	t.Run("ethereum key rotation begin block with submitter - success", testEthereumKeyRotationBeginBlockWithSubmitter)
}

func testRotateEthereumKeySuccess(t *testing.T) {
	top := getTestTopWithMockedSignatures(t)
	defer top.ctrl.Finish()
	top.timeService.EXPECT().GetTimeNow().AnyTimes()

	nr := commandspb.AnnounceNode{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
	}
	ctx := context.Background()
	err := top.AddNewNode(ctx, &nr, validators.ValidatorStatusTendermint)
	require.NoError(t, err)

	ekr := &commandspb.EthereumKeyRotateSubmission{
		TargetBlock:    15,
		NewAddress:     "new-eth-address",
		CurrentAddress: nr.EthereumAddress,
	}

	toRemove := []validators.NodeIDAddress{{NodeID: nr.Id, EthAddress: nr.EthereumAddress}}

	top.signatures.EXPECT().PrepareValidatorSignatures(
		gomock.Any(),
		toRemove,
		gomock.Any(),
		gomock.Any(),
	).Times(1)

	err = top.RotateEthereumKey(ctx, nr.VegaPubKey, 10, ekr)
	require.NoError(t, err)
}

func testRotateEthereumKeyFailsOnNonExistingNode(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.timeService.EXPECT().GetTimeNow().AnyTimes()

	err := top.RotateEthereumKey(
		context.Background(),
		"vega-nonexisting-pubkey",
		10,
		newEthereumKeyRotationSubmission("", "new-eth-addr", 10, ""),
	)

	assert.Error(t, err)
	assert.EqualError(t, err, "failed to rotate ethereum key for non existing validator \"vega-nonexisting-pubkey\"")
}

func testRotateEthereumKeyFailsWhenTargetBlockHeightIsLessThenCurrentBlockHeight(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()

	nr := commandspb.AnnounceNode{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
	}

	err := top.AddNewNode(context.Background(), &nr, validators.ValidatorStatusTendermint)
	require.NoError(t, err)

	err = top.RotateEthereumKey(
		context.Background(),
		nr.VegaPubKey,
		10,
		newEthereumKeyRotationSubmission("eth-address", "new-eth-addr", 5, ""),
	)
	assert.ErrorIs(t, err, validators.ErrTargetBlockHeightMustBeGraterThanCurrentHeight)
}

func testRotateEthereumKeyFailsWhenCurrentAddressDoesNotMatch(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.timeService.EXPECT().GetTimeNow().AnyTimes()

	nr := commandspb.AnnounceNode{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
		VegaPubKeyIndex: 1,
	}
	err := top.AddNewNode(context.Background(), &nr, validators.ValidatorStatusTendermint)
	require.NoError(t, err)

	err = top.RotateEthereumKey(
		context.Background(),
		nr.VegaPubKey,
		10,
		newEthereumKeyRotationSubmission("random-key", "new-eth-key", 20, ""),
	)
	assert.ErrorIs(t, err, validators.ErrCurrentEthAddressDoesNotMatch)
}

func newEthereumKeyRotationSubmission(currentAddr, newAddr string, targetBlock uint64, submitter string) *commandspb.EthereumKeyRotateSubmission {
	return &commandspb.EthereumKeyRotateSubmission{
		CurrentAddress:   currentAddr,
		NewAddress:       newAddr,
		TargetBlock:      targetBlock,
		SubmitterAddress: submitter,
	}
}

func testEthereumKeyRotationBeginBlock(t *testing.T) {
	top := getTestTopWithMockedSignatures(t)
	defer top.ctrl.Finish()
	top.timeService.EXPECT().GetTimeNow().AnyTimes()

	chainValidators := []string{"tm-pubkey-1", "tm-pubkey-2", "tm-pubkey-3", "tm-pubkey-4"}

	ctx := context.Background()
	for i := 0; i < len(chainValidators); i++ {
		j := i + 1
		id := fmt.Sprintf("vega-key-%d", j)
		nr := commandspb.AnnounceNode{
			Id:              fmt.Sprintf("vega-master-pubkey-%d", j),
			ChainPubKey:     chainValidators[i],
			VegaPubKey:      id,
			EthereumAddress: fmt.Sprintf("eth-address-%d", j),
		}

		err := top.AddNewNode(ctx, &nr, validators.ValidatorStatusTendermint)
		require.NoErrorf(t, err, "failed to add node registation %s", id)
	}

	top.signatures.EXPECT().ClearStaleSignatures().AnyTimes()
	top.signatures.EXPECT().PrepareValidatorSignatures(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Times(2 * len(chainValidators))

	// add ethereum key rotations
	err := top.RotateEthereumKey(ctx, "vega-key-1", 10, newEthereumKeyRotationSubmission("eth-address-1", "new-eth-address-1", 11, ""))
	require.NoError(t, err)
	err = top.RotateEthereumKey(ctx, "vega-key-2", 10, newEthereumKeyRotationSubmission("eth-address-2", "new-eth-address-2", 11, ""))
	require.NoError(t, err)
	err = top.RotateEthereumKey(ctx, "vega-key-3", 10, newEthereumKeyRotationSubmission("eth-address-3", "new-eth-address-3", 13, ""))
	require.NoError(t, err)
	err = top.RotateEthereumKey(ctx, "vega-key-4", 10, newEthereumKeyRotationSubmission("eth-address-4", "new-eth-address-4", 13, ""))
	require.NoError(t, err)

	// when
	top.BeginBlock(ctx, abcitypes.RequestBeginBlock{Header: types1.Header{Height: 11}})
	// then
	data1 := top.Get("vega-master-pubkey-1")
	require.NotNil(t, data1)
	assert.Equal(t, "new-eth-address-1", data1.EthereumAddress)
	data2 := top.Get("vega-master-pubkey-2")
	require.NotNil(t, data2)
	assert.Equal(t, "new-eth-address-2", data2.EthereumAddress)
	data3 := top.Get("vega-master-pubkey-3")
	require.NotNil(t, data3)
	assert.Equal(t, "eth-address-3", data3.EthereumAddress)
	data4 := top.Get("vega-master-pubkey-4")
	require.NotNil(t, data4)
	assert.Equal(t, "eth-address-4", data4.EthereumAddress)

	// when
	top.BeginBlock(ctx, abcitypes.RequestBeginBlock{Header: types1.Header{Height: 13}})
	// then
	data3 = top.Get("vega-master-pubkey-3")
	require.NotNil(t, data3)
	assert.Equal(t, "new-eth-address-3", data3.EthereumAddress)
	data4 = top.Get("vega-master-pubkey-4")
	require.NotNil(t, data4)
	assert.Equal(t, "new-eth-address-4", data4.EthereumAddress)
}

func testEthereumKeyRotationBeginBlockWithSubmitter(t *testing.T) {
	top := getTestTopWithMockedSignatures(t)
	defer top.ctrl.Finish()
	top.timeService.EXPECT().GetTimeNow().AnyTimes()

	chainValidators := []string{"tm-pubkey-1", "tm-pubkey-2", "tm-pubkey-3", "tm-pubkey-4"}

	ctx := context.Background()
	for i := 0; i < len(chainValidators); i++ {
		j := i + 1
		id := fmt.Sprintf("vega-master-pubkey-%d", j)
		nr := commandspb.AnnounceNode{
			Id:              id,
			ChainPubKey:     chainValidators[i],
			VegaPubKey:      fmt.Sprintf("vega-key-%d", j),
			EthereumAddress: fmt.Sprintf("eth-address-%d", j),
		}

		err := top.AddNewNode(ctx, &nr, validators.ValidatorStatusTendermint)
		require.NoErrorf(t, err, "failed to add node registation %s", id)
	}

	submitter := "some-eth-address"
	top.signatures.EXPECT().PrepareValidatorSignatures(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
	top.signatures.EXPECT().EmitValidatorRemovedSignatures(gomock.Any(), submitter, gomock.Any(), gomock.Any()).Times(1)
	top.signatures.EXPECT().EmitValidatorAddedSignatures(gomock.Any(), submitter, gomock.Any(), gomock.Any()).Times(1)
	top.signatures.EXPECT().ClearStaleSignatures().AnyTimes()

	// add ethereum key rotations
	err := top.RotateEthereumKey(ctx, "vega-key-1", 10, newEthereumKeyRotationSubmission("eth-address-1", "new-eth-address-1", 11, submitter))
	require.NoError(t, err)

	// when
	top.BeginBlock(ctx, abcitypes.RequestBeginBlock{Header: types1.Header{Height: 11}})
	// then
	data1 := top.Get("vega-master-pubkey-1")
	require.NotNil(t, data1)
	assert.Equal(t, "new-eth-address-1", data1.EthereumAddress)
}

type testTopWithSignatures struct {
	*testTop
	signatures *mocks.MockSignatures
}

func getTestTopWithMockedSignatures(t *testing.T) *testTopWithSignatures {
	t.Helper()

	top := getTestTopWithDefaultValidator(t)
	signatures := mocks.NewMockSignatures(top.ctrl)

	top.SetSignatures(signatures)

	return &testTopWithSignatures{
		testTop:    top,
		signatures: signatures,
	}
}
