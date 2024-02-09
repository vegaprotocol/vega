// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package events_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

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
	fromEpoch          = 5
	epochSeq           = 2
)

func TestValidatorUpdate(t *testing.T) {
	t.Run("returns public key", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, vegaPublicKeyIndex, ethAddress, tmPublicKey, infoURL, country, name, avatarURL, fromEpoch, true, epochSeq)

		assert.Equal(t, vegaPublicKey, vu.VegaPublicKey())
	})

	t.Run("returns Tendermint public key", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, vegaPublicKeyIndex, ethAddress, tmPublicKey, infoURL, country, name, avatarURL, fromEpoch, true, epochSeq)

		assert.Equal(t, tmPublicKey, vu.TendermintPublicKey())
	})

	t.Run("returns info url", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, vegaPublicKeyIndex, ethAddress, tmPublicKey, infoURL, country, name, avatarURL, fromEpoch, true, epochSeq)

		assert.Equal(t, infoURL, vu.InfoURL())
	})

	t.Run("returns country", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, vegaPublicKeyIndex, ethAddress, tmPublicKey, infoURL, country, name, avatarURL, fromEpoch, true, epochSeq)

		assert.Equal(t, country, vu.Country())
	})

	t.Run("returns validator update event proto", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, vegaPublicKeyIndex, ethAddress, tmPublicKey, infoURL, country, name, avatarURL, fromEpoch, true, epochSeq)

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
			EpochSeq:        epochSeq,
		}

		assert.Equal(t, expected, vu.Proto())
	})

	t.Run("returns stream message with validator update", func(t *testing.T) {
		ctx := context.Background()
		vu := events.NewValidatorUpdateEvent(ctx, nodeID, vegaPublicKey, vegaPublicKeyIndex, ethAddress, tmPublicKey, infoURL, country, name, avatarURL, fromEpoch, true, epochSeq)

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
			EpochSeq:        epochSeq,
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
