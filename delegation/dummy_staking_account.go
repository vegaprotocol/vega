package delegation

import (
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/types/num"
)

type DummyStakingAccounts struct {
	collateralEngine *collateral.Engine
	asset            string
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
func NewDummyStakingAccount(collateralEngine *collateral.Engine, asset string) *DummyStakingAccounts {
	return &DummyStakingAccounts{
		collateralEngine: collateralEngine,
		asset:            asset,
	}
}
