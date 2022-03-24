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
