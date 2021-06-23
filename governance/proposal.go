package governance

import (
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

// ProposalParameters stores proposal specific parameters
type ProposalParameters struct {
	MinClose              time.Duration
	MaxClose              time.Duration
	MinEnact              time.Duration
	MaxEnact              time.Duration
	RequiredParticipation num.Decimal
	RequiredMajority      num.Decimal
	MinProposerBalance    *num.Uint
	MinVoterBalance       *num.Uint
}

// ToEnact wraps the proposal in a type that has a convenient interface
// to quickly work out what change we're dealing with, and get the data
type ToEnact struct {
	p  *types.Proposal
	m  *ToEnactMarket
	a  *types.Asset
	n  *types.NetworkParameter
	as *types.AssetDetails
	u  *types.UpdateMarket
}

// ToEnactMarket is just a empty struct, to signal
// an enacted market. nothing to be done with it
// for now (later maybe add information to check
// end of opening auction or so)
type ToEnactMarket struct{}

func (t ToEnact) IsNewMarket() bool {
	return t.m != nil
}

func (t ToEnact) IsNewAsset() bool {
	a := t.p.Terms.GetNewAsset()
	return a != nil
}

func (t ToEnact) IsUpdateMarket() bool {
	return t.u != nil
}

func (t ToEnact) IsUpdateNetworkParameter() bool {
	return t.n != nil
}

func (t ToEnact) IsNewAssetDetails() bool {
	return t.IsNewAsset()
}

func (t *ToEnact) NewMarket() *ToEnactMarket {
	return t.m
}

func (t *ToEnact) NewAsset() *types.Asset {
	return t.a
}

func (t *ToEnact) NewAssetDetails() *types.AssetDetails {
	return t.as
}

func (t *ToEnact) UpdateNetworkParameter() *types.NetworkParameter {
	return t.n
}

func (t *ToEnact) UpdateMarket() *types.UpdateMarket {
	return t.u
}

func (t *ToEnact) Proposal() *types.Proposal {
	return t.p
}

// ToSubmit wraps the proposal in a type that has a convenient interface
// to quickly work out what change we're dealing with, and get the data
// This cover every kind of proposal which requires action after a
// a proposal is submitted
type ToSubmit struct {
	p *types.Proposal
	m *ToSubmitNewMarket
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

type ToSubmitNewMarket struct {
	m *types.Market
	l *types.LiquidityProvisionSubmission
}

func (t *ToSubmitNewMarket) Market() *types.Market {
	return t.m
}

func (t *ToSubmitNewMarket) LiquidityProvisionSubmission() *types.LiquidityProvisionSubmission {
	return t.l
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
