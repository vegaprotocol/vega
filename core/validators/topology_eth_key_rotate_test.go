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

package validators_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/core/validators/mocks"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopologyEthereumKeyRotate(t *testing.T) {
	t.Run("rotate ethereum key - success", testRotateEthereumKeySuccess)
	t.Run("rotate ethereum key - fails when rotating to the same key", testRotateEthereumKeyFailsRotatingToSameKey)
	t.Run("rotate ethereum key - fails if pending rotation already exists", testRotateEthereumKeyFailsIfPendingRotationExists)
	t.Run("rotate ethereum key - fails when node does not exists", testRotateEthereumKeyFailsOnNonExistingNode)
	t.Run("rotate ethereum key - fails when target block height is less then current block height", testRotateEthereumKeyFailsWhenTargetBlockHeightIsLessThenCurrentBlockHeight)
	t.Run("rotate ethereum key - fails when current address does not match", testRotateEthereumKeyFailsWhenCurrentAddressDoesNotMatch)
	t.Run("ethereum key rotation begin block - success", testEthereumKeyRotationBeginBlock)
	t.Run("ethereum key rotation begin block with submitter - success", TestEthereumKeyRotationBeginBlockWithSubmitter)
	t.Run("ethereum key rotation by pending or ersatz does not generate signatures", testNoSignaturesForNonTendermint)
}

func testRotateEthereumKeySuccess(t *testing.T) {
	top := getTestTopWithMockedSignatures(t)
	defer top.ctrl.Finish()

	nr := commandspb.AnnounceNode{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
	}
	ctx := context.Background()
	err := top.AddNewNode(ctx, &nr, validators.ValidatorStatusTendermint)
	require.NoError(t, err)

	ekr := newEthereumKeyRotationSubmission(nr.EthereumAddress, "new-eth-address", 15, "")

	toRemove := []validators.NodeIDAddress{{NodeID: nr.Id, EthAddress: nr.EthereumAddress}}

	top.timeService.EXPECT().GetTimeNow().AnyTimes()
	top.signatures.EXPECT().PrepareValidatorSignatures(
		gomock.Any(),
		toRemove,
		gomock.Any(),
		gomock.Any(),
	).Times(1)

	err = top.ProcessEthereumKeyRotation(ctx, nr.VegaPubKey, ekr, MockVerify)
	require.NoError(t, err)
}

func testRotateEthereumKeyFailsIfPendingRotationExists(t *testing.T) {
	top := getTestTopWithMockedSignatures(t)
	defer top.ctrl.Finish()

	nr := commandspb.AnnounceNode{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
	}
	ctx := context.Background()
	err := top.AddNewNode(ctx, &nr, validators.ValidatorStatusTendermint)
	require.NoError(t, err)

	ekr := newEthereumKeyRotationSubmission(nr.EthereumAddress, "new-eth-address", 15, "")

	toRemove := []validators.NodeIDAddress{{NodeID: nr.Id, EthAddress: nr.EthereumAddress}}

	top.timeService.EXPECT().GetTimeNow().AnyTimes()
	top.signatures.EXPECT().PrepareValidatorSignatures(
		gomock.Any(),
		toRemove,
		gomock.Any(),
		gomock.Any(),
	).Times(1)

	err = top.ProcessEthereumKeyRotation(ctx, nr.VegaPubKey, ekr, MockVerify)
	require.NoError(t, err)

	// now push in another rotation submission
	err = top.ProcessEthereumKeyRotation(ctx, nr.VegaPubKey, ekr, MockVerify)
	require.Error(t, err, validators.ErrNodeAlreadyHasPendingKeyRotation)
}

func testRotateEthereumKeyFailsRotatingToSameKey(t *testing.T) {
	top := getTestTopWithMockedSignatures(t)
	defer top.ctrl.Finish()

	nr := commandspb.AnnounceNode{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
	}
	ctx := context.Background()
	err := top.AddNewNode(ctx, &nr, validators.ValidatorStatusTendermint)
	require.NoError(t, err)

	top.timeService.EXPECT().GetTimeNow().AnyTimes()
	ekr := newEthereumKeyRotationSubmission(nr.EthereumAddress, nr.EthereumAddress, 15, "")
	err = top.ProcessEthereumKeyRotation(ctx, nr.VegaPubKey, ekr, MockVerify)
	require.Error(t, err, validators.ErrCannotRotateToSameKey)
}

func testRotateEthereumKeyFailsOnNonExistingNode(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()

	top.timeService.EXPECT().GetTimeNow().AnyTimes()
	err := top.ProcessEthereumKeyRotation(
		context.Background(),
		"vega-nonexisting-pubkey",
		newEthereumKeyRotationSubmission("", "new-eth-addr", 10, ""),
		MockVerify,
	)

	assert.Error(t, err)
	assert.EqualError(t, err, "failed to rotate ethereum key for non existing validator \"vega-nonexisting-pubkey\"")
}

func testRotateEthereumKeyFailsWhenTargetBlockHeightIsLessThenCurrentBlockHeight(t *testing.T) {
	top := getTestTopWithMockedSignatures(t)
	defer top.ctrl.Finish()

	nr := commandspb.AnnounceNode{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
	}

	err := top.AddNewNode(context.Background(), &nr, validators.ValidatorStatusTendermint)
	require.NoError(t, err)

	top.timeService.EXPECT().GetTimeNow().AnyTimes()
	err = top.ProcessEthereumKeyRotation(
		context.Background(),
		nr.VegaPubKey,
		newEthereumKeyRotationSubmission("eth-address", "new-eth-addr", 5, ""),
		MockVerify,
	)
	assert.ErrorIs(t, err, validators.ErrTargetBlockHeightMustBeGreaterThanCurrentHeight)
}

func testRotateEthereumKeyFailsWhenCurrentAddressDoesNotMatch(t *testing.T) {
	top := getTestTopWithMockedSignatures(t)
	defer top.ctrl.Finish()

	nr := commandspb.AnnounceNode{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
		VegaPubKeyIndex: 1,
	}
	err := top.AddNewNode(context.Background(), &nr, validators.ValidatorStatusTendermint)
	require.NoError(t, err)

	top.timeService.EXPECT().GetTimeNow().AnyTimes()
	err = top.ProcessEthereumKeyRotation(
		context.Background(),
		nr.VegaPubKey,
		newEthereumKeyRotationSubmission("random-key", "new-eth-key", 20, ""),
		MockVerify,
	)
	assert.ErrorIs(t, err, validators.ErrCurrentEthAddressDoesNotMatch)
}

func newEthereumKeyRotationSubmission(currentAddr, newAddr string, targetBlock uint64, submitter string) *commandspb.EthereumKeyRotateSubmission {
	return &commandspb.EthereumKeyRotateSubmission{
		CurrentAddress:   currentAddr,
		NewAddress:       newAddr,
		TargetBlock:      targetBlock,
		SubmitterAddress: submitter,
		EthereumSignature: &commandspb.Signature{
			Value: "deadbeef",
		},
	}
}

func testEthereumKeyRotationBeginBlock(t *testing.T) {
	top := getTestTopWithMockedSignatures(t)
	defer top.ctrl.Finish()

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
	top.timeService.EXPECT().GetTimeNow().AnyTimes()
	top.multisigTop.EXPECT().IsSigner(gomock.Any()).AnyTimes().Return(false)
	top.signatures.EXPECT().ClearStaleSignatures().AnyTimes()
	top.signatures.EXPECT().SetNonce(gomock.Any()).Times(2)
	top.signatures.EXPECT().PrepareValidatorSignatures(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Times(2 * len(chainValidators))

	// add ethereum key rotations
	err := top.ProcessEthereumKeyRotation(ctx, "vega-key-1", newEthereumKeyRotationSubmission("eth-address-1", "new-eth-address-1", 11, ""), MockVerify)
	require.NoError(t, err)
	err = top.ProcessEthereumKeyRotation(ctx, "vega-key-2", newEthereumKeyRotationSubmission("eth-address-2", "new-eth-address-2", 11, ""), MockVerify)
	require.NoError(t, err)
	err = top.ProcessEthereumKeyRotation(ctx, "vega-key-3", newEthereumKeyRotationSubmission("eth-address-3", "new-eth-address-3", 13, ""), MockVerify)
	require.NoError(t, err)
	err = top.ProcessEthereumKeyRotation(ctx, "vega-key-4", newEthereumKeyRotationSubmission("eth-address-4", "new-eth-address-4", 13, ""), MockVerify)
	require.NoError(t, err)

	// when
	top.BeginBlock(ctx, 11, "")
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
	top.BeginBlock(ctx, 13, "")
	// then
	data3 = top.Get("vega-master-pubkey-3")
	require.NotNil(t, data3)
	assert.Equal(t, "new-eth-address-3", data3.EthereumAddress)
	data4 = top.Get("vega-master-pubkey-4")
	require.NotNil(t, data4)
	assert.Equal(t, "new-eth-address-4", data4.EthereumAddress)
}

func TestEthereumKeyRotationBeginBlockWithSubmitter(t *testing.T) {
	top := getTestTopWithMockedSignatures(t)
	defer top.ctrl.Finish()

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

	top.multisigTop.EXPECT().ChainID().Times(1)
	top.multisigTop2.EXPECT().ChainID().Times(1)
	top.signatures.EXPECT().PrepareValidatorSignatures(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(3)
	top.signatures.EXPECT().EmitValidatorRemovedSignatures(gomock.Any(), submitter, gomock.Any(), gomock.Any(), gomock.Any()).Times(4)
	top.signatures.EXPECT().EmitValidatorAddedSignatures(gomock.Any(), submitter, gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
	top.signatures.EXPECT().ClearStaleSignatures().AnyTimes()
	top.timeService.EXPECT().GetTimeNow().Times(2)

	// add ethereum key rotations
	err := top.ProcessEthereumKeyRotation(ctx, "vega-key-1", newEthereumKeyRotationSubmission("eth-address-1", "new-eth-address-1", 11, submitter), MockVerify)
	require.NoError(t, err)

	// when
	now := time.Unix(666, 666)
	top.signatures.EXPECT().SetNonce(now).Times(1)
	top.timeService.EXPECT().GetTimeNow().Times(6).Return(now)
	top.multisigTop.EXPECT().ChainID().Times(1)
	top.multisigTop2.EXPECT().ChainID().Times(1)
	top.BeginBlock(ctx, 11, "")

	// then
	data1 := top.Get("vega-master-pubkey-1")
	require.NotNil(t, data1)
	assert.Equal(t, "new-eth-address-1", data1.EthereumAddress)

	// now try to add a new rotation before resolving the contract
	err = top.ProcessEthereumKeyRotation(ctx, "vega-key-1", newEthereumKeyRotationSubmission("eth-address-1", "new-eth-address-1", 13, submitter), MockVerify)
	require.Error(t, err, validators.ErrNodeHasUnresolvedRotation)

	// Now make it look like the old key is removed from the multisig contract
	top.multisigTop.EXPECT().IsSigner(gomock.Any()).Return(false).Times(1)
	top.multisigTop.EXPECT().IsSigner(gomock.Any()).Return(true).Times(1)

	now = now.Add(time.Second)
	top.signatures.EXPECT().SetNonce(now).Times(1)
	top.timeService.EXPECT().GetTimeNow().Times(6).Return(now)
	top.multisigTop.EXPECT().ChainID().Times(1)
	top.multisigTop2.EXPECT().ChainID().Times(1)
	top.BeginBlock(ctx, 140, "")

	// try to submit again
	err = top.ProcessEthereumKeyRotation(ctx, "vega-key-1", newEthereumKeyRotationSubmission("new-eth-address-1", "new-eth-address-2", 150, submitter), MockVerify)
	require.NoError(t, err)
}

func testNoSignaturesForNonTendermint(t *testing.T) {
	ctx := context.Background()

	tcs := []struct {
		name   string
		status validators.ValidatorStatus
	}{
		{
			name:   "no signatures when pending",
			status: validators.ValidatorStatusPending,
		},
		{
			name:   "no signatures when ersatz",
			status: validators.ValidatorStatusErsatz,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			top := getTestTopWithMockedSignatures(t)
			defer top.ctrl.Finish()

			nr := &commandspb.AnnounceNode{
				Id:              "vega-master-pubkey",
				ChainPubKey:     tmPubKey,
				VegaPubKey:      "vega-key",
				EthereumAddress: "eth-address",
			}

			err := top.AddNewNode(ctx, nr, tc.status)
			require.NoError(t, err)

			ekr := newEthereumKeyRotationSubmission(nr.EthereumAddress, "new-eth-address", 150, "")
			err = top.ProcessEthereumKeyRotation(ctx, nr.VegaPubKey, ekr, MockVerify)
			require.NoError(t, err)
		})
	}
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

	// set a reasonable block height
	top.timeService.EXPECT().GetTimeNow().Times(3)
	signatures.EXPECT().ClearStaleSignatures().Times(1)
	signatures.EXPECT().SetNonce(gomock.Any()).Times(1)
	signatures.EXPECT().OfferSignatures().AnyTimes()
	top.BeginBlock(context.Background(), 10, "")

	return &testTopWithSignatures{
		testTop:    top,
		signatures: signatures,
	}
}

func MockVerify(message, signature []byte, hexAddress string) error {
	return nil
}
