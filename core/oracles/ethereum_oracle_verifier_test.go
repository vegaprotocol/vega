package oracles_test

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/oracles/mocks"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type etheriumOracleVerifierTest struct {
	*oracles.EthereumOracleVerifier

	ctrl              *gomock.Controller
	witness           *mocks.MockWitness
	ts                *mocks.MockTimeService
	oracleBroadcaster *mocks.MockOracleDataBroadcaster
	ethCallSpecSource *mocks.MockEthCallSpecSource
	ethContractCaller *mocks.MockContractCaller
	ethConfirmations  *mocks.MockEthereumConfirmations

	onTick func(context.Context, time.Time)
}

func getTestEthereumOracleVerifier(t *testing.T) *etheriumOracleVerifierTest {
	t.Helper()
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	witness := mocks.NewMockWitness(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	broadcaster := mocks.NewMockOracleDataBroadcaster(ctrl)

	ethCallSpecSource := mocks.NewMockEthCallSpecSource(ctrl)
	ethContractCaller := mocks.NewMockContractCaller(ctrl)

	confirmations := mocks.NewMockEthereumConfirmations(ctrl)

	evt := &etheriumOracleVerifierTest{
		EthereumOracleVerifier: oracles.NewEthereumOracleVerifier(log, witness, ts, broadcaster, ethCallSpecSource,
			ethContractCaller,
			confirmations),
		ctrl: ctrl,

		witness:           witness,
		ts:                ts,
		oracleBroadcaster: broadcaster,
		ethCallSpecSource: ethCallSpecSource,
		ethContractCaller: ethContractCaller,
		ethConfirmations:  confirmations,
	}
	evt.onTick = evt.EthereumOracleVerifier.OnTick

	return evt
}

func TestEthereumOracleVerifier(t *testing.T) {
	t.Run("testProcessEthereumOracleQueryOK", testProcessEthereumOracleQueryOK)
	t.Run("testProcessEthereumOracleQueryResultMismatch", testProcessEthereumOracleQueryResultMismatch)
	t.Run("testProcessEthereumOracleFilterMismatch", testProcessEthereumOracleFilterMismatch)
	t.Run("testProcessEthereumOracleInsufficientConfirmations", testProcessEthereumOracleInsufficientConfirmations)
	t.Run("testProcessEthereumOracleQueryDuplicateIgnored", testProcessEthereumOracleQueryDuplicateIgnored)
}

func testProcessEthereumOracleQueryOK(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	callspec := mocks.NewMockEthCallSpec(eov.ctrl)

	eov.ethCallSpecSource.EXPECT().GetCall("testspec").Times(2).Return(callspec, nil)

	callspec.EXPECT().Call(gomock.Any(), eov.ethContractCaller, big.NewInt(1)).Return([]byte("testbytes"), nil)
	callspec.EXPECT().PassesFilters([]byte("testbytes"), uint64(1), uint64(100)).Return(true)
	callspec.EXPECT().RequiredConfirmations().Return(uint64(5))

	eov.ethConfirmations.EXPECT().Check(uint64(5)).Return(nil)
	eov.ts.EXPECT().GetTimeNow().Times(1)

	var onQueryResultVerified func(interface{}, bool)
	var checkResult error
	var resourceToCheck interface{}
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			resourceToCheck = toCheck
			onQueryResultVerified = fn
			checkResult = toCheck.Check()
			return nil
		})

	callEvent := types.EthContractCallEvent{
		BlockHeight: 1,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}

	err := eov.ProcessEthereumContractCallResult(callEvent)
	assert.NoError(t, err)
	assert.NoError(t, checkResult)

	// result verified
	onQueryResultVerified(resourceToCheck, true)

	normalisedData := map[string]string{"price": "12"}
	oracleData := oracles.OracleData{
		Signers: nil,
		Data:    normalisedData,
	}

	callspec.EXPECT().Normalise([]byte("testbytes")).Return(normalisedData, nil)
	eov.oracleBroadcaster.EXPECT().BroadcastData(gomock.Any(), oracleData)

	eov.onTick(context.Background(), time.Unix(10, 0))
}

func testProcessEthereumOracleQueryResultMismatch(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	callspec := mocks.NewMockEthCallSpec(eov.ctrl)

	eov.ethCallSpecSource.EXPECT().GetCall("testspec").Times(1).Return(callspec, nil)

	callspec.EXPECT().Call(gomock.Any(), eov.ethContractCaller, big.NewInt(1)).Return([]byte("someothertestbytes"), nil)

	eov.ts.EXPECT().GetTimeNow().Times(1)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check()
			return nil
		})

	callEvent := types.EthContractCallEvent{
		BlockHeight: 1,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}

	err := eov.ProcessEthereumContractCallResult(callEvent)

	assert.NoError(t, err)
	assert.NotNil(t, checkResult)
}

func testProcessEthereumOracleFilterMismatch(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	callspec := mocks.NewMockEthCallSpec(eov.ctrl)

	eov.ethCallSpecSource.EXPECT().GetCall("testspec").Times(1).Return(callspec, nil)

	callspec.EXPECT().Call(gomock.Any(), eov.ethContractCaller, big.NewInt(1)).Return([]byte("testbytes"), nil)
	callspec.EXPECT().PassesFilters([]byte("testbytes"), uint64(1), uint64(100)).Return(false)

	eov.ts.EXPECT().GetTimeNow().Times(1)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check()
			return nil
		})

	callEvent := types.EthContractCallEvent{
		BlockHeight: 1,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}

	err := eov.ProcessEthereumContractCallResult(callEvent)

	assert.NoError(t, err)
	assert.NotNil(t, checkResult)
}

func testProcessEthereumOracleInsufficientConfirmations(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	callspec := mocks.NewMockEthCallSpec(eov.ctrl)

	eov.ethCallSpecSource.EXPECT().GetCall("testspec").Times(1).Return(callspec, nil)

	callspec.EXPECT().Call(gomock.Any(), eov.ethContractCaller, big.NewInt(1)).Return([]byte("testbytes"), nil)
	callspec.EXPECT().PassesFilters([]byte("testbytes"), uint64(1), uint64(100)).Return(true)

	callspec.EXPECT().RequiredConfirmations().Return(uint64(5))

	eov.ethConfirmations.EXPECT().Check(uint64(5)).Return(errors.New("insufficient confirmations"))

	eov.ts.EXPECT().GetTimeNow().Times(1)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check()
			return nil
		})

	callEvent := types.EthContractCallEvent{
		BlockHeight: 1,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}

	err := eov.ProcessEthereumContractCallResult(callEvent)

	assert.NoError(t, err)
	assert.NotNil(t, checkResult)
}

func testProcessEthereumOracleQueryDuplicateIgnored(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	callspec := mocks.NewMockEthCallSpec(eov.ctrl)

	eov.ethCallSpecSource.EXPECT().GetCall("testspec").Times(1).Return(callspec, nil)

	callspec.EXPECT().Call(gomock.Any(), eov.ethContractCaller, big.NewInt(1)).Return([]byte("testbytes"), nil)
	callspec.EXPECT().PassesFilters([]byte("testbytes"), uint64(1), uint64(100)).Return(true)
	callspec.EXPECT().RequiredConfirmations().Return(uint64(5))

	eov.ethConfirmations.EXPECT().Check(uint64(5)).Return(nil)
	eov.ts.EXPECT().GetTimeNow().Times(1)

	var checkResult error

	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check()
			return nil
		})

	callEvent := types.EthContractCallEvent{
		BlockHeight: 1,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}

	err := eov.ProcessEthereumContractCallResult(callEvent)
	assert.NoError(t, err)
	assert.NoError(t, checkResult)

	err = eov.ProcessEthereumContractCallResult(callEvent)
	assert.NotNil(t, err)
}
