package builtin

import (
	"fmt"

	"code.vegaprotocol.io/vega/assets/common"
	types "code.vegaprotocol.io/vega/proto"
)

type Builtin struct {
	asset *types.Asset
}

func New(id string, asset *types.BuiltinAsset) *Builtin {
	return &Builtin{
		asset: &types.Asset{
			Id:          id,
			Name:        asset.Name,
			Symbol:      asset.Symbol,
			TotalSupply: asset.TotalSupply,
			Decimals:    asset.Decimals,
			Source: &types.AssetSource{
				Source: &types.AssetSource_BuiltinAsset{
					BuiltinAsset: asset,
				},
			},
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
		b.asset.Id, b.asset.Name, b.asset.Symbol, b.asset.TotalSupply,
		b.asset.Decimals)
}
