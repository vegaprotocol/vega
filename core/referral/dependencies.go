// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package referral

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/referral EpochEngine,Broker,TimeService,MarketActivityTracker,StakingBalances

type StakingBalances interface {
	GetAvailableBalance(party string) (*num.Uint, error)
}

type TimeService interface {
	GetTimeNow() time.Time
}

// EpochEngine is used to know when to apply the team switches.
type EpochEngine interface {
	NotifyOnEpoch(func(context.Context, types.Epoch), func(context.Context, types.Epoch))
}

// Broker is used to notify administrative actions on teams and members.
type Broker interface {
	Send(events.Event)
}

// MarketActivityTracker is used to retrieve the trading statistics about a party
// to compute referral program related data.
type MarketActivityTracker interface {
	NotionalTakerVolumeForParty(string) *num.Uint
}
