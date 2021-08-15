package staking_test

import (
	"testing"

	bmocks "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/staking/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type stakeVerifierTest struct {
	*staking.StakeVerifier

	ctrl    *gomock.Controller
	broker  *bmocks.MockBroker
	tt      *mocks.MockTimeTicker
	witness *mocks.MockWitness
	ocv     *mocks.MockEthOnChainVerifier
}

func getStakeVerifierTest(t *testing.T) *stakeVerifierTest {
	ctrl := gomock.NewController(t)
	broker := bmocks.NewMockBroker(ctrl)
	tt := mocks.NewMockTimeTicker(ctrl)
	witness := mocks.NewMockWitness(ctrl)
	ocv := mocks.NewMockEthOnChainVerifier(ctrl)

	tt.EXPECT().NotifyOnTick(gomock.Any()).AnyTimes()

	stakeV := staking.NewStakeVerifier(
		logging.NewTestLogger(),
		staking.NewDefaultConfig(),
		staking.NewAccounting(
			logging.NewTestLogger(),
			staking.NewDefaultConfig(),
			broker,
		),
		tt,
		witness,
		broker,
		ocv,
	)
	return &stakeVerifierTest{
		StakeVerifier: stakeV,
		ctrl:          ctrl,
		broker:        broker,
		tt:            tt,
		witness:       witness,
		ocv:           ocv,
	}
}

func TestStakeVerifier(t *testing.T) {
	t.Run("can process stake event deposited", testProcessStakeEventDeposited)
}

func testProcessStakeEventDeposited(t *testing.T) {
	stakeV := getStakeVerifierTest(t)
	assert.NotNil(t, stakeV)
}
