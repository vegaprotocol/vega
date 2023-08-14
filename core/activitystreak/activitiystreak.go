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

package activitystreak

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"golang.org/x/exp/maps"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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
}

func New(
	log *logging.Logger,
	marketStats MarketsStatsAggregator,
	broker Broker,
) *Engine {
	return &Engine{
		log:             log,
		partiesActivity: map[string]*PartyActivity{},
		marketStats:     marketStats,
		broker:          broker,
	}
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
		if _, ok := parties[party]; !ok {
			e.updateStreak(party, num.UintZero(), num.UintZero())
		}

		evt := e.makeEvent(party, epochSeq)
		evts = append(evts, events.NewPartyActivityStreakEvent(ctx, evt))
	}
	e.broker.SendBatch(evts)
}

func (e *Engine) makeEvent(party string, epochSeq uint64) *eventspb.PartyActivityStreak {
	partyActivity := e.partiesActivity[party]
	return &eventspb.PartyActivityStreak{
		Party:                                party,
		ActiveFor:                            partyActivity.Active,
		InactiveFor:                          partyActivity.Inactive,
		IsActive:                             partyActivity.IsActive(),
		RewardDistributionActivityMultiplier: partyActivity.RewardDistributionActivityMultiplier.String(),
		RewardVestingActivityMultiplier:      partyActivity.RewardVestingActivityMultiplier.String(),
		Epoch:                                epochSeq,
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

		if partyActivity.Inactive >= partyActivity.Active {
			partyActivity.Active = 0
			partyActivity.ResetMultipliers()
		}
	}

	partyActivity.UpdateMultipliers(e.benefitTiers)
}
