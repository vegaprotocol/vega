package assets

import (
	"code.vegaprotocol.io/vega/assets/builtin"
	"code.vegaprotocol.io/vega/assets/common"
	"code.vegaprotocol.io/vega/assets/erc20"
	"code.vegaprotocol.io/vega/types"
)

type isAsset interface {
	// Type get information about the asset itself
	Type() *types.Asset
	// GetAssetClass get the internal asset class
	GetAssetClass() common.AssetClass
	// IsValid is the order valid / validated with the target chain?
	IsValid() bool
	// Validate this is used to validate that the asset
	// exist on the target chain
	Validate() error
	String() string
}

type Asset struct {
	isAsset
}

func NewAsset(a isAsset) *Asset {
	return &Asset{a}
}

func (a *Asset) IsERC20() bool {
	_, ok := a.isAsset.(*erc20.ERC20)
	return ok
}

func (a *Asset) IsBuiltinAsset() bool {
	_, ok := a.isAsset.(*builtin.Builtin)
	return ok
}

func (a *Asset) ERC20() (*erc20.ERC20, bool) {
	asset, ok := a.isAsset.(*erc20.ERC20)
	return asset, ok
}

func (a *Asset) BuiltinAsset() (*builtin.Builtin, bool) {
	asset, ok := a.isAsset.(*builtin.Builtin)
	return asset, ok
}

func (a *Asset) ToAssetType() *types.Asset {
	return a.Type()
}
