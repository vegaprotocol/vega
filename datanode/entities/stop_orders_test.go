package entities_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"
	commandpb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	pbevents "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStopOrderFromProto(t *testing.T) {
	t.Run("Should set ExpiresAt if ExpiryStrategy is set", func(t *testing.T) {
		createdAt := time.Now().Round(time.Microsecond)
		expiresAt := createdAt.Add(time.Minute)
		sop := vega.StopOrder{
			Id:               "deadbeef",
			ExpiresAt:        ptr.From(expiresAt.UnixNano()),
			ExpiryStrategy:   ptr.From(vega.StopOrder_EXPIRY_STRATEGY_CANCELS),
			TriggerDirection: vega.StopOrder_TRIGGER_DIRECTION_RISES_ABOVE,
			Status:           vega.StopOrder_STATUS_PENDING,
			CreatedAt:        createdAt.UnixNano(),
			UpdatedAt:        nil,
			OrderId:          "deadbeef",
			PartyId:          "deadbaad",
			MarketId:         "dead2bad",
			Trigger: &vega.StopOrder_Price{
				Price: "100",
			},
		}
		sub := commandpb.OrderSubmission{
			MarketId:    "dead2bad",
			Price:       "100",
			Size:        100,
			Side:        vega.Side_SIDE_BUY,
			TimeInForce: vega.Order_TIME_IN_FORCE_GTC,
			Reference:   "some-reference",
		}
		soEvent := pbevents.StopOrderEvent{
			Submission: &sub,
			StopOrder:  &sop,
		}

		vegaTime := time.Now().Round(time.Microsecond)
		seqNum := uint64(0)
		txHash := entities.TxHash(`deadbaad`)

		got, err := entities.StopOrderFromProto(&soEvent, vegaTime, seqNum, txHash)
		require.NoError(t, err)

		want := entities.StopOrder{
			ID:                   entities.StopOrderID("deadbeef"),
			ExpiresAt:            ptr.From(expiresAt),
			ExpiryStrategy:       entities.StopOrderExpiryStrategyCancels,
			TriggerDirection:     entities.StopOrderTriggerDirectionRisesAbove,
			Status:               entities.StopOrderStatusPending,
			CreatedAt:            createdAt,
			UpdatedAt:            nil,
			OrderID:              "deadbeef",
			TriggerPrice:         ptr.From("100"),
			TriggerPercentOffset: nil,
			PartyID:              "deadbaad",
			MarketID:             "dead2bad",
			VegaTime:             vegaTime,
			SeqNum:               0,
			TxHash:               entities.TxHash(`deadbaad`),
			Submission:           &sub,
		}

		assert.Equal(t, want, got)
	})
}
