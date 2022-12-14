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
