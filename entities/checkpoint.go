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

type Checkpoint struct {
	Hash        string
	BlockHash   string
	BlockHeight int64
	VegaTime    time.Time
}

func (cp *Checkpoint) ToProto() *protoapi.Checkpoint {
	pcp := protoapi.Checkpoint{
		Hash:      cp.Hash,
		BlockHash: cp.BlockHash,
		AtBlock:   uint64(cp.BlockHeight),
	}
	return &pcp
}

func CheckpointFromProto(cpe *eventspb.CheckpointEvent) (Checkpoint, error) {
	cp := Checkpoint{
		Hash:        cpe.Hash,
		BlockHash:   cpe.BlockHash,
		BlockHeight: int64(cpe.BlockHeight),
	}
	return cp, nil
}
