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

package activitystreak

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"golang.org/x/exp/maps"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/activitystreak Broker,MarketsStatsAggregator

type PartyActivity struct {
	Active                               uint64
	Inactive                             uint64
	RewardDistributionActivityMultiplier num.Decimal
	RewardVestingActivityMultiplier      num.Decimal
}

func (p PartyActivity) IsActive() bool {
	return p.Active != 0 && p.Inactive == 0
}

func (p *PartyActivity) ResetMultipliers() {
	p.RewardDistributionActivityMultiplier = num.DecimalOne()
	p.RewardVestingActivityMultiplier = num.DecimalOne()
}

func (p *PartyActivity) UpdateMultipliers(benefitTiers []*types.ActivityStreakBenefitTier) {
	if !p.IsActive() {
		// we are not active, nothing changes
		return
	}

	// these are sorted properly already
	for _, b := range benefitTiers {
		if p.Active < b.MinimumActivityStreak {
			break
		}

		p.RewardDistributionActivityMultiplier = b.RewardMultiplier
		p.RewardVestingActivityMultiplier = b.VestingMultiplier
	}
}

type MarketsStatsAggregator interface {
	GetMarketStats() map[string]*types.MarketStats
}

type Broker interface {
	SendBatch(evt []events.Event)
}

type Engine struct {
	log *logging.Logger

	marketStats     MarketsStatsAggregator
	partiesActivity map[string]*PartyActivity
	broker          Broker

	benefitTiers                 []*types.ActivityStreakBenefitTier
	minQuantumOpenNotionalVolume *num.Uint
	minQuantumTradeVolume        *num.Uint
	inactivityLimit              uint64
}

func New(
	log *logging.Logger,
	marketStats MarketsStatsAggregator,
	broker Broker,
) *Engine {
	return &Engine{
		log:                          log,
		marketStats:                  marketStats,
		partiesActivity:              map[string]*PartyActivity{},
		broker:                       broker,
		minQuantumOpenNotionalVolume: num.UintZero(),
		minQuantumTradeVolume:        num.UintZero(),
	}
}

func (e *Engine) OnRewardsActivityStreakInactivityLimit(
	_ context.Context, v *num.Uint,
) error {
	e.inactivityLimit = v.Uint64()
	return nil
}

func (e *Engine) OnMinQuantumOpenNationalVolumeUpdate(
	_ context.Context, v *num.Uint,
) error {
	e.minQuantumOpenNotionalVolume = v.Clone()
	return nil
}

func (e *Engine) OnMinQuantumTradeVolumeUpdate(
	_ context.Context, v *num.Uint,
) error {
	e.minQuantumTradeVolume = v.Clone()
	return nil
}

func (e *Engine) OnBenefitTiersUpdate(
	_ context.Context, v interface{},
) error {
	tiers, err := types.ActivityStreakBenefitTiersFromUntypedProto(v)
	if err != nil {
		return err
	}

	e.benefitTiers = tiers.Clone().Tiers
	sort.Slice(e.benefitTiers, func(i, j int) bool {
		return e.benefitTiers[i].MinimumActivityStreak < e.benefitTiers[j].MinimumActivityStreak
	})
	return nil
}

func (e *Engine) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	if epoch.Action == vegapb.EpochAction_EPOCH_ACTION_END {
		e.update(ctx, epoch.Seq)
	}
}

func (e *Engine) OnEpochRestore(ctx context.Context, epoch types.Epoch) {}

func (e *Engine) GetRewardsDistributionMultiplier(party string) num.Decimal {
	if _, ok := e.partiesActivity[party]; !ok {
		return num.DecimalOne()
	}

	return e.partiesActivity[party].RewardDistributionActivityMultiplier
}

func (e *Engine) GetRewardsVestingMultiplier(party string) num.Decimal {
	if _, ok := e.partiesActivity[party]; !ok {
		return num.DecimalOne()
	}

	return e.partiesActivity[party].RewardVestingActivityMultiplier
}

type partyStats struct {
	OpenVolume  *num.Uint
	TradeVolume *num.Uint
}

func (e *Engine) update(ctx context.Context, epochSeq uint64) {
	stats := e.marketStats.GetMarketStats()

	// first accumulate the stats

	// party -> volume across all markets
	parties := map[string]*partyStats{}
	for _, v := range stats {
		for p, vol := range v.PartiesOpenNotionalVolume {
			party := parties[p]
			if party == nil {
				party = &partyStats{
					OpenVolume:  num.UintZero(),
					TradeVolume: num.UintZero(),
				}
				parties[p] = party
			}
			party.OpenVolume.Add(party.OpenVolume, vol.Clone())
		}

		for p, vol := range v.PartiesTotalTradeVolume {
			party := parties[p]
			if party == nil {
				party = &partyStats{
					OpenVolume:  num.UintZero(),
					TradeVolume: num.UintZero(),
				}
				parties[p] = party
			}
			party.TradeVolume.Add(party.TradeVolume, vol.Clone())
		}
	}

	partiesKey := maps.Keys(parties)
	sort.Strings(partiesKey)

	for _, party := range partiesKey {
		v := parties[party]
		e.updateStreak(party, v.OpenVolume, v.TradeVolume)
	}

	// now iterate over all existing parties,
	// and update the ones for which nothing happen during the epoch
	// and send the events
	partiesKey = maps.Keys(e.partiesActivity)
	sort.Strings(partiesKey)
	evts := []events.Event{}
	for _, party := range partiesKey {
		ps := parties[party]
		if _, ok := parties[party]; !ok {
			e.updateStreak(party, num.UintZero(), num.UintZero())
			ps = &partyStats{
				OpenVolume:  num.UintZero(),
				TradeVolume: num.UintZero(),
			}
		}

		evt := e.makeEvent(party, epochSeq, ps.OpenVolume, ps.TradeVolume)
		evts = append(evts, events.NewPartyActivityStreakEvent(ctx, evt))
	}
	e.broker.SendBatch(evts)
}

func (e *Engine) makeEvent(party string, epochSeq uint64, openVolume, tradedVolume *num.Uint) *eventspb.PartyActivityStreak {
	partyActivity := e.partiesActivity[party]
	return &eventspb.PartyActivityStreak{
		Party:                                party,
		ActiveFor:                            partyActivity.Active,
		InactiveFor:                          partyActivity.Inactive,
		IsActive:                             partyActivity.IsActive(),
		RewardDistributionActivityMultiplier: partyActivity.RewardDistributionActivityMultiplier.String(),
		RewardVestingActivityMultiplier:      partyActivity.RewardVestingActivityMultiplier.String(),
		Epoch:                                epochSeq,
		TradedVolume:                         tradedVolume.String(),
		OpenVolume:                           openVolume.String(),
	}
}

func (e *Engine) updateStreak(party string, openVolume, tradeVolume *num.Uint) {
	partyActivity, ok := e.partiesActivity[party]
	if !ok {
		partyActivity = &PartyActivity{
			RewardDistributionActivityMultiplier: num.DecimalOne(),
			RewardVestingActivityMultiplier:      num.DecimalOne(),
		}
		e.partiesActivity[party] = partyActivity
	}

	if openVolume.GT(e.minQuantumOpenNotionalVolume) || tradeVolume.GT(e.minQuantumTradeVolume) {
		partyActivity.Active++
		partyActivity.Inactive = 0
	} else {
		partyActivity.Inactive++

		if partyActivity.Inactive >= e.inactivityLimit {
			partyActivity.Active = 0
			partyActivity.ResetMultipliers()
		}
	}

	partyActivity.UpdateMultipliers(e.benefitTiers)
}
