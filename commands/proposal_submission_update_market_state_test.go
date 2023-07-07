package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/require"
)

func TestCheckProposalSubmissionForUpdateMarketState(t *testing.T) {
	t.Run("Submitting a market state update change without an update fails", testUpdateMarketStateChangeSubmissionWithoutUpdateFails)
	t.Run("Submitting a market state update change without a change configuration fails", testUpdateMarketStateChangeSubmissionWithoutConfigurationFails)
	t.Run("Submitting a market state update change without a market id fails", testUpdateMarketStateChangeSubmissionWithoutMarketIDFails)
	t.Run("Submitting a market state update change without an update type fails", testUpdateMarketStateChangeSubmissionWithoutUpdateTypeFails)
	t.Run("Submitting a market state update change for anything but termination and passing a price fails", testUpdateMarketStateChangeSubmissionPriceNotExpectedFails)
	t.Run("Submitting a market state update change for terminating a market with invalid price fails", testUpdateMarketStateChangeSubmissionWithInvalidPrice)
}

func testUpdateMarketStateChangeSubmissionWithoutUpdateFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarketState{},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.update_market_state"), commands.ErrIsRequired)
}

func testUpdateMarketStateChangeSubmissionWithoutConfigurationFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarketState{
				UpdateMarketState: &types.UpdateMarketState{},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.update_market_state.changes"), commands.ErrIsRequired)
}

func testUpdateMarketStateChangeSubmissionWithoutMarketIDFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarketState{
				UpdateMarketState: &types.UpdateMarketState{
					Changes: &types.UpdateMarketStateConfiguration{},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.update_market_state.changes.marketId"), commands.ErrIsRequired)
}

func testUpdateMarketStateChangeSubmissionWithoutUpdateTypeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarketState{
				UpdateMarketState: &types.UpdateMarketState{
					Changes: &types.UpdateMarketStateConfiguration{
						MarketId: "marketID",
					},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.update_market_state.changes.updateType"), commands.ErrIsRequired)
}

func testUpdateMarketStateChangeSubmissionPriceNotExpectedFails(t *testing.T) {
	price := "123"
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarketState{
				UpdateMarketState: &types.UpdateMarketState{
					Changes: &types.UpdateMarketStateConfiguration{
						MarketId:   "marketID",
						UpdateType: types.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_RESUME,
						Price:      &price,
					},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.update_market_state.changes.price"), commands.ErrMustBeEmpty)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarketState{
				UpdateMarketState: &types.UpdateMarketState{
					Changes: &types.UpdateMarketStateConfiguration{
						MarketId:   "marketID",
						UpdateType: types.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_SUSPEND,
						Price:      &price,
					},
				},
			},
		},
	})
	require.Contains(t, err.Get("proposal_submission.terms.change.update_market_state.changes.price"), commands.ErrMustBeEmpty)
}

func testUpdateMarketStateChangeSubmissionWithInvalidPrice(t *testing.T) {
	invalidPrices := []string{"aaa", "-1", "1.234"}
	for _, inv := range invalidPrices {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_UpdateMarketState{
					UpdateMarketState: &types.UpdateMarketState{
						Changes: &types.UpdateMarketStateConfiguration{
							MarketId:   "marketID",
							UpdateType: types.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_TERMINATE,
							Price:      &inv,
						},
					},
				},
			},
		})
		require.Contains(t, err.Get("proposal_submission.terms.change.update_market_state.changes.price"), commands.ErrIsNotValid)
	}
}
