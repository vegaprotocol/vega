package nullchain

import (
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type Collateral interface {
	GetPartyGeneralAccount(party, asset string) (*types.Account, error)
}

type Assets interface {
	Get(assetID string) (*assets.Asset, error)
}

type StakingLoop struct {
	col    Collateral
	assets Assets

	// The built-in asset which when deposited into the collateral is to be used to pretend that is was
	// staked on the bridge
	stakingAsset string
}

func NewStakingLoop(col Collateral, assets Assets) *StakingLoop {
	return &StakingLoop{
		col:          col,
		assets:       assets,
		stakingAsset: "VOTE",
	}
}

func (s *StakingLoop) GetAvailableBalance(party string) (*num.Uint, error) {
	acc, err := s.col.GetPartyGeneralAccount(party, s.stakingAsset)
	if err != nil {
		return nil, err
	}
	return acc.Balance.Clone(), nil
}

func (s *StakingLoop) GetAvailableBalanceInRange(party string, from, to time.Time) (*num.Uint, error) {
	// We're just going to have to say we have no notion of time range and whatever is has be deposited by the faucet
	// has always been there.
	balance, err := s.GetAvailableBalance(party)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (s *StakingLoop) GetStakingAssetTotalSupply() *num.Uint {
	asset, err := s.assets.Get(s.stakingAsset)
	if err != nil {
		return nil // its really should exist
	}
	return asset.Type().GetAssetTotalSupply()
}
