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

type EthereumKeyRotation struct {
	NodeID      NodeID
	OldAddress  EthereumAddress
	NewAddress  EthereumAddress
	BlockHeight uint64
	TxHash      TxHash
	VegaTime    time.Time
	SeqNum      uint64
}

func EthereumKeyRotationFromProto(kr *eventspb.EthereumKeyRotation, txHash TxHash, vegaTime time.Time,
	seqNum uint64,
) (EthereumKeyRotation, error) {
	return EthereumKeyRotation{
		NodeID:      NodeID(kr.NodeId),
		OldAddress:  EthereumAddress(kr.OldAddress),
		NewAddress:  EthereumAddress(kr.NewAddress),
		BlockHeight: kr.BlockHeight,
		TxHash:      txHash,
		VegaTime:    vegaTime,
		SeqNum:      seqNum,
	}, nil
}

func (kr EthereumKeyRotation) ToProto() *eventspb.EthereumKeyRotation {
	return &eventspb.EthereumKeyRotation{
		NodeId:      kr.NodeID.String(),
		OldAddress:  kr.OldAddress.String(),
		NewAddress:  kr.NewAddress.String(),
		BlockHeight: kr.BlockHeight,
	}
}

func (kr EthereumKeyRotation) Cursor() *Cursor {
	cursor := EthereumKeyRotationCursor{
		VegaTime:   kr.VegaTime,
		NodeID:     kr.NodeID,
		OldAddress: kr.OldAddress,
		NewAddress: kr.NewAddress,
	}
	return NewCursor(cursor.String())
}

func (kr EthereumKeyRotation) ToProtoEdge(_ ...any) (*v2.EthereumKeyRotationEdge, error) {
	return &v2.EthereumKeyRotationEdge{
		Node:   kr.ToProto(),
		Cursor: kr.Cursor().Encode(),
	}, nil
}

type EthereumKeyRotationCursor struct {
	VegaTime   time.Time       `json:"vegaTime"`
	NodeID     NodeID          `json:"nodeID"`
	OldAddress EthereumAddress `json:"oldAddress"`
	NewAddress EthereumAddress `json:"newAddress"`
}

func (ec EthereumKeyRotationCursor) String() string {
	bs, err := json.Marshal(ec)
	if err != nil {
		// This should never happen.
		panic(fmt.Errorf("couldn't marshal deposit cursor: %w", err))
	}
	return string(bs)
}

func (ec *EthereumKeyRotationCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), ec)
}
