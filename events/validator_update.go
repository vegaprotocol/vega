package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

// ValidatorUpdate ...
type ValidatorUpdate struct {
	*Base
	nodeID           string
	vegaPubKey       string
	vegaPubKeyNumber uint32
	ethAddress       string
	tmPubKey         string
	infoURL          string
	country          string
	name             string
	avatarURL        string
}

func NewValidatorUpdateEvent(
	ctx context.Context,
	nodeID string,
	vegaPubKey string,
	vegaPubKeyNumber uint32,
	ethAddress string,
	tmPubKey string,
	infoURL string,
	country string,
	name string,
	avatarURL string,
) *ValidatorUpdate {
	return &ValidatorUpdate{
		Base:             newBase(ctx, ValidatorUpdateEvent),
		nodeID:           nodeID,
		vegaPubKey:       vegaPubKey,
		vegaPubKeyNumber: vegaPubKeyNumber,
		ethAddress:       ethAddress,
		tmPubKey:         tmPubKey,
		infoURL:          infoURL,
		country:          country,
		name:             name,
		avatarURL:        avatarURL,
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

// VegaPublicKey returns validator's vega public key number.
func (vu ValidatorUpdate) VegaPublicKeyNumber() uint32 {
	return vu.vegaPubKeyNumber
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
		NodeId:           vu.nodeID,
		VegaPubKey:       vu.vegaPubKey,
		VegaPubKeyNumber: vu.vegaPubKeyNumber,
		EthereumAddress:  vu.ethAddress,
		TmPubKey:         vu.tmPubKey,
		InfoUrl:          vu.infoURL,
		Country:          vu.country,
		Name:             vu.name,
		AvatarUrl:        vu.avatarURL,
	}
}

func (vu ValidatorUpdate) StreamMessage() *eventspb.BusEvent {
	vuproto := vu.Proto()

	return &eventspb.BusEvent{
		Version: eventspb.Version,
		Id:      vu.eventID(),
		Block:   vu.TraceID(),
		ChainId: vu.ChainID(),
		Type:    vu.et.ToProto(),
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
		Base:             newBaseFromStream(ctx, ValidatorUpdateEvent, be),
		nodeID:           event.GetNodeId(),
		vegaPubKey:       event.GetVegaPubKey(),
		vegaPubKeyNumber: event.GetVegaPubKeyNumber(),
		ethAddress:       event.GetEthereumAddress(),
		tmPubKey:         event.GetTmPubKey(),
		infoURL:          event.GetInfoUrl(),
		country:          event.GetCountry(),
		name:             event.GetName(),
		avatarURL:        event.GetAvatarUrl(),
	}
}
