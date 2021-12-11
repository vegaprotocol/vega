package statevar_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/statevar"
	"code.vegaprotocol.io/vega/statevar/mocks"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/types/num"
	types "code.vegaprotocol.io/vega/types/statevar"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

type testEngine struct {
	engine      *statevar.Engine
	topology    *mocks.MockTopology
	broker      *mocks.MockBroker
	commander   *mocks.MockCommander
	epoch       *mocks.MockEpochEngine
	timeService *mocks.MockTimeService
}

func TestAddStateVar(t *testing.T) {
	startTime := time.Now().Add(-24 * time.Hour)
	tEngine := getTestEngine(t, startTime)
	engine := tEngine.engine

	kvb1 := &types.KeyValueBundle{}
	kvb1.KVT = append(kvb1.KVT, types.KeyValueTol{
		Key:       "scalar value",
		Val:       &types.FloatValue{Val: 1.23456},
		Tolerance: num.DecimalFromInt64(1),
	})
	kvb1.KVT = append(kvb1.KVT, types.KeyValueTol{
		Key:       "vector value",
		Val:       &types.FloatVector{Val: []float64{1.1, 2.2, 3.3, 4.4}},
		Tolerance: num.DecimalFromInt64(2),
	})
	kvb1.KVT = append(kvb1.KVT, types.KeyValueTol{
		Key:       "matrix value",
		Val:       &types.FloatMatrix{Val: [][]float64{{1.1, 2.2, 3.3, 4.4}, {4.4, 3.3, 2.2, 1.1}}},
		Tolerance: num.DecimalFromInt64(3),
	})

	calcFunc := func() (*types.KeyValueBundle, error) {
		return kvb1, nil
	}

	countResults := 0
	var resultValue *types.KeyValueResult
	resultCallBack := func(r *types.KeyValueResult) error {
		resultValue = r
		countResults++
		return nil
	}
	defaultValue := &types.KeyValueBundle{}
	defaultValue.KVT = append(defaultValue.KVT, types.KeyValueTol{
		Key:       "scalar value",
		Val:       &types.FloatValue{Val: 2.2},
		Tolerance: num.DecimalFromInt64(1),
	})
	defaultValue.KVT = append(defaultValue.KVT, types.KeyValueTol{
		Key:       "vector value",
		Val:       &types.FloatVector{Val: []float64{3, 4, 1.3000000001, 4}},
		Tolerance: num.DecimalFromInt64(2),
	})
	defaultValue.KVT = append(defaultValue.KVT, types.KeyValueTol{
		Key:       "matrix value",
		Val:       &types.FloatMatrix{Val: [][]float64{{-1.1, 1.1, 0.31, 2}, {4.4, 3.3, 2.2, 1.1}}},
		Tolerance: num.DecimalFromInt64(3),
	})
	defaultResult := defaultValue.ToDecimal()
	defaultResult.Validity = types.StateValidityDefault

	engine.AddStateVariable("12345", calcFunc, []statevar.StateVarEventType{}, 10*time.Minute, resultCallBack, defaultResult)
	require.Equal(t, 1, countResults)
	require.Equal(t, defaultResult, resultValue)

	// advance time by 10 seconds so
	tt := startTime.Add(10 * time.Second)
	eventID := tt.Format("20060102_150405.999999999")
	f := func(ctx context.Context, cmd txn.Command, payload proto.Message, f func(error)) {
		engine.ProposedValueReceived(context.Background(), "12345", "n1", eventID, kvb1)
	}
	tEngine.topology.EXPECT().SelfNodeID().Return("n1").Times(2)
	tEngine.topology.EXPECT().IsValidator().Return(true)
	tEngine.topology.EXPECT().IsValidatorNodeID(gomock.Any()).Return(true)
	tEngine.topology.EXPECT().AllNodeIDs().Return([]string{"n1"})
	tEngine.commander.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Do(f)

	engine.OnTimeTick(context.Background(), tt)

	require.Equal(t, 2, countResults)
	res := kvb1.ToDecimal()
	res.Validity = types.StateValidityConsensus
	require.Equal(t, res, resultValue)
}

func getTestEngine(t *testing.T, startTime time.Time) *testEngine {
	t.Helper()
	conf := statevar.NewDefaultConfig()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	logger := logging.NewTestLogger()
	topology := mocks.NewMockTopology(ctrl)
	epoch := mocks.NewMockEpochEngine(ctrl)
	ts := mocks.NewMockTimeService(ctrl)
	commander := mocks.NewMockCommander(ctrl)

	ts.EXPECT().NotifyOnTick(gomock.Any()).Times(1)
	epoch.EXPECT().NotifyOnEpoch(gomock.Any()).Times(1)
	engine := statevar.New(logger, conf, broker, topology, commander, epoch, ts)
	engine.OnTimeTick(context.Background(), startTime)

	return &testEngine{
		engine:      engine,
		topology:    topology,
		broker:      broker,
		commander:   commander,
		epoch:       epoch,
		timeService: ts,
	}
}
