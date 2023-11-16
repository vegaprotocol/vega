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

type CoreSnapshotData struct {
	BlockHeight     uint64
	BlockHash       string
	VegaCoreVersion string
	TxHash          TxHash
	VegaTime        time.Time
}

func CoreSnapshotDataFromProto(s *eventspb.CoreSnapshotData, txHash TxHash, vegaTime time.Time) CoreSnapshotData {
	return CoreSnapshotData{
		BlockHeight:     s.BlockHeight,
		BlockHash:       s.BlockHash,
		VegaCoreVersion: s.CoreVersion,
		TxHash:          txHash,
		VegaTime:        vegaTime,
	}
}

func (s *CoreSnapshotData) ToProto() *eventspb.CoreSnapshotData {
	return &eventspb.CoreSnapshotData{
		BlockHeight: s.BlockHeight,
		BlockHash:   s.BlockHash,
		CoreVersion: s.VegaCoreVersion,
	}
}

func (s CoreSnapshotData) Cursor() *Cursor {
	pc := CoreSnapshotDataCursor{
		VegaTime:        s.VegaTime,
		BlockHeight:     s.BlockHeight,
		BlockHash:       s.BlockHash,
		VegaCoreVersion: s.VegaCoreVersion,
	}
	return NewCursor(pc.String())
}

func (s CoreSnapshotData) ToProtoEdge(_ ...any) (*v2.CoreSnapshotEdge, error) {
	return &v2.CoreSnapshotEdge{
		Node:   s.ToProto(),
		Cursor: s.Cursor().Encode(),
	}, nil
}

type CoreSnapshotDataCursor struct {
	VegaTime        time.Time
	BlockHeight     uint64
	BlockHash       string
	VegaCoreVersion string
}

func (sc CoreSnapshotDataCursor) String() string {
	bs, err := json.Marshal(sc)
	if err != nil {
		panic(fmt.Errorf("failed to marshal core snapshot data cursor: %w", err))
	}
	return string(bs)
}

func (sc *CoreSnapshotDataCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), sc)
}
