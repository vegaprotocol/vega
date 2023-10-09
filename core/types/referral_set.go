package types

import (
	"time"

	"code.vegaprotocol.io/vega/libs/num"
)

type ReferralSetID string

type ReferralSet struct {
	ID ReferralSetID

	CreatedAt time.Time
	UpdatedAt time.Time

	Referrer *Membership
	Referees []*Membership

	CurrentRewardFactor            num.Decimal
	CurrentRewardsMultiplier       num.Decimal
	CurrentRewardsFactorMultiplier num.Decimal
}

type ReferralSetStats struct {
	AtEpoch                  uint64
	SetID                    ReferralSetID
	ReferralSetRunningVolume *num.Uint
	RefereesStats            map[PartyID]*RefereeStats
	RewardFactor             num.Decimal
	RewardsMultiplier        num.Decimal
	RewardsFactorMultiplier  num.Decimal
}

type RefereeStats struct {
	DiscountFactor num.Decimal
	TakerVolume    *num.Uint
}
