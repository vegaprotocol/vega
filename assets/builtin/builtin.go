// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package builtin

import (
	"fmt"

	"code.vegaprotocol.io/data-node/assets/common"
	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
)

type Builtin struct {
	asset *types.Asset
}

func New(id string, asset *types.AssetDetails) *Builtin {
	return &Builtin{
		asset: &types.Asset{
			ID:      id,
			Details: asset,
		},
	}
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

func (b *Builtin) Validate() error {
	return nil
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
	return fmt.Sprintf("id(%v) name(%v) symbol(%v) totalSupply(%v) decimals(%v)",
		b.asset.ID, b.asset.Details.Name, b.asset.Details.Symbol, b.asset.Details.TotalSupply,
		b.asset.Details.Decimals)
}
