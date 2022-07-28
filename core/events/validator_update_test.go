// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package events_test

import (
	"context"
	"testing"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/core/events"
	"github.com/stretchr/testify/assert"
)

const (
	nodeID             = "vega-master-public-key"
	vegaPublicKey      = "vega-public-key"
	vegaPublicKeyIndex = 1
	ethAddress         = "eth-address"
	tmPublicKey        = "tm-public-key"
	infoURL            = "no1.xyz.vega/nodes/a"
	country            = "GB"
	name               = "Validator"
	avatarURL          = "https://not-an-avatar.com"
	fromEpoch          = 1
)

func TestValidatorUpdate(t *testing.T) {
	t.Run("returns public key", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, vegaPublicKeyIndex, ethAddress, tmPublicKey, infoURL, country, name, avatarURL, fromEpoch, true)

		assert.Equal(t, vegaPublicKey, vu.VegaPublicKey())
	})

	t.Run("returns Tendermint public key", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, vegaPublicKeyIndex, ethAddress, tmPublicKey, infoURL, country, name, avatarURL, fromEpoch, true)

		assert.Equal(t, tmPublicKey, vu.TendermintPublicKey())
	})

	t.Run("returns info url", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, vegaPublicKeyIndex, ethAddress, tmPublicKey, infoURL, country, name, avatarURL, fromEpoch, true)

		assert.Equal(t, infoURL, vu.InfoURL())
	})

	t.Run("returns country", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, vegaPublicKeyIndex, ethAddress, tmPublicKey, infoURL, country, name, avatarURL, fromEpoch, true)

		assert.Equal(t, country, vu.Country())
	})

	t.Run("returns validator update event proto", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, vegaPublicKeyIndex, ethAddress, tmPublicKey, infoURL, country, name, avatarURL, fromEpoch, true)

		expected := eventspb.ValidatorUpdate{
			NodeId:          nodeID,
			VegaPubKey:      vegaPublicKey,
			VegaPubKeyIndex: vegaPublicKeyIndex,
			EthereumAddress: ethAddress,
			TmPubKey:        tmPublicKey,
			InfoUrl:         infoURL,
			Country:         country,
			Name:            name,
			AvatarUrl:       avatarURL,
			FromEpoch:       fromEpoch,
			Added:           true,
		}

		assert.Equal(t, expected, vu.Proto())
	})

	t.Run("returns stream message with validator update", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, vegaPublicKeyIndex, ethAddress, tmPublicKey, infoURL, country, name, avatarURL, fromEpoch, true)

		vuProto := eventspb.ValidatorUpdate{
			NodeId:          nodeID,
			VegaPubKey:      vegaPublicKey,
			VegaPubKeyIndex: vegaPublicKeyIndex,
			EthereumAddress: ethAddress,
			TmPubKey:        tmPublicKey,
			InfoUrl:         infoURL,
			Country:         country,
			Name:            name,
			AvatarUrl:       avatarURL,
			FromEpoch:       fromEpoch,
			Added:           true,
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
