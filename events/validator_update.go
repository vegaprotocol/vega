package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

// ValidatorUpdate ...
type ValidatorUpdate struct {
	*Base
	vegaPubKey string
	ethAddress string
	tmPubKey   string
	infoURL    string
	country    string
}

func NewValidatorUpdateEvent(
	ctx context.Context,
	vegaPubKey string,
	ethAddress string,
	tmPubKey string,
	infoURL string,
	country string,
) *ValidatorUpdate {
	return &ValidatorUpdate{
		Base:       newBase(ctx, ValidatorUpdateEvent),
		vegaPubKey: vegaPubKey,
		ethAddress: ethAddress,
		tmPubKey:   tmPubKey,
		infoURL:    infoURL,
		country:    country,
	}
}

// VegaPublicKey returns validator's vega public key
func (vu ValidatorUpdate) VegaPublicKey() string {
	return vu.vegaPubKey
}

// EthereumAddress returns validator's ethereum address
func (vu ValidatorUpdate) EthereumAddress() string {
	return vu.ethAddress
}

// TendermintPublicKey returns Tendermint nodes public key
func (vu ValidatorUpdate) TendermintPublicKey() string {
	return vu.tmPubKey
}

// InfoURL returns an url with information about validator node
func (vu ValidatorUpdate) InfoURL() string {
	return vu.infoURL
}

// Country returns country code of node's location
func (vu ValidatorUpdate) Country() string {
	return vu.country
}

func (vu ValidatorUpdate) Proto() eventspb.ValidatorUpdate {
	return eventspb.ValidatorUpdate{
		VegaPubKey:      vu.vegaPubKey,
		EthereumAddress: vu.ethAddress,
		TmPubKey:        vu.tmPubKey,
		InfoUrl:         vu.infoURL,
		Country:         vu.country,
	}
}

func (vu ValidatorUpdate) StreamMessage() *eventspb.BusEvent {
	vuproto := vu.Proto()

	return &eventspb.BusEvent{
		Id:    vu.eventID(),
		Block: vu.TraceID(),
		Type:  vu.et.ToProto(),
		Event: &eventspb.BusEvent_ValidatorUpdate{
			ValidatorUpdate: &vuproto,
		},
	}
}

func ValidatorUpdateEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ValidatorUpdate {
	event := be.GetValidatorUpdate()
	if event == nil {
		return nil
	}

	return &ValidatorUpdate{
		Base:       newBaseFromStream(ctx, ValidatorUpdateEvent, be),
		vegaPubKey: event.GetVegaPubKey(),
		ethAddress: event.GetEthereumAddress(),
		tmPubKey:   event.GetTmPubKey(),
		infoURL:    event.GetInfoUrl(),
		country:    event.GetCountry(),
	}
}
