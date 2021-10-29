package validators_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	brokerMocks "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/events"
	vgtesting "code.vegaprotocol.io/vega/libs/testing"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/validators/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var tmPubKey = "tm-pub-key"

type testTop struct {
	*validators.Topology
	ctrl   *gomock.Controller
	wallet *mocks.MockWallet
	broker *brokerMocks.MockBroker
}

func getTestTopology(t *testing.T) *testTop {
	t.Helper()
	ctrl := gomock.NewController(t)

	broker := brokerMocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	dummyPubKey := "iamapubkey"
	pubKey := crypto.NewPublicKey(dummyPubKey, []byte(dummyPubKey))

	wallet := mocks.NewMockWallet(ctrl)
	wallet.EXPECT().PubKey().Return(pubKey).AnyTimes()
	wallet.EXPECT().ID().Return(pubKey).AnyTimes()

	top := validators.NewTopology(logging.NewTestLogger(), validators.NewDefaultConfig(), wallet, broker)
	return &testTop{
		Topology: top,
		ctrl:     ctrl,
		wallet:   wallet,
		broker:   broker,
	}
}

func getTestTopWithDefaultValidator(t *testing.T) *testTop {
	t.Helper()

	top := getTestTopology(t)

	// Add Tendermint public key to validator set

	defaultTmPubKey := "default-tm-public-key"
	defaultTmPubKeyBase64 := base64.StdEncoding.EncodeToString([]byte(defaultTmPubKey))
	top.UpdateValidatorSet([]string{defaultTmPubKeyBase64})

	state := struct {
		Validators map[string]validators.ValidatorData
	}{
		Validators: map[string]validators.ValidatorData{
			defaultTmPubKeyBase64: {
				ID:              top.wallet.PubKey().Hex(),
				VegaPubKey:      top.wallet.PubKey().Hex(),
				TmPubKey:        "asdasd",
				EthereumAddress: "0x123456",
				InfoURL:         "n0.xyz.vega/node/123",
				Country:         "GB",
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

	return top
}

func TestValidatorTopology(t *testing.T) {
	t.Run("add node registration - success", testAddNodeRegistrationSuccess)
	t.Run("add node registration - failure", testAddNodeRegistrationFailure)
	t.Run("test add node registration send event to broker", testAddNodeRegistrationSendsValidatorUpdateEventToBroker)
	t.Run("topology validators length is equal to number of added validators", testGetLen)
	t.Run("added validator exists in topology", testExists)
	t.Run("test get by key", testGetByKey)
}

func testAddNodeRegistrationSuccess(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([]string{tmPubKey})

	nr := commandspb.NodeRegistration{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
	}
	ctx := context.Background()
	err := top.AddNodeRegistration(ctx, &nr)
	assert.NoError(t, err)
}

func testAddNodeRegistrationFailure(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([]string{"tm-pub-key-1", "tm-pub-key-2"})

	nr := commandspb.NodeRegistration{
		Id:              "vega-master-pubkey",
		ChainPubKey:     "tm-pub-key-1",
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
	}
	ctx := context.Background()
	err := top.AddNodeRegistration(ctx, &nr)
	assert.NoError(t, err)

	// Add node with existing VegaPubKey
	nr = commandspb.NodeRegistration{
		Id:              "vega-master-pubkey",
		ChainPubKey:     "tm-pub-key-2",
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address-2",
	}
	err = top.AddNodeRegistration(ctx, &nr)
	assert.Error(t, err)
}

func testGetLen(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([]string{tmPubKey})

	// first the len is 1 since the default validator loaded from genenesis
	assert.Equal(t, 1, top.Len())

	nr := commandspb.NodeRegistration{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
	}
	ctx := context.Background()
	err := top.AddNodeRegistration(ctx, &nr)
	assert.NoError(t, err)

	assert.Equal(t, 2, top.Len())
}

func testExists(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([]string{tmPubKey})

	assert.False(t, top.IsValidatorVegaPubKey("vega-key"))
	assert.False(t, top.IsValidatorNode("vega-master-pubkey"))

	nr := commandspb.NodeRegistration{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
	}
	ctx := context.Background()
	err := top.AddNodeRegistration(ctx, &nr)
	assert.NoError(t, err)

	assert.True(t, top.IsValidatorVegaPubKey("vega-key"))
	assert.True(t, top.IsValidatorNode("vega-master-pubkey"))
}

func testGetByKey(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([]string{tmPubKey})

	assert.False(t, top.IsValidatorVegaPubKey("vega-key"))
	assert.False(t, top.IsValidatorNode("vega-master-pubkey"))

	nr := commandspb.NodeRegistration{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
		InfoUrl:         "n0.xyz.vega/node/url/random",
		Country:         "CZ",
	}
	ctx := context.Background()
	err := top.AddNodeRegistration(ctx, &nr)
	assert.NoError(t, err)

	expectedData := &validators.ValidatorData{
		ID:              "vega-master-pubkey",
		VegaPubKey:      nr.VegaPubKey,
		EthereumAddress: "eth-address",
		TmPubKey:        nr.ChainPubKey,
		InfoURL:         nr.InfoUrl,
		Country:         nr.Country,
	}

	actualData := top.Get(nr.Id)
	assert.NotNil(t, actualData)

	assert.Equal(t, expectedData, actualData)
}

func testAddNodeRegistrationSendsValidatorUpdateEventToBroker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	_, err := nodewallets.GenerateVegaWallet(vegaPaths, "pass", "pass", false)
	require.NoError(t, err)
	wallet, err := nodewallets.GetVegaWallet(vegaPaths, "pass")
	require.NoError(t, err)

	broker := brokerMocks.NewMockBroker(ctrl)
	top := validators.NewTopology(logging.NewTestLogger(), validators.NewDefaultConfig(), wallet, broker)
	top.UpdateValidatorSet([]string{tmPubKey})

	ctx := context.Background()
	nr := commandspb.NodeRegistration{
		Id:              "vega-master-pubkey",
		ChainPubKey:     tmPubKey,
		VegaPubKey:      "vega-key",
		EthereumAddress: "eth-address",
		InfoUrl:         "n0.xyz.vega/node/url/random",
		Country:         "CZ",
		Name:            "validator",
		AvatarUrl:       "http://n0.xyz/avatar",
	}

	updateEvent := events.NewValidatorUpdateEvent(
		ctx,
		nr.Id,
		nr.VegaPubKey,
		nr.EthereumAddress,
		nr.ChainPubKey,
		nr.InfoUrl,
		nr.Country,
		nr.Name,
		nr.AvatarUrl,
	)

	broker.EXPECT().Send(updateEvent).Times(1)

	assert.NoError(t, top.AddNodeRegistration(ctx, &nr))
}
