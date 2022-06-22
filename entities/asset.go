package entities

import (
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	pb "code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type AssetID struct{ ID }

func NewAssetID(id string) AssetID {
	return AssetID{ID: ID(id)}
}

type Asset struct {
	ID                AssetID
	Name              string
	Symbol            string
	TotalSupply       decimal.Decimal // Maybe num.Uint if we can figure out how to add support to pgx
	Decimals          int
	Quantum           int
	Source            string
	ERC20Contract     string
	VegaTime          time.Time
	LifetimeLimit     decimal.Decimal
	WithdrawThreshold decimal.Decimal
}

func (a Asset) ToProto() *pb.Asset {
	pbAsset := &pb.Asset{
		Id: a.ID.String(),
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
				ContractAddress:   a.ERC20Contract,
				LifetimeLimit:     a.LifetimeLimit.String(),
				WithdrawThreshold: a.WithdrawThreshold.String(),
			},
		}
	}

	return pbAsset
}

func (a Asset) Cursor() *Cursor {
	return NewCursor(a.ID.String())
}

func (a Asset) ToProtoEdge(_ ...any) *v2.AssetEdge {
	return &v2.AssetEdge{
		Node:   a.ToProto(),
		Cursor: a.Cursor().Encode(),
	}
}
