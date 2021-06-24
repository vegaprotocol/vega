package builtin

import (
	"fmt"

	"code.vegaprotocol.io/vega/assets/common"
	"code.vegaprotocol.io/vega/types"
)

type Builtin struct {
	asset *types.Asset
}

func New(id string, asset *types.AssetDetails) *Builtin {
	return &Builtin{
		asset: &types.Asset{
			Id:      id,
			Details: asset,
		},
	}
}

func (b *Builtin) ProtoAsset() *types.Asset {
	return b.asset
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
		b.asset.Id, b.asset.Details.Name, b.asset.Details.Symbol, b.asset.Details.TotalSupply,
		b.asset.Details.Decimals)
}
