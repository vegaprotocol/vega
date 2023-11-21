// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package ethverifier_test

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
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

	ctrl              *gomock.Controller
	witness           *mocks.MockWitness
	ts                *omocks.MockTimeService
	oracleBroadcaster *mocks.MockOracleDataBroadcaster
	ethCallEngine     *mocks.MockEthCallEngine
	ethConfirmations  *mocks.MockEthereumConfirmations
	broker            *mocks2.MockBroker

	onTick func(context.Context, time.Time)
}

func getTestEthereumOracleVerifier(t *testing.T) *verifierTest {
	t.Helper()
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	witness := mocks.NewMockWitness(ctrl)
	ts := omocks.NewMockTimeService(ctrl)
	broadcaster := mocks.NewMockOracleDataBroadcaster(ctrl)
	ethCallEngine := mocks.NewMockEthCallEngine(ctrl)
	ethConfirmations := mocks.NewMockEthereumConfirmations(ctrl)
	broker := mocks2.NewMockBroker(ctrl)

	evt := &verifierTest{
		Verifier:          ethverifier.New(log, witness, ts, broker, broadcaster, ethCallEngine, ethConfirmations),
		ctrl:              ctrl,
		witness:           witness,
		ts:                ts,
		oracleBroadcaster: broadcaster,
		ethCallEngine:     ethCallEngine,
		ethConfirmations:  ethConfirmations,
		broker:            broker,
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
	t.Run("testProcessEthereumOracleQueryWithBlockTimeBeforeInitialTime", testProcessEthereumOracleQueryWithBlockTimeBeforeInitialTime)
	t.Run("testSpoofedEthTimeFails", testSpoofedEthTimeFails)
}

func testSpoofedEthTimeFails(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	eov.ethCallEngine.EXPECT().GetEthTime(gomock.Any(), uint64(1)).Return(uint64(50), nil)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations(gomock.Any()).Return(uint64(0), nil)
	eov.ts.EXPECT().GetTimeNow().Times(1)

	var checkResult error
	eov.witness.EXPECT().StartCheckWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time, _ int64) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	err := eov.ProcessEthereumContractCallResult(generateDummyCallEvent())
	assert.NoError(t, err)
	assert.Error(t, checkResult)
}

func testProcessEthereumOracleChainEventWithGlobalError(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	testError := "test error"
	eov.ethCallEngine.EXPECT().GetEthTime(gomock.Any(), uint64(1)).Return(uint64(100), nil)
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(ethcall.Result{}, errors.New(testError))
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations(gomock.Any()).Return(uint64(0), nil)

	now := time.Now()
	eov.ts.EXPECT().GetTimeNow().Return(now).Times(1)

	var onQueryResultVerified func(interface{}, bool)
	var checkResult error
	var resourceToCheck interface{}
	eov.witness.EXPECT().StartCheckWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time, _ int64) error {
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
				MetaData: []*datapb.Property{
					{
						Name:  "vega-time",
						Value: strconv.FormatInt(tickTime.Unix(), 10),
					},
				},
			},
		},
	}
	eov.broker.EXPECT().Send(events.NewOracleDataEvent(context.Background(), vegapb.OracleData{ExternalData: dataProto.ExternalData}))

	eov.onTick(context.Background(), tickTime)
}

func testProcessEthereumOracleChainEventWithLocalError(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	testError := "test error"

	eov.ethCallEngine.EXPECT().GetEthTime(gomock.Any(), uint64(1)).Return(uint64(100), nil)
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(ethcall.Result{}, nil)

	now := time.Now()
	eov.ts.EXPECT().GetTimeNow().Return(now).Times(1)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations(gomock.Any()).Return(uint64(0), nil)

	var checkResult error
	eov.witness.EXPECT().StartCheckWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time, _ int64) error {
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
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	testError := "test error"

	eov.ethCallEngine.EXPECT().GetEthTime(gomock.Any(), uint64(1)).Return(uint64(100), nil)
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(ethcall.Result{}, errors.New("another error"))

	now := time.Now()
	eov.ts.EXPECT().GetTimeNow().Return(now).Times(1)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations(gomock.Any()).Return(uint64(0), nil)

	var checkResult error
	eov.witness.EXPECT().StartCheckWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time, _ int64) error {
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
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()
	eov.ethCallEngine.EXPECT().GetEthTime(gomock.Any(), uint64(1)).Return(uint64(100), nil)
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ethCallEngine.EXPECT().MakeResult("testspec", []byte("testbytes")).Return(result, nil)

	eov.ethCallEngine.EXPECT().GetRequiredConfirmations("testspec").Return(uint64(5), nil).Times(2)

	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethCallEngine.EXPECT().GetInitialTriggerTime("testspec").Return(uint64(90), nil)
	eov.ethConfirmations.EXPECT().CheckRequiredConfirmations(uint64(1), uint64(5)).Return(nil)

	var onQueryResultVerified func(interface{}, bool)
	var checkResult error
	var resourceToCheck interface{}
	eov.witness.EXPECT().StartCheckWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time, _ int64) error {
			resourceToCheck = toCheck
			onQueryResultVerified = fn
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	err := eov.ProcessEthereumContractCallResult(generateDummyCallEvent())
	assert.NoError(t, err)
	assert.NoError(t, checkResult)

	// result verified
	onQueryResultVerified(resourceToCheck, true)

	tick := time.Unix(10, 0)
	oracleData := common.Data{
		EthKey:  "testspec",
		Signers: nil,
		Data:    okResult().Normalised,
		MetaData: map[string]string{
			"eth-block-height": "1",
			"eth-block-time":   "100",
			"vega-time":        strconv.FormatInt(tick.Unix(), 10),
		},
	}

	eov.oracleBroadcaster.EXPECT().BroadcastData(gomock.Any(), oracleData)

	eov.onTick(context.Background(), tick)
}

func testProcessEthereumOracleQueryWithBlockTimeBeforeInitialTime(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	eov.ethCallEngine.EXPECT().GetEthTime(gomock.Any(), uint64(1)).Return(uint64(100), nil)
	result := okResult()
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)

	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethCallEngine.EXPECT().GetInitialTriggerTime("testspec").Return(uint64(110), nil)

	eov.ethCallEngine.EXPECT().GetRequiredConfirmations(gomock.Any()).Return(uint64(0), nil)

	var checkResult error
	eov.witness.EXPECT().StartCheckWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time, _ int64) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	err := eov.ProcessEthereumContractCallResult(generateDummyCallEvent())
	assert.NoError(t, err)
	assert.ErrorContains(t, checkResult, "is before the specification's initial time")
}

func testProcessEthereumOracleQueryResultMismatch(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()

	eov.ethCallEngine.EXPECT().GetEthTime(gomock.Any(), uint64(1)).Return(uint64(100), nil)
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations(gomock.Any()).Return(uint64(0), nil)

	var checkResult error
	eov.witness.EXPECT().StartCheckWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time, _ int64) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	err := eov.ProcessEthereumContractCallResult(generateIncorrectDummyCallEvent())
	assert.NoError(t, err)
	assert.ErrorContains(t, checkResult, "mismatched")
}

func testProcessEthereumOracleFilterMismatch(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := filterMismatchResult()
	eov.ethCallEngine.EXPECT().GetEthTime(gomock.Any(), uint64(1)).Return(uint64(100), nil)
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations("testspec").Return(uint64(5), nil).Times(2)

	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethCallEngine.EXPECT().GetInitialTriggerTime("testspec").Return(uint64(90), nil)
	eov.ethConfirmations.EXPECT().CheckRequiredConfirmations(uint64(1), uint64(5)).Return(nil)

	var checkResult error
	eov.witness.EXPECT().StartCheckWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time, _ int64) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	err := eov.ProcessEthereumContractCallResult(generateDummyCallEvent())
	assert.NoError(t, err)
	assert.ErrorContains(t, checkResult, "failed filter")
}

func testProcessEthereumOracleInsufficientConfirmations(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()
	eov.ethCallEngine.EXPECT().GetEthTime(gomock.Any(), uint64(1)).Return(uint64(100), nil)
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations("testspec").Return(uint64(5), nil).Times(2)

	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethCallEngine.EXPECT().GetInitialTriggerTime("testspec").Return(uint64(90), nil)
	eov.ethConfirmations.EXPECT().CheckRequiredConfirmations(uint64(1), uint64(5)).Return(eth.ErrMissingConfirmations)

	var checkResult error
	eov.witness.EXPECT().StartCheckWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time, _ int64) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	err := eov.ProcessEthereumContractCallResult(generateDummyCallEvent())

	assert.ErrorIs(t, checkResult, eth.ErrMissingConfirmations)
	assert.Nil(t, err)
}

func testProcessEthereumOracleQueryDuplicateIgnored(t *testing.T) {
	eov := getTestEthereumOracleVerifier(t)
	defer eov.ctrl.Finish()
	assert.NotNil(t, eov)

	result := okResult()
	eov.ethCallEngine.EXPECT().GetEthTime(gomock.Any(), uint64(1)).Return(uint64(100), nil)
	eov.ethCallEngine.EXPECT().CallSpec(gomock.Any(), "testspec", uint64(1)).Return(result, nil)
	eov.ethCallEngine.EXPECT().GetRequiredConfirmations("testspec").Return(uint64(5), nil).Times(2)

	eov.ts.EXPECT().GetTimeNow().Times(1)
	eov.ethCallEngine.EXPECT().GetInitialTriggerTime("testspec").Return(uint64(90), nil)
	eov.ethConfirmations.EXPECT().CheckRequiredConfirmations(uint64(1), uint64(5)).Return(nil)

	var checkResult error
	eov.witness.EXPECT().StartCheckWithDelay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(func(toCheck validators.Resource, fn func(interface{}, bool), _ time.Time, _ int64) error {
			checkResult = toCheck.Check(context.Background())
			return nil
		})

	err := eov.ProcessEthereumContractCallResult(generateDummyCallEvent())
	assert.NoError(t, checkResult)
	assert.NoError(t, err)

	err = eov.ProcessEthereumContractCallResult(generateDummyCallEvent())
	assert.ErrorContains(t, err, "duplicated")
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
