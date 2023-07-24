package governance_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/types"
	"github.com/stretchr/testify/require"
)

func TestSubmittingProposalForTerminateMarketSucceeds(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newProposalForUpdateMarketState(party.Id, eng.tsvc.GetTimeNow().Add(2*time.Hour), types.MarketStateUpdateTypeTerminate, nil)

	// setup
	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, party.Id, proposal.ID)
	eng.ensureEquityLikeShareForMarketAndParty(t, proposal.UpdateMarketState().Changes.MarketID, party.Id, 0.1)
	eng.markets.EXPECT().MarketExists(proposal.UpdateMarketState().Changes.MarketID).Times(2).Return(true)
	eng.markets.EXPECT().GetMarketState(proposal.UpdateMarketState().Changes.MarketID).Times(1).Return(types.MarketStateActive, nil)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
	require.True(t, toSubmit.Proposal().IsMarketStateUpdate())
}

func TestSubmittingProposalForUpdateMarketStateInTerminalStateFails(t *testing.T) {
	states := []types.MarketState{
		types.MarketStateCancelled, types.MarketStateClosed, types.MarketStateTradingTerminated, types.MarketStateSettled, types.MarketStateProposed,
	}
	updateType := []types.MarketStateUpdateType{types.MarketStateUpdateTypeResume, types.MarketStateUpdateTypeSuspend, types.MarketStateUpdateTypeTerminate}
	// given
	eng := getTestEngine(t, time.Now())
	party := eng.newValidParty("a-valid-party", 123456789)
	for _, invalidState := range states {
		for _, msu := range updateType {
			proposal := eng.newProposalForUpdateMarketState(party.Id, eng.tsvc.GetTimeNow().Add(2*time.Hour), msu, nil)
			// setup
			eng.ensureAllAssetEnabled(t)
			eng.ensureEquityLikeShareForMarketAndParty(t, proposal.UpdateMarketState().Changes.MarketID, party.Id, 0.1)
			eng.markets.EXPECT().MarketExists(proposal.UpdateMarketState().Changes.MarketID).Times(2).Return(true)
			eng.markets.EXPECT().GetMarketState(proposal.UpdateMarketState().Changes.MarketID).Times(1).Return(invalidState, nil)
			eng.expectRejectedProposalEvent(t, party.Id, proposal.ID, types.ProposalErrorInvalidMarket)

			// when
			toSubmit, err := eng.submitProposal(t, proposal)

			// then
			require.Equal(t, "market state does not allow for state update", err.Error())
			require.Nil(t, toSubmit)
		}
	}
}

func TestSubmittingProposalForUpdateMarketStateForUnknownMarketFails(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	updateType := []types.MarketStateUpdateType{types.MarketStateUpdateTypeResume, types.MarketStateUpdateTypeSuspend, types.MarketStateUpdateTypeTerminate}
	party := eng.newValidParty("a-valid-party", 123456789)

	for _, msu := range updateType {
		proposal := eng.newProposalForUpdateMarketState(party.Id, eng.tsvc.GetTimeNow().Add(2*time.Hour), msu, nil)
		// given
		proposer := party.Id
		marketID := proposal.UpdateMarketState().Changes.MarketID

		// setup
		eng.ensureTokenBalanceForParty(t, proposer, 123456789)
		eng.ensureNonExistingMarket(t, marketID)

		// expect
		eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidMarket)

		// when
		toSubmit, err := eng.submitProposal(t, proposal)

		// then
		require.ErrorIs(t, governance.ErrMarketDoesNotExist, err)
		require.Nil(t, toSubmit)
	}
}
