package erc20multisig_test

import (
	"context"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/validators/erc20multisig"
	"code.vegaprotocol.io/vega/validators/erc20multisig/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testTopology struct {
	*erc20multisig.Topology

	ctrl    *gomock.Controller
	broker  *bmocks.MockBrokerI
	witness *mocks.MockWitness
	ocv     *mocks.MockMultiSigOnChainVerifier
}

func getTestTopology(t *testing.T) *testTopology {
	ctrl := gomock.NewController(t)
	witness := mocks.NewMockWitness(ctrl)
	ocv := mocks.NewMockMultiSigOnChainVerifier(ctrl)
	broker := bmocks.NewMockBrokerI(ctrl)

	return &testTopology{
		Topology: erc20multisig.NewTopology(
			erc20multisig.NewDefaultConfig(),
			logging.NewTestLogger(),
			witness,
			ocv,
			broker,
		),
		ctrl:    ctrl,
		broker:  broker,
		witness: witness,
		ocv:     ocv,
	}
}

func TestERC20Topology(t *testing.T) {
	t.Run("valid threshold set event, threshold  updated", testValidThresholdEvent)
	t.Run("invalid threshold set event, threshold not updated", testInvalidThresholdEvent)
	t.Run("valid signer event, signers set updated", testValidSignerEvents)
	t.Run("invalid signer event, signers set not updated", testInvalidSignerEvents)
	t.Run("error on duplicate signer event", testErrorOnDuplicteSignerEvent)
	t.Run("error on duplicate threshold set event", testErrorOnDuplicteThesholdSetEvent)
}

func testValidThresholdEvent(t *testing.T) {
	top := getTestTopology(t)
	defer top.ctrl.Finish()

	top.OnTick(context.Background(), time.Unix(10, 0))

	// first assert we have no signers
	assert.Equal(t, top.GetThreshold(), uint32(0))

	event := types.SignerThresholdSetEvent{
		Threshold:   666,
		BlockNumber: 10,
		LogIndex:    11,
		TxHash:      "0xacbde",
		ID:          "someid",
		Nonce:       "123",
		BlockTime:   123456789,
	}

	var cb func(interface{}, bool)
	var res validators.Resource
	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		cb = f
		res = r
		return nil
	})

	assert.NoError(t, top.ProcessThresholdEvent(&event))

	// now we can call the callback
	cb(res, true)

	// now we can update the time
	top.broker.EXPECT().Send(gomock.Any()).Times(1)
	top.OnTick(context.Background(), time.Unix(11, 0))
	assert.Equal(t, top.GetThreshold(), uint32(666))

	// now update it again at a later time
	event2 := types.SignerThresholdSetEvent{
		Threshold:   900,
		BlockNumber: 11,
		LogIndex:    7,
		TxHash:      "0xedcba",
		ID:          "someid2",
		Nonce:       "321",
		BlockTime:   123456790,
	}

	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		cb = f
		res = r
		return nil
	})

	assert.NoError(t, top.ProcessThresholdEvent(&event2))

	// now we can call the callback
	cb(res, true)

	// now we can update the time
	top.broker.EXPECT().Send(gomock.Any()).Times(1)
	top.OnTick(context.Background(), time.Unix(11, 0))
	assert.Equal(t, top.GetThreshold(), uint32(900))

}

func testInvalidThresholdEvent(t *testing.T) {
	top := getTestTopology(t)
	defer top.ctrl.Finish()

	top.OnTick(context.Background(), time.Unix(10, 0))

	// first assert we have no signers
	assert.Equal(t, top.GetThreshold(), uint32(0))

	event := types.SignerThresholdSetEvent{
		Threshold:   666,
		BlockNumber: 10,
		LogIndex:    11,
		TxHash:      "0xacbde",
		ID:          "someid",
		Nonce:       "123",
		BlockTime:   123456789,
	}

	var cb func(interface{}, bool)
	var res validators.Resource
	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		cb = f
		res = r
		return nil
	})

	assert.NoError(t, top.ProcessThresholdEvent(&event))

	// now we can call the callback
	cb(res, false)

	// now we can update the time
	top.OnTick(context.Background(), time.Unix(11, 0))
	assert.Equal(t, top.GetThreshold(), uint32(0))
}

func testInvalidSignerEvents(t *testing.T) {
	top := getTestTopology(t)
	defer top.ctrl.Finish()

	top.OnTick(context.Background(), time.Unix(10, 0))

	// first assert we have no signers
	assert.Len(t, top.GetSigners(), 0)

	event := types.SignerEvent{
		BlockNumber: 10,
		LogIndex:    11,
		TxHash:      "0xacbde",
		ID:          "someid",
		Address:     "0x123456",
		Nonce:       "123",
		BlockTime:   123456789,
		Kind:        types.SignerEventKindAdded,
	}

	var cb func(interface{}, bool)
	var res validators.Resource
	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		cb = f
		res = r
		return nil
	})

	assert.NoError(t, top.ProcessSignerEvent(&event))

	// now we can call the callback
	cb(res, false)

	// now we can update the time
	top.OnTick(context.Background(), time.Unix(11, 0))
	assert.Len(t, top.GetSigners(), 0)
}

func testValidSignerEvents(t *testing.T) {
	top := getTestTopology(t)
	defer top.ctrl.Finish()

	top.OnTick(context.Background(), time.Unix(10, 0))

	// first assert we have no signers
	assert.Len(t, top.GetSigners(), 0)

	event := types.SignerEvent{
		BlockNumber: 10,
		LogIndex:    11,
		TxHash:      "0xacbde",
		ID:          "someid",
		Address:     "0x123456",
		Nonce:       "123",
		BlockTime:   123456789,
		Kind:        types.SignerEventKindAdded,
	}

	var cb func(interface{}, bool)
	var res validators.Resource
	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		cb = f
		res = r
		return nil
	})

	assert.NoError(t, top.ProcessSignerEvent(&event))

	// now we can call the callback
	cb(res, true)

	// now we can update the time
	top.broker.EXPECT().Send(gomock.Any()).Times(1)
	top.OnTick(context.Background(), time.Unix(11, 0))

	t.Run("ensure the signer list is updated", func(t *testing.T) {
		signers := top.GetSigners()
		assert.Len(t, signers, 1)
		assert.Equal(t, "0x123456", signers[0])
	})

	t.Run("check if our party IsSigner", func(t *testing.T) {
		assert.True(t, top.IsSigner("0x123456"))
	})

	t.Run("check excess signers", func(t *testing.T) {
		okAddresses := []string{"0x123456"}
		koAddresses := []string{}

		assert.True(t, top.ExcessSigners(koAddresses))
		assert.False(t, top.ExcessSigners(okAddresses))
	})

	// now we try to delete it yeay!
	event2 := types.SignerEvent{
		BlockNumber: 11,
		LogIndex:    4,
		TxHash:      "0xedcba",
		ID:          "someid2",
		Address:     "0x123456",
		Nonce:       "321",
		BlockTime:   123456790,
		Kind:        types.SignerEventKindRemoved,
	}

	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		cb = f
		res = r
		return nil
	})

	assert.NoError(t, top.ProcessSignerEvent(&event2))

	// now we can call the callback again!
	cb(res, true)

	// now we can update the time
	top.broker.EXPECT().Send(gomock.Any()).Times(1)
	top.OnTick(context.Background(), time.Unix(12, 0))

	t.Run("ensure all signers have been removed", func(t *testing.T) {
		signers := top.GetSigners()
		assert.Len(t, signers, 0)
	})

}

func testErrorOnDuplicteSignerEvent(t *testing.T) {
	top := getTestTopology(t)
	defer top.ctrl.Finish()

	event := types.SignerEvent{
		BlockNumber: 10,
		LogIndex:    11,
		TxHash:      "0xacbde",
		ID:          "someid",
		Address:     "0x123456",
		Nonce:       "123",
		BlockTime:   123456789,
		Kind:        types.SignerEventKindAdded,
	}

	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
	assert.NoError(t, top.ProcessSignerEvent(&event))

	assert.EqualError(t,
		top.ProcessSignerEvent(&event),
		erc20multisig.ErrDuplicatedSignerEvent.Error(),
	)
}

func testErrorOnDuplicteThesholdSetEvent(t *testing.T) {
	top := getTestTopology(t)
	defer top.ctrl.Finish()

	event := types.SignerThresholdSetEvent{
		BlockNumber: 10,
		LogIndex:    11,
		TxHash:      "0xacbde",
		ID:          "someid",
		Threshold:   666,
		Nonce:       "123",
		BlockTime:   123456789,
	}

	top.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil)
	assert.NoError(t, top.ProcessThresholdEvent(&event))

	assert.EqualError(t,
		top.ProcessThresholdEvent(&event),
		erc20multisig.ErrDuplicatedThresholdEvent.Error(),
	)
}
