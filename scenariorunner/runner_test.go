package scenariorunner_test

import (
	"log"
	"testing"

	types "code.vegaprotocol.io/vega/proto"
	sr "code.vegaprotocol.io/vega/scenariorunner"
	"code.vegaprotocol.io/vega/scenariorunner/core"

	"github.com/stretchr/testify/assert"
)

func TestProcessInstructions(t *testing.T) {

	runner, err := sr.NewScenarioRunner()
	if err != nil {
		t.Fatal(err)
	}

	trader1 := "trader1"
	instr1, err := core.NewInstruction(
		"NotifyTraderAccount",
		&types.NotifyTraderAccount{
			TraderID: trader1,
		},
	)

	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
	}

	sellOrderId := "myId1"
	marketId := "ONLKZ6XIXYKWFDNHBWKZUAM7DFLQ42DZ"
	sell := types.Side_Sell
	instr2, err := core.NewInstruction(
		"SubmitOrder",
		&types.Order{
			Id:          sellOrderId,
			MarketID:    marketId,
			PartyID:     trader1,
			Price:       100,
			Size:        3,
			Side:        sell,
			TimeInForce: types.Order_GTC,
			ExpiresAt:   1924991999000000000,
		},
	)
	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
	}
	instr2.Description = "Submit a sell order"

	instr3, err := core.NewInstruction(
		"CancelOrder",
		&types.Order{
			Id:       sellOrderId,
			MarketID: "ONLKZ6XIXYKWFDNHBWKZUAM7DFLQ42DZ",
			Side:     sell,
		},
	)
	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
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
		log.Fatalln("Failed to create a new instruction: ", err)
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
		log.Fatalln("Failed to create a new instruction: ", err)
	}

	instr6, err := core.NewInstruction(
		"Withdraw",
		&types.Withdraw{
			PartyID: trader2,
			Amount:  1000,
		},
	)
	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
	}

	instructionSet := &core.InstructionSet{
		Instructions: []*core.Instruction{
			instr1,
			instr2,
			instr3,
			instr4,
			instr5,
			instr6,
		},
		Description: "Test instructions",
	}

	result, err := runner.ProcessInstructions(*instructionSet)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.EqualValues(t, len(instructionSet.Instructions), result.Summary.InstructionsProcessed)
	assert.EqualValues(t, 0, result.Summary.InstructionsOmitted)

}
