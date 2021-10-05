package events_test

import (
	"context"
	"testing"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/stretchr/testify/assert"
)

const (
	nodeID        = "vega-master-public-key"
	vegaPublicKey = "vega-public-key"
	ethAddress    = "eth-address"
	tmPublicKey   = "tm-public-key"
	infoURL       = "no1.xyz.vega/nodes/a"
	country       = "GB"
	name          = "Validator"
	avatarURL     = "https://not-an-avatar.com"
)

func TestValidatorUpdate(t *testing.T) {
	t.Run("returns public key", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, ethAddress, tmPublicKey, infoURL, country, name, avatarURL)

		assert.Equal(t, vegaPublicKey, vu.VegaPublicKey())
	})

	t.Run("returns Tendermint public key", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, ethAddress, tmPublicKey, infoURL, country, name, avatarURL)

		assert.Equal(t, tmPublicKey, vu.TendermintPublicKey())
	})

	t.Run("returns info url", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, ethAddress, tmPublicKey, infoURL, country, name, avatarURL)

		assert.Equal(t, infoURL, vu.InfoURL())
	})

	t.Run("returns country", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, ethAddress, tmPublicKey, infoURL, country, name, avatarURL)

		assert.Equal(t, country, vu.Country())
	})

	t.Run("returns validator update event proto", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, ethAddress, tmPublicKey, infoURL, country, name, avatarURL)

		expected := eventspb.ValidatorUpdate{
			NodeId:          nodeID,
			VegaPubKey:      vegaPublicKey,
			EthereumAddress: ethAddress,
			TmPubKey:        tmPublicKey,
			InfoUrl:         infoURL,
			Country:         country,
			Name:            name,
			AvatarUrl:       avatarURL,
		}

		assert.Equal(t, expected, vu.Proto())
	})

	t.Run("returns stream message with validator update", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, ethAddress, tmPublicKey, infoURL, country, name, avatarURL)

		vuProto := eventspb.ValidatorUpdate{
			NodeId:          nodeID,
			VegaPubKey:      vegaPublicKey,
			EthereumAddress: ethAddress,
			TmPubKey:        tmPublicKey,
			InfoUrl:         infoURL,
			Country:         country,
			Name:            name,
			AvatarUrl:       avatarURL,
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
