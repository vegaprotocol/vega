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

package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	VestingBalancesSummaryEvent interface {
		events.Event
		VestingBalancesSummary() *eventspb.VestingBalancesSummary
	}

	VestingBalancesStore interface {
		Add(ctx context.Context, balance entities.PartyVestingBalance) error
	}

	LockedBalancesStore interface {
		Add(ctx context.Context, balance entities.PartyLockedBalance) error
		Prune(ctx context.Context, currentEpoch uint64) error
	}

	VestingBalancesSummary struct {
		subscriber
		vestingStore VestingBalancesStore
		lockedStore  LockedBalancesStore
	}
)

func NewVestingBalancesSummary(
	vestingStore VestingBalancesStore,
	lockedStore LockedBalancesStore,
) *VestingBalancesSummary {
	return &VestingBalancesSummary{
		vestingStore: vestingStore,
		lockedStore:  lockedStore,
	}
}

func (v *VestingBalancesSummary) Types() []events.Type {
	return []events.Type{
		events.VestingBalancesSummaryEvent,
	}
}

func (v *VestingBalancesSummary) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case VestingBalancesSummaryEvent:
		return v.consumeVestingBalancesSummaryEvent(ctx, e, v.vegaTime)
	default:
		return nil
	}
}

func (v *VestingBalancesSummary) consumeVestingBalancesSummaryEvent(ctx context.Context, e VestingBalancesSummaryEvent, t time.Time) error {
	evt := e.VestingBalancesSummary()

	for _, pvs := range evt.PartiesVestingSummary {
		for _, ppvb := range pvs.PartyVestingBalances {
			pvb, err := entities.PartyVestingBalanceFromProto(
				pvs.Party, evt.EpochSeq, ppvb, t)
			if err != nil {
				return err
			}

			if err := v.vestingStore.Add(ctx, *pvb); err != nil {
				return err
			}
		}

		for _, pplb := range pvs.PartyLockedBalances {
			plb, err := entities.PartyLockedBalanceFromProto(
				pvs.Party, evt.EpochSeq, pplb, t)
			if err != nil {
				return err
			}

			if err := v.lockedStore.Add(ctx, *plb); err != nil {
				return err
			}
		}
	}

	return v.lockedStore.Prune(ctx, evt.EpochSeq)
}
