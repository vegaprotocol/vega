package validators_test

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	vgcrypto "code.vegaprotocol.io/shared/libs/crypto"
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
	"github.com/stretchr/testify/mock"
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
	assert.False(t, top.IsValidateNodeID("vega-master-pubkey"))

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
	assert.True(t, top.IsValidateNodeID("vega-master-pubkey"))
}

func testGetByKey(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([]string{tmPubKey})

	assert.False(t, top.IsValidatorVegaPubKey("vega-key"))
	assert.False(t, top.IsValidateNodeID("vega-master-pubkey"))

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
		nr.VegaPubKeyNumber,
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

func TestValidatorTopologyKeyRotate(t *testing.T) {
	t.Run("add key rotate - success", testAddKeyRotateSuccess)
	t.Run("add key rotate - fails when node does not exists", testAddKeyRotateSuccessFailsOnNonExistingNode)
	t.Run("add key rotate - fails when target block height is less then current block height", testAddKeyRotateSuccessFailsWhenTargetBlockHeightIsLessThenCurrentBlockHeight)
	t.Run("add key rotate - fails when new key number is less then current current key number", testAddKeyRotateSuccessFailsWhenNewKeyNumberIsLessThenCurrentKeyNumber)
	t.Run("add key rotate - fails when key rotation for node already exists", testAddKeyRotateSuccessFailsWhenKeyRotationForNodeAlreadyExists)
	t.Run("add key rotate - fails when current pub key hash does not match", testAddKeyRotateSuccessFailsWhenCurrentPubKeyHashDoesNotMatch)
	t.Run("beginning of block - success", testBeginBlockSuccess)
	t.Run("beginning of block - notify key change", testBeginBlockNotifyKeyChange)
	t.Run("beginning of block - adds to processed key rotations", testBeginBlockAddsToProcessedRotations)
}

func testAddKeyRotateSuccess(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([]string{tmPubKey})

	id := "vega-master-pubkey"
	vegaPubKey := "vega-key"
	newVegaPubKey := fmt.Sprintf("new-%s", vegaPubKey)

	nr := commandspb.NodeRegistration{
		Id:              id,
		ChainPubKey:     tmPubKey,
		VegaPubKey:      vegaPubKey,
		EthereumAddress: "eth-address",
	}
	ctx := context.TODO()
	err := top.AddNodeRegistration(ctx, &nr)
	assert.NoError(t, err)

	kr := &commandspb.KeyRotateSubmission{
		KeyNumber:         1,
		TargetBlock:       15,
		NewPubKey:         newVegaPubKey,
		CurrentPubKeyHash: hashKey(vegaPubKey),
	}

	err = top.AddKeyRotate(ctx, id, 10, kr)
	assert.NoError(t, err)
}

func testAddKeyRotateSuccessFailsOnNonExistingNode(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([]string{tmPubKey})

	id := "vega-master-pubkey"
	newVegaPubKey := "new-ega-key"

	ctx := context.TODO()

	err := top.AddKeyRotate(ctx, id, 10, newKeyRotationSubmission("", newVegaPubKey, 1, 10))
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to add key rotate for non existing node \"vega-master-pubkey\"")
}

func testAddKeyRotateSuccessFailsWhenTargetBlockHeightIsLessThenCurrentBlockHeight(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([]string{tmPubKey})

	id := "vega-master-pubkey"
	vegaPubKey := "vega-key"
	newVegaPubKey := fmt.Sprintf("new-%s", vegaPubKey)

	nr := commandspb.NodeRegistration{
		Id:              id,
		ChainPubKey:     tmPubKey,
		VegaPubKey:      vegaPubKey,
		EthereumAddress: "eth-address",
	}
	ctx := context.TODO()
	err := top.AddNodeRegistration(ctx, &nr)
	assert.NoError(t, err)

	err = top.AddKeyRotate(ctx, id, 15, newKeyRotationSubmission(vegaPubKey, newVegaPubKey, 1, 10))
	assert.ErrorIs(t, err, validators.ErrTargetBlockHeightMustBeGraterThanCurrentHeight)
}

func testAddKeyRotateSuccessFailsWhenNewKeyNumberIsLessThenCurrentKeyNumber(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([]string{tmPubKey})

	id := "vega-master-pubkey"
	vegaPubKey := "vega-key"
	newVegaPubKey := fmt.Sprintf("new-%s", vegaPubKey)

	nr := commandspb.NodeRegistration{
		Id:               id,
		ChainPubKey:      tmPubKey,
		VegaPubKey:       vegaPubKey,
		EthereumAddress:  "eth-address",
		VegaPubKeyNumber: 2,
	}
	ctx := context.TODO()
	err := top.AddNodeRegistration(ctx, &nr)
	assert.NoError(t, err)

	// test less then
	err = top.AddKeyRotate(ctx, id, 10, newKeyRotationSubmission(vegaPubKey, newVegaPubKey, 1, 15))
	assert.ErrorIs(t, err, validators.ErrNewVegaPubKeyNumberMustBeGreaterThenCurrentPubKeyNumber)
}

func testAddKeyRotateSuccessFailsWhenKeyRotationForNodeAlreadyExists(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([]string{tmPubKey})

	id := "vega-master-pubkey"
	vegaPubKey := "vega-key"
	newVegaPubKey := fmt.Sprintf("new-%s", vegaPubKey)

	nr := commandspb.NodeRegistration{
		Id:               id,
		ChainPubKey:      tmPubKey,
		VegaPubKey:       vegaPubKey,
		EthereumAddress:  "eth-address",
		VegaPubKeyNumber: 1,
	}
	ctx := context.TODO()
	err := top.AddNodeRegistration(ctx, &nr)
	assert.NoError(t, err)

	// add first
	err = top.AddKeyRotate(ctx, id, 10, newKeyRotationSubmission(vegaPubKey, newVegaPubKey, 2, 12))
	assert.NoError(t, err)

	// add second
	err = top.AddKeyRotate(ctx, id, 10, newKeyRotationSubmission(vegaPubKey, newVegaPubKey, 2, 13))
	assert.ErrorIs(t, err, validators.ErrNodeAlreadyHasPendingKeyRotation)
}

func testAddKeyRotateSuccessFailsWhenCurrentPubKeyHashDoesNotMatch(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()
	top.UpdateValidatorSet([]string{tmPubKey})

	id := "vega-master-pubkey"
	vegaPubKey := "vega-key"
	newVegaPubKey := fmt.Sprintf("new-%s", vegaPubKey)

	nr := commandspb.NodeRegistration{
		Id:               id,
		ChainPubKey:      tmPubKey,
		VegaPubKey:       vegaPubKey,
		EthereumAddress:  "eth-address",
		VegaPubKeyNumber: 1,
	}
	ctx := context.TODO()
	err := top.AddNodeRegistration(ctx, &nr)
	assert.NoError(t, err)

	err = top.AddKeyRotate(ctx, id, 10, newKeyRotationSubmission("random-key", newVegaPubKey, 2, 12))
	assert.ErrorIs(t, err, validators.ErrCurrentPubKeyHashDoesNotMatch)
}

func hashKey(key string) string {
	return hex.EncodeToString(vgcrypto.Hash([]byte(key)))
}

func newKeyRotationSubmission(currentPubKey, newVegaPubKey string, keyNumber uint32, targetBlock uint64) *commandspb.KeyRotateSubmission {
	return &commandspb.KeyRotateSubmission{
		KeyNumber:         keyNumber,
		TargetBlock:       targetBlock,
		NewPubKey:         newVegaPubKey,
		CurrentPubKeyHash: hashKey(currentPubKey),
	}
}

func testBeginBlockSuccess(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()

	chainValidators := []string{"tm-pubkey-1", "tm-pubkey-2", "tm-pubkey-3", "tm-pubkey-4"}
	top.UpdateValidatorSet(chainValidators)

	ctx := context.TODO()
	for i := 0; i < len(chainValidators); i++ {
		j := i + 1
		id := fmt.Sprintf("vega-master-pubkey-%d", j)
		nr := commandspb.NodeRegistration{
			Id:              id,
			ChainPubKey:     chainValidators[i],
			VegaPubKey:      fmt.Sprintf("vega-key-%d", j),
			EthereumAddress: fmt.Sprintf("eth-address-%d", j),
		}

		err := top.AddNodeRegistration(ctx, &nr)
		assert.NoErrorf(t, err, "failed to add node registation %s", id)
	}

	// add key rotations
	err := top.AddKeyRotate(ctx, "vega-master-pubkey-1", 10, newKeyRotationSubmission("vega-key-1", "new-vega-key-1", 1, 11))
	assert.NoError(t, err)
	err = top.AddKeyRotate(ctx, "vega-master-pubkey-2", 10, newKeyRotationSubmission("vega-key-2", "new-vega-key-2", 1, 11))
	assert.NoError(t, err)
	err = top.AddKeyRotate(ctx, "vega-master-pubkey-3", 10, newKeyRotationSubmission("vega-key-3", "new-vega-key-3", 1, 13))
	assert.NoError(t, err)
	err = top.AddKeyRotate(ctx, "vega-master-pubkey-4", 10, newKeyRotationSubmission("vega-key-4", "new-vega-key-4", 1, 13))
	assert.NoError(t, err)

	// when
	top.BeginBlock(ctx, 11)
	// then
	data1 := top.Get("vega-master-pubkey-1")
	assert.NotNil(t, data1)
	assert.Equal(t, "new-vega-key-1", data1.VegaPubKey)
	data2 := top.Get("vega-master-pubkey-2")
	assert.NotNil(t, data2)
	assert.Equal(t, "new-vega-key-2", data2.VegaPubKey)
	data3 := top.Get("vega-master-pubkey-3")
	assert.NotNil(t, data3)
	assert.Equal(t, "vega-key-3", data3.VegaPubKey)
	data4 := top.Get("vega-master-pubkey-4")
	assert.NotNil(t, data4)
	assert.Equal(t, "vega-key-4", data4.VegaPubKey)

	// when
	top.BeginBlock(ctx, 13)
	// then
	data3 = top.Get("vega-master-pubkey-3")
	assert.NotNil(t, data3)
	assert.Equal(t, "new-vega-key-3", data3.VegaPubKey)
	data4 = top.Get("vega-master-pubkey-4")
	assert.NotNil(t, data4)
	assert.Equal(t, "new-vega-key-4", data4.VegaPubKey)
}

type Callback struct {
	mock.Mock
}

func (m *Callback) Call(ctx context.Context, a, b string) {
	m.Called(ctx, a, b)
}

func newCallback(times int) Callback {
	c := Callback{}
	c.On("Call", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Times(times)
	return c
}

func testBeginBlockNotifyKeyChange(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()

	chainValidators := []string{"tm-pubkey-1", "tm-pubkey-2"}
	top.UpdateValidatorSet(chainValidators)

	ctx := context.TODO()
	for i := 0; i < len(chainValidators); i++ {
		j := i + 1
		id := fmt.Sprintf("vega-master-pubkey-%d", j)
		nr := commandspb.NodeRegistration{
			Id:              id,
			ChainPubKey:     chainValidators[i],
			VegaPubKey:      fmt.Sprintf("vega-key-%d", j),
			EthereumAddress: fmt.Sprintf("eth-address-%d", j),
		}

		err := top.AddNodeRegistration(ctx, &nr)
		assert.NoErrorf(t, err, "failed to add node registation %s", id)
	}

	// add key rotations
	err := top.AddKeyRotate(ctx, "vega-master-pubkey-1", 10, newKeyRotationSubmission("vega-key-1", "new-vega-key-1", 1, 11))
	assert.NoError(t, err)
	err = top.AddKeyRotate(ctx, "vega-master-pubkey-2", 10, newKeyRotationSubmission("vega-key-2", "new-vega-key-2", 1, 11))
	assert.NoError(t, err)

	// register callbacks
	c1 := newCallback(2)
	c2 := newCallback(2)
	top.NotifyOnKeyChange(c1.Call, c2.Call)

	// when
	top.BeginBlock(ctx, 11)

	// then
	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
}

func testBeginBlockAddsToProcessedRotations(t *testing.T) {
	top := getTestTopWithDefaultValidator(t)
	defer top.ctrl.Finish()

	chainValidators := []string{"tm-pubkey-1", "tm-pubkey-2"}
	top.UpdateValidatorSet(chainValidators)

	ctx := context.TODO()
	for i := 0; i < len(chainValidators); i++ {
		j := i + 1
		id := fmt.Sprintf("vega-master-pubkey-%d", j)
		nr := commandspb.NodeRegistration{
			Id:              id,
			ChainPubKey:     chainValidators[i],
			VegaPubKey:      fmt.Sprintf("vega-key-%d", j),
			EthereumAddress: fmt.Sprintf("eth-address-%d", j),
		}

		err := top.AddNodeRegistration(ctx, &nr)
		assert.NoErrorf(t, err, "failed to add node registation %s", id)
	}

	// add key rotations
	err := top.AddKeyRotate(ctx, "vega-master-pubkey-1", 10, newKeyRotationSubmission("vega-key-1", "new-vega-key-1", 1, 11))
	assert.NoError(t, err)
	err = top.AddKeyRotate(ctx, "vega-master-pubkey-2", 10, newKeyRotationSubmission("vega-key-2", "new-vega-key-2", 1, 12))
	assert.NoError(t, err)

	// when
	top.BeginBlock(ctx, 11)

	// then
	rotations := top.GetKeyRotations("vega-master-pubkey-2")
	assert.Len(t, rotations, 0)
	rotations = top.GetKeyRotations("vega-master-pubkey-1")
	assert.Len(t, rotations, 1)
	assert.Equal(t,
		validators.KeyRotation{
			NodeID:      "vega-master-pubkey-1",
			OldPubKey:   "vega-key-1",
			NewPubKey:   "new-vega-key-1",
			BlockHeight: 11,
		},
		rotations[0],
	)

	// add key rotation to previous node
	err = top.AddKeyRotate(ctx, "vega-master-pubkey-1", 10, newKeyRotationSubmission("new-vega-key-1", "new-2-vega-key-1", 2, 12))
	assert.NoError(t, err)

	// when
	top.BeginBlock(ctx, 12)

	// then
	rotations = top.GetKeyRotations("vega-master-pubkey-2")
	assert.Len(t, rotations, 1)
	assert.Equal(t,
		validators.KeyRotation{
			NodeID:      "vega-master-pubkey-2",
			OldPubKey:   "vega-key-2",
			NewPubKey:   "new-vega-key-2",
			BlockHeight: 12,
		},
		rotations[0],
	)

	rotations = top.GetKeyRotations("vega-master-pubkey-1")
	assert.Len(t, rotations, 2)
	assert.Equal(t,
		validators.KeyRotation{
			NodeID:      "vega-master-pubkey-1",
			OldPubKey:   "vega-key-1",
			NewPubKey:   "new-vega-key-1",
			BlockHeight: 11,
		},
		rotations[0],
	)
	assert.Equal(t,
		validators.KeyRotation{
			NodeID:      "vega-master-pubkey-1",
			OldPubKey:   "new-vega-key-1",
			NewPubKey:   "new-2-vega-key-1",
			BlockHeight: 12,
		},
		rotations[1],
	)

}
