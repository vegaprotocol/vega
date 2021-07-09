package assets

import (
	"code.vegaprotocol.io/data-node/assets/builtin"
	"code.vegaprotocol.io/data-node/assets/common"
	"code.vegaprotocol.io/data-node/types"
)

type isAsset interface {
	// ProtoAsset get information about the asset itself
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

func (a *Asset) IsBuiltinAsset() bool {
	_, ok := a.isAsset.(*builtin.Builtin)
	return ok
}

func (a *Asset) BuiltinAsset() (*builtin.Builtin, bool) {
	asset, ok := a.isAsset.(*builtin.Builtin)
	return asset, ok
}

func (a *Asset) ToAssetType() *types.Asset {
	return a.Type()
}
