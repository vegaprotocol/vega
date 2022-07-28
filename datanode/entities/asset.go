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
	Quantum           decimal.Decimal
	Source            string
	ERC20Contract     string
	VegaTime          time.Time
	LifetimeLimit     decimal.Decimal
	WithdrawThreshold decimal.Decimal
	Status            AssetStatus
}

func (a Asset) ToProto() *pb.Asset {
	pbAsset := &pb.Asset{
		Id: a.ID.String(),
		Details: &pb.AssetDetails{
			Name:        a.Name,
			Symbol:      a.Symbol,
			TotalSupply: a.TotalSupply.BigInt().String(),
			Decimals:    uint64(a.Decimals),
			Quantum:     a.Quantum.String(),
		},
		Status: pb.Asset_Status(a.Status),
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

func (a Asset) ToProtoEdge(_ ...any) (*v2.AssetEdge, error) {
	return &v2.AssetEdge{
		Node:   a.ToProto(),
		Cursor: a.Cursor().Encode(),
	}, nil
}
