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

type Checkpoint struct {
	VegaTime    time.Time
	Hash        string
	BlockHash   string
	TxHash      TxHash
	BlockHeight int64
	SeqNum      uint64
}

func (cp *Checkpoint) ToProto() *v2.Checkpoint {
	pcp := v2.Checkpoint{
		Hash:      cp.Hash,
		BlockHash: cp.BlockHash,
		AtBlock:   uint64(cp.BlockHeight),
	}
	return &pcp
}

func (cp Checkpoint) Cursor() *Cursor {
	return NewCursor(CheckpointCursor{BlockHeight: cp.BlockHeight}.String())
}

func (cp Checkpoint) ToProtoEdge(_ ...any) (*v2.CheckpointEdge, error) {
	return &v2.CheckpointEdge{
		Node:   cp.ToProto(),
		Cursor: cp.Cursor().Encode(),
	}, nil
}

func CheckpointFromProto(cpe *eventspb.CheckpointEvent, txHash TxHash) (Checkpoint, error) {
	cp := Checkpoint{
		Hash:        cpe.Hash,
		BlockHash:   cpe.BlockHash,
		BlockHeight: int64(cpe.BlockHeight),
		TxHash:      txHash,
	}
	return cp, nil
}

type CheckpointCursor struct {
	BlockHeight int64 `json:"blockHeight"`
}

func (cp CheckpointCursor) String() string {
	bs, err := json.Marshal(cp)
	if err != nil {
		panic(fmt.Errorf("couldn't marshal CheckpointCursor: %w", err))
	}
	return string(bs)
}

func (cp *CheckpointCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), cp)
}
