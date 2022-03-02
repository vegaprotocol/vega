package validators_test

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/validators/mocks"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testSignatures struct {
	*validators.ERC20Signatures
	notary *mocks.MockNotary
	ctrl   *gomock.Controller
	broker *bmocks.MockBroker
	signer testSigner
}

func getTestSignatures(t *testing.T) *testSignatures {
	ctrl := gomock.NewController(t)
	notary := mocks.NewMockNotary(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	tsigner := testSigner{}

	return &testSignatures{
		ERC20Signatures: validators.NewSignatures(
			logging.NewTestLogger(),
			notary,
			tsigner,
			broker,
		),
		ctrl:   ctrl,
		notary: notary,
		broker: broker,
		signer: tsigner,
	}
}

func TestPromotionSignatures(t *testing.T) {
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

	// just agregate all events
	// we'll verify their content after
	evts := []events.Event{}
	signatures.broker.EXPECT().SendBatch(gomock.Any()).Times(2).DoAndReturn(func(newEvts []events.Event) {
		evts = append(evts, newEvts...)
	})

	signatures.notary.EXPECT().StartAggregate(gomock.Any(), gomock.Any(), gomock.Any()).Times(5)

	// now, there's no assertion to do just now, this only send a sh*t ton of events
	signatures.EmitPromotionsSignatures(
		context.Background(),
		currentTime,
		previousState,
		newState,
	)

	assert.Len(t, evts, 3)

	t.Run("ensure all correct events are sent", func(t *testing.T) {
		add1, ok := evts[0].(*events.ERC20MultiSigSignerAdded)
		assert.True(t, ok, "invalid event, expected SignedAdded")
		assert.Equal(t, add1.ERC20MultiSigSignerAdded().NewSigner, "0x7629Faf5B7a3BB167B6f2F86DB5fB7f13B20Ee90")

		remove1, ok := evts[1].(*events.ERC20MultiSigSignerRemoved)
		assert.True(t, ok, "invalid event, expected SignedRemoved")
		assert.Equal(t, remove1.ERC20MultiSigSignerRemoved().OldSigner, "0x539ac90d9523f878779491D4175dc11AD09972F0")
		// also ensure 2 signature are expected
		assert.Len(t, remove1.ERC20MultiSigSignerRemoved().SignatureSubmitters, 2)

		remove2, ok := evts[2].(*events.ERC20MultiSigSignerRemoved)
		assert.True(t, ok, "invalid event, expected SignedRemoved")
		assert.Equal(t, remove2.ERC20MultiSigSignerRemoved().OldSigner, "0x5945ae02D5EE15181cc4AC0f5EaeF4C25Dc17Aa8")
		// also ensure 2 signature are expected
		assert.Len(t, remove2.ERC20MultiSigSignerRemoved().SignatureSubmitters, 2)
	})
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
