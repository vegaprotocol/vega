// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package referral

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/referral EpochEngine,Broker,TeamsEngine

// EpochEngine is used to know when to apply the team switches.
type EpochEngine interface {
	NotifyOnEpoch(func(context.Context, types.Epoch), func(context.Context, types.Epoch))
}

// Broker is used to notify administrative actions on teams and members.
type Broker interface {
	Send(event events.Event)
}

// TeamsEngine is used to retrieve statistics about a team member to compute
// referral program related data.
type TeamsEngine interface {
	IsTeamMember(party types.PartyID) bool
	NumberOfEpochInTeamForParty(party types.PartyID) uint64
}
