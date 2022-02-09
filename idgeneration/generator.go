package idgeneration

import (
	"encoding/hex"
	"strings"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
)

// idGenerator no mutex required, markets work deterministically, and sequentially.
type idGenerator struct {
	rootId      string
	nextIdBytes []byte
}

// NewDeterministicIDGenerator returns an idGenerator, and is used to abstract this type.
func NewDeterministicIDGenerator(rootId string) *idGenerator {
	nextIdBytes, err := hex.DecodeString(rootId)
	if err != nil {
		panic("failed to create new deterministic id generator: " + err.Error())
	}

	return &idGenerator{
		rootId:      rootId,
		nextIdBytes: nextIdBytes,
	}
}

func (i *idGenerator) SetID(order *types.Order) {
	order.ID = strings.ToUpper(hex.EncodeToString(i.nextIdBytes))
	i.nextIdBytes = crypto.Hash(i.nextIdBytes)
}
