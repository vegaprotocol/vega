package scenariorunner_test

import (
	"log"
	"testing"

	types "code.vegaprotocol.io/vega/proto"
	sr "code.vegaprotocol.io/vega/scenariorunner"

	"github.com/stretchr/testify/assert"
)

func TestProcessInstructions(t *testing.T) {

	runner, err := sr.NewScenarionRunner()
	if err != nil {
		t.Fatal(err)
	}

	instr1, err := sr.NewInstruction(
		"NotifyTraderAccount",
		&types.NotifyTraderAccount{
			TraderID: "trader1",
		},
	)

	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
	}
	instr2, err := sr.NewInstruction(
		"SubmitOrder",
		&types.OrderSubmission{
			MarketID:    "ONLKZ6XIXYKWFDNHBWKZUAM7DFLQ42DZ",
			PartyID:     "trader1",
			Price:       100,
			Size:        3,
			Side:        types.Side_Sell,
			TimeInForce: types.Order_GTC,
			ExpiresAt:   1924991999000000000,
		},
	)

	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
	}
	instr2.Description = "Submit a sell order"
	instructionSet := &sr.InstructionSet{
		Instructions: []*sr.Instruction{
			instr1,
			instr2,
		},
		Description: "Test instructions",
	}

	result, err := runner.ProcessInstructions(*instructionSet)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.EqualValues(t, 2, result.Summary.InstructionsProcessed)
	assert.EqualValues(t, 0, result.Summary.InstructionsOmitted)

}
