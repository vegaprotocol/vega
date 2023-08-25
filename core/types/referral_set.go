package types

import "time"

type ReferralSet struct {
	ID string

	CreatedAt time.Time
	UpdatedAt time.Time

	Referrer  *Membership
	Referrees []*Membership
}
