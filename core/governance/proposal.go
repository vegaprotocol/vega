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
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type batchProposal struct {
	*types.BatchProposal
	yes          map[string]*types.Vote
	no           map[string]*types.Vote
	invalidVotes map[string]*types.Vote
}

// AddVote registers the last vote casted by a party. The proposal has to be
// open, it returns an error otherwise.
func (p *batchProposal) AddVote(vote types.Vote) error {
	if !p.IsOpenForVotes() {
		return ErrProposalNotOpenForVotes
	}

	if vote.Value == types.VoteValueYes {
		delete(p.no, vote.PartyID)
		p.yes[vote.PartyID] = &vote
	} else {
		delete(p.yes, vote.PartyID)
		p.no[vote.PartyID] = &vote
	}

	return nil
}

func (p *batchProposal) IsOpenForVotes() bool {
	// It's allowed to vote during the validation of the proposal by the node.
	return p.State == types.ProposalStateOpen || p.State == types.ProposalStateWaitingForNodeVote
}

type proposal struct {
	*types.Proposal
	yes          map[string]*types.Vote
	no           map[string]*types.Vote
	invalidVotes map[string]*types.Vote
}

// ShouldClose tells if the proposal should be closed or not.
// We also check the "open" state, alongside the closing timestamp as solely
// relying on the closing timestamp could lead to call Close() on an
// already-closed proposal.
func (p *proposal) ShouldClose(now int64) bool {
	return p.IsOpen() && p.Terms.ClosingTimestamp < now
}

func (p *proposal) IsTimeToEnact(now int64) bool {
	return p.Terms.EnactmentTimestamp < now
}

func (p *proposal) SucceedsMarket(parentID string) bool {
	nm := p.NewMarket()
	if nm == nil {
		return false
	}
	if pid, ok := nm.ParentMarketID(); !ok || pid != parentID {
		return false
	}
	return true
}

func (p *proposal) IsOpenForVotes() bool {
	// It's allowed to vote during the validation of the proposal by the node.
	return p.State == types.ProposalStateOpen || p.State == types.ProposalStateWaitingForNodeVote
}

// AddVote registers the last vote casted by a party. The proposal has to be
// open, it returns an error otherwise.
func (p *proposal) AddVote(vote types.Vote) error {
	if !p.IsOpenForVotes() {
		return ErrProposalNotOpenForVotes
	}

	if vote.Value == types.VoteValueYes {
		delete(p.no, vote.PartyID)
		p.yes[vote.PartyID] = &vote
	} else {
		delete(p.yes, vote.PartyID)
		p.no[vote.PartyID] = &vote
	}

	return nil
}

// Close determines the state of the proposal, passed or declined based on the
// vote balance and weight.
// Warning: this method should only be called once. Use ShouldClose() to know
// when to call.
func (p *proposal) Close(accounts StakingAccounts, markets Markets) {
	if !p.IsOpen() {
		return
	}

	defer func() {
		p.purgeBlankVotes(p.yes)
		p.purgeBlankVotes(p.no)
	}()

	tokenVoteState, tokenVoteError := p.computeVoteStateUsingTokens(accounts)

	p.State = tokenVoteState
	p.Reason = tokenVoteError

	// Proposals, other than market updates, solely relies on votes using the
	// governance tokens. So, only proposals for market update can go beyond this
	// guard.
	if !p.IsMarketUpdate() && !p.IsSpotMarketUpdate() {
		return
	}

	if tokenVoteState == types.ProposalStateDeclined && tokenVoteError == types.ProposalErrorParticipationThresholdNotReached {
		elsVoteState, elsVoteError := p.computeVoteStateUsingEquityLikeShare(markets)
		p.State = elsVoteState
		p.Reason = elsVoteError
	}
}

func (p *proposal) computeVoteStateUsingTokens(accounts StakingAccounts) (types.ProposalState, types.ProposalError) {
	totalStake := accounts.GetStakingAssetTotalSupply()

	yes := p.countTokens(p.yes, accounts)
	yesDec := num.DecimalFromUint(yes)
	no := p.countTokens(p.no, accounts)
	totalTokens := num.Sum(yes, no)
	totalTokensDec := num.DecimalFromUint(totalTokens)
	p.weightVotesFromToken(p.yes, totalTokensDec)
	p.weightVotesFromToken(p.no, totalTokensDec)
	majorityThreshold := totalTokensDec.Mul(p.RequiredMajority)
	totalStakeDec := num.DecimalFromUint(totalStake)
	participationThreshold := totalStakeDec.Mul(p.RequiredParticipation)

	// if we have 0 votes, then just return straight away,
	// prevents a proposal to go through if the participation is set to 0
	if totalTokens.IsZero() {
		return types.ProposalStateDeclined, types.ProposalErrorParticipationThresholdNotReached
	}

	if yesDec.GreaterThanOrEqual(majorityThreshold) && totalTokensDec.GreaterThanOrEqual(participationThreshold) {
		return types.ProposalStatePassed, types.ProposalErrorUnspecified
	}

	if totalTokensDec.LessThan(participationThreshold) {
		return types.ProposalStateDeclined, types.ProposalErrorParticipationThresholdNotReached
	}

	return types.ProposalStateDeclined, types.ProposalErrorMajorityThresholdNotReached
}

func (p *proposal) computeVoteStateUsingEquityLikeShare(markets Markets) (types.ProposalState, types.ProposalError) {
	yes := p.countEquityLikeShare(p.yes, markets)
	no := p.countEquityLikeShare(p.no, markets)
	totalEquityLikeShare := yes.Add(no)
	threshold := totalEquityLikeShare.Mul(p.RequiredLPMajority)

	if yes.GreaterThanOrEqual(threshold) && totalEquityLikeShare.GreaterThanOrEqual(p.RequiredLPParticipation) {
		return types.ProposalStatePassed, types.ProposalErrorUnspecified
	}

	if totalEquityLikeShare.LessThan(p.RequiredLPParticipation) {
		return types.ProposalStateDeclined, types.ProposalErrorParticipationThresholdNotReached
	}

	return types.ProposalStateDeclined, types.ProposalErrorMajorityThresholdNotReached
}

func (p *proposal) countTokens(votes map[string]*types.Vote, accounts StakingAccounts) *num.Uint {
	tally := num.UintZero()
	for _, v := range votes {
		v.TotalGovernanceTokenBalance = getTokensBalance(accounts, v.PartyID)
		tally.AddSum(v.TotalGovernanceTokenBalance)
	}

	return tally
}

func (p *proposal) countEquityLikeShare(votes map[string]*types.Vote, markets Markets) num.Decimal {
	tally := num.DecimalZero()
	for _, v := range votes {
		var marketID string
		if p.MarketUpdate() != nil {
			marketID = p.MarketUpdate().MarketID
		} else {
			marketID = p.SpotMarketUpdate().MarketID
		}
		v.TotalEquityLikeShareWeight, _ = markets.GetEquityLikeShareForMarketAndParty(marketID, v.PartyID)
		tally = tally.Add(v.TotalEquityLikeShareWeight)
	}

	return tally
}

func (p *proposal) weightVotesFromToken(votes map[string]*types.Vote, totalVotes num.Decimal) {
	if totalVotes.IsZero() {
		return
	}

	for _, v := range votes {
		tokenBalanceDec := num.DecimalFromUint(v.TotalGovernanceTokenBalance)
		v.TotalGovernanceTokenWeight = tokenBalanceDec.Div(totalVotes)
	}
}

// purgeBlankVotes removes votes that don't have tokens or equity-like share
// associated. The user may have withdrawn their governance token or their
// equity-like share before the end of the vote.
// We will then purge them from the map if it's the case.
func (p *proposal) purgeBlankVotes(votes map[string]*types.Vote) {
	for k, v := range votes {
		if v.TotalGovernanceTokenBalance.IsZero() && v.TotalEquityLikeShareWeight.IsZero() {
			p.invalidVotes[k] = v
			delete(votes, k)
			continue
		}
	}
}

// ToEnact wraps the proposal in a type that has a convenient interface
// to quickly work out what change we're dealing with, and get the data.
type ToEnact struct {
	p                      *proposal
	m                      *ToEnactNewMarket
	s                      *ToEnactNewSpotMarket
	newAsset               *types.Asset
	updatedAsset           *types.Asset
	n                      *types.NetworkParameter
	as                     *types.AssetDetails
	updatedMarket          *types.Market
	updatedSpotMarket      *types.Market
	f                      *ToEnactFreeform
	t                      *ToEnactTransfer
	c                      *ToEnactCancelTransfer
	msu                    *ToEnactMarketStateUpdate
	referralProgramChanges *types.ReferralProgram
	volumeDiscountProgram  *types.VolumeDiscountProgram
}

type ToEnactMarketStateUpdate struct{}

type ToEnactTransfer struct{}

type ToEnactCancelTransfer struct{}

// ToEnactNewMarket is just a empty struct, to signal
// an enacted market. nothing to be done with it
// for now (later maybe add information to check
// end of opening auction or so).
type ToEnactNewMarket struct{}

type ToEnactNewSpotMarket struct{}

// ToEnactFreeform there is nothing to enact with a freeform proposal.
type ToEnactFreeform struct{}

func (t ToEnact) IsVolumeDiscountProgramUpdate() bool {
	return t.volumeDiscountProgram != nil
}

func (t ToEnact) IsReferralProgramUpdate() bool {
	return t.referralProgramChanges != nil
}

func (t ToEnact) IsMarketStateUpdate() bool {
	return t.msu != nil
}

func (t ToEnact) IsCancelTransfer() bool {
	return t.c != nil
}

func (t ToEnact) IsNewTransfer() bool {
	return t.t != nil
}

func (t ToEnact) IsNewMarket() bool {
	return t.m != nil
}

func (t ToEnact) IsNewSpotMarket() bool {
	return t.s != nil
}

func (t ToEnact) IsNewAsset() bool {
	a := t.p.Terms.GetNewAsset()
	return a != nil
}

func (t ToEnact) IsUpdateMarket() bool {
	return t.updatedMarket != nil
}

func (t ToEnact) IsUpdateSpotMarket() bool {
	return t.updatedSpotMarket != nil
}

func (t ToEnact) IsUpdateNetworkParameter() bool {
	return t.n != nil
}

func (t ToEnact) IsNewAssetDetails() bool {
	return t.IsNewAsset()
}

func (t ToEnact) IsFreeform() bool {
	return t.f != nil
}

func (t *ToEnact) MarketStateUpdate() *ToEnactMarketStateUpdate {
	return t.msu
}

func (t *ToEnact) NewTransfer() *ToEnactTransfer {
	return t.t
}

func (t *ToEnact) CancelTransfer() *ToEnactCancelTransfer {
	return t.c
}

func (t *ToEnact) NewMarket() *ToEnactNewMarket {
	return t.m
}

func (t *ToEnact) NewAsset() *types.Asset {
	return t.newAsset
}

func (t *ToEnact) NewAssetDetails() *types.AssetDetails {
	return t.as
}

func (t *ToEnact) UpdateNetworkParameter() *types.NetworkParameter {
	return t.n
}

func (t *ToEnact) ReferralProgramChanges() *types.ReferralProgram {
	return t.referralProgramChanges
}

func (t *ToEnact) VolumeDiscountProgramUpdate() *types.VolumeDiscountProgram {
	return t.volumeDiscountProgram
}

func (t *ToEnact) UpdateMarket() *types.Market {
	return t.updatedMarket
}

func (t *ToEnact) UpdateSpotMarket() *types.Market {
	return t.updatedSpotMarket
}

func (t *ToEnact) NewFreeform() *ToEnactFreeform {
	return t.f
}

func (t *ToEnact) ProposalData() *proposal { //revive:disable:unexported-return
	return t.p
}

func (t *ToEnact) Proposal() *types.Proposal {
	return t.p.Proposal
}

func (t *ToEnact) IsUpdateAsset() bool {
	return t.updatedAsset != nil
}

func (t *ToEnact) UpdateAsset() *types.Asset {
	return t.updatedAsset
}

// ToSubmit wraps the proposal in a type that has a convenient interface
// to quickly work out what change we're dealing with, and get the data
// This cover every kind of proposal which requires action after a proposal
// is submitted.
type ToSubmit struct {
	p *types.Proposal
	m *ToSubmitNewMarket
	s *ToSubmitNewSpotMarket
}

func (t *ToSubmit) Proposal() *types.Proposal {
	return t.p
}

func (t ToSubmit) IsNewMarket() bool {
	return t.m != nil
}

func (t *ToSubmit) NewMarket() *ToSubmitNewMarket {
	return t.m
}

func (t ToSubmit) IsNewSpotMarket() bool {
	return t.s != nil
}

func (t *ToSubmit) NewSpotMarket() *ToSubmitNewSpotMarket {
	return t.s
}

type ToSubmitNewSpotMarket struct {
	m *types.Market
}

func (t *ToSubmitNewSpotMarket) Market() *types.Market {
	return t.m
}

func (t *ToSubmit) ParentMarketID() string {
	return t.m.m.ParentMarketID
}

func (t *ToSubmit) InsurancePoolFraction() *num.Decimal {
	if len(t.m.m.ParentMarketID) == 0 {
		return nil
	}
	ipf := t.m.m.InsurancePoolFraction
	return &ipf
}

type ToSubmitNewMarket struct {
	m   *types.Market
	oos time.Time // opening auction start
}

func (t *ToSubmitNewMarket) Market() *types.Market {
	return t.m
}

func (t *ToSubmitNewMarket) OpeningAuctionStart() time.Time {
	return t.oos
}

type VoteClosed struct {
	p *types.Proposal
	m *NewMarketVoteClosed
}

func (t *VoteClosed) Proposal() *types.Proposal {
	return t.p
}

func (t *VoteClosed) IsNewMarket() bool {
	return t.m != nil
}

func (t *VoteClosed) NewMarket() *NewMarketVoteClosed {
	return t.m
}

type NewMarketVoteClosed struct {
	// true if the auction is to be started
	// false if the vote did get a majority of true
	// and the market is to be rejected.
	startAuction bool
}

func (t *NewMarketVoteClosed) Rejected() bool {
	return !t.startAuction
}

func (t *NewMarketVoteClosed) StartAuction() bool {
	return t.startAuction
}
