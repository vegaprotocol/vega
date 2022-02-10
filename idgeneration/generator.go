package idgeneration

import (
	"encoding/hex"
	"strings"

	"code.vegaprotocol.io/vega/libs/crypto"
)

// idGenerator no mutex required, markets work deterministically, and sequentially.
type idGenerator struct {
	nextIdBytes []byte
}

// New returns an idGenerator, and is used to abstract this type.
func New(rootId string) *idGenerator {
	nextIdBytes, err := hex.DecodeString(rootId)
	if err != nil {
		panic("failed to create new deterministic id generator: " + err.Error())
	}

	return &idGenerator{
		nextIdBytes: nextIdBytes,
	}
}

func (i *idGenerator) NextID() string {
	if i == nil {
		panic("id generator instance is not initialised")
	}

	nextId := strings.ToUpper(hex.EncodeToString(i.nextIdBytes))
	i.nextIdBytes = crypto.Hash(i.nextIdBytes)
	return nextId
}
