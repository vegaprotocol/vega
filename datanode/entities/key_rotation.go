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

package entities

import (
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type KeyRotation struct {
	NodeID      NodeID
	OldPubKey   VegaPublicKey
	NewPubKey   VegaPublicKey
	BlockHeight uint64
	TxHash      TxHash
	VegaTime    time.Time
}

func KeyRotationFromProto(kr *eventspb.KeyRotation, txHash TxHash, vegaTime time.Time) (*KeyRotation, error) {
	return &KeyRotation{
		NodeID:      NodeID(kr.NodeId),
		OldPubKey:   VegaPublicKey(kr.OldPubKey),
		NewPubKey:   VegaPublicKey(kr.NewPubKey),
		BlockHeight: kr.BlockHeight,
		TxHash:      txHash,
		VegaTime:    vegaTime,
	}, nil
}

func (kr KeyRotation) ToProto() *eventspb.KeyRotation {
	return &eventspb.KeyRotation{
		NodeId:      kr.NodeID.String(),
		OldPubKey:   kr.OldPubKey.String(),
		NewPubKey:   kr.NewPubKey.String(),
		BlockHeight: kr.BlockHeight,
	}
}

func (kr KeyRotation) Cursor() *Cursor {
	cursor := KeyRotationCursor{
		VegaTime:  kr.VegaTime,
		NodeID:    kr.NodeID,
		OldPubKey: kr.OldPubKey,
		NewPubKey: kr.NewPubKey,
	}

	return NewCursor(cursor.String())
}

func (kr KeyRotation) ToProtoEdge(_ ...any) (*v2.KeyRotationEdge, error) {
	return &v2.KeyRotationEdge{
		Node:   kr.ToProto(),
		Cursor: kr.Cursor().Encode(),
	}, nil
}

type KeyRotationCursor struct {
	VegaTime  time.Time     `json:"vega_time"`
	NodeID    NodeID        `json:"node_id"`
	OldPubKey VegaPublicKey `json:"old_pub_key"`
	NewPubKey VegaPublicKey `json:"new_pub_key"`
}

func (c KeyRotationCursor) String() string {
	bs, err := json.Marshal(c)
	// This should never fail so if it does, we should panic
	if err != nil {
		panic(fmt.Errorf("could not marshal key rotation cursor: %w", err))
	}

	return string(bs)
}

func (c *KeyRotationCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), c)
}
