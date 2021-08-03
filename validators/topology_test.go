package validators_test

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/validators/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	pubkey = "0f9041e6d5b83d3577d02de3e92c39a2ce1e5aeeee2c40cfbd28a339a3e2e265"
)

var tmPubKey = []byte("tm-pub-key")

type testTop struct {
	*validators.Topology
	ctrl   *gomock.Controller
	wallet *mocks.MockWallet
}

func getTestTopWithDefaultValidator(t *testing.T) *testTop {
	ctrl := gomock.NewController(t)
	wallet := mocks.NewMockWallet(ctrl)

	bytesKey, _ := hex.DecodeString(pubkey)
	wallet.EXPECT().PubKeyOrAddress().Times(1).Return(crypto.NewPublicKeyOrAddress(pubkey, bytesKey))

	defaultTmPubKey := []byte("default-tm-public-key")
	defaultTmPubKeyBase64 := base64.StdEncoding.EncodeToString([]byte(defaultTmPubKey))

	top := validators.NewTopology(logging.NewTestLogger(), validators.NewDefaultConfig(), wallet)
	// Add Tendermint public key to validator set
	top.UpdateValidatorSet([][]byte{defaultTmPubKey})

	state := struct {
		Validators map[string]validators.ValidatorData
	}{
		Validators: map[string]validators.ValidatorData{
			defaultTmPubKeyBase64: {
				PubKey:  pubkey,
				InfoURL: "n0.xyz.vege/node/123",
				Country: "GB",
			},
		},
	}

	buf, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("error marshalling state %v", err)
	}

	if err := top.LoadValidatorsOnGenesis(context.Background(), buf); err != nil {
		t.Fatalf("error loading validators on genesis: %v", err)
	}

	return &testTop{
		Topology: top,
		ctrl:     ctrl,
		wallet:   wallet,
	}
}

func TestValidatorTopology(t *testing.T) {
	t.Run("add node registration - success", testAddNodeRegistrationSuccess)
	t.Run("add node registration - failure", testAddNodeRegistrationFailure)
	t.Run("topology validators length is equal to number of added validators", testGetLen)
	t.Run("added validator exists in topology", testExists)
	t.Run("test get by key", testGetByKey)
}

func testAddNodeRegistrationSuccess(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([][]byte{tmPubKey})

	nr := commandspb.NodeRegistration{
		ChainPubKey: []byte(tmPubKey),
		PubKey:      []byte("vega-key"),
	}
	err := top.AddNodeRegistration(&nr)
	assert.NoError(t, err)
}

func testAddNodeRegistrationFailure(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([][]byte{tmPubKey})

	nr := commandspb.NodeRegistration{
		ChainPubKey: []byte(tmPubKey),
		PubKey:      []byte("vega-key"),
	}
	err := top.AddNodeRegistration(&nr)
	assert.NoError(t, err)

	nr = commandspb.NodeRegistration{
		ChainPubKey: []byte(tmPubKey),
		PubKey:      []byte("vega-key-2"),
	}
	err = top.AddNodeRegistration(&nr)
	assert.Error(t, err)
}

func testGetLen(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([][]byte{tmPubKey})

	// first the len is 1 since the default validator loaded from genenesis
	assert.Equal(t, 1, top.Len())

	nr := commandspb.NodeRegistration{
		ChainPubKey: []byte(tmPubKey),
		PubKey:      []byte("vega-key"),
	}
	err := top.AddNodeRegistration(&nr)
	assert.NoError(t, err)

	assert.Equal(t, 2, top.Len())
}

func testExists(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([][]byte{tmPubKey})

	assert.False(t, top.Exists([]byte("vega-key")))

	nr := commandspb.NodeRegistration{
		ChainPubKey: []byte(tmPubKey),
		PubKey:      []byte("vega-key"),
	}
	err := top.AddNodeRegistration(&nr)
	assert.NoError(t, err)

	assert.True(t, top.Exists([]byte("vega-key")))
}

func testGetByKey(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([][]byte{tmPubKey})

	assert.False(t, top.Exists([]byte("vega-key")))

	nr := commandspb.NodeRegistration{
		ChainPubKey: []byte(tmPubKey),
		PubKey:      []byte("vega-key"),
		InfoUrl:     "n0.xyz.vega/node/url/random",
		Country:     "CZ",
	}
	err := top.AddNodeRegistration(&nr)
	assert.NoError(t, err)

	expectedData := &validators.ValidatorData{
		PubKey:  string(nr.PubKey),
		InfoURL: nr.InfoUrl,
		Country: nr.Country,
	}

	actualData := top.GetByKey(nr.PubKey)
	assert.NotNil(t, actualData)

	assert.Equal(t, expectedData, actualData)
}
