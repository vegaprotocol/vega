package processor_test

import (
	"context"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/processor"
	"code.vegaprotocol.io/vega/core/processor/mocks"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestBatchMarketInstructionsErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	exec := mocks.NewMockExecutionEngine(ctrl)
	proc := processor.NewBMIProcessor(logging.NewTestLogger(), exec, processor.Validate{})

	batch := commandspb.BatchMarketInstructions{
		Amendments:  []*commandspb.OrderAmendment{{}},
		Submissions: []*commandspb.OrderSubmission{{}},
	}

	stats := stats.New(logging.NewTestLogger(), stats.NewDefaultConfig())

	err := proc.ProcessBatch(
		context.Background(),
		&batch,
		"43f86066fe13743448442022c099c48abbd7e9c5eac1c2558fdac1fbf549e867",
		"62017b6ae543d2e699f41d37598b22dab025c57ed98ef3c237bb91b948c5f8fc",
		stats.Blockchain,
	)

	assert.EqualError(t, err, "0 (* (order_amendment does not amend anything), order_amendment.market_id (is required), order_amendment.order_id (is required)), 1 (order_submission.market_id (is required), order_submission.side (is required), order_submission.size (must be positive), order_submission.time_in_force (is required), order_submission.type (is required))")
}

func TestBatchMarketInstructionsCannotSubmitMultipleAmendForSameID(t *testing.T) {
	ctrl := gomock.NewController(t)
	exec := mocks.NewMockExecutionEngine(ctrl)
	proc := processor.NewBMIProcessor(logging.NewTestLogger(), exec, processor.Validate{})
	stats := stats.New(logging.NewTestLogger(), stats.NewDefaultConfig())

	batch := commandspb.BatchMarketInstructions{
		Amendments: []*commandspb.OrderAmendment{
			{
				MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				OrderId:     "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				TimeInForce: vega.Order_TIME_IN_FORCE_GTC,
			},
			{
				MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				OrderId:     "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				TimeInForce: vega.Order_TIME_IN_FORCE_GTC,
			},
			{
				MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				OrderId:     "87d4717b42796bda59870f53d6bcb1f57acd53e4236a941077aae8a860fd1bad",
				TimeInForce: vega.Order_TIME_IN_FORCE_GTC,
			},
		},
	}

	amendCnt := 0
	exec.EXPECT().AmendOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).DoAndReturn(
		func(ctx context.Context, order *types.OrderAmendment, party string, idgen common.IDGenerator) ([]*types.OrderConfirmation, error) {
			amendCnt++
			return nil, nil
		},
	)
	err := proc.ProcessBatch(
		context.Background(),
		&batch,
		"43f86066fe13743448442022c099c48abbd7e9c5eac1c2558fdac1fbf549e867",
		"62017b6ae543d2e699f41d37598b22dab025c57ed98ef3c237bb91b948c5f8fc",
		stats.Blockchain,
	)

	assert.Equal(t, 2, amendCnt)
	assert.EqualError(t, err, "1 (order already amended in batch)")
}

func TestBatchMarketInstructionsContinueProcessingOnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	exec := mocks.NewMockExecutionEngine(ctrl)
	proc := processor.NewBMIProcessor(logging.NewTestLogger(), exec, processor.Validate{})
	stats := stats.New(logging.NewTestLogger(), stats.NewDefaultConfig())

	batch := commandspb.BatchMarketInstructions{
		Cancellations: []*commandspb.OrderCancellation{
			{
				OrderId:  "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				MarketId: "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
			},
			{
				OrderId:  "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				MarketId: "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
			},
			{
				OrderId:  "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				MarketId: "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
			},
		},
		Amendments: []*commandspb.OrderAmendment{
			{
				MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				OrderId:     "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				TimeInForce: vega.Order_TIME_IN_FORCE_GTC,
			},
			{
				MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				OrderId:     "f31f922db56ee0ffee7695e358c5f6c253857b8e0656ddead6dc40474502bc22",
				TimeInForce: vega.Order_TIME_IN_FORCE_GTC,
			},
			{
				MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				OrderId:     "87d4717b42796bda59870f53d6bcb1f57acd53e4236a941077aae8a860fd1bad",
				TimeInForce: vega.Order_TIME_IN_FORCE_GTC,
			},
		},
		Submissions: []*commandspb.OrderSubmission{
			{
				MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				Side:        vega.Side_SIDE_BUY,
				Size:        10,
				TimeInForce: vega.Order_TIME_IN_FORCE_FOK,
				Type:        vega.Order_TYPE_MARKET,
			},
			{
				MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				Side:        vega.Side_SIDE_BUY,
				Size:        10,
				TimeInForce: vega.Order_TIME_IN_FORCE_FOK,
				Type:        vega.Order_TYPE_MARKET,
			},
			{
				MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				Side:        vega.Side_SIDE_BUY,
				Size:        10,
				TimeInForce: vega.Order_TIME_IN_FORCE_FOK,
				Type:        vega.Order_TYPE_MARKET,
			},
		},
	}

	cancelCnt := 0
	exec.EXPECT().CancelOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(3).DoAndReturn(
		func(ctx context.Context, order *types.OrderCancellation, party string, idgen common.IDGenerator) ([]*types.OrderCancellationConfirmation, error) {
			cancelCnt++

			// if the order is order 2 we return an error
			if order.OrderID == "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa" {
				return nil, errors.New("cannot cancel order")
			}
			return nil, nil
		},
	)
	amendCnt := 0
	exec.EXPECT().AmendOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(3).DoAndReturn(
		func(ctx context.Context, order *types.OrderAmendment, party string, idgen common.IDGenerator) ([]*types.OrderConfirmation, error) {
			amendCnt++

			// if the order is order 2 we return an error
			if order.OrderID == "f31f922db56ee0ffee7695e358c5f6c253857b8e0656ddead6dc40474502bc22" {
				return nil, errors.New("cannot amend order")
			}
			return nil, nil
		},
	)

	orderCnt := 0
	exec.EXPECT().SubmitOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(3).DoAndReturn(
		func(ctx context.Context, order *types.OrderSubmission, party string, idgen common.IDGenerator, orderID string) (*types.OrderConfirmation, error) {
			orderCnt++

			// if the order is order 2 we return an error
			if orderCnt == 2 {
				return nil, errors.New("cannot submit order")
			}
			return &types.OrderConfirmation{Order: nil, Trades: []*types.Trade{{ID: "1"}, {ID: "2"}}, PassiveOrdersAffected: []*types.Order{}}, nil
		},
	)

	err := proc.ProcessBatch(
		context.Background(),
		&batch,
		"43f86066fe13743448442022c099c48abbd7e9c5eac1c2558fdac1fbf549e867",
		"62017b6ae543d2e699f41d37598b22dab025c57ed98ef3c237bb91b948c5f8fc",
		stats.Blockchain,
	)

	assert.Equal(t, uint64(3), stats.Blockchain.TotalCancelOrder())
	assert.Equal(t, uint64(3), stats.Blockchain.TotalAmendOrder())
	stats.Blockchain.NewBatch()

	assert.Equal(t, 3, amendCnt)
	assert.Equal(t, 3, cancelCnt)
	assert.Equal(t, 3, orderCnt)
	assert.Equal(t, uint64(3), stats.Blockchain.TotalOrders())
	assert.Equal(t, uint64(2), stats.Blockchain.TotalOrdersLastBatch())
	assert.Equal(t, uint64(4), stats.Blockchain.TotalTradesLastBatch())

	assert.EqualError(t, err, "1 (cannot cancel order), 4 (cannot amend order), 7 (cannot submit order)")

	// ensure the errors is reported as partial
	perr, ok := err.(abci.MaybePartialError)
	assert.True(t, ok)
	assert.True(t, perr.IsPartial())
}

func TestBatchMarketInstructionsEnsureAllErrorReturnNonPartialError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	exec := mocks.NewMockExecutionEngine(ctrl)
	proc := processor.NewBMIProcessor(logging.NewTestLogger(), exec, processor.Validate{})
	stats := stats.New(logging.NewTestLogger(), stats.NewDefaultConfig())

	batch := commandspb.BatchMarketInstructions{
		Cancellations: []*commandspb.OrderCancellation{
			{
				OrderId:  "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				MarketId: "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
			},
		},
		Amendments: []*commandspb.OrderAmendment{
			{
				MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				OrderId:     "f31f922db56ee0ffee7695e358c5f6c253857b8e0656ddead6dc40474502bc22",
				TimeInForce: vega.Order_TIME_IN_FORCE_GTC,
			},
		},
		Submissions: []*commandspb.OrderSubmission{},
	}

	cancelCnt := 0
	exec.EXPECT().CancelOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, order *types.OrderCancellation, party string, idgen common.IDGenerator) ([]*types.OrderCancellationConfirmation, error) {
			cancelCnt++

			// if the order is order 2 we return an error
			if order.OrderID == "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa" {
				return nil, errors.New("cannot cancel order")
			}
			return nil, nil
		},
	)
	amendCnt := 0
	exec.EXPECT().AmendOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, order *types.OrderAmendment, party string, idgen common.IDGenerator) ([]*types.OrderConfirmation, error) {
			amendCnt++

			// if the order is order 2 we return an error
			if order.OrderID == "f31f922db56ee0ffee7695e358c5f6c253857b8e0656ddead6dc40474502bc22" {
				return nil, errors.New("cannot amend order")
			}
			return nil, nil
		},
	)

	err := proc.ProcessBatch(
		context.Background(),
		&batch,
		"43f86066fe13743448442022c099c48abbd7e9c5eac1c2558fdac1fbf549e867",
		"62017b6ae543d2e699f41d37598b22dab025c57ed98ef3c237bb91b948c5f8fc",
		stats.Blockchain,
	)

	assert.EqualError(t, err, "0 (cannot cancel order), 1 (cannot amend order)")

	// ensure the errors is reported as partial
	perr, ok := err.(abci.MaybePartialError)
	assert.True(t, ok)
	assert.False(t, perr.IsPartial())
}
