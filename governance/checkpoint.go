// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package governance

import (
	"context"
	"encoding/hex"

	"code.vegaprotocol.io/protos/vega"
	checkpointpb "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types"

	"code.vegaprotocol.io/vega/libs/crypto"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/proto"
)

func (e *Engine) Name() types.CheckpointName {
	return types.GovernanceCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	if len(e.enactedProposals) == 0 {
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

	for _, p := range cp.Proposals {
		prop, err := types.ProposalFromProto(p)
		if err != nil {
			return err
		}

		switch prop.Terms.Change.GetTermType() {
		case types.ProposalTermsTypeNewMarket:
			// if the proposal is for a new market we want to restore it such that it will be in opening auction
			if p.Terms.EnactmentTimestamp <= now.Unix() {
				prop.Terms.EnactmentTimestamp = now.Add(duration).Unix()
			}
			toSubmit, err := e.intoToSubmit(ctx, prop)
			if err != nil {
				e.log.Panic("Failed to convert proposal into market")
			}
			nm := toSubmit.NewMarket()
			lpid := hex.EncodeToString(vgcrypto.Hash([]byte(nm.Market().ID)))
			if lp := nm.LiquidityProvisionSubmission(); lp != nil {
				deterministicID := crypto.HashStr(nm.m.ID + lpid)
				err = e.markets.RestoreMarketWithLiquidityProvision(
					ctx, nm.Market(), nm.LiquidityProvisionSubmission(), lpid, deterministicID)
			} else {
				err = e.markets.RestoreMarket(ctx, nm.Market())
			}
			if err != nil {
				if err == execution.ErrMarketDoesNotExist {
					// market has been settled, we don't care
					continue
				}
				// any other error, panic
				e.log.Panic("failed to restore market from checkpoint", logging.Market(*nm.Market()), logging.Error(err))
			}

			if err := e.markets.StartOpeningAuction(ctx, prop.ID); err != nil {
				e.log.Panic("failed to start opening auction for market", logging.String("market-id", prop.ID), logging.Error(err))
			}
		}

		evts = append(evts, events.NewProposalEvent(ctx, *prop))
		e.enactedProposals = append(e.enactedProposals, &proposal{
			Proposal: prop,
		})
	}
	// send events for restored proposals
	e.broker.SendBatch(evts)
	// @TODO ensure OnTick is called
	return nil
}

func (e *Engine) getCheckpointProposals() []*vega.Proposal {
	ret := make([]*vega.Proposal, 0, len(e.enactedProposals))
	for _, p := range e.enactedProposals {
		ret = append(ret, p.IntoProto())
	}
	return ret
}
