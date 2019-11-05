package main

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"

	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/scenariorunner/core"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/stretchr/testify/assert"
)

func TestReadFilesFailsWithFakePaths(t *testing.T) {
	fakePaths := []string{"madeUp1", "madeUp2.txt", "abc/madeUp3.json"}
	readFiles, err := readFiles(fakePaths)

	assert.Error(t, err, "Expected an error when reading files from paths that don't exist")
	for i := 0; i < len(fakePaths); i++ {
		assert.Nil(t, readFiles[i], "Expected read files to be nil.")
	}
}

func TestReadFiles(t *testing.T) {
	files := []string{"exampleInstructions.json", "exampleInstructions.json"}
	instrSet, err := ProcessFiles(files)
	assert.NoError(t, err)
	assert.Equal(t, len(files), len(instrSet))
	assert.Equal(t, 2, len(files))
	assert.NotNil(t, instrSet[0])
	assert.EqualValues(t, instrSet[0], instrSet[1])
}

func TestUnmarshallApiTypes(t *testing.T) {

	instr1, err := core.NewInstruction(
		"trading.NotifyTraderAccount",
		&api.NotifyTraderAccountRequest{
			Notif: &types.NotifyTraderAccount{
				TraderID: "trader1",
			},
		})

	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
	}
	instr2, err := core.NewInstruction(
		"trading.SubmitOrder",
		&api.SubmitOrderRequest{
			Submission: &types.OrderSubmission{
				MarketID:    "Market1",
				PartyID:     "trader1",
				Price:       100,
				Size:        3,
				Side:        types.Side_Sell,
				TimeInForce: types.Order_GTC,
				ExpiresAt:   1924991999000000000,
			},
		})
	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
	}
	instr2.Description = "Submit a sell order"
	expected := &core.InstructionSet{
		Instructions: []*core.Instruction{
			instr1,
			instr2,
		},
		Description: "Test instructions",
	}
	data := strings.NewReader(`{
	"description": "Test instructions",
	"instructions": [
	{
	"request": "trading.NotifyTraderAccount",
	"message": {
		"@type": "api.NotifyTraderAccountRequest",
		"notif": {
		"traderID": "trader1"
		}
	}
	},
	{
	"description": "Submit a sell order",
	"request": "trading.SubmitOrder",
	"message": {
		"@type": "api.SubmitOrderRequest",
		"submission": {
		"marketID": "Market1",
		"partyID": "trader1",
		"price": 100,
		"size": 3,
		"side": "Sell",
		"TimeInForce": "GTC",
		"expiresAt": 1924991999000000000
		}
	}
	}
	]
	}`)

	actual, err := unmarshall(data)

	assert.NoError(t, err)
	assert.EqualValues(t, expected, actual)
}

func TestUnmarshallInternalTypes(t *testing.T) {

	instr1, err := core.NewInstruction(
		"NotifyTraderAccount",
		&types.NotifyTraderAccount{
			TraderID: "trader1",
		})

	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
	}
	instr2, err := core.NewInstruction(
		"SubmitOrder",
		&types.Order{
			MarketID:    "Market1",
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
	expected := &core.InstructionSet{
		Instructions: []*core.Instruction{
			instr1,
			instr2,
		},
		Description: "Test instructions",
	}
	data := strings.NewReader(`{
		"description": "Test instructions",
		"instructions": [
		  {
			"description": "",
			"request": "NotifyTraderAccount",
			"message": {
			  "@type": "vega.NotifyTraderAccount",
			  "traderID": "trader1"
			}
		  },
		  {
			"description": "Submit a sell order",
			"request": "SubmitOrder",
			"message": {
			  "@type": "vega.Order",
			  "marketID": "Market1",
			  "partyID": "trader1",
			  "side": "Sell",
			  "price": "100",
			  "size": "3",
			  "timeInForce": "GTC",
			  "expiresAt": "1924991999000000000"
			}
		  }
		]
	  }`)

	actual, err := unmarshall(data)

	assert.NoError(t, err)
	assert.EqualValues(t, expected, actual)
}

func TestMarshal(t *testing.T) {
	expected := string(
		`{
		"summary": {
		  "instructionsProcessed": "2",
		  "instructionsOmitted": "0",
		  "tradesGenerated": "1",
		  "processingTime": "3s",
		  "finalOrderBook": [
			{
			  "marketID": "Market1",
			  "buy": [
				{
				  "price": "100",
				  "numberOfOrders": "1",
				  "volume": "3",
				  "cumulativeVolume": "3"
				}
			  ],
			  "sell": [
				{
				  "price": "100",
				  "numberOfOrders": "1",
				  "volume": "3",
				  "cumulativeVolume": "3"
				}
			  ]
			}
		  ]
		},
		"results": [
		  {
			"instruction": {
			  "description": "",
			  "request": "trading.SubmitOrder",
			  "message": {
				"@type": "api.SubmitOrderRequest",
				"submission": {
				  "id": "",
				  "marketID": "Market1",
				  "partyID": "trader1",
				  "price": "100",
				  "size": "3",
				  "side": "Sell",
				  "TimeInForce": "GTC",
				  "expiresAt": "1924991999000000000",
				  "type": "LIMIT"
				},
				"token": ""
			  }
			},
			"error": "",
			"response": {
			  "@type": "vega.PendingOrder",
			  "reference": "",
			  "price": "100",
			  "TimeInForce": "GTC",
			  "side": "Sell",
			  "marketID": "Market1",
			  "size": "3",
			  "partyID": "trader1",
			  "status": "Active",
			  "id": "order1",
			  "type": "LIMIT"
			}
		  },
		  {
			"instruction": {
			  "description": "",
			  "request": "trading.SubmitOrder",
			  "message": {
				"@type": "api.SubmitOrderRequest",
				"submission": {
				  "id": "",
				  "marketID": "Market1",
				  "partyID": "trader2",
				  "price": "100",
				  "size": "3",
				  "side": "Buy",
				  "TimeInForce": "GTC",
				  "expiresAt": "1924991999000000000",
				  "type": "LIMIT"
				},
				"token": ""
			  }
			},
			"error": "",
			"response": {
			  "@type": "vega.PendingOrder",
			  "reference": "",
			  "price": "100",
			  "TimeInForce": "GTC",
			  "side": "Buy",
			  "marketID": "Market1",
			  "size": "3",
			  "partyID": "trader2",
			  "status": "Active",
			  "id": "order2",
			  "type": "LIMIT"
			}
		  }
		]
	  }`)

	instr1, err := core.NewInstruction(
		"trading.SubmitOrder",
		&api.SubmitOrderRequest{
			Submission: &types.OrderSubmission{
				MarketID:    "Market1",
				PartyID:     "trader1",
				Price:       100,
				Size:        3,
				Side:        types.Side_Sell,
				TimeInForce: types.Order_GTC,
				ExpiresAt:   1924991999000000000,
			},
		})
	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
	}

	instr2, err := core.NewInstruction(
		"trading.SubmitOrder",
		&api.SubmitOrderRequest{
			Submission: &types.OrderSubmission{
				MarketID:    "Market1",
				PartyID:     "trader2",
				Price:       100,
				Size:        3,
				Side:        types.Side_Buy,
				TimeInForce: types.Order_GTC,
				ExpiresAt:   1924991999000000000,
			},
		})
	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
	}

	resp1 := types.PendingOrder{
		Price:       100,
		TimeInForce: types.Order_GTC,
		Side:        types.Side_Sell,
		MarketID:    "Market1",
		Size:        3,
		PartyID:     "trader1",
		Status:      types.Order_Active,
		Id:          "order1",
		Type:        types.Order_LIMIT,
	}

	resp2 := types.PendingOrder{
		Price:       100,
		TimeInForce: types.Order_GTC,
		Side:        types.Side_Buy,
		MarketID:    "Market1",
		Size:        3,
		PartyID:     "trader2",
		Status:      types.Order_Active,
		Id:          "order2",
		Type:        types.Order_LIMIT,
	}

	result1, err := instr1.NewResult(&resp1, nil)
	if err != nil {
		log.Fatalln("Failed to create a new instruction result: ", err)
	}
	result2, err := instr2.NewResult(&resp2, nil)
	if err != nil {
		log.Fatalln("Failed to create a new instruction result: ", err)
	}

	resultSet := core.ResultSet{
		Summary: &core.Metadata{
			InstructionsProcessed: 2,
			InstructionsOmitted:   0,
			TradesGenerated:       1,
			ProcessingTime: &duration.Duration{
				Seconds: 3,
			},
			FinalOrderBook: []*types.MarketDepth{
				{
					MarketID: "Market1",
					Buy: []*types.PriceLevel{
						&types.PriceLevel{
							Price:            100,
							NumberOfOrders:   1,
							Volume:           3,
							CumulativeVolume: 3,
						},
					},
					Sell: []*types.PriceLevel{
						&types.PriceLevel{
							Price:            100,
							NumberOfOrders:   1,
							Volume:           3,
							CumulativeVolume: 3,
						},
					},
				},
			},
		},
		Results: []*core.InstructionResult{
			result1,
			result2,
		},
	}
	out := strings.Builder{}
	err = marshall(&resultSet, &out)
	assert.NoError(t, err)

	actual := out.String()
	equal, err := areEqualJSON(expected, actual)
	assert.NoError(t, err)
	assert.True(t, equal)
}

func areEqualJSON(s1, s2 string) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 2 :: %s", err.Error())
	}

	return reflect.DeepEqual(o1, o2), nil
}
