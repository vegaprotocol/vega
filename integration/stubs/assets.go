package stubs

import (
	"errors"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/assets/common"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type AssetStub struct {
	registered map[string]*assets.Asset
	permissive bool
	defaultDP  uint64
}

func NewAssetStub() *AssetStub {
	return &AssetStub{
		registered: map[string]*assets.Asset{},
		permissive: true,
		defaultDP:  0,
	}
}

func (a *AssetStub) Get(id string) (*assets.Asset, error) {
	r, ok := a.registered[id]
	if ok {
		// pre-registered, so simply return
		return r, nil
	}
	if !a.permissive {
		// we're in strict mode, unknown assets should result in errors
		return nil, errors.New("unknown asset")
	}
	// permissive, we return the default decimal asset
	stub := NewIsAssetStub(id, a.defaultDP)
	return stub, nil
}

func (a *AssetStub) Register(id string, decimals uint64) {
	a.registered[id] = NewIsAssetStub(id, decimals)
}

func (a *AssetStub) SetPermissive() {
	a.permissive = true
}

func (a *AssetStub) SetStrict() {
	a.permissive = false
}

func (AssetStub) Enable(assetID string) error {
	return nil
}

type isAssetStub struct {
	ID            string
	DecimalPlaces uint64
}

func NewIsAssetStub(id string, dp uint64) *assets.Asset {
	return assets.NewAsset(&isAssetStub{
		ID:            id,
		DecimalPlaces: dp,
	})
}

func (a isAssetStub) Type() *types.Asset {
	return &types.Asset{
		ID: a.ID,
		Details: &types.AssetDetails{
			Decimals: a.DecimalPlaces,
			Quantum:  num.NewUint(5000),
		},
	}
}

func (isAssetStub) GetAssetClass() common.AssetClass {
	return common.Builtin
}

func (isAssetStub) IsValid() bool {
	return true
}

func (isAssetStub) Validate() error {
	return nil
}

func (isAssetStub) SetValidNonValidator() {}

func (a isAssetStub) String() string {
	return a.ID
}
