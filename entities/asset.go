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

// Some assets have IDs that are not hex strings; this will be fixed but in the mean time
// if they are not hex strings store the name as-is with a prefix so we can identify them.
func MakeAssetId(stringId string) []byte {
	id, err := hex.DecodeString(stringId)
	if err != nil {
		id = []byte("bad_asset_" + stringId)
	}
	return id
}

func (a Asset) HexId() string {
	if string(a.ID[:10]) == "bad_asset_" {
		return string(a.ID[10:])
	}

	return hex.EncodeToString(a.ID)
}
