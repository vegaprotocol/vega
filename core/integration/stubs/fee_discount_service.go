package stubs

import (
	"errors"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type FeeDiscountRewardService struct{}

func (*FeeDiscountRewardService) ReferralDiscountFactorForParty(party types.PartyID) num.Decimal {
	return num.DecimalZero()
}

func (*FeeDiscountRewardService) VolumeDiscountFactorForParty(party types.PartyID) num.Decimal {
	return num.DecimalZero()
}

func (*FeeDiscountRewardService) RewardsFactorMultiplierAppliedForParty(party types.PartyID) num.Decimal {
	return num.DecimalZero()
}

func (*FeeDiscountRewardService) GetReferrer(referee types.PartyID) (types.PartyID, error) {
	return types.PartyID(""), errors.New("no referrer")
}
