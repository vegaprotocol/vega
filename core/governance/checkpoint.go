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

package governance

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	checkpointpb "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
)

type enactmentTime struct {
	current         int64
	shouldNotVerify bool
	cpLoad          bool
}

func (e *Engine) Name() types.CheckpointName {
	return types.GovernanceCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	if len(e.enactedProposals) == 0 && len(e.activeProposals) == 0 {
		return nil, nil
	}
	cp := &checkpointpb.Proposals{
		Proposals: e.getCheckpointProposals(),
	}
	return proto.Marshal(cp)
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	cp := &checkpointpb.Proposals{}
	if err := proto.Unmarshal(data, cp); err != nil {
		return err
	}

	evts := make([]events.Event, 0, len(cp.Proposals))
	now := e.timeService.GetTimeNow()
	minEnact, err := e.netp.GetDuration(netparams.GovernanceProposalMarketMinEnact)
	if err != nil {
		e.log.Panic("failed to get proposal market min enactment duration from network parameter")
	}
	minAuctionDuration, err := e.netp.GetDuration(netparams.MarketAuctionMinimumDuration)
	if err != nil {
		e.log.Panic("failed to get proposal market min auction duration from network parameter")
	}
	duration := minEnact
	// we have to choose the max between minEnact and minAuctionDuration otherwise we won't be able to submit the market successfully
	if int64(minEnact) < int64(minAuctionDuration) {
		duration = minAuctionDuration
	}

	latestUpdateMarketProposals := map[string]*types.Proposal{}
	updatedMarketIDs := []string{}
	for _, p := range cp.Proposals {
		prop, err := types.ProposalFromProto(p)
		if err != nil {
			return err
		}

		switch prop.Terms.Change.GetTermType() {
		case types.ProposalTermsTypeNewMarket:
			// before we mess around with enactment times, determine the time until enactment
			closeTime := time.Unix(p.Terms.ClosingTimestamp, 0)
			enactTime := time.Unix(p.Terms.EnactmentTimestamp, 0)
			auctionDuration := enactTime.Sub(closeTime)
			// check for successor proposals
			toEnact := false
			if p.Terms.EnactmentTimestamp <= now.Unix() {
				toEnact = true
			}
			enct := &enactmentTime{
				cpLoad: true,
			}
			// if the proposal is for a new market it should be restored it such that it will be in opening auction
			if toEnact {
				prop.Terms.ClosingTimestamp = now.Unix()
				if auctionDuration < duration {
					prop.Terms.EnactmentTimestamp = now.Add(duration).Unix()
				} else {
					prop.Terms.EnactmentTimestamp = now.Add(auctionDuration).Unix()
				}
				enct.shouldNotVerify = true
			}
			enct.current = prop.Terms.EnactmentTimestamp

			// handle markets that were proposed in older versions so will not have these new fields when we resubmit
			if prop.NewMarket().Changes.LiquiditySLAParameters == nil {
				prop.NewMarket().Changes.LiquiditySLAParameters = ptr.From(liquidity.DefaultSLAParameters)
			}
			if prop.NewMarket().Changes.LiquidityFeeSettings == nil {
				prop.NewMarket().Changes.LiquidityFeeSettings = &types.LiquidityFeeSettings{Method: types.LiquidityFeeMethodMarginalCost}
			}

			toSubmit, err := e.intoToSubmit(ctx, prop, enct, true)
			if err != nil {
				e.log.Panic("Failed to convert proposal into market", logging.Error(err))
			}
			nm := toSubmit.NewMarket()
			err = e.markets.RestoreMarket(ctx, nm.Market())
			if err != nil {
				if err == execution.ErrMarketDoesNotExist {
					// market has been settled, network doesn't care
					continue
				}
				// any other error, panic
				e.log.Panic("failed to restore market from checkpoint", logging.Market(*nm.Market()), logging.Error(err))
			}

			if err := e.markets.StartOpeningAuction(ctx, prop.ID); err != nil {
				e.log.Panic("failed to start opening auction for market", logging.String("market-id", prop.ID), logging.Error(err))
			}
		case types.ProposalTermsTypeUpdateMarket:
			marketID := prop.Terms.GetUpdateMarket().MarketID
			updatedMarketIDs = append(updatedMarketIDs, marketID)
			last, ok := latestUpdateMarketProposals[marketID]
			if !ok || prop.Terms.EnactmentTimestamp > last.Terms.EnactmentTimestamp {
				latestUpdateMarketProposals[marketID] = prop
			}
		}

		evts = append(evts, events.NewProposalEvent(ctx, *prop))
		e.enactedProposals = append(e.enactedProposals, &proposal{
			Proposal: prop,
		})
	}
	for _, v := range updatedMarketIDs {
		p := latestUpdateMarketProposals[v]
		mkt, _, err := e.updatedMarketFromProposal(&proposal{Proposal: p})
		if err != nil {
			continue
		}
		e.markets.UpdateMarket(ctx, mkt)
	}

	// send events for restored proposals
	e.broker.SendBatch(evts)
	// @TODO ensure OnTick is called
	return nil
}

func (e *Engine) isActiveMarket(marketID string) bool {
	mktState, err := e.markets.GetMarketState(marketID)
	// if the market is missing from the execution engine it means it's been already cancelled or settled or rejected
	if err == types.ErrInvalidMarketID {
		e.log.Info("not saving market proposal to checkpoint - market has already been removed", logging.String("market-id", marketID))
		return false
	}
	if mktState == types.MarketStateTradingTerminated {
		e.log.Info("not saving market proposal to checkpoint ", logging.String("market-id", marketID), logging.String("market-state", mktState.String()))
		return false
	}
	return true
}

func (e *Engine) getCheckpointProposals() []*vega.Proposal {
	ret := make([]*vega.Proposal, 0, len(e.enactedProposals))

	for _, p := range e.enactedProposals {
		switch p.Terms.Change.GetTermType() {
		case types.ProposalTermsTypeNewMarket:
			if !e.isActiveMarket(p.ID) {
				continue
			}
		case types.ProposalTermsTypeUpdateMarket:
			if !e.isActiveMarket(p.MarketUpdate().MarketID) {
				continue
			}
		}
		ret = append(ret, p.IntoProto())
	}

	// we also need to include new market proposals that have passed, but have no yet been enacted
	// this is because they will exist in the execution engine in an opening auction and should
	// be recreated on checkpoint restore.
	for _, p := range e.activeProposals {
		if !p.IsPassed() {
			continue
		}
		switch p.Terms.Change.GetTermType() {
		case types.ProposalTermsTypeNewMarket:
			if e.isActiveMarket(p.ID) {
				ret = append(ret, p.IntoProto())
			}
		}
	}

	return ret
}
