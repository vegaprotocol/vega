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
