package erc20multisig_test

import (
	"testing"

	bmocks "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
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
	t.Run("error on duplicate signer event", testErrorOnDuplicteSignerEvent)
	t.Run("error on duplicate threshold set event", testErrorOnDuplicteThesholdSetEvent)
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
