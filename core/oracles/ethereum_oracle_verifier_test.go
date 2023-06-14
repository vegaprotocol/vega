package oracles_test

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/client/eth"
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
	ethCallEngine     *mocks.MockEthCallEngine
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
	ethCallEngine := mocks.NewMockEthCallEngine(ctrl)
	ethConfirmations := mocks.NewMockEthereumConfirmations(ctrl)

	evt := &etheriumOracleVerifierTest{
		EthereumOracleVerifier: oracles.NewEthereumOracleVerifier(log, witness, ts, broadcaster, ethCallEngine, ethConfirmations),
		ctrl:                   ctrl,
		witness:                witness,
		ts:                     ts,
		oracleBroadcaster:      broadcaster,
		ethCallEngine:          ethCallEngine,
		ethConfirmations:       ethConfirmations,
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

	result := okResult()
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ethCallEngine.EXPECT().MakeResult("testspec", []byte("testbytes")).Return(result, nil)

	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethConfirmations.EXPECT().Check(uint64(1)).Return(nil)

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

	err := eov.ProcessEthereumContractCallResult(result.generateDummyCallEvent())
	assert.NoError(t, err)
	assert.NoError(t, checkResult)

	// result verified
	onQueryResultVerified(resourceToCheck, true)

	oracleData := oracles.OracleData{
		Signers: nil,
		Data:    okResult().normalised,
	}

	eov.oracleBroadcaster.EXPECT().BroadcastData(gomock.Any(), oracleData)

	eov.onTick(context.Background(), time.Unix(10, 0))
}

func testProcessEthereumOracleQueryResultMismatch(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()

	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ts.EXPECT().GetTimeNow().Times(1)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check()
			return nil
		})

	err := eov.ProcessEthereumContractCallResult(result.generateIncorrectDummyCallEvent())
	assert.NoError(t, err)
	assert.ErrorContains(t, checkResult, "mismatched")
}

func testProcessEthereumOracleFilterMismatch(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := filterMismatchResult()
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethConfirmations.EXPECT().Check(uint64(1)).Return(nil)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check()
			return nil
		})

	err := eov.ProcessEthereumContractCallResult(result.generateDummyCallEvent())
	assert.NoError(t, err)
	assert.ErrorContains(t, checkResult, "failed filter")
}

func testProcessEthereumOracleInsufficientConfirmations(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := insufficentConfirmationsResult()
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethConfirmations.EXPECT().Check(uint64(1)).Return(eth.ErrMissingConfirmations)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check()
			return nil
		})

	err := eov.ProcessEthereumContractCallResult(result.generateDummyCallEvent())

	assert.ErrorIs(t, checkResult, eth.ErrMissingConfirmations)
	assert.Nil(t, err)
}

func testProcessEthereumOracleQueryDuplicateIgnored(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethConfirmations.EXPECT().Check(uint64(1)).Return(nil)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check()
			return nil
		})

	err := eov.ProcessEthereumContractCallResult(result.generateDummyCallEvent())
	assert.NoError(t, checkResult)
	assert.NoError(t, err)

	err = eov.ProcessEthereumContractCallResult(result.generateDummyCallEvent())
	assert.ErrorContains(t, err, "duplicated")
}

type fakeResult struct {
	bytes                    []byte
	values                   []any
	normalised               map[string]string
	passesFilters            bool
	hasRequiredConfirmations bool
}

func (r fakeResult) Bytes() []byte                          { return r.bytes }
func (r fakeResult) Values() ([]any, error)                 { return r.values, nil }
func (r fakeResult) Normalised() (map[string]string, error) { return r.normalised, nil }
func (r fakeResult) PassesFilters() (bool, error)           { return r.passesFilters, nil }

func (r fakeResult) generateDummyCallEvent() types.EthContractCallEvent {
	return types.EthContractCallEvent{
		BlockHeight: 1,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}
}

func (r fakeResult) generateIncorrectDummyCallEvent() types.EthContractCallEvent {
	res := r.generateDummyCallEvent()
	res.Result = []byte("otherbytes")
	return res
}

func okResult() fakeResult {
	return fakeResult{
		bytes:         []byte("testbytes"),
		values:        []any{big.NewInt(42)},
		normalised:    map[string]string{"price": fmt.Sprintf("%s", big.NewInt(42))},
		passesFilters: true,
	}
}

func filterMismatchResult() fakeResult {
	r := okResult()
	r.passesFilters = false
	return r
}

func insufficentConfirmationsResult() fakeResult {
	r := okResult()
	r.hasRequiredConfirmations = false
	return r
}
