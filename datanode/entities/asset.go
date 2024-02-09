// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package entities

import (
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	pb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
)

type _Asset struct{}

type AssetID = ID[_Asset]

type Asset struct {
	ID                AssetID
	Name              string
	Symbol            string
	Decimals          int
	Quantum           decimal.Decimal
	Source            string
	ERC20Contract     string
	TxHash            TxHash
	VegaTime          time.Time
	LifetimeLimit     decimal.Decimal
	WithdrawThreshold decimal.Decimal
	Status            AssetStatus
}

func (a Asset) ToProto() *pb.Asset {
	pbAsset := &pb.Asset{
		Id: a.ID.String(),
		Details: &pb.AssetDetails{
			Name:     a.Name,
			Symbol:   a.Symbol,
			Decimals: uint64(a.Decimals),
			Quantum:  a.Quantum.String(),
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
	ac := AssetCursor{
		ID: a.ID,
	}
	return NewCursor(ac.String())
}

func (a Asset) ToProtoEdge(_ ...any) (*v2.AssetEdge, error) {
	return &v2.AssetEdge{
		Node:   a.ToProto(),
		Cursor: a.Cursor().Encode(),
	}, nil
}

type AssetCursor struct {
	ID AssetID `json:"id"`
}

func (ac AssetCursor) String() string {
	bs, err := json.Marshal(ac)
	if err != nil {
		panic(fmt.Errorf("couldn't marshal asset cursor: %w", err))
	}
	return string(bs)
}

func (ac *AssetCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), ac)
}
