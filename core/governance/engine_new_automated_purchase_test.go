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

package governance_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestProposalForNewProtocolAutomatedPurchase(t *testing.T) {
	t.Run("Submitting a proposal for new automated purchase succeeds", testSubmittingProposalForNewProtocolAutomatedPurchaseSucceeds)
	t.Run("Submitting a proposal for new automated purchase with invalid asset fails", testSubmittingProposalForNewProtocolAutomatedPurchaseInvalidAssetFails)
	t.Run("Submitting a proposal for new automated purchase with invalid market fails", testSubmittingProposalForNewProtocolAutomatedPurchaseInvalidMarketFails)
	t.Run("Submitting a proposal for new automated purchase with a market that is not a spot market fails", testSubmittingProposalForNewProtocolAutomatedPurchaseNotSpotMarketFails)
	t.Run("Submitting a proposal for new automated purchase with invalid asset for spot market fails", testSubmittingProposalForNewProtocolAutomatedPurchaseInvalidAssetForMarketFails)
	t.Run("Submitting a proposal for new automated purchase to a stopped market fails", testSubmittingProposalForNewProtocolAutomatedPurchaseStoppedMarketWithStateFails)
	t.Run("Submitting a proposal for new automated purchase to a market which already has an active pap fails", testSubmittingPAPToMarketWithActivePAPFails)
}

func testSubmittingProposalForNewProtocolAutomatedPurchaseSucceeds(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinProposerBalance, "1000")

	eng.markets.EXPECT().GetMarket(gomock.Any(), gomock.Any()).Return(types.Market{
		TradableInstrument: types.TradableInstrumentFromProto(&vega.TradableInstrument{
			RiskModel: &vega.TradableInstrument_SimpleRiskModel{
				SimpleRiskModel: &vega.SimpleRiskModel{
					Params: &vega.SimpleModelParams{},
				},
			},
			Instrument: &vega.Instrument{
				Product: &vega.Instrument_Spot{
					Spot: &vega.Spot{
						BaseAsset:  "base",
						QuoteAsset: "quote",
					},
				},
				Metadata: &vega.InstrumentMetadata{},
			},
		}),
	}, true).AnyTimes()
	eng.assets.EXPECT().IsEnabled(gomock.Any()).Return(true).AnyTimes()
	eng.markets.EXPECT().MarketHasActivePAP(gomock.Any()).Return(false, nil).AnyTimes()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewProtocolAutomatedPurchase(proposer, now, &types.NewProtocolAutomatedPurchaseChanges{
		ExpiryTimestamp: now.Add(4 * 48 * time.Hour),
		From:            "base",
		FromAccountType: types.AccountTypeBuyBackFees,
		ToAccountType:   types.AccountTypeBuyBackFees,
		MarketID:        crypto.RandomHash(),
		PriceOracle:     &vega.DataSourceDefinition{},
		PriceOracleBinding: &vega.SpecBindingForCompositePrice{
			PriceSourceProperty: "oracle.price",
		},
		OracleOffsetFactor:            num.DecimalFromFloat(0.1),
		AuctionSchedule:               &vega.DataSourceDefinition{},
		AuctionVolumeSnapshotSchedule: &vega.DataSourceDefinition{},
		AutomatedPurchaseSpecBinding:  &vega.DataSourceSpecToAutomatedPurchaseBinding{},
		AuctionDuration:               time.Hour,
		MinimumAuctionSize:            num.NewUint(1000),
		MaximumAuctionSize:            num.NewUint(2000),
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
}

func setupPAP(t *testing.T) (*tstEngine, types.Proposal) {
	t.Helper()
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinProposerBalance, "1000")

	eng.markets.EXPECT().GetMarket(gomock.Any(), gomock.Any()).Return(types.Market{
		TradableInstrument: types.TradableInstrumentFromProto(&vega.TradableInstrument{
			RiskModel: &vega.TradableInstrument_SimpleRiskModel{
				SimpleRiskModel: &vega.SimpleRiskModel{
					Params: &vega.SimpleModelParams{},
				},
			},
			Instrument: &vega.Instrument{
				Product: &vega.Instrument_Spot{
					Spot: &vega.Spot{
						BaseAsset:  "base",
						QuoteAsset: "quote",
					},
				},
				Metadata: &vega.InstrumentMetadata{},
			},
		}),
	}, true).AnyTimes()
	eng.assets.EXPECT().IsEnabled(gomock.Any()).Return(true).AnyTimes()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewProtocolAutomatedPurchase(proposer, now, &types.NewProtocolAutomatedPurchaseChanges{
		ExpiryTimestamp: now.Add(4 * 48 * time.Hour),
		From:            "base",
		FromAccountType: types.AccountTypeBuyBackFees,
		ToAccountType:   types.AccountTypeBuyBackFees,
		MarketID:        crypto.RandomHash(),
		PriceOracle:     &vega.DataSourceDefinition{},
		PriceOracleBinding: &vega.SpecBindingForCompositePrice{
			PriceSourceProperty: "oracle.price",
		},
		OracleOffsetFactor:            num.DecimalFromFloat(0.1),
		AuctionSchedule:               &vega.DataSourceDefinition{},
		AuctionVolumeSnapshotSchedule: &vega.DataSourceDefinition{},
		AutomatedPurchaseSpecBinding:  &vega.DataSourceSpecToAutomatedPurchaseBinding{},
		AuctionDuration:               time.Hour,
		MinimumAuctionSize:            num.NewUint(1000),
		MaximumAuctionSize:            num.NewUint(2000),
	})
	return eng, proposal
}

func testSubmittingPAPToMarketWithActivePAPFails(t *testing.T) {
	eng, proposal := setupPAP(t)
	// setup
	eng.markets.EXPECT().MarketHasActivePAP(gomock.Any()).Return(false, nil).Times(1)
	eng.ensureTokenBalanceForParty(t, proposal.Party, 1000)

	// expect
	eng.expectOpenProposalEvent(t, proposal.Party, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)

	// now resubmit to the same market when this one is already active
	proposal.ID = crypto.RandomHash()
	eng.markets.EXPECT().MarketHasActivePAP(gomock.Any()).Return(true, nil).Times(1)
	eng.ensureTokenBalanceForParty(t, proposal.Party, 1000)
	// expect
	eng.expectRejectedProposalEvent(t, proposal.Party, proposal.ID, types.ProposalErrorInvalidMarket)

	// when
	_, err = eng.submitProposal(t, proposal)

	// then
	require.Equal(t, "market already has an active protocol automated purchase program", err.Error())
}

func testSubmittingProposalForNewProtocolAutomatedPurchaseInvalidAssetFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinProposerBalance, "1000")

	eng.markets.EXPECT().GetMarket(gomock.Any(), gomock.Any()).Return(types.Market{
		TradableInstrument: types.TradableInstrumentFromProto(&vega.TradableInstrument{
			RiskModel: &vega.TradableInstrument_SimpleRiskModel{
				SimpleRiskModel: &vega.SimpleRiskModel{
					Params: &vega.SimpleModelParams{},
				},
			},
			Instrument: &vega.Instrument{
				Product: &vega.Instrument_Spot{
					Spot: &vega.Spot{
						BaseAsset:  "base",
						QuoteAsset: "quote",
					},
				},
				Metadata: &vega.InstrumentMetadata{},
			},
		}),
	}, true).AnyTimes()
	eng.assets.EXPECT().IsEnabled(gomock.Any()).Return(false).AnyTimes()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewProtocolAutomatedPurchase(proposer, now, &types.NewProtocolAutomatedPurchaseChanges{
		ExpiryTimestamp: now.Add(4 * 48 * time.Hour),
		From:            "base",
		FromAccountType: types.AccountTypeBuyBackFees,
		ToAccountType:   types.AccountTypeBuyBackFees,
		MarketID:        crypto.RandomHash(),
		PriceOracle:     &vega.DataSourceDefinition{},
		PriceOracleBinding: &vega.SpecBindingForCompositePrice{
			PriceSourceProperty: "oracle.price",
		},
		OracleOffsetFactor:            num.DecimalFromFloat(0.1),
		AuctionSchedule:               &vega.DataSourceDefinition{},
		AuctionVolumeSnapshotSchedule: &vega.DataSourceDefinition{},
		AutomatedPurchaseSpecBinding:  &vega.DataSourceSpecToAutomatedPurchaseBinding{},
		AuctionDuration:               time.Hour,
		MinimumAuctionSize:            num.NewUint(1000),
		MaximumAuctionSize:            num.NewUint(2000),
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidAsset)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	require.Equal(t, "asset does not exist", err.Error())
}

func testSubmittingProposalForNewProtocolAutomatedPurchaseInvalidMarketFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinProposerBalance, "1000")

	eng.markets.EXPECT().GetMarket(gomock.Any(), gomock.Any()).Return(types.Market{}, false).AnyTimes()
	eng.assets.EXPECT().IsEnabled(gomock.Any()).Return(true).AnyTimes()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewProtocolAutomatedPurchase(proposer, now, &types.NewProtocolAutomatedPurchaseChanges{
		ExpiryTimestamp: now.Add(4 * 48 * time.Hour),
		From:            "base",
		FromAccountType: types.AccountTypeBuyBackFees,
		ToAccountType:   types.AccountTypeBuyBackFees,
		MarketID:        crypto.RandomHash(),
		PriceOracle:     &vega.DataSourceDefinition{},
		PriceOracleBinding: &vega.SpecBindingForCompositePrice{
			PriceSourceProperty: "oracle.price",
		},
		OracleOffsetFactor:            num.DecimalFromFloat(0.1),
		AuctionSchedule:               &vega.DataSourceDefinition{},
		AuctionVolumeSnapshotSchedule: &vega.DataSourceDefinition{},
		AutomatedPurchaseSpecBinding:  &vega.DataSourceSpecToAutomatedPurchaseBinding{},
		AuctionDuration:               time.Hour,
		MinimumAuctionSize:            num.NewUint(1000),
		MaximumAuctionSize:            num.NewUint(2000),
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidMarket)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	require.Equal(t, "market does not exist", err.Error())
}

func testSubmittingProposalForNewProtocolAutomatedPurchaseInvalidAssetForMarketFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinProposerBalance, "1000")

	eng.markets.EXPECT().GetMarket(gomock.Any(), gomock.Any()).Return(types.Market{
		TradableInstrument: types.TradableInstrumentFromProto(&vega.TradableInstrument{
			RiskModel: &vega.TradableInstrument_SimpleRiskModel{
				SimpleRiskModel: &vega.SimpleRiskModel{
					Params: &vega.SimpleModelParams{},
				},
			},
			Instrument: &vega.Instrument{
				Product: &vega.Instrument_Spot{
					Spot: &vega.Spot{
						BaseAsset:  "base",
						QuoteAsset: "quote",
					},
				},
				Metadata: &vega.InstrumentMetadata{},
			},
		}),
	}, true).AnyTimes()
	eng.assets.EXPECT().IsEnabled(gomock.Any()).Return(true).AnyTimes()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewProtocolAutomatedPurchase(proposer, now, &types.NewProtocolAutomatedPurchaseChanges{
		ExpiryTimestamp: now.Add(4 * 48 * time.Hour),
		From:            "neither_base_nor_quote",
		FromAccountType: types.AccountTypeBuyBackFees,
		ToAccountType:   types.AccountTypeBuyBackFees,
		MarketID:        crypto.RandomHash(),
		PriceOracle:     &vega.DataSourceDefinition{},
		PriceOracleBinding: &vega.SpecBindingForCompositePrice{
			PriceSourceProperty: "oracle.price",
		},
		OracleOffsetFactor:            num.DecimalFromFloat(0.1),
		AuctionSchedule:               &vega.DataSourceDefinition{},
		AuctionVolumeSnapshotSchedule: &vega.DataSourceDefinition{},
		AutomatedPurchaseSpecBinding:  &vega.DataSourceSpecToAutomatedPurchaseBinding{},
		AuctionDuration:               time.Hour,
		MinimumAuctionSize:            num.NewUint(1000),
		MaximumAuctionSize:            num.NewUint(2000),
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidMarket)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	require.Equal(t, "mismatch between asset for automated purchase and the spot market configuration - asset is not one of base/quote assets of the market", err.Error())
}

func testSubmittingProposalForNewProtocolAutomatedPurchaseNotSpotMarketFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinProposerBalance, "1000")

	eng.markets.EXPECT().GetMarket(gomock.Any(), gomock.Any()).Return(types.Market{
		TradableInstrument: types.TradableInstrumentFromProto(&vega.TradableInstrument{
			RiskModel: &vega.TradableInstrument_SimpleRiskModel{
				SimpleRiskModel: &vega.SimpleRiskModel{
					Params: &vega.SimpleModelParams{},
				},
			},
			Instrument: &vega.Instrument{
				Product: &vega.Instrument_Future{
					Future: &vega.Future{
						SettlementAsset:                     "some_future",
						DataSourceSpecForSettlementData:     &vega.DataSourceSpec{},
						DataSourceSpecForTradingTermination: &vega.DataSourceSpec{},
						DataSourceSpecBinding:               &vega.DataSourceSpecToFutureBinding{},
					},
				},
				Metadata: &vega.InstrumentMetadata{},
			},
		}),
	}, true).AnyTimes()
	eng.assets.EXPECT().IsEnabled(gomock.Any()).Return(true).AnyTimes()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewProtocolAutomatedPurchase(proposer, now, &types.NewProtocolAutomatedPurchaseChanges{
		ExpiryTimestamp: now.Add(4 * 48 * time.Hour),
		From:            "neither_base_nor_quote",
		FromAccountType: types.AccountTypeBuyBackFees,
		ToAccountType:   types.AccountTypeBuyBackFees,
		MarketID:        crypto.RandomHash(),
		PriceOracle:     &vega.DataSourceDefinition{},
		PriceOracleBinding: &vega.SpecBindingForCompositePrice{
			PriceSourceProperty: "oracle.price",
		},
		OracleOffsetFactor:            num.DecimalFromFloat(0.1),
		AuctionSchedule:               &vega.DataSourceDefinition{},
		AuctionVolumeSnapshotSchedule: &vega.DataSourceDefinition{},
		AutomatedPurchaseSpecBinding:  &vega.DataSourceSpecToAutomatedPurchaseBinding{},
		AuctionDuration:               time.Hour,
		MinimumAuctionSize:            num.NewUint(1000),
		MaximumAuctionSize:            num.NewUint(2000),
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidMarket)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	require.Equal(t, "market for automated purchase must be a spot market", err.Error())
}

func testSubmittingProposalForNewProtocolAutomatedPurchaseStoppedMarketFailes(t *testing.T, state types.MarketState) {
	t.Helper()
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalAutomatedPurchaseConfigMinProposerBalance, "1000")

	eng.markets.EXPECT().GetMarket(gomock.Any(), gomock.Any()).Return(types.Market{
		State: state,
		TradableInstrument: types.TradableInstrumentFromProto(&vega.TradableInstrument{
			RiskModel: &vega.TradableInstrument_SimpleRiskModel{
				SimpleRiskModel: &vega.SimpleRiskModel{
					Params: &vega.SimpleModelParams{},
				},
			},
			Instrument: &vega.Instrument{
				Product: &vega.Instrument_Spot{
					Spot: &vega.Spot{
						BaseAsset:  "base",
						QuoteAsset: "quote",
					},
				},
				Metadata: &vega.InstrumentMetadata{},
			},
		}),
	}, true).AnyTimes()
	eng.assets.EXPECT().IsEnabled(gomock.Any()).Return(true).AnyTimes()
	eng.markets.EXPECT().MarketHasActivePAP(gomock.Any()).Return(false, nil).AnyTimes()

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForNewProtocolAutomatedPurchase(proposer, now, &types.NewProtocolAutomatedPurchaseChanges{
		ExpiryTimestamp: now.Add(4 * 48 * time.Hour),
		From:            "base",
		FromAccountType: types.AccountTypeBuyBackFees,
		ToAccountType:   types.AccountTypeBuyBackFees,
		MarketID:        crypto.RandomHash(),
		PriceOracle:     &vega.DataSourceDefinition{},
		PriceOracleBinding: &vega.SpecBindingForCompositePrice{
			PriceSourceProperty: "oracle.price",
		},
		OracleOffsetFactor:            num.DecimalFromFloat(0.1),
		AuctionSchedule:               &vega.DataSourceDefinition{},
		AuctionVolumeSnapshotSchedule: &vega.DataSourceDefinition{},
		AutomatedPurchaseSpecBinding:  &vega.DataSourceSpecToAutomatedPurchaseBinding{},
		AuctionDuration:               time.Hour,
		MinimumAuctionSize:            num.NewUint(1000),
		MaximumAuctionSize:            num.NewUint(2000),
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidMarket)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	require.Equal(t, "market for automated purchase must be active", err.Error())
}

func testSubmittingProposalForNewProtocolAutomatedPurchaseStoppedMarketWithStateFails(t *testing.T) {
	stoppedStates := []types.MarketState{types.MarketStateCancelled, types.MarketStateClosed, types.MarketStateRejected, types.MarketStateSettled, types.MarketStateTradingTerminated}
	for _, state := range stoppedStates {
		testSubmittingProposalForNewProtocolAutomatedPurchaseStoppedMarketFailes(t, state)
	}
}
