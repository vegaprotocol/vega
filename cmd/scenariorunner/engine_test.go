package main

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/cmd/scenariorunner/core"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
)

const marketId string = "JXGQYDVQAP5DJUAQBCB4PACVJPFJR4XI"

func TestSubmitOrderAndReadStores(t *testing.T) {

	party1 := "V@d3r"
	party2 := "Luk39"

	notifyParty1, err := core.NewInstruction(
		core.RequestType_NOTIFY_TRADER_ACCOUNT,
		&protoapi.NotifyTraderAccountRequest{
			Notif: &types.NotifyTraderAccount{
				TraderID: party1,
				Amount:   10,
			},
		},
	)
	assert.NoError(t, err)

	notifyParty2, err := core.NewInstruction(
		core.RequestType_NOTIFY_TRADER_ACCOUNT,
		&protoapi.NotifyTraderAccountRequest{
			Notif: &types.NotifyTraderAccount{
				TraderID: party2,
				Amount:   10,
			},
		},
	)
	assert.NoError(t, err)

	sellOrderParty1, err := core.NewInstruction(
		core.RequestType_SUBMIT_ORDER,
		&protoapi.SubmitOrderRequest{
			Submission: &types.OrderSubmission{
				MarketID:    marketId,
				PartyID:     party1,
				Price:       100,
				Size:        3,
				Type:        types.Order_LIMIT,
				Side:        types.Side_Sell,
				TimeInForce: types.Order_GTC,
			},
		},
	)
	assert.NoError(t, err)

	buyOrderParty2, err := core.NewInstruction(
		core.RequestType_SUBMIT_ORDER,
		&protoapi.SubmitOrderRequest{
			Submission: &types.OrderSubmission{
				MarketID:    marketId,
				PartyID:     party1,
				Price:       100,
				Size:        2,
				Type:        types.Order_LIMIT,
				Side:        types.Side_Buy,
				TimeInForce: types.Order_GTC,
			},
		},
	)
	assert.NoError(t, err)

	instructionSet := core.InstructionSet{
		Instructions: []*core.Instruction{
			notifyParty1, notifyParty2, sellOrderParty1, buyOrderParty2,
		},
		Description: "Submit two orders, expect one trade and stores updated",
	}
	log := logging.NewTestLogger()
	storageConfig, err := storage.NewTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	defer storage.FlushStores(log, storageConfig)
	runner, err := NewEngine(log, NewDefaultConfig(), storageConfig, "test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = runner.ProcessInstructions(instructionSet)

	result, err := runner.ExtractData()
	assert.NoError(t, err)
	assert.True(t, len(result.Summary.Parties) > 0)

	anyOrders := false
	anyTrades := false
	for _, mkt := range result.Summary.Markets {
		if len(mkt.Orders) > 0 {
			anyOrders = true
		}
		if len(mkt.Trades) > 0 {
			anyTrades = true
		}
	}
	assert.True(t, anyOrders)
	assert.True(t, anyTrades)
}

func TestExtractData(t *testing.T) {

	instructions, err := getExecutionEngineInstructions(marketId, "trader1")
	if err != nil {
		t.Fatal(err)
	}
	instructionSet := core.InstructionSet{
		Instructions: instructions,
		Description:  "Executing a trade",
	}
	log := logging.NewTestLogger()
	storageConfig, err := storage.NewTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	defer storage.FlushStores(log, storageConfig)
	runner, err := NewEngine(log, NewDefaultConfig(), storageConfig, "test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = runner.ProcessInstructions(instructionSet)

	result, err := runner.ExtractData()
	assert.NoError(t, err)
	assert.True(t, len(result.Summary.Parties) > 0)

	anyOrders := false
	anyTrades := false
	for _, mkt := range result.Summary.Markets {
		if len(mkt.Orders) > 0 {
			anyOrders = true
		}
		if len(mkt.Trades) > 0 {
			anyTrades = true
		}
	}
	assert.True(t, anyOrders)
	assert.True(t, anyTrades)

}

// TODO (WG 08/11/2019) The tests below are integration tests used during development. They should be moved to where we keep integration tests and executed with dependencies injected from outside.
func TestProcessInstructionsAll(t *testing.T) {

	partyId := "party1"
	orderId := "order1"
	instructions1, err := getExecutionEngineInstructions(marketId, partyId)
	if err != nil {
		t.Fatal(err)
	}
	instructions2, err := getTradingDataInstructions(marketId, partyId, orderId)
	if err != nil {
		t.Fatal(err)
	}
	instructions3, err := getInternalInstructions(marketId)
	if err != nil {
		t.Fatal(err)
	}
	instructions4, err := getAccountInstructions(marketId, partyId)
	if err != nil {
		t.Fatal(err)
	}
	instructions5, err := getCandleInstructions(marketId)
	if err != nil {
		t.Fatal(err)
	}
	instructions6, err := getPositionInstructions(marketId)
	if err != nil {
		t.Fatal(err)
	}
	instructions7, err := getPositionInstructions(marketId)
	if err != nil {
		t.Fatal(err)
	}

	instructions := append(instructions1, instructions2...)
	instructions = append(instructions, instructions3...)
	instructions = append(instructions, instructions4...)
	instructions = append(instructions, instructions5...)
	instructions = append(instructions, instructions6...)
	instructions = append(instructions, instructions7...)

	instructionSet := core.InstructionSet{
		Instructions: instructions,
		Description:  "Test instructions",
	}

	testInstructionSet(t, instructionSet)
}

func TestProcessInstructionsExecution(t *testing.T) {
	instructions, err := getExecutionEngineInstructions(marketId, "party1")
	if err != nil {
		t.Fatal(err)
	}

	instructionSet := core.InstructionSet{
		Instructions: instructions,
		Description:  "Test instructions",
	}

	testInstructionSet(t, instructionSet)
}

func TestProcessInstructionsTradingData(t *testing.T) {

	instructions, err := getTradingDataInstructions(marketId, "party1", "order1")
	if err != nil {
		t.Fatal(err)
	}

	instructionSet := core.InstructionSet{
		Instructions: instructions,
		Description:  "Test instructions",
	}

	testInstructionSet(t, instructionSet)
}

func TestProcessInstructionsTime(t *testing.T) {

	instructions, err := getInternalInstructions(marketId)
	if err != nil {
		t.Fatal(err)
	}

	instructionSet := core.InstructionSet{
		Instructions: instructions,
		Description:  "Test instructions",
	}

	testInstructionSet(t, instructionSet)

}

func TestProcessInstructionsInternal(t *testing.T) {

	instructions, err := getInternalInstructions(marketId)
	if err != nil {
		t.Fatal(err)
	}

	instructionSet := core.InstructionSet{
		Instructions: instructions,
		Description:  "Test instructions",
	}

	testInstructionSet(t, instructionSet)

}

func testInstructionSet(t *testing.T, instructionSet core.InstructionSet) {
	log := logging.NewTestLogger()
	storageConfig, err := storage.NewTestConfig()
	if err != nil {
		t.Fatal(err)
	}
	defer storage.FlushStores(log, storageConfig)
	runner, err := NewEngine(log, NewDefaultConfig(), storageConfig, "test")
	if err != nil {
		t.Fatal(err)
	}

	result, err := runner.ProcessInstructions(instructionSet)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Metadata)
	assert.EqualValues(t, len(instructionSet.Instructions), result.Metadata.InstructionsProcessed)
	assert.EqualValues(t, 0, result.Metadata.InstructionsOmitted)
	assert.NotNil(t, result.Metadata.FinalMarketDepth)
	assert.True(t, result.Metadata.ProcessingTime.GetNanos() > 0)
	assert.EqualValues(t, len(instructionSet.Instructions), len(result.Results))
}

func getExecutionEngineInstructions(marketId string, trader1Id string) ([]*core.Instruction, error) {
	instr1, err := core.NewInstruction(
		core.RequestType_NOTIFY_TRADER_ACCOUNT,
		&protoapi.NotifyTraderAccountRequest{
			Notif: &types.NotifyTraderAccount{
				TraderID: trader1Id,
				Amount:   1000,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	sell := types.Side_Sell
	instr2, err := core.NewInstruction(
		core.RequestType_SUBMIT_ORDER,
		&protoapi.SubmitOrderRequest{
			Submission: &types.OrderSubmission{
				MarketID:    marketId,
				PartyID:     trader1Id,
				Price:       100,
				Size:        3,
				Side:        sell,
				TimeInForce: types.Order_GTC,
				ExpiresAt:   1924991999000000000,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	instr2.Description = "Submit a sell order"

	//instr3, err := core.NewInstruction(
	//	core.RequestType_CANCEL_ORDER,
	//	&protoapi.CancelOrderRequest{
	//		Cancellation: &types.OrderCancellation{
	//			OrderID:  "",
	//			MarketID: marketId,
	//			PartyID:  trader1Id,
	//		},
	//	},
	//)
	//if err != nil {
	//	return nil, err
	//}

	trader2 := "trader2"
	instr4, err := core.NewInstruction(
		core.RequestType_NOTIFY_TRADER_ACCOUNT,
		&protoapi.NotifyTraderAccountRequest{
			Notif: &types.NotifyTraderAccount{
				TraderID: trader2,
				Amount:   1000,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	buy := types.Side_Buy
	instr5, err := core.NewInstruction(
		core.RequestType_SUBMIT_ORDER,
		&protoapi.SubmitOrderRequest{
			Submission: &types.OrderSubmission{
				MarketID:    marketId,
				PartyID:     trader2,
				Price:       100,
				Size:        3,
				Side:        buy,
				TimeInForce: types.Order_GTC,
				ExpiresAt:   1924991999000000000,
			},
		},
	)

	if err != nil {
		return nil, err
	}

	instr6, err := core.NewInstruction(
		core.RequestType_AMEND_ORDER,
		&protoapi.AmendOrderRequest{
			Amendment: &types.OrderAmendment{
				PartyID:   trader2,
				Price:     100,
				Size:      30,
				ExpiresAt: 1924991999000000000,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	instr7, err := core.NewInstruction(
		core.RequestType_WITHDRAW,
		&protoapi.WithdrawRequest{
			Withdraw: &types.Withdraw{
				PartyID: trader2,
				Amount:  1000,
				Asset:   "BTC",
			},
		},
	)
	if err != nil {
		return nil, err
	}

	instr8, err := core.NewInstruction(
		core.RequestType_NOTIFY_TRADER_ACCOUNT,
		&protoapi.NotifyTraderAccountRequest{
			Notif: &types.NotifyTraderAccount{
				TraderID: "trader3",
				Amount:   1000,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	instr9, err := core.NewInstruction(
		core.RequestType_SUBMIT_ORDER,
		&protoapi.SubmitOrderRequest{
			Submission: &types.OrderSubmission{
				MarketID:    marketId,
				PartyID:     "trader3",
				Price:       100,
				Size:        3,
				Side:        types.Side_Sell,
				TimeInForce: types.Order_GTC,
				ExpiresAt:   1924991999000000000,
			},
		},
	)

	if err != nil {
		return nil, err
	}
	instr10, err := core.NewInstruction(
		core.RequestType_NOTIFY_TRADER_ACCOUNT,
		&protoapi.NotifyTraderAccountRequest{
			Notif: &types.NotifyTraderAccount{
				TraderID: "trader4",
				Amount:   1000,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	instr11, err := core.NewInstruction(
		core.RequestType_SUBMIT_ORDER,
		&protoapi.SubmitOrderRequest{
			Submission: &types.OrderSubmission{
				MarketID:    marketId,
				PartyID:     "trader4",
				Price:       100,
				Size:        3,
				Side:        types.Side_Buy,
				TimeInForce: types.Order_GTC,
				ExpiresAt:   1924991999000000000,
			},
		},
	)

	if err != nil {
		return nil, err
	}

	instructions := []*core.Instruction{
		instr1,
		instr2,
		//instr3,
		instr4,
		instr5,
		instr6,
		instr7,
		instr8,
		instr9,
		instr10,
		instr11,
	}

	return instructions, nil
}

func getTradingDataInstructions(marketId string, partyId string, orderId string) ([]*core.Instruction, error) {
	instr1, err := core.NewInstruction(
		core.RequestType_MARKET_DEPTH,
		&protoapi.MarketDepthRequest{
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr2, err := core.NewInstruction(
		core.RequestType_MARKET_BY_ID,
		&protoapi.MarketByIDRequest{
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr3, err := core.NewInstruction(
		core.RequestType_MARKETS,
		&empty.Empty{},
	)
	if err != nil {
		return nil, err
	}

	instr4, err := core.NewInstruction(
		core.RequestType_ORDERS_BY_MARKET,
		&protoapi.OrdersByMarketRequest{
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr5, err := core.NewInstruction(
		core.RequestType_ORDERS_BY_PARTY,
		&protoapi.OrdersByPartyRequest{
			PartyID: partyId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr6, err := core.NewInstruction(
		core.RequestType_ORDER_BY_MARKET_AND_ID,
		&protoapi.OrderByMarketAndIdRequest{
			MarketID: marketId,
			OrderID:  orderId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr7, err := core.NewInstruction(
		core.RequestType_ORDER_BY_REFERENCE,
		&protoapi.OrderByReferenceRequest{
			Reference: "testReference",
		},
	)
	if err != nil {
		return nil, err
	}

	instr8, err := core.NewInstruction(
		core.RequestType_TRADES_BY_MARKET,
		&protoapi.TradesByMarketRequest{
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr9, err := core.NewInstruction(
		core.RequestType_TRADES_BY_PARTY,
		&protoapi.TradesByPartyRequest{
			PartyID:  partyId,
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr10, err := core.NewInstruction(
		core.RequestType_TRADES_BY_ORDER,
		&protoapi.TradesByOrderRequest{
			OrderID: orderId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr11, err := core.NewInstruction(
		core.RequestType_LAST_TRADE,
		&protoapi.LastTradeRequest{
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instructions := []*core.Instruction{
		instr1,
		instr2,
		instr3,
		instr4,
		instr5,
		instr6,
		instr7,
		instr8,
		instr9,
		instr10,
		instr11,
	}

	return instructions, nil
}

func getInternalInstructions(marketId string) ([]*core.Instruction, error) {
	ts, err := ptypes.TimestampProto(time.Date(2019, 1, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		return nil, err
	}

	instr1, err := core.NewInstruction(
		core.RequestType_SET_TIME,
		&core.SetTimeRequest{
			Time: ts,
		},
	)
	if err != nil {
		return nil, err
	}

	instr2, err := core.NewInstruction(
		core.RequestType_ADVANCE_TIME,
		&core.AdvanceTimeRequest{
			TimeDelta: ptypes.DurationProto(time.Nanosecond),
		},
	)
	if err != nil {
		return nil, err
	}

	instr3, err := core.NewInstruction(
		core.RequestType_ADVANCE_TIME,
		&core.AdvanceTimeRequest{
			TimeDelta: ptypes.DurationProto(time.Hour),
		},
	)
	if err != nil {
		return nil, err
	}
	instr4, err := core.NewInstruction(
		core.RequestType_MARKET_SUMMARY,
		&core.MarketSummaryRequest{
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}
	instr5, err := core.NewInstruction(
		core.RequestType_SUMMARY,
		&core.SummaryRequest{},
	)
	if err != nil {
		return nil, err
	}

	instructions := []*core.Instruction{
		instr1,
		instr2,
		instr3,
		instr4,
		instr5,
	}

	return instructions, nil
}

func getAccountInstructions(marketId string, partyId string) ([]*core.Instruction, error) {
	instr1, err := core.NewInstruction(
		core.RequestType_ACCOUNTS_BY_PARTY,
		&protoapi.AccountsByPartyRequest{
			PartyID: partyId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr2, err := core.NewInstruction(
		core.RequestType_ACCOUNTS_BY_PARTY_AND_ASSET,
		&protoapi.AccountsByPartyAndAssetRequest{
			PartyID: partyId,
			Asset:   "",
		},
	)
	if err != nil {
		return nil, err
	}

	instr3, err := core.NewInstruction(
		core.RequestType_ACCOUNTS_BY_PARTY_AND_MARKET,
		&protoapi.AccountsByPartyAndMarketRequest{
			PartyID:  partyId,
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instructions := []*core.Instruction{
		instr1,
		instr2,
		instr3,
	}

	return instructions, nil
}

func getCandleInstructions(marketId string) ([]*core.Instruction, error) {
	instr1, err := core.NewInstruction(
		core.RequestType_CANDLES,
		&protoapi.CandlesRequest{
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instructions := []*core.Instruction{
		instr1,
	}

	return instructions, nil
}

func getPositionInstructions(partyId string) ([]*core.Instruction, error) {
	instr1, err := core.NewInstruction(
		core.RequestType_POSITIONS_BY_PARTY,
		&protoapi.PositionsByPartyRequest{
			PartyID: partyId,
		},
	)
	if err != nil {
		return nil, err
	}

	instructions := []*core.Instruction{
		instr1,
	}

	return instructions, nil
}

func getPartyInstructions(partyId string) ([]*core.Instruction, error) {
	instr1, err := core.NewInstruction(
		core.RequestType_PARTY_BY_ID,
		&protoapi.PartyByIDRequest{
			PartyID: partyId,
		},
	)
	if err != nil {
		return nil, err
	}
	instr2, err := core.NewInstruction(
		core.RequestType_PARTIES,
		&empty.Empty{},
	)
	if err != nil {
		return nil, err
	}

	instructions := []*core.Instruction{
		instr1,
		instr2,
	}

	return instructions, nil
}
