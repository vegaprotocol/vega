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

package processor_test

import (
	"context"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/processor"
	"code.vegaprotocol.io/vega/core/processor/mocks"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestBatchMarketInstructionsContinueFailsAllOrdersForMarketOnSwitchFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	exec := mocks.NewMockExecutionEngine(ctrl)
	proc := processor.NewBMIProcessor(logging.NewTestLogger(), exec, processor.Validate{})
	stats := stats.New(logging.NewTestLogger(), stats.NewDefaultConfig())

	batch := commandspb.BatchMarketInstructions{
		UpdateMarginMode: []*commandspb.UpdateMarginMode{
			{
				MarketId: "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				Mode:     commandspb.UpdateMarginMode_MODE_ISOLATED_MARGIN,
			},
		},
		Cancellations: []*commandspb.OrderCancellation{
			{
				OrderId:  "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
				MarketId: "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
			},
			{
				OrderId:  "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				MarketId: "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
			},
		},
		Amendments: []*commandspb.OrderAmendment{
			{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				OrderId:     "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				TimeInForce: vega.Order_TIME_IN_FORCE_GTC,
			},
			{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
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
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				Side:        vega.Side_SIDE_BUY,
				Size:        10,
				TimeInForce: vega.Order_TIME_IN_FORCE_FOK,
				Type:        vega.Order_TYPE_MARKET,
			},
			{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				Side:        vega.Side_SIDE_BUY,
				Size:        10,
				TimeInForce: vega.Order_TIME_IN_FORCE_FOK,
				Type:        vega.Order_TYPE_MARKET,
			},
		},
	}

	cancelCnt := 0
	exec.EXPECT().CancelOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, order *types.OrderCancellation, party string, idgen common.IDGenerator) ([]*types.OrderCancellationConfirmation, error) {
			cancelCnt++

			if order.OrderID == "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa" {
				return nil, errors.New("cannot cancel order")
			}
			return nil, nil
		},
	)
	amendCnt := 0
	exec.EXPECT().AmendOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).DoAndReturn(
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
	exec.EXPECT().SubmitOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).DoAndReturn(
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
	errors := err.(*processor.BMIError).Errors
	require.Equal(t, 7, len(errors))
	require.Equal(t, 1, len(errors["updateMarginMode"]))
	require.Equal(t, "update_margin_mode.margin_factor (margin factor must be defined when margin mode is isolated margin)", errors["updateMarginMode"][0].Error())

	require.Equal(t, 1, len(errors["0"])) // cancellation for market with failed update margin mode
	require.Equal(t, "Update margin mode transaction failed for market 926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23. Ignoring all transactions for the market", errors["0"][0].Error())

	require.Equal(t, 1, len(errors["1"])) // cancellation tx failed
	require.Equal(t, "cannot cancel order", errors["1"][0].Error())

	require.Equal(t, 1, len(errors["3"])) // amend tx failed
	require.Equal(t, "cannot amend order", errors["3"][0].Error())

	require.Equal(t, 1, len(errors["4"])) // amend for market with failed update margin mode
	require.Equal(t, "Update margin mode transaction failed for market 926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23. Ignoring all transactions for the market", errors["4"][0].Error())

	require.Equal(t, 1, len(errors["5"])) // submit for market with failed update margin mode
	require.Equal(t, "Update margin mode transaction failed for market 926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23. Ignoring all transactions for the market", errors["5"][0].Error())

	require.Equal(t, 1, len(errors["7"])) // submit tx failed
	require.Equal(t, "cannot submit order", errors["7"][0].Error())

	// one cancellation gets through, the other one is for 926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23
	// which had a failure in updating margin mode
	assert.Equal(t, uint64(1), stats.Blockchain.TotalCancelOrder())

	// 3 amends:
	// 1 is rejected for the market's switch margin mode failing
	assert.Equal(t, uint64(2), stats.Blockchain.TotalAmendOrder())

	// 3 submits:
	// 1 is rejected for the market's switch margin mode failing
	assert.Equal(t, uint64(2), stats.Blockchain.TotalCreateOrder())

	stats.Blockchain.NewBatch()

	assert.Equal(t, 2, amendCnt)
	assert.Equal(t, 1, cancelCnt)
	assert.Equal(t, 2, orderCnt)

	// // ensure the errors is reported as partial
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

func TestBatchMarketInstructionInvalidStopOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	exec := mocks.NewMockExecutionEngine(ctrl)
	proc := processor.NewBMIProcessor(logging.NewTestLogger(), exec, processor.Validate{})
	stats := stats.New(logging.NewTestLogger(), stats.NewDefaultConfig())

	batch := commandspb.BatchMarketInstructions{
		StopOrdersSubmission: []*commandspb.StopOrdersSubmission{
			{}, // this one is invalid
			{
				RisesAbove: &commandspb.StopOrderSetup{
					Trigger: &commandspb.StopOrderSetup_Price{Price: "1000"},
					OrderSubmission: &commandspb.OrderSubmission{
						MarketId:    "92eea9006eaa51154cb9110b9fe982f37d3bd50f62ee9d0a7709d9c74de329aa",
						Size:        1,
						Side:        vega.Side_SIDE_SELL,
						Type:        vega.Order_TYPE_MARKET,
						TimeInForce: types.OrderTimeInForceFOK,
						ReduceOnly:  true,
					},
				},
			},
		},
	}
	stopCnt := 0
	exec.EXPECT().SubmitStopOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context, stop *types.StopOrdersSubmission, party string, idgen common.IDGenerator, id1, id2 *string) ([]*types.OrderConfirmation, error) {
			stopCnt++
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

	assert.EqualError(t, err, "0 (* (must have at least one of rises above or falls below))")

	// ensure the errors is reported as partial
	perr, ok := err.(abci.MaybePartialError)
	assert.True(t, ok)
	assert.True(t, perr.IsPartial())
	assert.Equal(t, 1, stopCnt)
}

func TestConvertProto(t *testing.T) {
	t.Run("with success", func(t *testing.T) {
		txResult := events.NewTransactionResultEventSuccess(context.Background(), "0xDEADBEEF", "p1", &commandspb.BatchMarketInstructions{})
		assert.Nil(t, ptr.From(txResult.Proto()).GetFailure())
		assert.Equal(t, txResult.Proto().StatusDetail, eventspb.TransactionResult_STATUS_SUCCESS)
	})

	t.Run("with a normal error", func(t *testing.T) {
		err := errors.New("not a bmi error")

		txResult := events.NewTransactionResultEventFailure(context.Background(), "0xDEADBEEF", "p1", err, &commandspb.BatchMarketInstructions{})
		assert.Nil(t, ptr.From(txResult.Proto()).GetSuccess())
		assert.NotNil(t, ptr.From(txResult.Proto()).GetFailure())
		assert.Nil(t, ptr.From(txResult.Proto()).GetFailure().Errors)
		assert.NotNil(t, ptr.From(txResult.Proto()).GetFailure().Error)
		assert.False(t, txResult.Proto().Status)
		assert.Equal(t, txResult.Proto().StatusDetail, eventspb.TransactionResult_STATUS_FAILURE)
	})

	t.Run("with a partial BMI error", func(t *testing.T) {
		errs := &processor.BMIError{
			Errors: commands.NewErrors(),
		}

		errs.AddForProperty("1", errors.New("some error"))
		errs.AddForProperty("1", errors.New("some other error"))
		errs.AddForProperty("2", errors.New("another error again"))

		errs.Partial = true

		txResult := events.NewTransactionResultEventFailure(context.Background(), "0xDEADBEEF", "p1", errs, &commandspb.BatchMarketInstructions{})
		assert.Nil(t, ptr.From(txResult.Proto()).GetSuccess())
		assert.NotNil(t, ptr.From(txResult.Proto()).GetFailure())
		assert.NotNil(t, ptr.From(txResult.Proto()).GetFailure().Errors)
		assert.Equal(t, ptr.From(txResult.Proto()).GetFailure().Error, "")
		assert.False(t, txResult.Proto().Status)
		assert.Equal(t, txResult.Proto().StatusDetail, eventspb.TransactionResult_STATUS_PARTIAL_SUCCESS)
		assert.Equal(t, ptr.From(txResult.Proto()).GetFailure().Errors[0].Key, "1")
		assert.Equal(t, ptr.From(txResult.Proto()).GetFailure().Errors[0].Errors, []string{"some error", "some other error"})
		assert.Equal(t, ptr.From(txResult.Proto()).GetFailure().Errors[1].Key, "2")
		assert.Equal(t, ptr.From(txResult.Proto()).GetFailure().Errors[1].Errors, []string{"another error again"})
	})

	t.Run("with a full BMI error", func(t *testing.T) {
		errs := &processor.BMIError{
			Errors: commands.NewErrors(),
		}

		errs.AddForProperty("1", errors.New("some error"))
		errs.AddForProperty("1", errors.New("some other error"))
		errs.AddForProperty("2", errors.New("another error again"))

		errs.Partial = false

		txResult := events.NewTransactionResultEventFailure(context.Background(), "0xDEADBEEF", "p1", errs, &commandspb.BatchMarketInstructions{})
		assert.Nil(t, ptr.From(txResult.Proto()).GetSuccess())
		assert.NotNil(t, ptr.From(txResult.Proto()).GetFailure())
		assert.NotNil(t, ptr.From(txResult.Proto()).GetFailure().Errors)
		assert.Equal(t, ptr.From(txResult.Proto()).GetFailure().Error, "")
		assert.False(t, txResult.Proto().Status)
		assert.Equal(t, txResult.Proto().StatusDetail, eventspb.TransactionResult_STATUS_FAILURE)
		assert.Equal(t, ptr.From(txResult.Proto()).GetFailure().Errors[0].Key, "1")
		assert.Equal(t, ptr.From(txResult.Proto()).GetFailure().Errors[0].Errors, []string{"some error", "some other error"})
		assert.Equal(t, ptr.From(txResult.Proto()).GetFailure().Errors[1].Key, "2")
		assert.Equal(t, ptr.From(txResult.Proto()).GetFailure().Errors[1].Errors, []string{"another error again"})
	})
}
