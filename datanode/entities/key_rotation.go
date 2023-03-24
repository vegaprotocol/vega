// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
