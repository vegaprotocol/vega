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
}

type ReferralSetStats struct {
	AtEpoch                  uint64
	SetID                    ReferralSetID
	ReferralSetRunningVolume *num.Uint
	RefereesStats            map[PartyID]*RefereeStats
}

type RefereeStats struct {
	DiscountFactor num.Decimal
	RewardFactor   num.Decimal
}
