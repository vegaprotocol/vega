// Copyright (c) 2023 Gobalsky Labs Limited
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

package ethverifier_test

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/ethverifier"
	"code.vegaprotocol.io/vega/core/datasource/external/ethverifier/mocks"
	omocks "code.vegaprotocol.io/vega/core/datasource/spec/mocks"
	"code.vegaprotocol.io/vega/core/events"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	mocks2 "code.vegaprotocol.io/vega/core/broker/mocks"

	"code.vegaprotocol.io/vega/core/client/eth"
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"

	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type verifierTest struct {
	*ethverifier.Verifier

	ctrl                     *gomock.Controller
	witness                  *mocks.MockWitness
	ts                       *omocks.MockTimeService
	oracleBroadcaster        *mocks.MockOracleDataBroadcaster
	ethCallEngine            *mocks.MockEthCallEngine
	ethConfirmations         *mocks.MockEthereumConfirmations
	broker                   *mocks2.MockBroker
	ethContractCallEventChan chan ethcall.ContractCallEvent

	onTick func(context.Context, time.Time)
}

func getTestEthereumOracleVerifier(ctx context.Context, t *testing.T) *verifierTest {
	t.Helper()
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	witness := mocks.NewMockWitness(ctrl)
	ts := omocks.NewMockTimeService(ctrl)
	broadcaster := mocks.NewMockOracleDataBroadcaster(ctrl)
	ethCallEngine := mocks.NewMockEthCallEngine(ctrl)
	ethConfirmations := mocks.NewMockEthereumConfirmations(ctrl)
	broker := mocks2.NewMockBroker(ctrl)
	ethContractCallEventChan := make(chan ethcall.ContractCallEvent)

	evt := &verifierTest{
		Verifier: ethverifier.New(ctx, log, witness, ts, broker, broadcaster, ethCallEngine, ethConfirmations,
			ethContractCallEventChan),
		ctrl:                     ctrl,
		witness:                  witness,
		ts:                       ts,
		oracleBroadcaster:        broadcaster,
		ethCallEngine:            ethCallEngine,
		ethConfirmations:         ethConfirmations,
		broker:                   broker,
		ethContractCallEventChan: ethContractCallEventChan,
	}
	evt.onTick = evt.Verifier.OnTick

	return evt
}

func TestVerifier(t *testing.T) {
	t.Run("testProcessEthereumOracleQueryOK", testProcessEthereumOracleQueryOK)
	t.Run("testProcessEthereumOracleQueryResultMismatch", testProcessEthereumOracleQueryResultMismatch)
	t.Run("testProcessEthereumOracleFilterMismatch", testProcessEthereumOracleFilterMismatch)
	t.Run("testProcessEthereumOracleInsufficientConfirmations", testProcessEthereumOracleInsufficientConfirmations)
	t.Run("testProcessEthereumOracleQueryDuplicateIgnored", testProcessEthereumOracleQueryDuplicateIgnored)
	t.Run("testProcessEthereumOracleChainEventWithGlobalError", testProcessEthereumOracleChainEventWithGlobalError)
	t.Run("testProcessEthereumOracleChainEventWithLocalError", testProcessEthereumOracleChainEventWithLocalError)
	t.Run("testProcessEthereumOracleChainEventWithMismatchedError", testProcessEthereumOracleChainEventWithMismatchedError)
	t.Run("testProcessEthereumOracleQueryResultWithoutCorrespondingLocalCallEventFails", testProcessEthereumOracleQueryResultWithoutCorrespondingLocalCallEventFails)
}

func testProcessEthereumOracleChainEventWithGlobalError(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	eov := getTestEthereumOracleVerifier(ctx, t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	testError := "test error"

	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(ethcall.Result{}, errors.New(testError))

	now := time.Now()
	eov.ts.EXPECT().GetTimeNow().Return(now).Times(1)

	var onQueryResultVerified func(interface{}, bool)
	var checkResult error
	var resourceToCheck interface{}
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			resourceToCheck = toCheck
			onQueryResultVerified = fn
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	errCallEvent := ethcall.ContractCallEvent{
		BlockHeight: 1,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      nil,
		Error:       &testError,
	}

	err := eov.ProcessEthereumContractCallResult(errCallEvent)
	assert.NoError(t, err)
	assert.NoError(t, checkResult)

	// result verified
	onQueryResultVerified(resourceToCheck, true)

	tickTime := time.Unix(10, 0)

	dataProto := vegapb.OracleData{
		ExternalData: &datapb.ExternalData{
			Data: &datapb.Data{
				MatchedSpecIds: []string{"testspec"},
				BroadcastAt:    tickTime.UnixNano(),
				Error:          &testError,
			},
		},
	}
	eov.broker.EXPECT().Send(events.NewOracleDataEvent(context.Background(), vegapb.OracleData{ExternalData: dataProto.ExternalData}))

	eov.onTick(context.Background(), tickTime)
}

func testProcessEthereumOracleChainEventWithLocalError(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	eov := getTestEthereumOracleVerifier(ctx, t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	testError := "test error"

	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(ethcall.Result{}, nil)

	now := time.Now()
	eov.ts.EXPECT().GetTimeNow().Return(now).Times(1)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	errCallEvent := ethcall.ContractCallEvent{
		BlockHeight: 1,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      nil,
		Error:       &testError,
	}

	err := eov.ProcessEthereumContractCallResult(errCallEvent)
	assert.NoError(t, err)
	assert.Error(t, checkResult)
}

func testProcessEthereumOracleChainEventWithMismatchedError(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	eov := getTestEthereumOracleVerifier(ctx, t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	testError := "test error"

	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(ethcall.Result{}, errors.New("another error"))

	now := time.Now()
	eov.ts.EXPECT().GetTimeNow().Return(now).Times(1)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	errCallEvent := ethcall.ContractCallEvent{
		BlockHeight: 1,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      nil,
		Error:       &testError,
	}

	err := eov.ProcessEthereumContractCallResult(errCallEvent)
	assert.NoError(t, err)
	assert.Error(t, checkResult)
}

func testProcessEthereumOracleQueryOK(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	eov := getTestEthereumOracleVerifier(ctx, t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ethCallEngine.EXPECT().MakeResult("testspec", []byte("testbytes")).Return(result, nil)

	eov.ethCallEngine.EXPECT().GetRequiredConfirmations("testspec").Return(uint64(5), nil)

	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethConfirmations.EXPECT().CheckRequiredConfirmations(uint64(1), uint64(5)).Return(nil)

	var onQueryResultVerified func(interface{}, bool)
	var checkResult error
	var resourceToCheck interface{}
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			resourceToCheck = toCheck
			onQueryResultVerified = fn
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	dummyEvent := generateDummyCallEvent()

	eov.ethContractCallEventChan <- dummyEvent
	err := eov.ProcessEthereumContractCallResult(dummyEvent)
	assert.NoError(t, err)
	assert.NoError(t, checkResult)

	// result verified
	onQueryResultVerified(resourceToCheck, true)

	oracleData := common.Data{
		Signers:  nil,
		Data:     okResult().Normalised,
		MetaData: map[string]string{"eth-block-height": "1", "eth-block-time": "100"},
	}

	eov.oracleBroadcaster.EXPECT().BroadcastData(gomock.Any(), oracleData)

	eov.onTick(context.Background(), time.Unix(10, 0))
}

func testProcessEthereumOracleQueryResultMismatch(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	eov := getTestEthereumOracleVerifier(ctx, t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()

	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ts.EXPECT().GetTimeNow().Times(1)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	dummyEvent := generateIncorrectDummyCallEvent()
	eov.ethContractCallEventChan <- dummyEvent
	err := eov.ProcessEthereumContractCallResult(dummyEvent)
	assert.NoError(t, err)
	assert.ErrorContains(t, checkResult, "mismatched")
}

func testProcessEthereumOracleFilterMismatch(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	eov := getTestEthereumOracleVerifier(ctx, t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := filterMismatchResult()
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations("testspec").Return(uint64(5), nil)

	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethConfirmations.EXPECT().CheckRequiredConfirmations(uint64(1), uint64(5)).Return(nil)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	dummyEvent := generateDummyCallEvent()
	eov.ethContractCallEventChan <- dummyEvent
	err := eov.ProcessEthereumContractCallResult(dummyEvent)
	assert.NoError(t, err)
	assert.ErrorContains(t, checkResult, "failed filter")
}

func testProcessEthereumOracleInsufficientConfirmations(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	eov := getTestEthereumOracleVerifier(ctx, t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations("testspec").Return(uint64(5), nil)

	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethConfirmations.EXPECT().CheckRequiredConfirmations(uint64(1), uint64(5)).Return(eth.ErrMissingConfirmations)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	dummyEvent := generateDummyCallEvent()
	eov.ethContractCallEventChan <- dummyEvent
	err := eov.ProcessEthereumContractCallResult(dummyEvent)

	assert.ErrorIs(t, checkResult, eth.ErrMissingConfirmations)
	assert.Nil(t, err)
}

func testProcessEthereumOracleQueryDuplicateIgnored(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	eov := getTestEthereumOracleVerifier(ctx, t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations("testspec").Return(uint64(5), nil)

	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethConfirmations.EXPECT().CheckRequiredConfirmations(uint64(1), uint64(5)).Return(nil)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	dummyEvent := generateDummyCallEvent()
	eov.ethContractCallEventChan <- dummyEvent
	err := eov.ProcessEthereumContractCallResult(dummyEvent)
	assert.NoError(t, checkResult)
	assert.NoError(t, err)

	err = eov.ProcessEthereumContractCallResult(generateDummyCallEvent())
	assert.ErrorContains(t, err, "duplicated")
}

func testProcessEthereumOracleQueryResultWithoutCorrespondingLocalCallEventFails(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	eov := getTestEthereumOracleVerifier(ctx, t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()

	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ts.EXPECT().GetTimeNow().Times(1)

	var checkResult error
	eov.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	dummyEvent := generateDummyCallEvent()
	err := eov.ProcessEthereumContractCallResult(dummyEvent)
	assert.NoError(t, err)
	assert.ErrorContains(t, checkResult, "event not found in local contract calls")
}

func generateDummyCallEvent() ethcall.ContractCallEvent {
	return ethcall.ContractCallEvent{
		BlockHeight: 1,
		BlockTime:   100,
		SpecId:      "testspec",
		Result:      []byte("testbytes"),
	}
}

func generateIncorrectDummyCallEvent() ethcall.ContractCallEvent {
	res := generateDummyCallEvent()
	res.Result = []byte("otherbytes")
	return res
}

func okResult() ethcall.Result {
	return ethcall.Result{
		Bytes:         []byte("testbytes"),
		Values:        []any{big.NewInt(42)},
		Normalised:    map[string]string{"price": fmt.Sprintf("%s", big.NewInt(42))},
		PassesFilters: true,
	}
}

func filterMismatchResult() ethcall.Result {
	r := okResult()
	r.PassesFilters = false
	return r
}
