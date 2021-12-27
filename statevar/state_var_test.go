package statevar_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/statevar"
	"code.vegaprotocol.io/vega/statevar/mocks"
	"code.vegaprotocol.io/vega/types/num"
	types "code.vegaprotocol.io/vega/types/statevar"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type testEngine struct {
	engine      *statevar.Engine
	topology    *mocks.MockTopology
	broker      *mocks.MockBroker
	commander   *mocks.MockCommander
	timeService *mocks.MockTimeService
}

// this is how state param bundles would be created:
// a native data structure
// and a converter to/from bundle type.
type sampleParams struct {
	param1 num.Decimal
	param2 []num.Decimal
}

type converter struct{}

var now = time.Date(2021, time.Month(2), 21, 1, 10, 30, 0, time.UTC)

func (converter) BundleToInterface(kvb *types.KeyValueBundle) types.StateVariableResult {
	return &sampleParams{
		param1: kvb.KVT[0].Val.(*types.DecimalScalar).Val,
		param2: kvb.KVT[1].Val.(*types.DecimalVector).Val,
	}
}

func (converter) InterfaceToBundle(res types.StateVariableResult) *types.KeyValueBundle {
	value := res.(*sampleParams)
	return &types.KeyValueBundle{
		KVT: []types.KeyValueTol{
			{Key: "param1", Val: &types.DecimalScalar{Val: value.param1}, Tolerance: num.DecimalFromFloat(1)},
			{Key: "param2", Val: &types.DecimalVector{Val: value.param2}, Tolerance: num.DecimalFromFloat(2)},
		},
	}
}

func getTestEngine(t *testing.T, startTime time.Time) *testEngine {
	t.Helper()
	conf := statevar.NewDefaultConfig()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	logger := logging.NewTestLogger()
	topology := mocks.NewMockTopology(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	commander := mocks.NewMockCommander(ctrl)

	ts.EXPECT().NotifyOnTick(gomock.Any()).Times(1)
	engine := statevar.New(logger, conf, broker, topology, commander, ts)
	engine.OnTimeTick(context.Background(), startTime)

	return &testEngine{
		engine:      engine,
		topology:    topology,
		broker:      broker,
		commander:   commander,
		timeService: ts,
	}
}

func getValidators(t *testing.T, now time.Time, numValidators int) []*testEngine {
	t.Helper()
	validators := make([]*testEngine, 0, numValidators)
	for i := 0; i < numValidators; i++ {
		validators = append(validators, getTestEngine(t, now))
		validators[i].engine.OnDefaultValidatorsVoteRequiredUpdate(context.Background(), num.DecimalFromFloat(0.67))
		validators[i].engine.OnFloatingPointUpdatesDurationUpdate(context.Background(), 10*time.Second)
		validators[i].engine.OnTimeTick(context.Background(), now)
	}
	return validators
}

func generateStateVariableForValidator(t *testing.T, testEngine *testEngine, startTime time.Time, startCalc func(string, types.FinaliseCalculation), resultCallback func(context.Context, types.StateVariableResult) error) error {
	t.Helper()
	kvb1 := &types.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, types.KeyValueTol{
		Key:       "scalar value",
		Val:       &types.DecimalScalar{Val: num.DecimalFromFloat(1.23456)},
		Tolerance: num.DecimalFromInt64(1),
	})

	return testEngine.engine.AddStateVariable("asset", "market", converter{}, startCalc, []types.StateVarEventType{types.StateVarEventTypeMarketEnacatment, types.StateVarEventTypeTimeTrigger}, resultCallback)
}

func defaultStartCalc() func(string, types.FinaliseCalculation) {
	return func(string, types.FinaliseCalculation) {}
}

func defaultResultBack() func(context.Context, types.StateVariableResult) error {
	return func(context.Context, types.StateVariableResult) error { return nil }
}

func setupValidators(t *testing.T, offset int, numValidators int, startCalc func(string, types.FinaliseCalculation), resultCallback func(context.Context, types.StateVariableResult) error) []*testEngine {
	t.Helper()
	validators := getValidators(t, now, numValidators)
	allNodeIds := []string{"0", "1", "2", "3", "4"}
	for _, v := range validators {
		err := generateStateVariableForValidator(t, v, now, startCalc, resultCallback)
		require.NoError(t, err)
		v.topology.EXPECT().IsValidator().Return(true).AnyTimes()
		v.topology.EXPECT().IsValidatorVegaPubKey(gomock.Any()).DoAndReturn(func(nodeID string) bool {
			ID, err := strconv.Atoi(nodeID)
			return err == nil && ID >= 0 && ID < len(allNodeIds)
		}).AnyTimes()
		v.topology.EXPECT().AllNodeIDs().Return(allNodeIds).AnyTimes()
		v.commander.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	}
	return validators
}

func TestStateVar(t *testing.T) {
	t.Run("test converters from/to native data type/key value bundle", testConverters)
	t.Run("new event comes in, no previous active event - triggers calculation", testEventTriggeredNoPreviousEvent)
	t.Run("new event comes in aborting an existing event", testEventTriggeredWithPreviousEvent)
	t.Run("new event comes in and triggers a calculation that resulst in an error", testEventTriggeredCalculationError)
	t.Run("perfect match through quorum", testBundleReceivedPerfectMatchOfQuorum)
	t.Run("reach consensus through random selection of one that is within reach of 2/3+1 of the others", testBundleReceivedReachingConsensusSuccessfuly)
	t.Run("no consensus can be reached", testBundleReceivedReachingConsensusNotSuccessful)
	t.Run("time based trigger", testTimeBasedEvent)
}

func testConverters(t *testing.T) {
	c := converter{}
	sampleP := &sampleParams{
		param1: num.DecimalFromFloat(1.23456),
		param2: []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)},
	}

	asBundle := c.InterfaceToBundle(sampleP)
	require.Equal(t, 2, len(asBundle.KVT))
	require.Equal(t, "param1", asBundle.KVT[0].Key)
	require.Equal(t, num.DecimalFromFloat(1), asBundle.KVT[0].Tolerance)
	require.Equal(t, "param2", asBundle.KVT[1].Key)
	require.Equal(t, num.DecimalFromFloat(2), asBundle.KVT[1].Tolerance)

	// check roundtrip - f^(f(a)) = a
	backToInterface := c.BundleToInterface(asBundle)
	require.Equal(t, sampleP, backToInterface)
	require.Equal(t, num.DecimalFromFloat(1.23456), backToInterface.(*sampleParams).param1)
	require.Equal(t, []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}, backToInterface.(*sampleParams).param2)

	// check double roundtrip - g^(g(b)) = b
	backAsBundle := c.InterfaceToBundle(backToInterface)
	require.Equal(t, asBundle, backAsBundle)
}

func testEventTriggeredNoPreviousEvent(t *testing.T) {
	validators := setupValidators(t, 0, 4, defaultStartCalc(), defaultResultBack())
	brokerEvents := make([]events.Event, 0, len(validators))
	for _, v := range validators {
		v.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(func(events []events.Event) {
			brokerEvents = append(brokerEvents, events...)
		})
	}

	for _, v := range validators {
		v.engine.NewEvent("asset", "market", types.StateVarEventTypeMarketEnacatment)
	}

	for _, v := range validators {
		v.engine.OnTimeTick(context.Background(), now.Add(1*time.Second))
	}

	require.Equal(t, len(validators), len(brokerEvents))
	for _, bes := range brokerEvents {
		evt := events.StateVarEventFromStream(context.Background(), bes.StreamMessage())
		require.Equal(t, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", evt.EventID)
		require.Equal(t, "consensus_calc_started", evt.State)
	}
}

func testEventTriggeredWithPreviousEvent(t *testing.T) {
	validators := setupValidators(t, 0, 4, defaultStartCalc(), defaultResultBack())

	brokerEvents := make([]events.Event, 0, len(validators))
	for _, v := range validators {
		v.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(func(events []events.Event) {
			brokerEvents = append(brokerEvents, events...)
		})
	}

	for _, v := range validators {
		v.engine.NewEvent("asset", "market", types.StateVarEventTypeMarketEnacatment)
	}

	for _, v := range validators {
		v.engine.OnTimeTick(context.Background(), now.Add(1*time.Second))
	}

	require.Equal(t, len(validators), len(brokerEvents))
	for _, bes := range brokerEvents {
		evt := events.StateVarEventFromStream(context.Background(), bes.StreamMessage())
		require.Equal(t, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", evt.EventID)
		require.Equal(t, "consensus_calc_started", evt.State)
	}

	for _, v := range validators {
		v.engine.NewEvent("asset", "market", types.StateVarEventTypeMarketEnacatment)
	}

	for _, v := range validators {
		v.engine.OnTimeTick(context.Background(), now.Add(2*time.Second))
	}

	require.Equal(t, 3*len(validators), len(brokerEvents))

	for i := 4; i < 3*len(validators); i += 2 {
		evt1 := events.StateVarEventFromStream(context.Background(), brokerEvents[i].StreamMessage())
		require.Equal(t, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", evt1.EventID)
		require.Equal(t, "consensus_calc_aborted", evt1.State)

		evt2 := events.StateVarEventFromStream(context.Background(), brokerEvents[i+1].StreamMessage())
		require.Equal(t, "asset_market_G8FFe2zipFM1jPoS3X31grPi7QrcJ1QF_2", evt2.EventID)
		require.Equal(t, "consensus_calc_started", evt2.State)
	}
}

func testEventTriggeredCalculationError(t *testing.T) {
	startCalc := func(eventID string, f types.FinaliseCalculation) {
		f.CalculationFinished(eventID, nil, errors.New("error"))
	}
	validators := setupValidators(t, 0, 4, startCalc, defaultResultBack())

	brokerEvents := make([]events.Event, 0, len(validators))
	for _, v := range validators {
		v.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(func(events []events.Event) {
			brokerEvents = append(brokerEvents, events...)
		})
	}

	for _, v := range validators {
		v.engine.NewEvent("asset", "market", types.StateVarEventTypeMarketEnacatment)
	}

	for _, v := range validators {
		v.engine.OnTimeTick(context.Background(), now.Add(1*time.Second))
	}

	require.Equal(t, 2*len(validators), len(brokerEvents))
	for i := 0; i < 2*len(validators); i += 2 {
		evt1 := events.StateVarEventFromStream(context.Background(), brokerEvents[i].StreamMessage())
		require.Equal(t, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", evt1.EventID)
		require.Equal(t, "consensus_calc_started", evt1.State)

		evt2 := events.StateVarEventFromStream(context.Background(), brokerEvents[i+1].StreamMessage())
		require.Equal(t, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", evt2.EventID)
		require.Equal(t, "error", evt2.State)
	}
}

func testBundleReceivedPerfectMatchOfQuorum(t *testing.T) {
	res := &sampleParams{
		param1: num.DecimalFromFloat(1.23456),
		param2: []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)},
	}
	startCalc := func(eventID string, f types.FinaliseCalculation) {
		f.CalculationFinished(eventID, res, nil)
	}

	counter := 0
	resultCallback := func(_ context.Context, r types.StateVariableResult) error {
		counter++
		require.Equal(t, res, r)
		return nil
	}

	// start sending publishing the results from each validators (they all would match so after 2/3+1 we should get the result back)
	validators := setupValidators(t, 0, 4, startCalc, resultCallback)
	brokerEvents := make([]events.Event, 0, len(validators))
	for _, v := range validators {
		v.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(func(events []events.Event) {
			brokerEvents = append(brokerEvents, events...)
		})
	}

	// event for the right asset/market
	for _, v := range validators {
		v.engine.NewEvent("asset", "market", types.StateVarEventTypeMarketEnacatment)
	}

	// send an unexpected results from all validators to all others, so that there would have been a quorum had it been the right event id
	c := converter{}
	bundle := c.InterfaceToBundle(res)
	for i := 0; i < len(validators); i++ {
		iAsString := strconv.Itoa(i)

		for j := 0; j < len(validators); j++ {
			validators[j].engine.ProposedValueReceived(context.Background(), "asset_market_8SQcDlWbkRMBvCoawjhbLStINMoO9wwo", iAsString, "eventID2", bundle)
		}
	}
	require.Equal(t, 0, counter)

	// send 3 results from non validator nodes, should be all ignored although it's for theh right event
	for i := 5; i < 10; i++ {
		iAsString := strconv.Itoa(i)

		for j := 0; j < len(validators); j++ {
			validators[j].engine.ProposedValueReceived(context.Background(), "asset_market_8SQcDlWbkRMBvCoawjhbLStINMoO9wwo", iAsString, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", bundle)
		}
	}
	require.Equal(t, 0, counter)

	// send bundles from 3/4 validators
	for i := 0; i < len(validators)-1; i++ {
		iAsString := strconv.Itoa(i)

		for j := 0; j < len(validators); j++ {
			validators[j].engine.ProposedValueReceived(context.Background(), "asset_market_8SQcDlWbkRMBvCoawjhbLStINMoO9wwo", iAsString, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", bundle)
		}
	}
	// this means that the result callback has been called with the same result for all of them
	require.Equal(t, 4, counter)

	for _, v := range validators {
		v.engine.OnTimeTick(context.Background(), now.Add(1*time.Second))
	}

	// we exepct there to be 8 events emitted, 4 for starting and 4 for perfect match
	require.Equal(t, 8, len(brokerEvents))
	for i := 0; i < len(validators); i++ {
		evt := events.StateVarEventFromStream(context.Background(), brokerEvents[2*i].StreamMessage())
		require.Equal(t, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", evt.EventID)
		require.Equal(t, "consensus_calc_started", evt.State)

		evt2 := events.StateVarEventFromStream(context.Background(), brokerEvents[2*i+1].StreamMessage())
		require.Equal(t, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", evt2.EventID)
		require.Equal(t, "perfect_match", evt2.State)
	}
}

func testBundleReceivedReachingConsensusSuccessfuly(t *testing.T) {
	// 3 of the results are within the acceptable tolerance, the other two are far off and are received first, so will require all 5 results to be received to reach consensus
	// therefore consensus is possible
	validatorResults := []*sampleParams{
		{param1: num.DecimalFromFloat(100.23456), param2: []num.Decimal{num.DecimalFromFloat(30), num.DecimalFromFloat(0.2), num.DecimalFromFloat(1.3), num.DecimalFromFloat(2.4)}},
		{param1: num.DecimalFromFloat(1.23456), param2: []num.Decimal{num.DecimalFromFloat(30), num.DecimalFromFloat(0.2), num.DecimalFromFloat(1.3), num.DecimalFromFloat(2.4)}},
		{param1: num.DecimalFromFloat(1.23456), param2: []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}},
		{param1: num.DecimalFromFloat(2.23456), param2: []num.Decimal{num.DecimalFromFloat(3), num.DecimalFromFloat(0.2), num.DecimalFromFloat(1.3), num.DecimalFromFloat(2.4)}},
		{param1: num.DecimalFromFloat(0.23456), param2: []num.Decimal{num.DecimalFromFloat(0.11), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}},
	}

	startCalcFuncs := make([]func(eventID string, f types.FinaliseCalculation), 0, len(validatorResults))
	for i := range validatorResults {
		startCalcFuncs = append(startCalcFuncs, func(eventID string, f types.FinaliseCalculation) {
			f.CalculationFinished(eventID, validatorResults[i], nil)
		})
	}

	counter := 0
	resultCallback := func(_ context.Context, r types.StateVariableResult) error {
		counter++
		require.Equal(t, validatorResults[2], r)
		return nil
	}

	validators := make([]*testEngine, 0, 5)
	for i := range startCalcFuncs {
		validators = append(validators, setupValidators(t, i, 1, startCalcFuncs[i], resultCallback)[0])
	}

	// start sending publishing the results from each validators (they all would match so after 2/3+1 we should get the result back)
	brokerEvents := make([]events.Event, 0, len(validators))
	for _, v := range validators {
		v.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(func(events []events.Event) {
			brokerEvents = append(brokerEvents, events...)
		})
	}

	for _, v := range validators {
		v.engine.NewEvent("asset", "market", types.StateVarEventTypeMarketEnacatment)
	}

	c := converter{}

	for i := 0; i < len(validators); i++ {
		iAsString := strconv.Itoa(i)

		for j := 0; j < len(validators); j++ {
			validators[j].engine.ProposedValueReceived(context.Background(), "asset_market_8SQcDlWbkRMBvCoawjhbLStINMoO9wwo", iAsString, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", c.InterfaceToBundle(validatorResults[i]))
		}
	}
	// this means that the result callback has been called with the same result for all of them
	require.Equal(t, 5, counter)

	for _, v := range validators {
		v.engine.OnTimeTick(context.Background(), now.Add(1*time.Second))
	}

	// we exepct there to be 10 events emitted, 5 for starting and 5 for consensus reached
	require.Equal(t, 10, len(brokerEvents))
	for i := 0; i < len(validators); i++ {
		evt := events.StateVarEventFromStream(context.Background(), brokerEvents[2*i].StreamMessage())
		require.Equal(t, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", evt.EventID)
		require.Equal(t, "consensus_calc_started", evt.State)

		evt2 := events.StateVarEventFromStream(context.Background(), brokerEvents[2*i+1].StreamMessage())
		require.Equal(t, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", evt2.EventID)
		require.Equal(t, "consensus_reached", evt2.State)
	}
}

func testBundleReceivedReachingConsensusNotSuccessful(t *testing.T) {
	// no 3 are within the required tolerance so consensus cannot be reached
	validatorResults := []*sampleParams{
		{param1: num.DecimalFromFloat(100.23456), param2: []num.Decimal{num.DecimalFromFloat(30), num.DecimalFromFloat(0.2), num.DecimalFromFloat(1.3), num.DecimalFromFloat(2.4)}},
		{param1: num.DecimalFromFloat(25.23456), param2: []num.Decimal{num.DecimalFromFloat(30), num.DecimalFromFloat(0.2), num.DecimalFromFloat(1.3), num.DecimalFromFloat(2.4)}},
		{param1: num.DecimalFromFloat(10.23456), param2: []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}},
		{param1: num.DecimalFromFloat(5.23456), param2: []num.Decimal{num.DecimalFromFloat(3), num.DecimalFromFloat(0.2), num.DecimalFromFloat(1.3), num.DecimalFromFloat(2.4)}},
		{param1: num.DecimalFromFloat(0.23456), param2: []num.Decimal{num.DecimalFromFloat(0.11), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}},
	}

	startCalcFuncs := make([]func(eventID string, f types.FinaliseCalculation), 0, len(validatorResults))
	for i := range validatorResults {
		startCalcFuncs = append(startCalcFuncs, func(eventID string, f types.FinaliseCalculation) {
			f.CalculationFinished(eventID, validatorResults[i], nil)
		})
	}

	resultCallback := func(_ context.Context, r types.StateVariableResult) error {
		require.Fail(t, "expecting no consensus")
		return nil
	}

	validators := make([]*testEngine, 0, 5)
	for i := range startCalcFuncs {
		validators = append(validators, setupValidators(t, i, 1, startCalcFuncs[i], resultCallback)[0])
	}

	// start sending publishing the results from each validators (they all would match so after 2/3+1 we should get the result back)
	brokerEvents := make([]events.Event, 0, len(validators))
	for _, v := range validators {
		v.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(func(events []events.Event) {
			brokerEvents = append(brokerEvents, events...)
		})
	}

	for _, v := range validators {
		v.engine.NewEvent("asset", "market", types.StateVarEventTypeMarketEnacatment)
	}

	// send an unexpected results from all validators to all others, so that there would have been a quorum had it been the right event id
	c := converter{}

	for i := 0; i < len(validators); i++ {
		iAsString := strconv.Itoa(i)

		for j := 0; j < len(validators); j++ {
			validators[j].engine.ProposedValueReceived(context.Background(), "asset_market_8SQcDlWbkRMBvCoawjhbLStINMoO9wwo", iAsString, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", c.InterfaceToBundle(validatorResults[i]))
		}
	}

	for _, v := range validators {
		v.engine.OnTimeTick(context.Background(), now.Add(1*time.Second))
	}

	// we exepct there to be 5 events emitted
	require.Equal(t, 5, len(brokerEvents))
	for i := 0; i < len(validators); i++ {
		evt := events.StateVarEventFromStream(context.Background(), brokerEvents[i].StreamMessage())
		require.Equal(t, "asset_market_OcskiC47WpCBO63KYKtLbEUctsTRRkwF_1", evt.EventID)
		require.Equal(t, "consensus_calc_started", evt.State)
	}
}

func testTimeBasedEvent(t *testing.T) {
	// 3 of the results are within the acceptable tolerance, the other two are far off and are received first, so will require all 5 results to be received to reach consensus
	// therefore consensus is possible
	validatorResults := []*sampleParams{
		{param1: num.DecimalFromFloat(100.23456), param2: []num.Decimal{num.DecimalFromFloat(30), num.DecimalFromFloat(0.2), num.DecimalFromFloat(1.3), num.DecimalFromFloat(2.4)}},
		{param1: num.DecimalFromFloat(1.23456), param2: []num.Decimal{num.DecimalFromFloat(30), num.DecimalFromFloat(0.2), num.DecimalFromFloat(1.3), num.DecimalFromFloat(2.4)}},
		{param1: num.DecimalFromFloat(1.23456), param2: []num.Decimal{num.DecimalFromFloat(1.1), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}},
		{param1: num.DecimalFromFloat(2.23456), param2: []num.Decimal{num.DecimalFromFloat(3), num.DecimalFromFloat(0.2), num.DecimalFromFloat(1.3), num.DecimalFromFloat(2.4)}},
		{param1: num.DecimalFromFloat(0.23456), param2: []num.Decimal{num.DecimalFromFloat(0.11), num.DecimalFromFloat(2.2), num.DecimalFromFloat(3.3), num.DecimalFromFloat(4.4)}},
	}

	startCalcFuncs := make([]func(eventID string, f types.FinaliseCalculation), 0, len(validatorResults))
	for i := range validatorResults {
		startCalcFuncs = append(startCalcFuncs, func(eventID string, f types.FinaliseCalculation) {
			f.CalculationFinished(eventID, validatorResults[i], nil)
		})
	}

	counter := 0
	resultCallback := func(_ context.Context, r types.StateVariableResult) error {
		counter++
		require.Equal(t, validatorResults[2], r)
		return nil
	}

	validators := make([]*testEngine, 0, 5)
	for i := range startCalcFuncs {
		validators = append(validators, setupValidators(t, i, 1, startCalcFuncs[i], resultCallback)[0])
	}

	for _, validator := range validators {
		validator.engine.ReadyForTimeTrigger("asset", "market")
	}

	// start sending publishing the results from each validators (they all would match so after 2/3+1 we should get the result back)
	brokerEvents := make([]events.Event, 0, len(validators))
	for _, v := range validators {
		v.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(func(events []events.Event) {
			brokerEvents = append(brokerEvents, events...)
		})
	}

	now = now.Add(time.Second * 10)
	for _, v := range validators {
		v.engine.OnTimeTick(context.Background(), now)
	}

	// send an unexpected results from all validators to all others, so that there would have been a quorum had it been the right event id
	c := converter{}

	for i := 0; i < len(validators); i++ {
		iAsString := strconv.Itoa(i)

		for j := 0; j < len(validators); j++ {
			validators[j].engine.ProposedValueReceived(context.Background(), "asset_market_8SQcDlWbkRMBvCoawjhbLStINMoO9wwo", iAsString, "20210221_011040", c.InterfaceToBundle(validatorResults[i]))
		}
	}

	now = now.Add(time.Second * 1)
	for _, v := range validators {
		v.engine.OnTimeTick(context.Background(), now)
	}

	// this means that the result callback has been called with the same result for all of them
	require.Equal(t, 5, counter)

	// we exepct there to be 10 events emitted, 5 for starting and 5 for consensus reached
	require.Equal(t, 10, len(brokerEvents))
	for i := 0; i < len(validators); i++ {
		evt := events.StateVarEventFromStream(context.Background(), brokerEvents[2*i].StreamMessage())
		require.Equal(t, "20210221_011040", evt.EventID)
		require.Equal(t, "consensus_calc_started", evt.State)

		evt2 := events.StateVarEventFromStream(context.Background(), brokerEvents[2*i+1].StreamMessage())
		require.Equal(t, "20210221_011040", evt2.EventID)
		require.Equal(t, "consensus_reached", evt2.State)
	}

	// advance 9 more seconds to get another time trigger
	now = now.Add(time.Second * 9)
	for _, v := range validators {
		v.engine.OnTimeTick(context.Background(), now)
	}
	brokerEvents = []events.Event{}

	// start another block for events to be emitted
	now = now.Add(time.Second * 1)
	for _, v := range validators {
		v.engine.OnTimeTick(context.Background(), now)
	}

	require.Equal(t, 5, len(brokerEvents))
	for i := 0; i < len(validators); i++ {
		evt := events.StateVarEventFromStream(context.Background(), brokerEvents[i].StreamMessage())
		require.Equal(t, "20210221_011050", evt.EventID)
		require.Equal(t, "consensus_calc_started", evt.State)
	}
}
