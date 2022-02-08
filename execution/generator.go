package execution

import (
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"encoding/hex"
	"fmt"
)

// idGenerator no mutex required, markets work deterministically, and sequentially.
type idGenerator struct {
	rootId      string
	nextIdBytes []byte
}

// NewDeterministicIDGenerator returns an idGenerator, and is used to abstract this type.
func NewDeterministicIDGenerator(rootId string) (*idGenerator, error) {

	nextIdBytes, err := hex.DecodeString(rootId)
	if err != nil {
		return nil, fmt.Errorf("failed to create new deterministic id generator:%w", err)
	}

	return &idGenerator{
		rootId:      rootId,
		nextIdBytes: nextIdBytes,
	}, nil
}

func (i *idGenerator) SetID(order *types.Order) {
	order.ID = hex.EncodeToString(i.nextIdBytes)
	i.nextIdBytes = crypto.Hash(i.nextIdBytes)
}
