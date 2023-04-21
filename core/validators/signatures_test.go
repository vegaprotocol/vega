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
	"crypto/ed25519"
	"encoding/hex"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/core/validators/mocks"
	"code.vegaprotocol.io/vega/logging"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSignatures struct {
	*validators.ERC20Signatures
	notary           *mocks.MockNotary
	ctrl             *gomock.Controller
	broker           *bmocks.MockBroker
	signer           testSigner
	multisigTopology *mocks.MockMultiSigTopology
}

func getTestSignatures(t *testing.T) *testSignatures {
	t.Helper()
	ctrl := gomock.NewController(t)
	notary := mocks.NewMockNotary(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	nodewallet := mocks.NewMockNodeWallets(ctrl)
	multisigTopology := mocks.NewMockMultiSigTopology(ctrl)
	tsigner := testSigner{}
	nodewallet.EXPECT().GetEthereum().AnyTimes().Return(tsigner)

	return &testSignatures{
		ERC20Signatures: validators.NewSignatures(
			logging.NewTestLogger(),
			multisigTopology,
			notary,
			nodewallet,
			broker,
			true,
		),
		ctrl:             ctrl,
		notary:           notary,
		broker:           broker,
		signer:           tsigner,
		multisigTopology: multisigTopology,
	}
}

func TestPromotionSignatures(t *testing.T) {
	ctx := context.Background()
	signatures := getTestSignatures(t)
	defer signatures.ctrl.Finish()

	// previous state, 2 validators, 1 non validator
	previousState := map[string]validators.StatusAddress{
		"8fd85dac403623ea3b894e9e342571716eedf550b3b1953e2c29eb58a6da683a": {
			Status:     validators.ValidatorStatusTendermint,
			EthAddress: "0xddDFA1974b156336b9c49579A2bC4e0a7059CAD0",
		},
		"927cbf8d5909cc017cf78ea9806fd57c3115d37e481eaf9d866f526b356f3ced": {
			Status:     validators.ValidatorStatusTendermint,
			EthAddress: "0x5945ae02D5EE15181cc4AC0f5EaeF4C25Dc17Aa8",
		},
		"95893347980299679883f817f118718f949826d1a0a1c2e4f22ba5f0cd6d1f5d": {
			Status:     validators.ValidatorStatusTendermint,
			EthAddress: "0x539ac90d9523f878779491D4175dc11AD09972F0",
		},
		"4554375ce61b6828c6f7b625b7735034496b7ea19951509cccf4eb2ba35011b0": {
			Status:     validators.ValidatorStatusErsatz,
			EthAddress: "0x7629Faf5B7a3BB167B6f2F86DB5fB7f13B20Ee90",
		},
	}
	// based on the previous state, the validators in order:
	// - 1 stays a validator
	// - 1 validators became erzatz
	// - 1 validators completely removed
	// - 1 erzatz became validator

	newState := map[string]validators.StatusAddress{
		"8fd85dac403623ea3b894e9e342571716eedf550b3b1953e2c29eb58a6da683a": {
			Status:     validators.ValidatorStatusTendermint,
			EthAddress: "0xddDFA1974b156336b9c49579A2bC4e0a7059CAD0",
		},
		"927cbf8d5909cc017cf78ea9806fd57c3115d37e481eaf9d866f526b356f3ced": {
			Status:     validators.ValidatorStatusErsatz,
			EthAddress: "0x5945ae02D5EE15181cc4AC0f5EaeF4C25Dc17Aa8",
		},
		"4554375ce61b6828c6f7b625b7735034496b7ea19951509cccf4eb2ba35011b0": {
			Status:     validators.ValidatorStatusTendermint,
			EthAddress: "0x7629Faf5B7a3BB167B6f2F86DB5fB7f13B20Ee90",
		},
	}

	currentTime := time.Unix(10, 0)

	// just aggregate all events
	// we'll verify their content after
	evts := []events.Event{}
	signatures.broker.EXPECT().SendBatch(gomock.Any()).Times(5).DoAndReturn(func(newEvts []events.Event) {
		evts = append(evts, newEvts...)
	})

	signatures.notary.EXPECT().StartAggregate(gomock.Any(), gomock.Any(), gomock.Any()).Times(5)

	// now, there's no assertion to do just now, this only send a sh*t ton of events
	signatures.PreparePromotionsSignatures(
		ctx,
		currentTime,
		12,
		previousState,
		newState,
	)

	assert.Len(t, evts, 0)

	// now request the signature bundle for the adding
	err := signatures.EmitValidatorAddedSignatures(ctx, "0x7629Faf5B7a3BB167B6f2F86DB5fB7f13B20Ee90", "4554375ce61b6828c6f7b625b7735034496b7ea19951509cccf4eb2ba35011b0", currentTime)
	require.NoError(t, err)

	// now ask for all the removes, each tendermint validator will ask
	toAsk := []string{
		"0xddDFA1974b156336b9c49579A2bC4e0a7059CAD0",
		"0x5945ae02D5EE15181cc4AC0f5EaeF4C25Dc17Aa8",
	}

	for _, v := range toAsk {
		err = signatures.EmitValidatorRemovedSignatures(ctx, v, "927cbf8d5909cc017cf78ea9806fd57c3115d37e481eaf9d866f526b356f3ced", currentTime)
		require.NoError(t, err)
	}

	for _, v := range toAsk {
		err = signatures.EmitValidatorRemovedSignatures(ctx, v, "95893347980299679883f817f118718f949826d1a0a1c2e4f22ba5f0cd6d1f5d", currentTime)
		require.NoError(t, err)
	}

	require.Len(t, evts, 5)

	t.Run("ensure all correct events are sent", func(t *testing.T) {
		add1, ok := evts[0].(*events.ERC20MultiSigSignerAdded)
		assert.True(t, ok, "invalid event, expected SignedAdded")
		assert.Equal(t, add1.ERC20MultiSigSignerAdded().NewSigner, "0x7629Faf5B7a3BB167B6f2F86DB5fB7f13B20Ee90")

		remove1, ok := evts[1].(*events.ERC20MultiSigSignerRemoved)
		assert.True(t, ok, "invalid event, expected SignedRemoved")
		assert.Equal(t, remove1.ERC20MultiSigSignerRemoved().OldSigner, "0x5945ae02D5EE15181cc4AC0f5EaeF4C25Dc17Aa8")

		remove2, ok := evts[1].(*events.ERC20MultiSigSignerRemoved)
		assert.True(t, ok, "invalid event, expected SignedRemoved")
		assert.Equal(t, remove2.ERC20MultiSigSignerRemoved().OldSigner, "0x5945ae02D5EE15181cc4AC0f5EaeF4C25Dc17Aa8")

		// check the two removes on the same node have the same nonce
		assert.Equal(t, remove1.ERC20MultiSigSignerRemoved().Nonce, remove2.ERC20MultiSigSignerRemoved().Nonce)

		remove3, ok := evts[3].(*events.ERC20MultiSigSignerRemoved)
		assert.True(t, ok, "invalid event, expected SignedRemoved")
		assert.Equal(t, remove3.ERC20MultiSigSignerRemoved().OldSigner, "0x539ac90d9523f878779491D4175dc11AD09972F0")

		remove4, ok := evts[4].(*events.ERC20MultiSigSignerRemoved)
		assert.True(t, ok, "invalid event, expected SignedRemoved")
		assert.Equal(t, remove4.ERC20MultiSigSignerRemoved().OldSigner, "0x539ac90d9523f878779491D4175dc11AD09972F0")

		assert.Equal(t, remove3.ERC20MultiSigSignerRemoved().Nonce, remove4.ERC20MultiSigSignerRemoved().Nonce)
	})

	t.Run("test snapshots", func(t *testing.T) {
		state := signatures.SerialisePendingSignatures()
		snap := getTestSignatures(t)
		snap.RestorePendingSignatures(state)

		snap.broker.EXPECT().SendBatch(gomock.Any()).Times(len(toAsk) + 1)

		// check the pending signatures still exist (we get no error) and that "already issued" is restored (notary mock should not expect anything)
		require.NoError(t, snap.EmitValidatorAddedSignatures(ctx, "0x7629Faf5B7a3BB167B6f2F86DB5fB7f13B20Ee90", "4554375ce61b6828c6f7b625b7735034496b7ea19951509cccf4eb2ba35011b0", currentTime))
		for _, v := range toAsk {
			require.NoError(t, snap.EmitValidatorRemovedSignatures(ctx, v, "95893347980299679883f817f118718f949826d1a0a1c2e4f22ba5f0cd6d1f5d", currentTime))
		}
	})

	t.Run("clear stale remove signatures", func(t *testing.T) {
		// return that the signers are not on the contract
		signatures.multisigTopology.EXPECT().IsSigner(gomock.Any()).Return(false).Times(3)
		signatures.ClearStaleSignatures()

		// we should get no signatures for the removed nodes
		require.Error(t, validators.ErrNoPendingSignaturesForNodeID, signatures.EmitValidatorRemovedSignatures(ctx, "submitter", "927cbf8d5909cc017cf78ea9806fd57c3115d37e481eaf9d866f526b356f3ced", currentTime))

		// now for the add signatures
		signatures.multisigTopology.EXPECT().IsSigner(gomock.Any()).Return(true).Times(1)
		signatures.ClearStaleSignatures()
		require.Error(t, validators.ErrNoPendingSignaturesForNodeID, signatures.EmitValidatorAddedSignatures(ctx, "0x7629Faf5B7a3BB167B6f2F86DB5fB7f13B20Ee90", "4554375ce61b6828c6f7b625b7735034496b7ea19951509cccf4eb2ba35011b0", currentTime))
	})
}

func TestOfferSignatures(t *testing.T) {
	ctx := context.Background()
	signatures := getTestSignatures(t)
	defer signatures.ctrl.Finish()

	signatures.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	var addSigResID string
	var addSig []byte
	signatures.notary.EXPECT().
		StartAggregate(gomock.Any(), types.NodeSignatureKindERC20MultiSigSignerAdded, gomock.Any()).DoAndReturn(
		func(resID string, kind types.NodeSignatureKind, signature []byte) {
			addSigResID = resID
			addSig = signature
		},
	)

	var removeSigResID string
	var removeSig []byte
	signatures.notary.EXPECT().
		StartAggregate(gomock.Any(), types.NodeSignatureKindERC20MultiSigSignerRemoved, gomock.Any()).DoAndReturn(
		func(resID string, kind types.NodeSignatureKind, signature []byte) {
			removeSigResID = resID
			removeSig = signature
		},
	)

	submitter := "node_operator"
	now := time.Now()
	validator := validators.NodeIDAddress{NodeID: "node_1", EthAddress: "eth_address"}

	signatures.PrepareValidatorSignatures(ctx, []validators.NodeIDAddress{validator}, 1, false)
	err := signatures.EmitValidatorRemovedSignatures(ctx, submitter, validator.NodeID, now)
	require.NoError(t, err)

	validator.EthAddress = "updated_eth_address"

	signatures.PrepareValidatorSignatures(ctx, []validators.NodeIDAddress{validator}, 1, true)
	err = signatures.EmitValidatorAddedSignatures(ctx, submitter, validator.NodeID, now)
	require.NoError(t, err)

	signatures.notary.EXPECT().
		OfferSignatures(types.NodeSignatureKindERC20MultiSigSignerAdded, gomock.Any()).DoAndReturn(
		func(kind types.NodeSignatureKind, f func(id string) []byte) {
			require.Equal(t, addSig, f(addSigResID))
		},
	)

	signatures.notary.EXPECT().
		OfferSignatures(types.NodeSignatureKindERC20MultiSigSignerRemoved, gomock.Any()).DoAndReturn(
		func(kind types.NodeSignatureKind, f func(id string) []byte) {
			require.Equal(t, removeSig, f(removeSigResID))
		},
	)

	signatures.OfferSignatures()
}

const (
	privKey = "9feb9cbee69c1eeb30db084544ff8bf92166bf3fddefa6a021b458b4de04c66758a127387b1dff15b71fd7d0a9fd104ed75da4aac549efd5d149051ea57cefaf"
	pubKey  = "58a127387b1dff15b71fd7d0a9fd104ed75da4aac549efd5d149051ea57cefaf"
)

type testSigner struct{}

func (s testSigner) Algo() string { return "ed25519" }

func (s testSigner) Sign(msg []byte) ([]byte, error) {
	priv, _ := hex.DecodeString(privKey)

	return ed25519.Sign(ed25519.PrivateKey(priv), msg), nil
}

func (s testSigner) Verify(msg, sig []byte) bool {
	pub, _ := hex.DecodeString(pubKey)
	hash := crypto.Keccak256(msg)

	return ed25519.Verify(ed25519.PublicKey(pub), hash, sig)
}
