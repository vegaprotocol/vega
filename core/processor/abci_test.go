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
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/genesis"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/processor"
	"code.vegaprotocol.io/vega/core/processor/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	proto "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	tmtypes "github.com/cometbft/cometbft/abci/types"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gproto "google.golang.org/protobuf/proto"
)

func TestListSnapshots(t *testing.T) {
	ctx, cfunc := context.WithCancel(context.Background())
	app := getTestApp(t, cfunc, stopDummy, true, true)
	defer app.ctrl.Finish()

	app.snap.EXPECT().ListLatestSnapshots().Times(1).Return([]*tmtypes.Snapshot{
		{
			Height:   123,
			Format:   1,
			Chunks:   3,
			Hash:     []byte("0xDEADBEEF"),
			Metadata: []byte("test"),
		},
	}, nil)
	resp, err := app.ListSnapshots(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, 1, len(resp.GetSnapshots()))
}

func TestAppInfo(t *testing.T) {
	ctx, cfunc := context.WithCancel(context.Background())
	app := getTestApp(t, cfunc, stopDummy, true, true)
	defer app.ctrl.Finish()
	// first, the broker streaming stuff
	app.broker.EXPECT().SetStreaming(false).Times(1).Return(true)
	app.broker.EXPECT().SetStreaming(true).Times(1).Return(true)
	// snapshot engine
	app.snap.EXPECT().HasSnapshots().Times(1).Return(true, nil)
	// hash, height, chainID := app.snapshotEngine.Info()
	app.snap.EXPECT().Info().Times(1).Return([]byte("43f86066fe13743448442022c099c48abbd7e9c5eac1c2558fdac1fbf549e867"), int64(123), fmt.Sprintf("%d", app.pChainID))
	info, err := app.Info(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, info)
}

func getTransaction(t *testing.T, inputData *commandspb.InputData) *commandspb.Transaction {
	t.Helper()
	rawInputData, err := gproto.Marshal(inputData)
	if err != nil {
		t.Fatal(err)
	}
	return &commandspb.Transaction{
		InputData: rawInputData,
		Signature: &commandspb.Signature{
			Algo:    "vega/ed25519",
			Value:   "876e46defc40030391b5feb2c9bb0b6b68b2d95a6b5fd17a730a46ea73f3b1808420c8c609be6f1c6156e472ecbcd09202f750da000dee41429947a4b7eca00b",
			Version: 1,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
		},
		Version: 2,
	}
}

// A batch transaction including only cancellations and/or post-only limit orders is executed at
// the top of the block alongside standalone post-only limit orders and cancellations (0093-TRTO-001).
func TestBatchOnlyCancelsAndPostOnly(t *testing.T) {
	_, cfunc := context.WithCancel(context.Background())
	app := getTestApp(t, cfunc, stopDummy, false, false)
	defer app.ctrl.Finish()

	// setup some order as the first tx
	tx1InputData := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderSubmission{
			OrderSubmission: &commandspb.OrderSubmission{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				Side:        proto.Side_SIDE_BUY,
				Size:        1,
				TimeInForce: proto.Order_TIME_IN_FORCE_FOK,
				Type:        proto.Order_TYPE_LIMIT,
				Price:       "123",
			},
		},
	}
	tx1 := getTransaction(t, tx1InputData)
	marshalledTx1, err := gproto.Marshal(tx1)
	require.NoError(t, err)

	// setup a batch transaction with cancellation and post only
	tx2InputData := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_BatchMarketInstructions{
			BatchMarketInstructions: &commandspb.BatchMarketInstructions{
				Cancellations: []*commandspb.OrderCancellation{
					{
						OrderId:  "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
						MarketId: "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
					},
				},
				Submissions: []*commandspb.OrderSubmission{
					{
						MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
						TimeInForce: proto.Order_TIME_IN_FORCE_GTC,
						PostOnly:    true,
					},
				},
			},
		},
	}
	tx2 := getTransaction(t, tx2InputData)
	marshalledTx2, err := gproto.Marshal(tx2)
	require.NoError(t, err)

	rawTxs := [][]byte{marshalledTx1, marshalledTx2}
	txs := []abci.Tx{}
	for _, tx := range rawTxs {
		decodedTx, err := app.codec.Decode(tx, "1")
		require.NoError(t, err)
		txs = append(txs, decodedTx)
	}

	app.txCache.EXPECT().GetRawTxs(gomock.Any()).Return(nil).Times(1)
	app.txCache.EXPECT().IsDelayRequired(gomock.Any()).Return(true).AnyTimes()
	app.txCache.EXPECT().NewDelayedTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("123")).Times(1)
	app.limits.EXPECT().CanTrade().Return(true).AnyTimes()
	blockTxs := app.Abci().OnPrepareProposal(100, txs, rawTxs)
	require.Equal(t, 2, len(blockTxs))
	require.Equal(t, rawTxs[1], blockTxs[0])
	require.Equal(t, []byte("123"), blockTxs[1])
}

// A batch transaction including either a non-post-only order or an amendment is delayed by one block
// and then executed after the expedited transactions in that later block (0093-TRTO-002).
func TestBatchDelayed(t *testing.T) {
	_, cfunc := context.WithCancel(context.Background())
	app := getTestApp(t, cfunc, stopDummy, false, false)
	defer app.ctrl.Finish()

	// setup some order as the first tx that doesn't get delayed
	tx1InputData := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderSubmission{
			OrderSubmission: &commandspb.OrderSubmission{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				Side:        proto.Side_SIDE_BUY,
				Size:        1,
				TimeInForce: proto.Order_TIME_IN_FORCE_GTC,
				Type:        proto.Order_TYPE_LIMIT,
				Price:       "123",
				PostOnly:    true,
			},
		},
	}
	tx1 := getTransaction(t, tx1InputData)
	marshalledTx1, err := gproto.Marshal(tx1)
	require.NoError(t, err)

	// setup a cancellation
	tx2InputData := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderCancellation{
			OrderCancellation: &commandspb.OrderCancellation{
				MarketId: "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
			},
		},
	}
	tx2 := getTransaction(t, tx2InputData)
	marshalledTx2, err := gproto.Marshal(tx2)
	require.NoError(t, err)

	// now get a batch transaction with submission such that will get it delayed by 1 block
	// setup a batch transaction with cancellation and post only
	tx3InputData := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_BatchMarketInstructions{
			BatchMarketInstructions: &commandspb.BatchMarketInstructions{
				Submissions: []*commandspb.OrderSubmission{
					{
						MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
						Side:        proto.Side_SIDE_BUY,
						Size:        1,
						TimeInForce: proto.Order_TIME_IN_FORCE_FOK,
						Type:        proto.Order_TYPE_LIMIT,
						Price:       "123",
					},
					{
						MarketId:    "926df3b689a5440fe21cad7069ebcedc46f75b2b23ce11002a1ee2254e339f23",
						Side:        proto.Side_SIDE_BUY,
						Size:        2,
						TimeInForce: proto.Order_TIME_IN_FORCE_FOK,
						Type:        proto.Order_TYPE_LIMIT,
						Price:       "234",
					},
				},
			},
		},
	}
	tx3 := getTransaction(t, tx3InputData)
	marshalledTx3, err := gproto.Marshal(tx3)
	require.NoError(t, err)

	rawTxs := [][]byte{marshalledTx1, marshalledTx2, marshalledTx3}
	txs := []abci.Tx{}
	for _, tx := range rawTxs {
		decodedTx, err := app.codec.Decode(tx, "1")
		require.NoError(t, err)
		txs = append(txs, decodedTx)
	}

	app.txCache.EXPECT().GetRawTxs(gomock.Any()).Return(nil).Times(1)
	app.txCache.EXPECT().IsDelayRequired(gomock.Any()).Return(true).AnyTimes()
	app.txCache.EXPECT().NewDelayedTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("123")).Times(1)
	app.limits.EXPECT().CanTrade().Return(true).AnyTimes()
	blockTxs := app.Abci().OnPrepareProposal(100, txs, rawTxs)
	// the first two transactions and the delayed wrapped transaction
	require.Equal(t, 3, len(blockTxs))
	require.Equal(t, rawTxs[1], blockTxs[0])
	require.Equal(t, rawTxs[0], blockTxs[1])
	require.Equal(t, []byte("123"), blockTxs[2])

	// setup a cancellation
	tx4InputData := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderCancellation{
			OrderCancellation: &commandspb.OrderCancellation{
				MarketId: "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
			},
		},
	}
	tx4 := getTransaction(t, tx4InputData)
	marshalledTx4, err := gproto.Marshal(tx4)
	require.NoError(t, err)

	rawTxs = [][]byte{marshalledTx4}
	txs = []abci.Tx{}
	for _, tx := range rawTxs {
		decodedTx, err := app.codec.Decode(tx, "1")
		require.NoError(t, err)
		txs = append(txs, decodedTx)
	}
	// now lets go to the next block and have the postponed transactions executed after the
	app.txCache.EXPECT().GetRawTxs(gomock.Any()).Return([][]byte{marshalledTx3}).Times(1)
	blockTxs = app.Abci().OnPrepareProposal(101, txs, rawTxs)
	require.Equal(t, 2, len(blockTxs))
	require.Equal(t, marshalledTx4, blockTxs[0])
	// the delayed transaction gets in execution order after the expedited
	require.Equal(t, marshalledTx3, blockTxs[1])
}

// Cancellation transactions always occur before:
// Market orders (0093-TRTO-003)
// Non post-only limit orders (0093-TRTO-004)
// Order Amends (0093-TRTO-005)
// post-only limit orders (0093-TRTO-013).
func TestCancelledOrdersGoFirst(t *testing.T) {
	_, cfunc := context.WithCancel(context.Background())
	app := getTestApp(t, cfunc, stopDummy, false, false)
	defer app.ctrl.Finish()

	// cancel 1
	// setup a cancellation
	cancel1 := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderCancellation{
			OrderCancellation: &commandspb.OrderCancellation{
				MarketId: "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
			},
		},
	}
	cancel1Tx := getTransaction(t, cancel1)
	marshalledCancel1Tx, err := gproto.Marshal(cancel1Tx)
	require.NoError(t, err)

	// cancel 2
	cancel2 := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderCancellation{
			OrderCancellation: &commandspb.OrderCancellation{
				MarketId: "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
			},
		},
	}
	cancel2Tx := getTransaction(t, cancel2)
	marshalledCancel2Tx, err := gproto.Marshal(cancel2Tx)
	require.NoError(t, err)

	// now lets set up one order of each of the desired types
	marketOrder := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderSubmission{
			OrderSubmission: &commandspb.OrderSubmission{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				Side:        proto.Side_SIDE_BUY,
				Size:        1,
				TimeInForce: proto.Order_TIME_IN_FORCE_FOK,
				Type:        proto.Order_TYPE_MARKET,
			},
		},
	}
	marketOrderTx := getTransaction(t, marketOrder)
	marshalledMarketOrderTx, err := gproto.Marshal(marketOrderTx)
	require.NoError(t, err)

	limitOrder := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderSubmission{
			OrderSubmission: &commandspb.OrderSubmission{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				Side:        proto.Side_SIDE_BUY,
				Size:        1,
				TimeInForce: proto.Order_TIME_IN_FORCE_GTC,
				Type:        proto.Order_TYPE_LIMIT,
				Price:       "123",
			},
		},
	}
	limitOrderTx := getTransaction(t, limitOrder)
	marshalledLimitOrderTx, err := gproto.Marshal(limitOrderTx)
	require.NoError(t, err)

	amend := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderAmendment{
			OrderAmendment: &commandspb.OrderAmendment{
				MarketId:  "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				OrderId:   "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				SizeDelta: 5,
			},
		},
	}
	amendTx := getTransaction(t, amend)
	marshalledAmendTx, err := gproto.Marshal(amendTx)
	require.NoError(t, err)

	postOnly := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderSubmission{
			OrderSubmission: &commandspb.OrderSubmission{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				Side:        proto.Side_SIDE_BUY,
				Size:        1,
				TimeInForce: proto.Order_TIME_IN_FORCE_GTC,
				Type:        proto.Order_TYPE_LIMIT,
				Price:       "123",
				PostOnly:    true,
			},
		},
	}
	postOnlyTx := getTransaction(t, postOnly)
	marshalledPostOnlyTx, err := gproto.Marshal(postOnlyTx)
	require.NoError(t, err)

	rawTxs := [][]byte{marshalledCancel1Tx, marshalledCancel2Tx, marshalledAmendTx, marshalledMarketOrderTx, marshalledLimitOrderTx, marshalledPostOnlyTx}
	txs := []abci.Tx{}
	for _, tx := range rawTxs {
		decodedTx, err := app.codec.Decode(tx, "1")
		require.NoError(t, err)
		txs = append(txs, decodedTx)
	}

	app.txCache.EXPECT().GetRawTxs(gomock.Any()).Return(nil).Times(1)
	app.txCache.EXPECT().IsDelayRequired(gomock.Any()).Return(true).AnyTimes()
	app.txCache.EXPECT().NewDelayedTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("123")).Times(1)
	app.limits.EXPECT().CanTrade().Return(true).AnyTimes()
	blockTxs := app.Abci().OnPrepareProposal(100, txs, rawTxs)
	// cancel 1, then cancel 2, then post only, then wrapped delayed
	require.Equal(t, 4, len(blockTxs))
	require.Equal(t, marshalledCancel1Tx, blockTxs[0])
	require.Equal(t, marshalledCancel2Tx, blockTxs[1])
	require.Equal(t, marshalledPostOnlyTx, blockTxs[2])
	require.Equal(t, []byte("123"), blockTxs[3])

	// now, in the following block we expect the delayed transactions to be executed, still after expedited transactions
	// cancel 3
	cancel3 := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderCancellation{
			OrderCancellation: &commandspb.OrderCancellation{
				MarketId: "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
			},
		},
	}
	cancel3Tx := getTransaction(t, cancel3)
	marshalledCancel3Tx, err := gproto.Marshal(cancel3Tx)
	require.NoError(t, err)

	rawTxs = [][]byte{marshalledCancel3Tx}
	txs = []abci.Tx{}
	for _, tx := range rawTxs {
		decodedTx, err := app.codec.Decode(tx, "1")
		require.NoError(t, err)
		txs = append(txs, decodedTx)
	}
	// now lets go to the next block and have the postponed transactions executed after the
	app.txCache.EXPECT().GetRawTxs(gomock.Any()).Return([][]byte{marshalledAmendTx, marshalledMarketOrderTx, marshalledLimitOrderTx}).Times(1)
	blockTxs = app.Abci().OnPrepareProposal(101, txs, rawTxs)
	require.Equal(t, 4, len(blockTxs))
	require.Equal(t, marshalledCancel3Tx, blockTxs[0])
	require.Equal(t, marshalledAmendTx, blockTxs[1])
	require.Equal(t, marshalledMarketOrderTx, blockTxs[2])
	require.Equal(t, marshalledLimitOrderTx, blockTxs[3])
}

// Post-only transactions always occur before:
// Market orders (0093-TRTO-006)
// Non post-only limit orders (0093-TRTO-007)
// Order Amends (0093-TRTO-008).
// Potentially aggressive orders take effect on the market exactly one block after they are included
// in a block (i.e for an order which is included in block N it hits the market in block N+1). This is true for:
// Market orders (0093-TRTO-009)
// Non post-only limit orders (0093-TRTO-010)
// Order Amends (0093-TRTO-011).
func TestPostOnlyGoBeforeAggressive(t *testing.T) {
	_, cfunc := context.WithCancel(context.Background())
	app := getTestApp(t, cfunc, stopDummy, false, false)
	defer app.ctrl.Finish()

	postOnly1 := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderSubmission{
			OrderSubmission: &commandspb.OrderSubmission{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				Side:        proto.Side_SIDE_BUY,
				Size:        1,
				TimeInForce: proto.Order_TIME_IN_FORCE_GTC,
				Type:        proto.Order_TYPE_LIMIT,
				Price:       "123",
				PostOnly:    true,
			},
		},
	}
	postOnly1Tx := getTransaction(t, postOnly1)
	marshalledPostOnly1Tx, err := gproto.Marshal(postOnly1Tx)
	require.NoError(t, err)

	postOnly2 := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderSubmission{
			OrderSubmission: &commandspb.OrderSubmission{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				Side:        proto.Side_SIDE_BUY,
				Size:        1,
				TimeInForce: proto.Order_TIME_IN_FORCE_GTC,
				Type:        proto.Order_TYPE_LIMIT,
				Price:       "123",
				PostOnly:    true,
			},
		},
	}
	postOnly2Tx := getTransaction(t, postOnly2)
	marshalledPostOnly2Tx, err := gproto.Marshal(postOnly2Tx)
	require.NoError(t, err)

	// cancel 1
	cancel1 := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderCancellation{
			OrderCancellation: &commandspb.OrderCancellation{
				MarketId: "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
			},
		},
	}
	cancel1Tx := getTransaction(t, cancel1)
	marshalledCancel1Tx, err := gproto.Marshal(cancel1Tx)
	require.NoError(t, err)

	// now lets set up one order of each of the desired types
	marketOrder := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderSubmission{
			OrderSubmission: &commandspb.OrderSubmission{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				Side:        proto.Side_SIDE_BUY,
				Size:        1,
				TimeInForce: proto.Order_TIME_IN_FORCE_FOK,
				Type:        proto.Order_TYPE_MARKET,
			},
		},
	}
	marketOrderTx := getTransaction(t, marketOrder)
	marshalledMarketOrderTx, err := gproto.Marshal(marketOrderTx)
	require.NoError(t, err)

	limitOrder := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderSubmission{
			OrderSubmission: &commandspb.OrderSubmission{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				Side:        proto.Side_SIDE_BUY,
				Size:        1,
				TimeInForce: proto.Order_TIME_IN_FORCE_GTC,
				Type:        proto.Order_TYPE_LIMIT,
				Price:       "123",
			},
		},
	}
	limitOrderTx := getTransaction(t, limitOrder)
	marshalledLimitOrderTx, err := gproto.Marshal(limitOrderTx)
	require.NoError(t, err)

	amend := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderAmendment{
			OrderAmendment: &commandspb.OrderAmendment{
				MarketId:  "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				OrderId:   "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				SizeDelta: 5,
			},
		},
	}
	amendTx := getTransaction(t, amend)
	marshalledAmendTx, err := gproto.Marshal(amendTx)
	require.NoError(t, err)

	rawTxs := [][]byte{marshalledAmendTx, marshalledMarketOrderTx, marshalledLimitOrderTx, marshalledPostOnly1Tx, marshalledCancel1Tx, marshalledPostOnly2Tx}
	txs := []abci.Tx{}
	for _, tx := range rawTxs {
		decodedTx, err := app.codec.Decode(tx, "1")
		require.NoError(t, err)
		txs = append(txs, decodedTx)
	}

	app.txCache.EXPECT().GetRawTxs(gomock.Any()).Return(nil).Times(1)
	app.txCache.EXPECT().IsDelayRequired(gomock.Any()).Return(true).AnyTimes()
	app.txCache.EXPECT().NewDelayedTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("123")).Times(1)
	app.limits.EXPECT().CanTrade().Return(true).AnyTimes()
	blockTxs := app.Abci().OnPrepareProposal(100, txs, rawTxs)
	// cancel, then post only 1, then post only 2, then wrapped delayed
	require.Equal(t, 4, len(blockTxs))
	require.Equal(t, marshalledCancel1Tx, blockTxs[0])
	require.Equal(t, marshalledPostOnly1Tx, blockTxs[1])
	require.Equal(t, marshalledPostOnly2Tx, blockTxs[2])
	require.Equal(t, []byte("123"), blockTxs[3])

	// now, in the following block we expect the delayed transactions to be executed, still after expedited transactions
	// cancel 3
	cancel3 := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderCancellation{
			OrderCancellation: &commandspb.OrderCancellation{
				MarketId: "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
			},
		},
	}
	cancel3Tx := getTransaction(t, cancel3)
	marshalledCancel3Tx, err := gproto.Marshal(cancel3Tx)
	require.NoError(t, err)

	postOnly3 := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderSubmission{
			OrderSubmission: &commandspb.OrderSubmission{
				MarketId:    "47076f002ddd9bfeb7f4679fc75b4686f64446d5a5afcb84584e7c7166d13efa",
				Side:        proto.Side_SIDE_BUY,
				Size:        1,
				TimeInForce: proto.Order_TIME_IN_FORCE_GTC,
				Type:        proto.Order_TYPE_LIMIT,
				Price:       "123",
				PostOnly:    true,
			},
		},
	}
	postOnly3Tx := getTransaction(t, postOnly3)
	marshalledPostOnly3Tx, err := gproto.Marshal(postOnly3Tx)
	require.NoError(t, err)

	amend2 := &commandspb.InputData{
		Nonce:       123456789,
		BlockHeight: 1789,
		Command: &commandspb.InputData_OrderAmendment{
			OrderAmendment: &commandspb.OrderAmendment{
				MarketId:  "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				OrderId:   "b5fd9d3c4ad553cb3196303b6e6df7f484cf7f5331a572a45031239fd71ad8a0",
				SizeDelta: 5,
			},
		},
	}
	amend2Tx := getTransaction(t, amend2)
	marshalledAmend2Tx, err := gproto.Marshal(amend2Tx)
	require.NoError(t, err)

	rawTxs = [][]byte{marshalledPostOnly3Tx, marshalledAmend2Tx, marshalledCancel3Tx}
	txs = []abci.Tx{}
	for _, tx := range rawTxs {
		decodedTx, err := app.codec.Decode(tx, "1")
		require.NoError(t, err)
		txs = append(txs, decodedTx)
	}
	// now lets go to the next block and have the postponed transactions executed after the new cancellation and post only from this block
	// plus throw in another amend this block that gets delayed to the next block
	app.txCache.EXPECT().GetRawTxs(gomock.Any()).Return([][]byte{marshalledAmendTx, marshalledMarketOrderTx, marshalledLimitOrderTx}).Times(1)
	app.txCache.EXPECT().NewDelayedTransaction(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("456")).Times(1)
	blockTxs = app.Abci().OnPrepareProposal(101, txs, rawTxs)
	require.Equal(t, 6, len(blockTxs))
	require.Equal(t, marshalledCancel3Tx, blockTxs[0])
	require.Equal(t, marshalledPostOnly3Tx, blockTxs[1])
	require.Equal(t, marshalledAmendTx, blockTxs[2])
	require.Equal(t, marshalledMarketOrderTx, blockTxs[3])
	require.Equal(t, marshalledLimitOrderTx, blockTxs[4])
	require.Equal(t, []byte("456"), blockTxs[5])
}

type tstApp struct {
	*processor.App
	ctrl               *gomock.Controller
	timeSvc            *mocks.MockTimeService
	epochSvc           *mocks.MockEpochService
	delegation         *mocks.MockDelegationEngine
	exec               *mocks.MockExecutionEngine
	gov                *mocks.MockGovernanceEngine
	stats              *mocks.MockStats
	assets             *mocks.MockAssets
	validator          *mocks.MockValidatorTopology
	notary             *mocks.MockNotary
	evtForwarder       *mocks.MockEvtForwarder
	witness            *mocks.MockWitness
	banking            *mocks.MockBanking
	netParams          *mocks.MockNetworkParameters
	oracleEngine       *mocks.MockOraclesEngine
	oracleAdaptors     *mocks.MockOracleAdaptors
	l1Verifier         *mocks.MockEthereumOracleVerifier
	l2Verifier         *mocks.MockEthereumOracleVerifier
	limits             *mocks.MockLimits
	stakeVerifier      *mocks.MockStakeVerifier
	stakingAccs        *mocks.MockStakingAccounts
	primaryERC20       *mocks.MockERC20MultiSigTopology
	secondaryERC20     *mocks.MockERC20MultiSigTopology
	cp                 *mocks.MockCheckpoint
	broker             *mocks.MockBroker
	evtForwarderHB     *mocks.MockEvtForwarderHeartbeat
	spam               *mocks.MockSpamEngine
	pow                *mocks.MockPoWEngine
	snap               *mocks.MockSnapshotEngine
	stateVar           *mocks.MockStateVarEngine
	teams              *mocks.MockTeamsEngine
	referral           *mocks.MockReferralProgram
	volDiscount        *mocks.MockVolumeDiscountProgram
	volRebate          *mocks.MockVolumeRebateProgram
	bClient            *mocks.MockBlockchainClient
	puSvc              *mocks.MockProtocolUpgradeService
	ethCallEng         *mocks.MockEthCallEngine
	balance            *mocks.MockBalanceChecker
	parties            *mocks.MockPartiesEngine
	txCache            *mocks.MockTxCache
	codec              processor.NullBlockchainTxCodec
	onTickCB           []func(context.Context, time.Time)
	pChainID, sChainID uint64
}

func TestProtocolUpgradeFailedBrokerStreamError(t *testing.T) {
	streamClient := newBrokerClient(0)
	wg := sync.WaitGroup{}
	wg.Add(1)
	stop := func() error {
		wg.Done()
		return nil
	}
	ctx, cfunc := context.WithCancel(context.Background())
	app := getTestAppWithInit(t, cfunc, stop, false, false)
	defer func() {
		wg.Wait()
		streamClient.finish()
		app.ctrl.Finish()
	}()

	// vars
	blockHeight := uint64(123)
	blockHash := "0xDEADBEEF"
	proposer := "0xCAFECAFE1"
	updateTime := time.Now()
	brokerErr := errors.Errorf("pretend something went wrong")

	// let's make this look like a protocol upgrade
	app.txCache.EXPECT().SetRawTxs(nil, blockHeight).Times(1)

	// These are the calls made in the startProtocolUpgrade func:
	app.puSvc.EXPECT().CoreReadyForUpgrade().Times(1).Return(true)
	// broker streaming is enabled, so let's set up the channels and push some data
	app.broker.EXPECT().StreamingEnabled().Times(1).Return(true)
	app.broker.EXPECT().SocketClient().Times(1).Return(streamClient)
	// we expect the protocol upgraded started event to be sent once at least.
	app.broker.EXPECT().Send(gomock.Any()).Times(1)
	// as part of the event data sent here, we call stats to get the height:
	app.stats.EXPECT().Height().Times(1).Return(blockHeight - 1)
	// now we set the upgrade service as ready:
	app.puSvc.EXPECT().SetReadyForUpgrade().Times(0)

	// start stream client routine, throw some events on the channels
	go func() {
		// this event should be read, but not have any effect on the logic.
		streamClient.evtCh <- events.NewTime(ctx, updateTime)
		require.True(t, blockHeight > 0) // just check we reach this part
		streamClient.errCh <- brokerErr
	}()

	// see if we can recover here?
	defer func() {
		if r := recover(); r != nil {
			expect := "failed to wait for data node to get ready for upgrade"
			msg := fmt.Sprintf("Test likely passed, recovered: %v\n", r)
			require.Contains(t, msg, expect, msg)
		}
	}()

	// start upgrade
	_ = app.OnBeginBlock(blockHeight, blockHash, updateTime, proposer, nil)
}

func TestOnBeginBlock(t *testing.T) {
	// keeping the test for reference, as it can be used for reference with regular OnBeginBlock calls
	// or error cases with for protocol upgrade stuff.
	// set up stop call, so we can ensure the chain is stopped as part of protocol upgrade.
	_, cfunc := context.WithCancel(context.Background())
	app := getTestAppWithInit(t, cfunc, stopDummy, true, true)
	defer app.ctrl.Finish()

	// vars
	blockHeight := uint64(123)
	blockHash := "0xDEADBEEF"
	proposer := "0xCAFECAFE1"
	updateTime := time.Now()
	prevTime := updateTime.Add(-1 * time.Second)

	// let's make this look like a protocol upgrade
	app.txCache.EXPECT().SetRawTxs(nil, blockHeight).Times(1)

	// check for upgrade
	app.puSvc.EXPECT().CoreReadyForUpgrade().Times(1).Return(false)

	// now we're back to the OnBeginBlock call, set mocks for the remainder of that func:
	app.broker.EXPECT().Send(gomock.Any()).Times(1)
	// we're not passing any transactions ATM, so no setTxStats to worry about
	// now PoW
	app.pow.EXPECT().BeginBlock(blockHeight, blockHash, nil).Times(1)
	// spam:
	app.spam.EXPECT().BeginBlock(nil).Times(1)
	// now do stats:
	app.stats.EXPECT().SetHash(blockHash).Times(1)
	app.stats.EXPECT().SetHeight(blockHeight).Times(1)
	// now the calls to time service:
	app.timeSvc.EXPECT().SetTimeNow(gomock.Any(), updateTime).Times(1)
	app.timeSvc.EXPECT().GetTimeNow().Times(1).Return(updateTime)
	app.timeSvc.EXPECT().GetTimeLastBatch().Times(1).Return(prevTime)
	// begin upgrade:
	app.puSvc.EXPECT().BeginBlock(gomock.Any(), blockHeight).Times(1)
	// topology:
	app.validator.EXPECT().BeginBlock(gomock.Any(), blockHeight, proposer).Times(1)
	// balance checker:
	app.balance.EXPECT().BeginBlock(gomock.Any()).Times(1)
	// exec engine:
	app.exec.EXPECT().BeginBlock(gomock.Any(), updateTime.Sub(prevTime)).Times(1)

	// actually make the call now:
	ctx := app.OnBeginBlock(blockHeight, blockHash, updateTime, proposer, nil)
	// we should get a context back, this just checks the call returned.
	require.NotNil(t, ctx)
}

func stopDummy() error {
	return nil
}

func getTestApp(t *testing.T, cfunc func(), stop func() error, PoW, Spam bool) *tstApp {
	t.Helper()
	pChain := "1"
	sChain := "2"
	gHandler := genesis.New(
		logging.NewTestLogger(),
		genesis.NewDefaultConfig(),
	)
	pChainID, err := strconv.ParseUint(pChain, 10, 64)
	if err != nil {
		t.Fatalf("Could not get test app instance, chain ID parse error: %v", err)
	}
	sChainID, err := strconv.ParseUint(sChain, 10, 64)
	if err != nil {
		t.Fatalf("Could not get test app instance, chain ID parse error: %v", err)
	}
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	timeSvc := mocks.NewMockTimeService(ctrl)
	epochSvc := mocks.NewMockEpochService(ctrl)
	delegation := mocks.NewMockDelegationEngine(ctrl)
	exec := mocks.NewMockExecutionEngine(ctrl)
	gov := mocks.NewMockGovernanceEngine(ctrl)
	stats := mocks.NewMockStats(ctrl)
	assets := mocks.NewMockAssets(ctrl)
	validator := mocks.NewMockValidatorTopology(ctrl)
	notary := mocks.NewMockNotary(ctrl)
	evtForwarder := mocks.NewMockEvtForwarder(ctrl)
	evtForwarderHB := mocks.NewMockEvtForwarderHeartbeat(ctrl)
	witness := mocks.NewMockWitness(ctrl)
	banking := mocks.NewMockBanking(ctrl)
	netParams := mocks.NewMockNetworkParameters(ctrl)
	oracleEngine := mocks.NewMockOraclesEngine(ctrl)
	oracleAdaptors := mocks.NewMockOracleAdaptors(ctrl)
	l1Verifier := mocks.NewMockEthereumOracleVerifier(ctrl)
	l2Verifier := mocks.NewMockEthereumOracleVerifier(ctrl)
	limits := mocks.NewMockLimits(ctrl)
	stakeVerifier := mocks.NewMockStakeVerifier(ctrl)
	stakingAccs := mocks.NewMockStakingAccounts(ctrl)
	pERC20 := mocks.NewMockERC20MultiSigTopology(ctrl)
	sERC20 := mocks.NewMockERC20MultiSigTopology(ctrl)
	cp := mocks.NewMockCheckpoint(ctrl)
	vaultService := mocks.NewMockVaultService(ctrl)
	var (
		spam *mocks.MockSpamEngine
		pow  *mocks.MockPoWEngine
	)
	if Spam {
		spam = mocks.NewMockSpamEngine(ctrl)
	}
	if PoW {
		pow = mocks.NewMockPoWEngine(ctrl)
	}
	snap := mocks.NewMockSnapshotEngine(ctrl)
	stateVar := mocks.NewMockStateVarEngine(ctrl)
	teams := mocks.NewMockTeamsEngine(ctrl)
	referral := mocks.NewMockReferralProgram(ctrl)
	volDiscount := mocks.NewMockVolumeDiscountProgram(ctrl)
	volRebate := mocks.NewMockVolumeRebateProgram(ctrl)
	bClient := mocks.NewMockBlockchainClient(ctrl)
	puSvc := mocks.NewMockProtocolUpgradeService(ctrl)
	ethCallEng := mocks.NewMockEthCallEngine(ctrl)
	balance := mocks.NewMockBalanceChecker(ctrl)
	parties := mocks.NewMockPartiesEngine(ctrl)
	txCache := mocks.NewMockTxCache(ctrl)
	codec := processor.NullBlockchainTxCodec{}
	// paths, config, gastimator, ...
	vp := paths.New("/tmp")
	conf := processor.NewDefaultConfig()
	gastimator := processor.NewGastimator(exec)
	// test wrapper
	tstApp := &tstApp{
		ctrl:           ctrl,
		broker:         broker,
		timeSvc:        timeSvc,
		epochSvc:       epochSvc,
		delegation:     delegation,
		exec:           exec,
		gov:            gov,
		stats:          stats,
		assets:         assets,
		validator:      validator,
		notary:         notary,
		evtForwarder:   evtForwarder,
		evtForwarderHB: evtForwarderHB,
		witness:        witness,
		banking:        banking,
		netParams:      netParams,
		oracleEngine:   oracleEngine,
		oracleAdaptors: oracleAdaptors,
		l1Verifier:     l1Verifier,
		l2Verifier:     l2Verifier,
		limits:         limits,
		stakeVerifier:  stakeVerifier,
		stakingAccs:    stakingAccs,
		primaryERC20:   pERC20,
		secondaryERC20: sERC20,
		cp:             cp,
		spam:           spam,
		pow:            pow,
		snap:           snap,
		stateVar:       stateVar,
		teams:          teams,
		referral:       referral,
		volDiscount:    volDiscount,
		volRebate:      volRebate,
		bClient:        bClient,
		puSvc:          puSvc,
		ethCallEng:     ethCallEng,
		balance:        balance,
		parties:        parties,
		txCache:        txCache,
		codec:          codec,
		onTickCB:       []func(context.Context, time.Time){},
		pChainID:       pChainID,
		sChainID:       sChainID,
	}
	// timeSvc will be set up to the onTick callback
	timeSvc.EXPECT().NotifyOnTick(gomock.Any()).AnyTimes().Do(func(cbs ...func(context.Context, time.Time)) {
		if cbs == nil {
			return
		}
		tstApp.onTickCB = append(tstApp.onTickCB, cbs...)
	})
	// ensureConfig calls netparams
	netParams.EXPECT().GetJSONStruct(netparams.BlockchainsPrimaryEthereumConfig, gomock.Any()).Times(1).DoAndReturn(func(_ string, v netparams.Reset) error {
		vt, ok := v.(*proto.EthereumConfig)
		if !ok {
			return errors.Errorf("invalid type %t", v)
		}
		vt.ChainId = pChain
		return nil
	})
	netParams.EXPECT().GetJSONStruct(netparams.BlockchainsEVMBridgeConfigs, gomock.Any()).Times(1).DoAndReturn(func(_ string, v netparams.Reset) error {
		vt, ok := v.(*proto.EVMBridgeConfigs)
		if !ok {
			return errors.Errorf("invalid type %t", v)
		}
		if vt.Configs == nil {
			vt.Configs = []*proto.EVMBridgeConfig{}
		}
		vt.Configs = append(vt.Configs, &proto.EVMBridgeConfig{
			ChainId: sChain,
		})
		return nil
	})
	// set primary chain ID
	exec.EXPECT().OnChainIDUpdate(pChainID).Times(1).Return(nil)
	gov.EXPECT().OnChainIDUpdate(pChainID).Times(1).Return(nil)
	app := processor.NewApp(
		logging.NewTestLogger(),
		vp,
		conf,
		cfunc,
		stop,
		assets,
		banking,
		broker,
		witness,
		evtForwarder,
		evtForwarderHB,
		exec,
		gHandler,
		gov,
		notary,
		stats,
		timeSvc,
		epochSvc,
		validator,
		netParams,
		&processor.Oracle{
			Engine:                    oracleEngine,
			Adaptors:                  oracleAdaptors,
			EthereumOraclesVerifier:   l1Verifier,
			EthereumL2OraclesVerifier: l2Verifier,
		},
		delegation,
		limits,
		stakeVerifier,
		cp,
		spam,
		pow,
		stakingAccs,
		snap,
		stateVar,
		teams,
		referral,
		volDiscount,
		volRebate,
		bClient,
		pERC20,
		sERC20,
		"0",
		puSvc,
		&codec,
		gastimator,
		ethCallEng,
		balance,
		parties,
		txCache,
		vaultService,
	)

	// embed the app
	tstApp.App = app
	// return wrapper
	return tstApp
}

func getTestAppWithInit(t *testing.T, cfunc func(), stop func() error, PoW, Spam bool) *tstApp {
	t.Helper()
	tstApp := getTestApp(t, cfunc, stop, PoW, Spam)
	// now set up the OnInitChain stuff
	req := &tmtypes.RequestInitChain{
		ChainId:       "1",
		InitialHeight: 0,
		Time:          time.Now().Add(-1 * time.Hour), // some time in the past
	}
	// set up mock calls:
	tstApp.broker.EXPECT().Send(gomock.Any()).Times(2) // once before, and once after gHandler is called
	tstApp.ethCallEng.EXPECT().Start().Times(1)
	tstApp.validator.EXPECT().GetValidatorPowerUpdates().Times(1).Return(nil)
	resp, err := tstApp.OnInitChain(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	return tstApp
}

// fake socket client.
type brokerClient struct {
	errCh chan error
	evtCh chan events.Event
}

func newBrokerClient(buf int) *brokerClient {
	return &brokerClient{
		errCh: make(chan error, buf),
		evtCh: make(chan events.Event, buf),
	}
}

func (b *brokerClient) finish() {
	if b.errCh != nil {
		close(b.errCh)
		b.errCh = nil
	}
	if b.evtCh != nil {
		close(b.evtCh)
		b.evtCh = nil
	}
}

func (b *brokerClient) SendBatch(events []events.Event) error {
	return nil
}

func (b *brokerClient) Receive(ctx context.Context) (<-chan events.Event, <-chan error) {
	return b.evtCh, b.errCh
}
