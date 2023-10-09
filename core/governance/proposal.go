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
	p                     *proposal
	m                     *ToEnactNewMarket
	s                     *ToEnactNewSpotMarket
	newAsset              *types.Asset
	updatedAsset          *types.Asset
	n                     *types.NetworkParameter
	as                    *types.AssetDetails
	updatedMarket         *types.Market
	updatedSpotMarket     *types.Market
	f                     *ToEnactFreeform
	t                     *ToEnactTransfer
	c                     *ToEnactCancelTransfer
	msu                   *ToEnactMarketStateUpdate
	referralProgram       *types.ReferralProgram
	volumeDiscountProgram *types.VolumeDiscountProgram
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
	return t.referralProgram != nil
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

func (t *ToEnact) ReferralProgramUpdate() *types.ReferralProgram {
	return t.referralProgram
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
