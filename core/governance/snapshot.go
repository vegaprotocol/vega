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
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"golang.org/x/exp/maps"
)

var (
	activeKey         = (&types.PayloadGovernanceActive{}).Key()
	enactedKey        = (&types.PayloadGovernanceEnacted{}).Key()
	nodeValidationKey = (&types.PayloadGovernanceNode{}).Key()
	batchActiveKey    = (&types.PayloadGovernanceBatchActive{}).Key()

	hashKeys = []string{
		activeKey,
		enactedKey,
		nodeValidationKey,
		batchActiveKey,
	}
	defaultMarkPriceConfig = &types.CompositePriceConfiguration{
		DecayWeight:        num.DecimalZero(),
		DecayPower:         num.DecimalZero(),
		CashAmount:         num.UintZero(),
		CompositePriceType: types.CompositePriceTypeByLastTrade,
	}
)

type governanceSnapshotState struct {
	serialisedActive         []byte
	serialisedEnacted        []byte
	serialisedNodeValidation []byte
	serialisedBatchActive    []byte
}

func (e *Engine) OnStateLoaded(ctx context.Context) error {
	// previously new market proposals that passed but where not enacted existed in both
	// the active and enacted slices, but now this has changed and it is only ever in one
	// or the other.

	// so for upgrade purposes any active proposals in the enacted slice needs to be removed
	// from the enacted slice
	for _, p := range e.activeProposals {
		for i := range e.enactedProposals {
			if p.ID == e.enactedProposals[i].ID {
				e.log.Warn("removing proposal from enacted since it is also in active", logging.String("id", p.ID))
				e.enactedProposals = append(e.enactedProposals[:i], e.enactedProposals[i+1:]...)
				break
			}
		}
	}

	// update market events may require updating to set the liquidation strategy slippage
	if vgcontext.InProgressUpgradeFromMultiple(ctx, "v0.75.8", "v0.75.7") {
		evts := make([]events.Event, 0, len(e.activeProposals)/2)
		for _, p := range e.activeProposals {
			if !p.Proposal.IsMarketUpdate() {
				continue
			}
			mID := p.Proposal.MarketUpdate().MarketID
			changes := p.Proposal.MarketUpdate().Changes
			if changes.LiquidationStrategy != nil && changes.LiquidationStrategy.DisposalSlippage.IsZero() {
				existingMarket, ok := e.markets.GetMarket(mID, false)
				if !ok {
					continue
				}
				// execution engine has already been restored at this point, so we can get the current slippage value from the market itself.
				changes.LiquidationStrategy.DisposalSlippage = existingMarket.LiquidationStrategy.DisposalSlippage
				evts = append(evts, events.NewProposalEvent(ctx, *p.Proposal))
			}
		}
		if len(evts) > 0 {
			e.broker.SendBatch(evts)
		}
	}

	return nil
}

// serialiseActiveProposals returns the engine's active proposals as marshalled bytes.
func (e *Engine) serialiseActiveProposals() ([]byte, error) {
	pending := make([]*types.ProposalData, 0, len(e.activeProposals))
	for _, p := range e.activeProposals {
		pp := &types.ProposalData{
			Proposal: p.Proposal,
			Yes:      votesAsSlice(p.yes),
			No:       votesAsSlice(p.no),
			Invalid:  votesAsSlice(p.invalidVotes),
		}
		pending = append(pending, pp)
	}

	pl := types.Payload{
		Data: &types.PayloadGovernanceActive{
			GovernanceActive: &types.GovernanceActive{
				Proposals: pending,
			},
		},
	}

	return proto.Marshal(pl.IntoProto())
}

// serialiseBatchActiveProposals returns the engine's batch active proposals as marshalled bytes.
func (e *Engine) serialiseBatchActiveProposals() ([]byte, error) {
	batchIDs := maps.Keys(e.activeBatchProposals)
	sort.Strings(batchIDs)

	batchProposals := make([]*snapshotpb.BatchProposalData, 0, len(batchIDs))
	for _, batchID := range batchIDs {
		batchProposal := e.activeBatchProposals[batchID]

		bpd := &snapshotpb.BatchProposalData{
			BatchProposal: &snapshotpb.ProposalData{
				Proposal: batchProposal.ToProto(),
				Yes:      votesAsProtoSlice(batchProposal.yes),
				No:       votesAsProtoSlice(batchProposal.no),
				Invalid:  votesAsProtoSlice(batchProposal.invalidVotes),
			},
			Proposals: make([]*vegapb.Proposal, 0, len(batchProposal.Proposals)),
		}

		for _, proposal := range batchProposal.Proposals {
			bpd.Proposals = append(bpd.Proposals, proposal.IntoProto())
		}

		batchProposals = append(batchProposals, bpd)
	}

	pl := types.Payload{
		Data: &types.PayloadGovernanceBatchActive{
			GovernanceBatchActive: &types.GovernanceBatchActive{
				BatchProposals: batchProposals,
			},
		},
	}

	return proto.Marshal(pl.IntoProto())
}

// serialiseEnactedProposals returns the engine's enacted proposals as marshalled bytes.
func (e *Engine) serialiseEnactedProposals() ([]byte, error) {
	enacted := make([]*types.ProposalData, 0, len(e.activeProposals))
	for _, p := range e.enactedProposals {
		pp := &types.ProposalData{
			Proposal: p.Proposal,
			Yes:      votesAsSlice(p.yes),
			No:       votesAsSlice(p.no),
			Invalid:  votesAsSlice(p.invalidVotes),
		}
		enacted = append(enacted, pp)
	}

	pl := types.Payload{
		Data: &types.PayloadGovernanceEnacted{
			GovernanceEnacted: &types.GovernanceEnacted{
				Proposals: enacted,
			},
		},
	}
	return proto.Marshal(pl.IntoProto())
}

// serialiseNodeProposals returns the engine's proposals waiting for node validation.
func (e *Engine) serialiseNodeProposals() ([]byte, error) {
	nodeProposals := e.nodeProposalValidation.getProposals()
	nodeBatchProposals := e.nodeProposalValidation.getBatchProposals()
	proposals := make([]*types.ProposalData, 0, len(nodeProposals))
	batchProposals := make([]*snapshotpb.BatchProposalData, 0, len(nodeBatchProposals))

	for _, np := range nodeProposals {
		proposals = append(proposals, &types.ProposalData{
			Proposal: np.Proposal,
			Yes:      votesAsSlice(np.yes),
			No:       votesAsSlice(np.no),
			Invalid:  votesAsSlice(np.invalidVotes),
		})
	}

	for _, proposal := range nodeBatchProposals {
		bp := &snapshotpb.BatchProposalData{
			BatchProposal: &snapshotpb.ProposalData{
				Proposal: proposal.ToProto(),
				Yes:      votesAsProtoSlice(proposal.yes),
				No:       votesAsProtoSlice(proposal.no),
				Invalid:  votesAsProtoSlice(proposal.invalidVotes),
			},
			Proposals: make([]*vegapb.Proposal, 0, len(proposal.Proposals)),
		}

		for _, proposal := range proposal.Proposals {
			bp.Proposals = append(bp.Proposals, proposal.IntoProto())
		}

		batchProposals = append(batchProposals, bp)
	}

	pl := types.Payload{
		Data: &types.PayloadGovernanceNode{
			GovernanceNode: &types.GovernanceNode{
				ProposalData:      proposals,
				BatchProposalData: batchProposals,
			},
		},
	}
	return proto.Marshal(pl.IntoProto())
}

func (e *Engine) serialiseK(serialFunc func() ([]byte, error), dataField *[]byte) ([]byte, error) {
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	*dataField = data
	return data, nil
}

func (e *Engine) serialise(k string) ([]byte, error) {
	switch k {
	case activeKey:
		return e.serialiseK(e.serialiseActiveProposals, &e.gss.serialisedActive)
	case enactedKey:
		return e.serialiseK(e.serialiseEnactedProposals, &e.gss.serialisedEnacted)
	case nodeValidationKey:
		return e.serialiseK(e.serialiseNodeProposals, &e.gss.serialisedNodeValidation)
	case batchActiveKey:
		return e.serialiseK(e.serialiseBatchActiveProposals, &e.gss.serialisedBatchActive)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.GovernanceSnapshot
}

func (e *Engine) Keys() []string {
	return hashKeys
}

func (e *Engine) Stopped() bool {
	return false
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, err := e.serialise(k)
	return data, nil, err
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := p.Data.(type) {
	case *types.PayloadGovernanceActive:
		return nil, e.restoreActiveProposals(ctx, pl.GovernanceActive, p)
	case *types.PayloadGovernanceEnacted:
		e.restoreEnactedProposals(ctx, pl.GovernanceEnacted, p)
		return nil, nil
	case *types.PayloadGovernanceNode:
		return nil, e.restoreNodeProposals(ctx, pl.GovernanceNode, p)
	case *types.PayloadGovernanceBatchActive:
		return nil, e.restoreBatchActiveProposals(ctx, pl.GovernanceBatchActive, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *Engine) restoreActiveProposals(ctx context.Context, active *types.GovernanceActive, p *types.Payload) error {
	e.activeProposals = make([]*proposal, 0, len(active.Proposals))
	evts := []events.Event{}
	vevts := []events.Event{}
	e.log.Debug("restoring active proposals snapshot", logging.Int("nproposals", len(active.Proposals)))
	for _, p := range active.Proposals {
		if vgcontext.InProgressUpgradeFromMultiple(ctx, "v0.75.8", "v0.75.7") {
			if p.Proposal.IsNewMarket() || p.Proposal.IsMarketUpdate() {
				setLiquidationSlippage(p.Proposal)
			}
		}
		pp := &proposal{
			Proposal:     p.Proposal,
			yes:          votesAsMap(p.Yes),
			no:           votesAsMap(p.No),
			invalidVotes: votesAsMap(p.Invalid),
		}

		e.log.Debug("proposals",
			logging.String("id", pp.ID),
			logging.Int("yes", len(pp.yes)),
			logging.Int("no", len(pp.no)),
			logging.Int("invalid", len(pp.invalidVotes)),
		)
		e.activeProposals = append(e.activeProposals, pp)
		evts = append(evts, events.NewProposalEvent(ctx, *pp.Proposal))

		for _, v := range pp.yes {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}
		for _, v := range pp.no {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}

		for _, v := range pp.invalidVotes {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}
	}

	var err error
	e.gss.serialisedActive, err = proto.Marshal(p.IntoProto())
	e.broker.SendBatch(evts)
	e.broker.SendBatch(vevts)
	return err
}

func setLiquidationSlippage(p *types.Proposal) {
	if p.IsNewMarket() {
		if !p.NewMarket().Changes.LiquidationStrategy.DisposalSlippage.IsZero() {
			return
		}
		changes := p.NewMarket().Changes
		changes.LiquidationStrategy.DisposalSlippage = changes.LiquiditySLAParameters.PriceRange
		return
	}
	// this must be a market update
	changes := p.MarketUpdate().Changes
	if changes.LiquidationStrategy != nil && changes.LiquiditySLAParameters != nil && changes.LiquidationStrategy.DisposalSlippage.IsZero() {
		changes.LiquidationStrategy.DisposalSlippage = changes.LiquiditySLAParameters.PriceRange
	}
}

func (e *Engine) restoreBatchActiveProposals(ctx context.Context, active *types.GovernanceBatchActive, p *types.Payload) error {
	e.activeBatchProposals = make(map[string]*batchProposal, len(active.BatchProposals))

	evts := []events.Event{}
	vevts := []events.Event{}
	e.log.Debug("restoring active proposals snapshot", logging.Int("nproposals", len(active.BatchProposals)))
	for _, bpp := range active.BatchProposals {
		bpt := types.BatchProposalFromSnapshotProto(bpp.BatchProposal.Proposal, bpp.Proposals)
		bp := &batchProposal{
			BatchProposal: bpt,
			yes:           votesAsMapFromProto(bpp.BatchProposal.Yes),
			no:            votesAsMapFromProto(bpp.BatchProposal.No),
			invalidVotes:  votesAsMapFromProto(bpp.BatchProposal.Invalid),
		}

		evts = append(evts, events.NewProposalEventFromProto(ctx, bp.BatchProposal.ToProto()))
		for _, p := range bp.BatchProposal.Proposals {
			if vgcontext.InProgressUpgradeFromMultiple(ctx, "v0.75.8", "v0.75.7") {
				if p.IsMarketUpdate() || p.IsNewMarket() {
					setLiquidationSlippage(p)
				}
			}
			evts = append(evts, events.NewProposalEvent(ctx, *p))
		}

		e.log.Debug("batch proposal",
			logging.String("id", bp.ID),
			logging.Int("yes", len(bp.yes)),
			logging.Int("no", len(bp.no)),
			logging.Int("invalid", len(bp.invalidVotes)),
		)

		e.activeBatchProposals[bp.ID] = bp

		for _, v := range bp.yes {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}
		for _, v := range bp.no {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}

		for _, v := range bp.invalidVotes {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}
	}

	var err error
	e.gss.serialisedBatchActive, err = proto.Marshal(p.IntoProto())
	e.broker.SendBatch(evts)
	e.broker.SendBatch(vevts)
	return err
}

func (e *Engine) restoreEnactedProposals(ctx context.Context, enacted *types.GovernanceEnacted, p *types.Payload) {
	evts := []events.Event{}
	vevts := []events.Event{}
	e.log.Debug("restoring enacted proposals snapshot", logging.Int("nproposals", len(enacted.Proposals)))
	for _, p := range enacted.Proposals {
		pp := &proposal{
			Proposal:     p.Proposal,
			yes:          votesAsMap(p.Yes),
			no:           votesAsMap(p.No),
			invalidVotes: votesAsMap(p.Invalid),
		}
		e.log.Debug("proposals",
			logging.String("id", pp.ID),
			logging.Int("yes", len(pp.yes)),
			logging.Int("no", len(pp.no)),
			logging.Int("invalid", len(pp.invalidVotes)),
		)
		e.enactedProposals = append(e.enactedProposals, pp)
		evts = append(evts, events.NewProposalEvent(ctx, *pp.Proposal))

		for _, v := range pp.yes {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}
		for _, v := range pp.no {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}

		for _, v := range pp.invalidVotes {
			vevts = append(vevts, events.NewVoteEvent(ctx, *v))
		}
	}
	e.gss.serialisedEnacted, _ = proto.Marshal(p.IntoProto())
	e.broker.SendBatch(evts)
	e.broker.SendBatch(vevts)
}

func (e *Engine) restoreNodeProposals(ctx context.Context, node *types.GovernanceNode, p *types.Payload) error {
	// node.Proposals should be empty for new snapshots because they are the old version that didn't include votes
	for _, p := range node.Proposals {
		e.nodeProposalValidation.restore(ctx, &types.ProposalData{Proposal: p})
		e.broker.Send(events.NewProposalEvent(ctx, *p))
	}

	for _, p := range node.ProposalData {
		e.nodeProposalValidation.restore(ctx, p)
		e.broker.Send(events.NewProposalEvent(ctx, *p.Proposal))
	}

	for _, p := range node.BatchProposalData {
		prop, _ := e.nodeProposalValidation.restoreBatch(ctx, p)
		e.broker.Send(events.NewProposalEventFromProto(ctx, prop.ToProto()))
	}

	var err error
	e.gss.serialisedNodeValidation, err = proto.Marshal(p.IntoProto())
	return err
}

// votesAsSlice returns a sorted slice of votes from a given map of votes.
func votesAsSlice(votes map[string]*types.Vote) []*types.Vote {
	ret := make([]*types.Vote, 0, len(votes))
	for _, v := range votes {
		ret = append(ret, v)
	}
	sort.SliceStable(ret, func(i, j int) bool { return ret[i].PartyID < ret[j].PartyID })
	return ret
}

// votesAsProtoSlice returns a sorted slice of proto votes from a given map of votes.
func votesAsProtoSlice(votes map[string]*types.Vote) []*vegapb.Vote {
	ret := make([]*vegapb.Vote, 0, len(votes))
	for _, v := range votes {
		ret = append(ret, v.IntoProto())
	}
	sort.SliceStable(ret, func(i, j int) bool { return ret[i].PartyId < ret[j].PartyId })
	return ret
}

// votesAsMap returns an partyID => Vote map from the given slice of votes.
func votesAsMap(votes []*types.Vote) map[string]*types.Vote {
	r := make(map[string]*types.Vote, len(votes))
	for _, v := range votes {
		r[v.PartyID] = v
	}
	return r
}

// votesAsMapFromProto returns an partyID => Vote map from the given slice of proto votes.
func votesAsMapFromProto(votes []*vegapb.Vote) map[string]*types.Vote {
	r := make(map[string]*types.Vote, len(votes))
	for _, v := range votes {
		v, _ := types.VoteFromProto(v)
		r[v.PartyID] = v
	}
	return r
}
