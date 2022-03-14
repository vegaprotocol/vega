package entities

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	pb "code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

const badAssetPrefix = "bad_asset_"

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
		id = []byte(badAssetPrefix + stringID)
	}
	return id
}

func (a Asset) HexID() string {
	if strings.HasPrefix(string(a.ID), badAssetPrefix) {
		return strings.TrimPrefix(string(a.ID), badAssetPrefix)
	}

	return hex.EncodeToString(a.ID)
}

func (a Asset) ToProto() *pb.Asset {
	pbAsset := &pb.Asset{
		Id: a.HexID(),
		Details: &pb.AssetDetails{
			Name:        a.Name,
			Symbol:      a.Symbol,
			TotalSupply: a.TotalSupply.BigInt().String(),
			Decimals:    uint64(a.Decimals),
			Quantum:     fmt.Sprintf("%d", a.Quantum),
		},
	}
	if a.Source != "" {
		pbAsset.Details.Source = &pb.AssetDetails_BuiltinAsset{
			BuiltinAsset: &pb.BuiltinAsset{
				MaxFaucetAmountMint: a.Source,
			},
		}
	} else if a.ERC20Contract != "" {
		pbAsset.Details.Source = &pb.AssetDetails_Erc20{
			Erc20: &pb.ERC20{
				ContractAddress: a.ERC20Contract,
			},
		}
	}

	return pbAsset
}
