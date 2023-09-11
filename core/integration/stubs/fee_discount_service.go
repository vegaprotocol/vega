package stubs

import (
	"errors"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type ReferralDiscountRewardService struct{}

func (*ReferralDiscountRewardService) ReferralDiscountFactorForParty(party types.PartyID) num.Decimal {
	return num.DecimalZero()
}

func (*ReferralDiscountRewardService) RewardsFactorMultiplierAppliedForParty(party types.PartyID) num.Decimal {
	return num.DecimalZero()
}

func (*ReferralDiscountRewardService) GetReferrer(referee types.PartyID) (types.PartyID, error) {
	return types.PartyID(""), errors.New("no referrer")
}

type VolumeDiscountService struct{}

func (*VolumeDiscountService) VolumeDiscountFactorForParty(party types.PartyID) num.Decimal {
	return num.DecimalZero()
}
