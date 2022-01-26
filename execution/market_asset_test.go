package execution_test

import (
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/assets/common"
	"code.vegaprotocol.io/vega/types"
)

type isAssetStub struct {
	ID            string
	DecimalPlaces uint64
}

func NewAssetStub(id string, dp uint64) *assets.Asset {
	return assets.NewAsset(&isAssetStub{
		ID:            id,
		DecimalPlaces: dp,
	})
}

func (a isAssetStub) Type() *types.Asset {
	return &types.Asset{
		ID: a.ID,
		Details: &types.AssetDetails{
			Symbol:   a.ID,
			Decimals: a.DecimalPlaces,
		},
	}
}

func (_ isAssetStub) GetAssetClass() common.AssetClass {
	return common.Builtin
}

func (_ isAssetStub) IsValid() bool {
	return true
}

func (_ isAssetStub) Validate() error {
	return nil
}

func (_ isAssetStub) SetValidNonValidator() {}

func (a isAssetStub) String() string {
	return a.ID
}
