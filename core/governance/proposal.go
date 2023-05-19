// Copyright (c) 2022 Gobalsky Labs Limited
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

package governance

import (
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

// ProposalParameters stores proposal specific parameters.
type ProposalParameters struct {
	MinClose                time.Duration
	MaxClose                time.Duration
	MinEnact                time.Duration
	MaxEnact                time.Duration
	RequiredParticipation   num.Decimal
	RequiredMajority        num.Decimal
	MinProposerBalance      *num.Uint
	MinVoterBalance         *num.Uint
	RequiredParticipationLP num.Decimal
	RequiredMajorityLP      num.Decimal
	MinEquityLikeShare      num.Decimal
}

// ToEnact wraps the proposal in a type that has a convenient interface
// to quickly work out what change we're dealing with, and get the data.
type ToEnact struct {
	p             *proposal
	m             *ToEnactNewMarket
	newAsset      *types.Asset
	updatedAsset  *types.Asset
	n             *types.NetworkParameter
	as            *types.AssetDetails
	updatedMarket *types.Market
	f             *ToEnactFreeform
}

// ToEnactNewMarket is just a empty struct, to signal
// an enacted market. nothing to be done with it
// for now (later maybe add information to check
// end of opening auction or so).
type ToEnactNewMarket struct{}

// ToEnactFreeform there is nothing to enact with a freeform proposal.
type ToEnactFreeform struct{}

func (t ToEnact) IsNewMarket() bool {
	return t.m != nil
}

func (t ToEnact) IsNewAsset() bool {
	a := t.p.Terms.GetNewAsset()
	return a != nil
}

func (t ToEnact) IsUpdateMarket() bool {
	return t.updatedMarket != nil
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

func (t *ToEnact) UpdateMarket() *types.Market {
	return t.updatedMarket
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
	m              *types.Market
	insuranceShare *num.Decimal
	succeeds       string
}

func (t *ToSubmitNewMarket) Market() *types.Market {
	return t.m
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
