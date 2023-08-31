package types

import "time"

type ReferralSetID string

type ReferralSet struct {
	ID ReferralSetID

	CreatedAt time.Time
	UpdatedAt time.Time

	Referrer *Membership
	Referees []*Membership
}
