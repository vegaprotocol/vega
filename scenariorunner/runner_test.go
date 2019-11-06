package scenariorunner_test

import (
	"testing"
	"time"

	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	sr "code.vegaprotocol.io/vega/scenariorunner"
	"code.vegaprotocol.io/vega/scenariorunner/core"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
)

const marketId string = "ONLKZ6XIXYKWFDNHBWKZUAM7DFLQ42DZ"

func TestProcessInstructionsAll(t *testing.T) {

	partyId := "party1"
	orderId := "order1"
	instructions1, err := getExecutionEngineInstructions(marketId, partyId, orderId)
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

	instructions := append(append(instructions1, instructions2...), instructions3...)

	instructionSet := core.InstructionSet{
		Instructions: instructions,
		Description:  "Test instructions",
	}

	testInstructionSet(t, instructionSet)
}

func TestProcessInstructionsExecution(t *testing.T) {
	instructions, err := getExecutionEngineInstructions(marketId, "party1", "order1")
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
	runner, err := sr.NewScenarioRunner()
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

func getExecutionEngineInstructions(marketId string, trader1Id string, order1Id string) ([]*core.Instruction, error) {
	instr1, err := core.NewInstruction(
		"NotifyTraderAccount",
		&types.NotifyTraderAccount{
			TraderID: trader1Id,
		},
	)

	if err != nil {
		return nil, err
	}

	sell := types.Side_Sell
	instr2, err := core.NewInstruction(
		"SubmitOrder",
		&types.Order{
			Id:          order1Id,
			MarketID:    marketId,
			PartyID:     trader1Id,
			Price:       100,
			Size:        3,
			Side:        sell,
			TimeInForce: types.Order_GTC,
			ExpiresAt:   1924991999000000000,
		},
	)
	if err != nil {
		return nil, err
	}
	instr2.Description = "Submit a sell order"

	instr3, err := core.NewInstruction(
		"CancelOrder",
		&types.Order{
			Id:       order1Id,
			MarketID: marketId,
			Side:     sell,
		},
	)
	if err != nil {
		return nil, err
	}

	buy := types.Side_Buy
	buyOrderID := "myId2"
	trader2 := "trader2"
	instr4, err := core.NewInstruction(
		"SubmitOrder",
		&types.Order{
			Id:          buyOrderID,
			MarketID:    marketId,
			PartyID:     trader2,
			Price:       100,
			Size:        3,
			Side:        buy,
			TimeInForce: types.Order_GTC,
			ExpiresAt:   1924991999000000000,
		},
	)
	if err != nil {
		return nil, err
	}

	instr5, err := core.NewInstruction(
		"AmendOrder",
		&types.OrderAmendment{
			OrderID:   buyOrderID,
			PartyID:   trader2,
			Price:     100,
			Size:      30,
			ExpiresAt: 1924991999000000000,
		},
	)
	if err != nil {
		return nil, err
	}

	instr6, err := core.NewInstruction(
		"Withdraw",
		&types.Withdraw{
			PartyID: trader2,
			Amount:  1000,
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
	}

	return instructions, nil
}

func getTradingDataInstructions(marketId string, partyId string, orderId string) ([]*core.Instruction, error) {
	instr1, err := core.NewInstruction(
		"MarketDepth",
		&protoapi.MarketDepthRequest{
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr2, err := core.NewInstruction(
		"MarketById",
		&protoapi.MarketByIDRequest{
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr3, err := core.NewInstruction(
		"Markets",
		&empty.Empty{},
	)
	if err != nil {
		return nil, err
	}

	instr4, err := core.NewInstruction(
		"OrdersByMarket",
		&protoapi.OrdersByMarketRequest{
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr5, err := core.NewInstruction(
		"OrdersByParty",
		&protoapi.OrdersByPartyRequest{
			PartyID: partyId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr6, err := core.NewInstruction(
		"OrderByMarketAndId",
		&protoapi.OrderByMarketAndIdRequest{
			MarketID: marketId,
			OrderID:  orderId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr7, err := core.NewInstruction(
		"OrderByReference",
		&protoapi.OrderByReferenceRequest{
			Reference: "testReference",
		},
	)
	if err != nil {
		return nil, err
	}

	instr8, err := core.NewInstruction(
		"TradesByMarket",
		&protoapi.TradesByMarketRequest{
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr9, err := core.NewInstruction(
		"TradesByParty",
		&protoapi.TradesByPartyRequest{
			PartyID:  partyId,
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr10, err := core.NewInstruction(
		"TradesByOrder",
		&protoapi.TradesByOrderRequest{
			OrderID: orderId,
		},
	)
	if err != nil {
		return nil, err
	}

	instr11, err := core.NewInstruction(
		"LastTrade",
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
		"SetTime",
		&core.SetTimeRequest{
			Time: ts,
		},
	)
	if err != nil {
		return nil, err
	}

	instr2, err := core.NewInstruction(
		"AdvanceTime",
		&core.AdvanceTimeRequest{
			TimeDelta: ptypes.DurationProto(time.Nanosecond),
		},
	)
	if err != nil {
		return nil, err
	}

	instr3, err := core.NewInstruction(
		"AdvanceTime",
		&core.AdvanceTimeRequest{
			TimeDelta: ptypes.DurationProto(time.Hour),
		},
	)
	if err != nil {
		return nil, err
	}
	instr4, err := core.NewInstruction(
		"MarketSummary",
		&core.MarketSummaryRequest{
			MarketID: marketId,
		},
	)
	if err != nil {
		return nil, err
	}
	instr5, err := core.NewInstruction(
		"ProtocolSummary",
		&core.ProtocolSummaryRequest{},
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
