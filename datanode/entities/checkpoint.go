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

type Checkpoint struct {
	Hash        string
	BlockHash   string
	BlockHeight int64
	TxHash      TxHash
	VegaTime    time.Time
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
