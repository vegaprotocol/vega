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

type EthereumKeyRotation struct {
	VegaTime    time.Time
	NodeID      NodeID
	OldAddress  EthereumAddress
	NewAddress  EthereumAddress
	TxHash      TxHash
	BlockHeight uint64
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
