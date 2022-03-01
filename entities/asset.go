package entities

import (
	"encoding/hex"
	"time"

	"github.com/shopspring/decimal"
)

type Asset struct {
	ID            []byte
	Name          string
	Symbol        string
	TotalSupply   decimal.Decimal // Maybe num.Uint if we can figure out how to add support to pgx
	Decimals      int
	Quantum       int
	Source        string
	ERC20Contract string
	VegaTime      time.Time
}

// MakeAssetID converts a string into a set if bytes. Normally this takes a hex encoded
// SHA256 string, which gets encoded into the corresponding binary representation.
// However some assets have IDs that are not hex strings; this will be fixed but
// in the mean time in that case, store the name as-is with a prefix.
func MakeAssetID(stringID string) []byte {
	id, err := hex.DecodeString(stringID)
	if err != nil {
		id = []byte("bad_asset_" + stringID)
	}
	return id
}

func (a Asset) HexID() string {
	if string(a.ID[:10]) == "bad_asset_" {
		return string(a.ID[10:])
	}

	return hex.EncodeToString(a.ID)
}
