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

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

// ValidatorUpdate ...
type ValidatorUpdate struct {
	*Base
	nodeID          string
	vegaPubKey      string
	vegaPubKeyIndex uint32
	ethAddress      string
	tmPubKey        string
	infoURL         string
	country         string
	name            string
	avatarURL       string
	fromEpoch       uint64
	added           bool
	epochSeq        uint64
}

func NewValidatorUpdateEvent(
	ctx context.Context,
	nodeID string,
	vegaPubKey string,
	vegaPubKeyIndex uint32,
	ethAddress string,
	tmPubKey string,
	infoURL string,
	country string,
	name string,
	avatarURL string,
	fromEpoch uint64,
	added bool,
	epochSeq uint64,
) *ValidatorUpdate {
	return &ValidatorUpdate{
		Base:            newBase(ctx, ValidatorUpdateEvent),
		nodeID:          nodeID,
		vegaPubKey:      vegaPubKey,
		vegaPubKeyIndex: vegaPubKeyIndex,
		ethAddress:      ethAddress,
		tmPubKey:        tmPubKey,
		infoURL:         infoURL,
		country:         country,
		name:            name,
		avatarURL:       avatarURL,
		fromEpoch:       fromEpoch,
		added:           added,
		epochSeq:        epochSeq,
	}
}

// NodeID returns nodes ID.
func (vu ValidatorUpdate) NodeID() string {
	return vu.nodeID
}

// VegaPublicKey returns validator's vega public key.
func (vu ValidatorUpdate) VegaPublicKey() string {
	return vu.vegaPubKey
}

// VegaPublicKey returns validator's vega public key index.
func (vu ValidatorUpdate) VegaPublicKeyIndex() uint32 {
	return vu.vegaPubKeyIndex
}

// EthereumAddress returns validator's ethereum address.
func (vu ValidatorUpdate) EthereumAddress() string {
	return vu.ethAddress
}

// TendermintPublicKey returns Tendermint nodes public key.
func (vu ValidatorUpdate) TendermintPublicKey() string {
	return vu.tmPubKey
}

// InfoURL returns an url with information about validator node.
func (vu ValidatorUpdate) InfoURL() string {
	return vu.infoURL
}

// Country returns country code of node's location.
func (vu ValidatorUpdate) Country() string {
	return vu.country
}

// Name return the name of the validator.
func (vu ValidatorUpdate) Name() string {
	return vu.name
}

// AvatarURL return an URL to the validator avatar for UI purpose.
func (vu ValidatorUpdate) AvatarURL() string {
	return vu.avatarURL
}

func (vu ValidatorUpdate) ValidatorUpdate() eventspb.ValidatorUpdate {
	return vu.Proto()
}

func (vu ValidatorUpdate) Proto() eventspb.ValidatorUpdate {
	return eventspb.ValidatorUpdate{
		NodeId:          vu.nodeID,
		VegaPubKey:      vu.vegaPubKey,
		VegaPubKeyIndex: vu.vegaPubKeyIndex,
		EthereumAddress: vu.ethAddress,
		TmPubKey:        vu.tmPubKey,
		InfoUrl:         vu.infoURL,
		Country:         vu.country,
		Name:            vu.name,
		AvatarUrl:       vu.avatarURL,
		FromEpoch:       vu.fromEpoch,
		Added:           vu.added,
		EpochSeq:        vu.epochSeq,
	}
}

func (vu ValidatorUpdate) StreamMessage() *eventspb.BusEvent {
	vuproto := vu.Proto()

	busEvent := newBusEventFromBase(vu.Base)
	busEvent.Event = &eventspb.BusEvent_ValidatorUpdate{
		ValidatorUpdate: &vuproto,
	}

	return busEvent
}

func ValidatorUpdateEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ValidatorUpdate {
	event := be.GetValidatorUpdate()
	if event == nil {
		return nil
	}

	return &ValidatorUpdate{
		Base:            newBaseFromBusEvent(ctx, ValidatorUpdateEvent, be),
		nodeID:          event.GetNodeId(),
		vegaPubKey:      event.GetVegaPubKey(),
		vegaPubKeyIndex: event.GetVegaPubKeyIndex(),
		ethAddress:      event.GetEthereumAddress(),
		tmPubKey:        event.GetTmPubKey(),
		infoURL:         event.GetInfoUrl(),
		country:         event.GetCountry(),
		name:            event.GetName(),
		avatarURL:       event.GetAvatarUrl(),
		fromEpoch:       event.FromEpoch,
		added:           event.Added,
		epochSeq:        event.EpochSeq,
	}
}
