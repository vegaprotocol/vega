package events_test

import (
	"context"
	"testing"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/stretchr/testify/assert"
)

const (
	publicKey   = "public-key"
	tmPublicKey = "tm-public-key"
	infoURL     = "no1.xyz.vega/nodes/a"
	country     = "GB"
)

func TestValidatorUpdate(t *testing.T) {
	t.Run("returns public key", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, publicKey, tmPublicKey, infoURL, country)

		assert.Equal(t, publicKey, vu.PublicKey())
	})

	t.Run("returns Tendermint public key", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, publicKey, tmPublicKey, infoURL, country)

		assert.Equal(t, tmPublicKey, vu.TendermintPublicKey())
	})

	t.Run("returns info url", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, publicKey, tmPublicKey, infoURL, country)

		assert.Equal(t, infoURL, vu.InfoURL())
	})

	t.Run("returns country", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, publicKey, tmPublicKey, infoURL, country)

		assert.Equal(t, country, vu.Country())
	})

	t.Run("returns validator update event proto", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, publicKey, tmPublicKey, infoURL, country)

		expected := eventspb.ValidatorUpdate{
			PubKey:   publicKey,
			TmPubKey: tmPublicKey,
			InfoUrl:  infoURL,
			Country:  country,
		}

		assert.Equal(t, expected, vu.Proto())
	})

	t.Run("returns stream message with validator update", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, publicKey, tmPublicKey, infoURL, country)

		vuProto := eventspb.ValidatorUpdate{
			PubKey:   publicKey,
			TmPubKey: tmPublicKey,
			InfoUrl:  infoURL,
			Country:  country,
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
