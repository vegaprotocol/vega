package events_test

import (
	"context"
	"testing"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/stretchr/testify/assert"
)

const (
	nodeID        = "vega-public-key"
	vegaPublicKey = "vega-public-key"
	ethAddress    = "eth-address"
	tmPublicKey   = "tm-public-key"
	infoURL       = "no1.xyz.vega/nodes/a"
	country       = "GB"
)

func TestValidatorUpdate(t *testing.T) {
	t.Run("returns public key", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, ethAddress, tmPublicKey, infoURL, country)

		assert.Equal(t, vegaPublicKey, vu.VegaPublicKey())
	})

	t.Run("returns Tendermint public key", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, ethAddress, tmPublicKey, infoURL, country)

		assert.Equal(t, tmPublicKey, vu.TendermintPublicKey())
	})

	t.Run("returns info url", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, ethAddress, tmPublicKey, infoURL, country)

		assert.Equal(t, infoURL, vu.InfoURL())
	})

	t.Run("returns country", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, ethAddress, tmPublicKey, infoURL, country)

		assert.Equal(t, country, vu.Country())
	})

	t.Run("returns validator update event proto", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, ethAddress, tmPublicKey, infoURL, country)

		expected := eventspb.ValidatorUpdate{
			NodeId:          nodeID,
			VegaPubKey:      vegaPublicKey,
			EthereumAddress: ethAddress,
			TmPubKey:        tmPublicKey,
			InfoUrl:         infoURL,
			Country:         country,
		}

		assert.Equal(t, expected, vu.Proto())
	})

	t.Run("returns stream message with validator update", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, ethAddress, tmPublicKey, infoURL, country)

		vuProto := eventspb.ValidatorUpdate{
			NodeId:          nodeID,
			VegaPubKey:      vegaPublicKey,
			EthereumAddress: ethAddress,
			TmPubKey:        tmPublicKey,
			InfoUrl:         infoURL,
			Country:         country,
		}

		expectedUpdate := &eventspb.BusEvent_ValidatorUpdate{
			ValidatorUpdate: &vuProto,
		}

		expectedType := eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_UPDATE

		sm := vu.StreamMessage()

		assert.Equal(t, expectedUpdate, sm.Event)
		assert.Equal(t, expectedType, sm.Type)
	})
}
