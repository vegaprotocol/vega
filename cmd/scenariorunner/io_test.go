package main

import (
	"log"
	"strings"
	"testing"
	"time"

	sr "code.vegaprotocol.io/vega/internal/scenariorunner"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/proto/api"

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

func TestUnmarshall(t *testing.T) {

	instr1, err := sr.NewInstruction(
		"trading.NotifyTraderAccount",
		&api.NotifyTraderAccountRequest{
			Notif: &types.NotifyTraderAccount{
				TraderID: "trader1",
			},
		})

	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
	}
	instr2, err := sr.NewInstruction(
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
	expected := sr.InstructionSet{
		Instructions: []*sr.Instruction{
			instr1,
			instr2,
		},
		Description: "Test instructions",
	}
	data := strings.NewReader(`{
	"Description": "Test instructions",
	"Instructions": [
	{
	"Request": "trading.NotifyTraderAccount",
	"Message": {
		"@type": "api.NotifyTraderAccountRequest",
		"notif": {
		"traderID": "trader1"
		}
	}
	},
	{
	"Description": "Submit a sell order",
	"Request": "trading.SubmitOrder",
	"Message": {
		"@type": "api.SubmitOrderRequest",
		"submission": {
		"marketID": "Market1",
		"partyID": "trader1",
		"price": "100",
		"size": "3",
		"side": "Sell",
		"expiresAt": "1924991999000000000"
		}
	}
	}
	]
	}`)

	actual, err := unmarshall(data)

	assert.NoError(t, err)
	assert.ObjectsAreEqualValues(expected, actual)
}

func TestMarshal(t *testing.T) {
	expected := string(`{
  "Summary": {
    "InstructionsProcessed": "2",
    "TradesGenerated": "1",
    "ProcessingTime": "3s",
    "FinalOrderBook": {
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
  },
  "Results": [
    {
      "Response": {
        "@type": "vega.PendingOrder",
        "price": "100",
        "side": "Sell",
        "marketID": "Market1",
        "partyID": "trader1",
        "id": "order1"
      },
      "Instruction": {
        "Request": "trading.SubmitOrder",
        "Message": {
          "@type": "api.SubmitOrderRequest",
          "submission": {
            "marketID": "Market1",
            "partyID": "trader1",
            "price": "100",
            "size": "3",
            "side": "Sell",
            "expiresAt": "1924991999000000000"
          }
        }
      }
    },
    {
      "Response": {
        "@type": "vega.PendingOrder",
        "price": "100",
        "marketID": "Market1",
        "partyID": "trader2",
        "id": "order2"
      },
      "Instruction": {
        "Request": "trading.SubmitOrder",
        "Message": {
          "@type": "api.SubmitOrderRequest",
          "submission": {
            "marketID": "Market1",
            "partyID": "trader2",
            "price": "100",
            "size": "3",
            "expiresAt": "1924991999000000000"
          }
        }
      }
    }
  ]
}`)

	instr1, err := sr.NewInstruction(
		"trading.SubmitOrder",
		&api.SubmitOrderRequest{
			Submission: &types.OrderSubmission{
				MarketID:    "Market1",
				PartyID:     "trader1",
				Price:       100,
				Size:        3,
				Side:        types.Side_Sell,
				TimeInForce: types.Order_GTC,
				ExpiresAt:   time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC).UnixNano(),
			},
		})
	if err != nil {
		log.Fatalln("Failed to create a new instruction: ", err)
	}

	instr2, err := sr.NewInstruction(
		"trading.SubmitOrder",
		&api.SubmitOrderRequest{
			Submission: &types.OrderSubmission{
				MarketID:    "Market1",
				PartyID:     "trader2",
				Price:       100,
				Size:        3,
				Side:        types.Side_Buy,
				TimeInForce: types.Order_GTC,
				ExpiresAt:   time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC).UnixNano(),
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

	resultSet := sr.ResultSet{
		Summary: &sr.Metadata{
			InstructionsProcessed: 2,
			InstructionsOmitted:   0,
			TradesGenerated:       1,
			ProcessingTime: &duration.Duration{
				Seconds: 3,
			},
			FinalOrderBook: &types.MarketDepth{
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
		Results: []*sr.InstructionResult{
			result1,
			result2,
		},
	}
	out := strings.Builder{}
	err = marshall(&resultSet, &out)
	assert.NoError(t, err)

	actual := out.String()
	assert.EqualValues(t, expected, actual)
}
