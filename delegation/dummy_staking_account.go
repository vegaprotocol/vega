package delegation

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/types/num"
)

type DummyStakingAccounts struct {
	collateralEngine *collateral.Engine
	asset            string
}

func (d *DummyStakingAccounts) GovAssetUpdated(ctx context.Context, asset string) error {
	d.asset = asset
	return nil
}

//GetBalanceNow returns the current party's governance token balance
func (d *DummyStakingAccounts) GetBalanceNow(party string) *num.Uint {
	if generalAcc, err := d.collateralEngine.GetPartyGeneralAccount(party, d.asset); err == nil {
		return generalAcc.Balance.Clone()
	}
	return nil
}

//GetBalanceForEpoch returns the current party's governance token balance
func (d *DummyStakingAccounts) GetBalanceForEpoch(party string, from, to time.Time) *num.Uint {
	if generalAcc, err := d.collateralEngine.GetPartyGeneralAccount(party, d.asset); err == nil {
		return generalAcc.Balance.Clone()
	}
	return nil
}

//NewDummyStakingAccount returns a new instance of a staking account backed by governance token account
func NewDummyStakingAccount(collateralEngine *collateral.Engine) *DummyStakingAccounts {
	return &DummyStakingAccounts{
		collateralEngine: collateralEngine,
	}
}
