package scenariorunner_test

import (
	"errors"
	"log"
	"testing"

	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/proto/api"
	sr "code.vegaprotocol.io/vega/scenariorunner"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestNewInstruction(t *testing.T) {
	request := "testRequest"
	trader := "testTrader"
	message := api.NotifyTraderAccountRequest{
		Notif: &types.NotifyTraderAccount{
			TraderID: trader,
		},
	}

	instruction, err := sr.NewInstruction(request, &message)
	assert.NoError(t, err)
	assert.Equal(t, request, instruction.Request)
	assert.Equal(t, "", instruction.Description)

	var messageReconstructed api.NotifyTraderAccountRequest
	err = proto.Unmarshal(instruction.Message.Value, &messageReconstructed)
	if err != nil {
		log.Fatalln("Failed to unmarshal response message: ", err)
	}
	assert.EqualValues(t, trader, messageReconstructed.Notif.TraderID)
}

func TestNewResult(t *testing.T) {
	request := "testRequest"
	trader := "testTrader"
	message := api.NotifyTraderAccountRequest{
		Notif: &types.NotifyTraderAccount{
			TraderID: trader,
		},
	}
	instruction, err := sr.NewInstruction(request, &message)
	assert.NoError(t, err)

	response := api.NotifyTraderAccountResponse{
		Submitted: true,
	}

	innerErr := errors.New("Mock error")
	res, err := instruction.NewResult(&response, innerErr)
	assert.NoError(t, err)
	assert.Equal(t, innerErr.Error(), res.Error)
	assert.EqualValues(t, instruction, res.Instruction)

	var responseReconstructed api.NotifyTraderAccountResponse
	err = proto.Unmarshal(res.Response.Value, &responseReconstructed)
	if err != nil {
		t.Fatalf("Failed to unmarshal response message: %s", err.Error())
	}
	assert.EqualValues(t, response.Submitted, responseReconstructed.Submitted)

}

func TestNewResultWithEmptyReposnse(t *testing.T) {
	request := "testRequest"
	trader := "testTrader"
	message := api.NotifyTraderAccountRequest{
		Notif: &types.NotifyTraderAccount{
			TraderID: trader,
		},
	}
	instruction, err := sr.NewInstruction(request, &message)
	if err != nil {
		t.Fatalf("Failed to create a new instruction: %s", err.Error())
	}

	result, err := instruction.NewResult(nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}
