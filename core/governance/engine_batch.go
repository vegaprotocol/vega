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
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	vgerrors "code.vegaprotocol.io/vega/libs/errors"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"golang.org/x/exp/maps"
)

func (e *Engine) SubmitBatchProposal(
	ctx context.Context,
	bpsub types.BatchProposalSubmission,
	batchID, party string,
) ([]*ToSubmit, error) {
	if _, ok := e.getBatchProposal(batchID); ok {
		return nil, ErrProposalIsDuplicate // state is not allowed to change externally
	}

	timeNow := e.timeService.GetTimeNow().UnixNano()

	bp := &types.BatchProposal{
		ID:               batchID,
		Timestamp:        timeNow,
		ClosingTimestamp: bpsub.Terms.ClosingTimestamp,
		Party:            party,
		State:            types.ProposalStateOpen,
		Reference:        bpsub.Reference,
		Rationale:        bpsub.Rationale,
		Proposals:        make([]*types.Proposal, 0, len(bpsub.Terms.Changes)),
	}

	var proposalsEvents []events.Event //nolint:prealloc
	defer func() {
		e.broker.Send(events.NewProposalEventFromProto(ctx, bp.ToProto()))

		if len(proposalsEvents) > 0 {
			e.broker.SendBatch(proposalsEvents)
		}
	}()

	proposalParamsPerProposalTermType := map[types.ProposalTermsType]*types.ProposalParameters{}

	for _, change := range bpsub.Terms.Changes {
		p := &types.Proposal{
			ID:        change.ID,
			BatchID:   &batchID,
			Timestamp: timeNow,
			Party:     party,
			State:     bp.State,
			Reference: bp.Reference,
			Rationale: bp.Rationale,
			Terms: &types.ProposalTerms{
				ClosingTimestamp:   bp.ClosingTimestamp,
				EnactmentTimestamp: change.EnactmentTime,
				Change:             change.Change,
			},
		}

		params, err := e.getProposalParams(change.Change)
		if err != nil {
			bp.RejectWithErr(types.ProposalErrorUnknownType, err)
			return nil, err
		}

		proposalParamsPerProposalTermType[change.Change.GetTermType()] = params

		bp.Proposals = append(bp.Proposals, p)
	}

	var toSubmits []*ToSubmit //nolint:prealloc
	errs := vgerrors.NewCumulatedErrors()

	for _, p := range bp.Proposals {
		perTypeParams := proposalParamsPerProposalTermType[p.Terms.Change.GetTermType()]
		submit, err := e.validateProposalFromBatch(ctx, p, perTypeParams)
		if err != nil {
			errs.Add(err)
			continue
		}

		toSubmits = append(toSubmits, submit)
	}

	for _, p := range bp.Proposals {
		if !p.IsRejected() && errs.HasAny() {
			p.Reject(types.ProposalErrorProposalInBatchRejected)
		}

		proposalsEvents = append(proposalsEvents, events.NewProposalEvent(ctx, *p))
	}

	if errs.HasAny() {
		bp.State = types.ProposalStateRejected
		bp.Reason = types.ProposalErrorProposalInBatchRejected

		return nil, errs
	}

	e.startBatchProposal(bp)

	return toSubmits, nil
}

func (e *Engine) RejectBatchProposal(
	ctx context.Context, proposalID string, r types.ProposalError, errorDetails error,
) error {
	bp, ok := e.getBatchProposal(proposalID)
	if !ok {
		return ErrProposalDoesNotExist
	}

	bp.RejectWithErr(r, errorDetails)

	evts := make([]events.Event, 0, len(bp.Proposals))
	for _, proposal := range bp.Proposals {
		e.rejectProposal(ctx, proposal, r, errorDetails)
		evts = append(evts, events.NewProposalEvent(ctx, *proposal))
	}

	e.broker.Send(events.NewProposalEventFromProto(ctx, bp.ToProto()))
	e.broker.SendBatch(evts)
	return nil
}

func (e *Engine) evaluateBatchProposals(
	ctx context.Context, now int64,
) (voteClosed []*VoteClosed, addToActiveProposals []*proposal) {
	batchIDs := maps.Keys(e.activeBatchProposals)
	sort.Strings(batchIDs)

	for _, batchID := range batchIDs {
		batchProposal := e.activeBatchProposals[batchID]

		var batchHasRejectedProposal bool
		var batchHasDeclinedProposal bool
		var closedProposals []*proposal
		for _, propType := range batchProposal.Proposals {
			proposal := &proposal{
				Proposal:     propType,
				yes:          batchProposal.yes,
				no:           batchProposal.no,
				invalidVotes: map[string]*types.Vote{},
			}

			// check if the market for successor proposals still exists, if not, reject the proposal
			// in case a single proposal is rejected we can reject the whole batch
			if nm := proposal.Terms.GetNewMarket(); nm != nil && nm.Successor() != nil {
				if _, err := e.markets.GetMarketState(proposal.ID); err != nil {
					proposal.RejectWithErr(types.ProposalErrorInvalidSuccessorMarket,
						ErrParentMarketSucceededByCompeting)
					batchHasRejectedProposal = true
					break
				}
			}

			// do not check parent market, the market was either rejected when the parent was succeeded
			// or, if the parent market state is gone (ie succession window has expired), the proposal simply
			// loses its parent market reference
			if proposal.ShouldClose(now) {
				proposal.Close(e.accs, e.markets)
				if proposal.IsPassed() {
					e.log.Debug("Proposal passed",
						logging.ProposalID(proposal.ID),
						logging.ProposalBatchID(batchID),
					)
				} else if proposal.IsDeclined() {
					e.log.Debug("Proposal declined",
						logging.ProposalID(proposal.ID),
						logging.String("details", proposal.ErrorDetails),
						logging.String("reason", proposal.Reason.String()),
						logging.ProposalBatchID(batchID),
					)
					batchHasDeclinedProposal = true
				}

				closedProposals = append(closedProposals, proposal)
				voteClosed = append(voteClosed, e.preVoteClosedProposal(proposal))
			}
		}

		if batchHasRejectedProposal {
			batchProposal.State = types.ProposalStateRejected
			batchProposal.Reason = types.ProposalErrorProposalInBatchRejected

			proposalsEvents := make([]events.Event, 0, len(batchProposal.Proposals))
			for _, proposal := range batchProposal.Proposals {
				if proposal.IsPassed() {
					proposal.Reject(types.ProposalErrorProposalInBatchRejected)
				}

				proposalsEvents = append(proposalsEvents, events.NewProposalEvent(ctx, *proposal))
			}

			e.broker.Send(events.NewProposalEventFromProto(ctx, batchProposal.ToProto()))
			e.broker.SendBatch(proposalsEvents)

			delete(e.activeBatchProposals, batchProposal.ID)
			continue
		}

		if len(closedProposals) < 1 {
			continue
		}

		// all the proposal in the batch should close at the same time so this should never happen
		if len(closedProposals) != len(batchProposal.Proposals) {
			e.log.Panic("Failed to close all proposals in batch proposal",
				logging.ProposalBatchID(batchID),
			)
		}

		proposalEvents := make([]events.Event, 0, len(closedProposals))
		for _, proposal := range closedProposals {
			if proposal.IsPassed() && batchHasDeclinedProposal {
				proposal.Decline(types.ProposalErrorProposalInBatchDeclined)
			} else if proposal.IsPassed() {
				addToActiveProposals = append(addToActiveProposals, proposal)
			}

			proposalEvents = append(proposalEvents, events.NewProposalEvent(ctx, *proposal.Proposal))
			proposalEvents = append(proposalEvents, newUpdatedProposalEvents(ctx, proposal)...)
		}

		batchProposal.State = types.ProposalStatePassed
		if batchHasDeclinedProposal {
			batchProposal.State = types.ProposalStateDeclined
			batchProposal.Reason = types.ProposalErrorProposalInBatchDeclined
		}

		e.broker.Send(events.NewProposalEventFromProto(ctx, batchProposal.ToProto()))
		e.broker.SendBatch(proposalEvents)
		delete(e.activeBatchProposals, batchProposal.ID)
	}

	return
}

func (e *Engine) getBatchProposal(id string) (*batchProposal, bool) {
	bp, ok := e.activeBatchProposals[id]
	return bp, ok
}

func (e *Engine) validateProposalFromBatch(
	ctx context.Context,
	p *types.Proposal,
	params *types.ProposalParameters,
) (*ToSubmit, error) {
	if proposalErr, err := e.validateOpenProposal(p, params); err != nil {
		p.RejectWithErr(proposalErr, err)

		if e.log.IsDebug() {
			e.log.Debug("Batch proposal rejected",
				logging.String("proposal-id", p.ID),
				logging.String("proposal details", p.String()),
				logging.Error(err),
			)
		}

		return nil, err
	}

	submit, err := e.intoToSubmit(ctx, p, &enactmentTime{current: p.Terms.EnactmentTimestamp}, false)
	if err != nil {
		if e.log.IsDebug() {
			e.log.Debug("Batch proposal rejected",
				logging.String("proposal-id", p.ID),
				logging.String("proposal details", p.String()),
				logging.Error(err),
			)
		}
		return nil, err
	}

	return submit, nil
}

func (e *Engine) startBatchProposal(p *types.BatchProposal) {
	e.activeBatchProposals[p.ID] = &batchProposal{
		BatchProposal: p,
		yes:           map[string]*types.Vote{},
		no:            map[string]*types.Vote{},
		invalidVotes:  map[string]*types.Vote{},
	}
}

func (e *Engine) addBatchVote(ctx context.Context, batchProposal *batchProposal, cmd types.VoteSubmission, party string) error {
	validationErrs := vgerrors.NewCumulatedErrors()

	proposalParamsPerProposalTermType := map[types.ProposalTermsType]*types.ProposalParameters{}
	for _, proposal := range batchProposal.Proposals {
		params, ok := proposalParamsPerProposalTermType[proposal.Terms.Change.GetTermType()]
		if !ok {
			p, err := e.getProposalParams(proposal.Terms.Change)
			if err != nil {
				validationErrs.Add(fmt.Errorf("proposal term %q has failed with: %w", proposal.Terms.Change.GetTermType(), err))
				continue
			}
			proposalParamsPerProposalTermType[proposal.Terms.Change.GetTermType()] = p
			params = p
		}

		if err := e.canVote(proposal, params, party); err != nil {
			validationErrs.Add(fmt.Errorf("proposal term %q has failed with: %w", proposal.Terms.Change.GetTermType(), err))
			continue
		}
	}

	if validationErrs.HasAny() {
		e.log.Debug("invalid vote submission",
			logging.PartyID(party),
			logging.String("vote", cmd.String()),
			logging.Error(validationErrs),
		)
		return validationErrs
	}

	vote := types.Vote{
		PartyID:                     party,
		ProposalID:                  cmd.ProposalID,
		Value:                       cmd.Value,
		Timestamp:                   e.timeService.GetTimeNow().UnixNano(),
		TotalGovernanceTokenBalance: getTokensBalance(e.accs, party),
		TotalGovernanceTokenWeight:  num.DecimalZero(),
		TotalEquityLikeShareWeight:  num.DecimalZero(),
	}

	if err := batchProposal.AddVote(vote); err != nil {
		return fmt.Errorf("couldn't cast the vote: %w", err)
	}

	if e.log.IsDebug() {
		e.log.Debug("vote submission accepted",
			logging.PartyID(party),
			logging.String("vote", cmd.String()),
		)
	}
	e.broker.Send(events.NewVoteEvent(ctx, vote))

	return nil
}
