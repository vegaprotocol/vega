package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

// ValidatorUpdate ...
type ValidatorUpdate struct {
	*Base
	pubKey   string
	tmPubKey string
	infoURL  string
	country  string
}

func NewValidatorUpdateEvent(
	ctx context.Context,
	pubKey string,
	tmPubKey string,
	infoURL string,
	country string,
) *ValidatorUpdate {
	return &ValidatorUpdate{
		Base:     newBase(ctx, ValidatorUpdateEvent),
		pubKey:   pubKey,
		tmPubKey: tmPubKey,
		infoURL:  infoURL,
		country:  country,
	}
}

// PublicKey returns validator's public key
func (vu ValidatorUpdate) PublicKey() string {
	return vu.pubKey
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
		PubKey:   vu.pubKey,
		TmPubKey: vu.tmPubKey,
		InfoUrl:  vu.infoURL,
		Country:  vu.country,
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
		Base:     newBaseFromStream(ctx, ValidatorUpdateEvent, be),
		pubKey:   event.GetPubKey(),
		tmPubKey: event.GetTmPubKey(),
		infoURL:  event.GetInfoUrl(),
		country:  event.GetCountry(),
	}
}
