package entities

import (
	"time"

	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type KeyRotation struct {
	NodeID      NodeID
	OldPubKey   VegaPublicKey
	NewPubKey   VegaPublicKey
	BlockHeight uint64
	VegaTime    time.Time
}

func KeyRotationFromProto(kr *eventspb.KeyRotation, vegaTime time.Time) (*KeyRotation, error) {
	return &KeyRotation{
		NodeID:      NewNodeID(kr.NodeId),
		OldPubKey:   VegaPublicKey(kr.OldPubKey),
		NewPubKey:   VegaPublicKey(kr.NewPubKey),
		BlockHeight: kr.BlockHeight,
		VegaTime:    vegaTime,
	}, nil
}

func (kr *KeyRotation) ToProto() *protoapi.KeyRotation {
	return &protoapi.KeyRotation{
		NodeId:      kr.NodeID.String(),
		OldPubKey:   kr.OldPubKey.String(),
		NewPubKey:   kr.NewPubKey.String(),
		BlockHeight: kr.BlockHeight,
	}
}
