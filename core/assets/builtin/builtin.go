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

package builtin

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/assets/common"
	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type Builtin struct {
	asset *types.Asset
}

func New(id string, asset *types.AssetDetails) *Builtin {
	return &Builtin{
		asset: &types.Asset{
			ID:      id,
			Details: asset,
			Status:  types.AssetStatusProposed,
		},
	}
}

func (e *Builtin) SetValid() {}

func (e *Builtin) SetPendingListing() {
	e.asset.Status = types.AssetStatusPendingListing
}

func (e *Builtin) SetRejected() {
	e.asset.Status = types.AssetStatusRejected
}

func (e *Builtin) SetEnabled() {
	e.asset.Status = types.AssetStatusEnabled
}

func (b *Builtin) ProtoAsset() *proto.Asset {
	return b.asset.IntoProto()
}

func (b Builtin) Type() *types.Asset {
	return b.asset.DeepClone()
}

func (b *Builtin) GetAssetClass() common.AssetClass {
	return common.Builtin
}

func (b *Builtin) IsValid() bool {
	return true
}

func (b *Builtin) SignBridgeWhitelisting() ([]byte, []byte, error) {
	return nil, nil, nil
}

func (b *Builtin) ValidateWithdrawal() error {
	return nil
}

func (b *Builtin) SignWithdrawal() ([]byte, error) {
	return nil, nil
}

func (b *Builtin) ValidateDeposit() error {
	return nil
}

func (b *Builtin) String() string {
	return fmt.Sprintf("id(%v) name(%v) symbol(%v) decimals(%v)",
		b.asset.ID, b.asset.Details.Name, b.asset.Details.Symbol,
		b.asset.Details.Decimals)
}
