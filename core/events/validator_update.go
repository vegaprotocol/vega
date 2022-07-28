// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
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
	}
}
