package commands_test

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
)

func TestCheckProposalSubmissionForUpdateMarket(t *testing.T) {
	t.Run("Submitting a market change without update market fails", testUpdateMarketChangeSubmissionWithoutUpdateMarketFails)
	t.Run("Submitting a market change without changes fails", testUpdateMarketChangeSubmissionWithoutChangesFails)
	t.Run("Submitting a market change without decimal places succeeds", testUpdateMarketChangeSubmissionWithoutDecimalPlacesSucceeds)
	t.Run("Submitting a update market without price monitoring succeeds", testUpdateMarketChangeSubmissionWithoutPriceMonitoringSucceeds)
	t.Run("Submitting a update market with price monitoring succeeds", testUpdateMarketChangeSubmissionWithPriceMonitoringSucceeds)
	t.Run("Submitting a price monitoring change without triggers succeeds", testUpdateMarketPriceMonitoringChangeSubmissionWithoutTriggersSucceeds)
	t.Run("Submitting a price monitoring change with triggers succeeds", testUpdateMarketPriceMonitoringChangeSubmissionWithTriggersSucceeds)
	t.Run("Submitting a price monitoring change without trigger horizon fails", testUpdateMarketPriceMonitoringChangeSubmissionWithoutTriggerHorizonFails)
	t.Run("Submitting a price monitoring change with trigger horizon succeeds", testUpdateMarketPriceMonitoringChangeSubmissionWithTriggerHorizonSucceeds)
	t.Run("Submitting a price monitoring change with wrong trigger probability fails", testUpdateMarketPriceMonitoringChangeSubmissionWithWrongTriggerProbabilityFails)
	t.Run("Submitting a price monitoring change with right trigger probability succeeds", testUpdateMarketPriceMonitoringChangeSubmissionWithRightTriggerProbabilitySucceeds)
	t.Run("Submitting a price monitoring change without trigger auction extension fails", testUpdateMarketPriceMonitoringChangeSubmissionWithoutTriggerAuctionExtensionFails)
	t.Run("Submitting a price monitoring change with trigger auction extension succeeds", testUpdateMarketPriceMonitoringChangeSubmissionWithTriggerAuctionExtensionSucceeds)
	t.Run("Submitting a update market without liquidity monitoring succeeds", testUpdateMarketChangeSubmissionWithoutLiquidityMonitoringSucceeds)
	t.Run("Submitting a update market with liquidity monitoring succeeds", testUpdateMarketChangeSubmissionWithLiquidityMonitoringSucceeds)
	t.Run("Submitting a liquidity monitoring change with wrong triggering ratio fails", testUpdateMarketLiquidityMonitoringChangeSubmissionWithWrongTriggeringRatioFails)
	t.Run("Submitting a liquidity monitoring change with right triggering ratio succeeds", testUpdateMarketLiquidityMonitoringChangeSubmissionWithRightTriggeringRatioSucceeds)
	t.Run("Submitting a liquidity monitoring change without triggering ratio parameter fails", testUpdateMarketLiquidityMonitoringChangeSubmissionWithoutTriggeringRatioParameterFails)
	t.Run("Submitting a liquidity monitoring change without target stake parameters fails", testUpdateMarketLiquidityMonitoringChangeSubmissionWithoutTargetStakeParametersFails)
	t.Run("Submitting a liquidity monitoring change with target stake parameters succeeds", testUpdateMarketLiquidityMonitoringChangeSubmissionWithTargetStakeParametersSucceeds)
	t.Run("Submitting a liquidity monitoring change with non-positive time window fails", testUpdateMarketLiquidityMonitoringChangeSubmissionWithNonPositiveTimeWindowFails)
	t.Run("Submitting a liquidity monitoring change with positive time window succeeds", testUpdateMarketLiquidityMonitoringChangeSubmissionWithPositiveTimeWindowSucceeds)
	t.Run("Submitting a liquidity monitoring change with non-positive scaling factor fails", testUpdateMarketLiquidityMonitoringChangeSubmissionWithNonPositiveScalingFactorFails)
	t.Run("Submitting a liquidity monitoring change with positive scaling factor succeeds", testUpdateMarketLiquidityMonitoringChangeSubmissionWithPositiveScalingFactorSucceeds)
	t.Run("Submitting a market change without instrument code fails", testUpdateMarketChangeSubmissionWithoutInstrumentCodeFails)
	t.Run("Submitting a market change with instrument code succeeds", testUpdateMarketChangeSubmissionWithInstrumentCodeSucceeds)
	t.Run("Submitting a market change without product fails", testUpdateMarketChangeSubmissionWithoutProductFails)
	t.Run("Submitting a market change with product succeeds", testUpdateMarketChangeSubmissionWithProductSucceeds)
	t.Run("Submitting a future market change without future fails", testUpdateFutureMarketChangeSubmissionWithoutFutureFails)
	t.Run("Submitting a future market change with future succeeds", testUpdateFutureMarketChangeSubmissionWithFutureSucceeds)
	t.Run("Submitting a future market change without quote name fails", testUpdateFutureMarketChangeSubmissionWithoutQuoteNameFails)
	t.Run("Submitting a future market change with quote name succeeds", testUpdateFutureMarketChangeSubmissionWithQuoteNameSucceeds)
	t.Run("Submitting a future market change without oracle spec fails", testUpdateFutureMarketChangeSubmissionWithoutOracleSpecFails)
	t.Run("Submitting a future market change without either of the required oracle spec fails", testUpdateFutureMarketChangeSubmissionMissingSingleOracleSpecFails)
	t.Run("Submitting a future market change with oracle spec succeeds", testUpdateFutureMarketChangeSubmissionWithOracleSpecSucceeds)
	t.Run("Submitting a future market change without pub-keys fails", testUpdateFutureMarketChangeSubmissionWithoutPubKeysFails)
	t.Run("Submitting a future market change with wrong pub-keys fails", testUpdateFutureMarketChangeSubmissionWithWrongPubKeysFails)
	t.Run("Submitting a future market change with pub-keys succeeds", testUpdateFutureMarketChangeSubmissionWithPubKeysSucceeds)
	t.Run("Submitting a future market change without filters fails", testUpdateFutureMarketChangeSubmissionWithoutFiltersFails)
	t.Run("Submitting a future market change with filters succeeds", testUpdateFutureMarketChangeSubmissionWithFiltersSucceeds)
	t.Run("Submitting a future market change with filter without key fails", testUpdateFutureMarketChangeSubmissionWithFilterWithoutKeyFails)
	t.Run("Submitting a future market change with filter with key succeeds", testUpdateFutureMarketChangeSubmissionWithFilterWithKeySucceeds)
	t.Run("Submitting a future market change with filter without key name fails", testUpdateFutureMarketChangeSubmissionWithFilterWithoutKeyNameFails)
	t.Run("Submitting a future market change with filter with key name succeeds", testUpdateFutureMarketChangeSubmissionWithFilterWithKeyNameSucceeds)
	t.Run("Submitting a future market change with filter without key type fails", testUpdateFutureMarketChangeSubmissionWithFilterWithoutKeyTypeFails)
	t.Run("Submitting a future market change with filter with key type succeeds", testUpdateFutureMarketChangeSubmissionWithFilterWithKeyTypeSucceeds)
	t.Run("Submitting a future market change with filter without condition succeeds", testUpdateFutureMarketChangeSubmissionWithFilterWithoutConditionsSucceeds)
	t.Run("Submitting a future market change with filter without condition operator fails", testUpdateFutureMarketChangeSubmissionWithFilterWithoutConditionOperatorFails)
	t.Run("Submitting a future market change with filter with condition operator succeeds", testUpdateFutureMarketChangeSubmissionWithFilterWithConditionOperatorSucceeds)
	t.Run("Submitting a future market change with filter without condition value fails", testUpdateFutureMarketChangeSubmissionWithFilterWithoutConditionValueFails)
	t.Run("Submitting a future market change with filter with condition value succeeds", testUpdateFutureMarketChangeSubmissionWithFilterWithConditionValueSucceeds)
	t.Run("Submitting a future market change without oracle spec bindings fails", testUpdateFutureMarketChangeSubmissionWithoutDataSourceSpecBindingFails)
	t.Run("Submitting a future market change with oracle spec binding succeeds", testUpdateFutureMarketChangeSubmissionWithDataSourceSpecBindingSucceeds)
	t.Run("Submitting a future market change without settlement data property fails", testUpdateFutureMarketChangeSubmissionWithoutSettlementDataPropertyFails)
	t.Run("Submitting a future market change without trading termination property fails", testUpdateFutureMarketChangeSubmissionWithoutTradingTerminationPropertyFails)
	t.Run("Submitting a future market change with a mismatch between binding property name and filter fails", testUpdateFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingFails)
	t.Run("Submitting a future market change with match between binding property name and filter succeeds", testUpdateFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingSucceeds)
	t.Run("Submitting a future market change with settlement data and trading termination properties succeeds", testUpdateFutureMarketChangeSubmissionWithSettlementDataPropertySucceeds)
	t.Run("Submitting a simple risk parameters change without simple risk parameters fails", testUpdateSimpleRiskParametersChangeSubmissionWithoutSimpleRiskParametersFails)
	t.Run("Submitting a simple risk parameters change with simple risk parameters succeeds", testUpdateSimpleRiskParametersChangeSubmissionWithSimpleRiskParametersSucceeds)
	t.Run("Submitting a simple risk parameters change with min move down fails", testUpdateSimpleRiskParametersChangeSubmissionWithPositiveMinMoveDownFails)
	t.Run("Submitting a simple risk parameters change with min move down succeeds", testUpdateSimpleRiskParametersChangeSubmissionWithNonPositiveMinMoveDownSucceeds)
	t.Run("Submitting a simple risk parameters change with max move up fails", testUpdateSimpleRiskParametersChangeSubmissionWithNegativeMaxMoveUpFails)
	t.Run("Submitting a simple risk parameters change with max move up succeeds", testUpdateSimpleRiskParametersChangeSubmissionWithNonNegativeMaxMoveUpSucceeds)
	t.Run("Submitting a simple risk parameters change with wrong probability of trading fails", testUpdateSimpleRiskParametersChangeSubmissionWithWrongProbabilityOfTradingFails)
	t.Run("Submitting a simple risk parameters change with right probability of trading succeeds", testUpdateSimpleRiskParametersChangeSubmissionWithRightProbabilityOfTradingSucceeds)
	t.Run("Submitting a log normal risk parameters change without log normal risk parameters fails", testUpdateLogNormalRiskParametersChangeSubmissionWithoutLogNormalRiskParametersFails)
	t.Run("Submitting a log normal risk parameters change with log normal risk parameters succeeds", testUpdateLogNormalRiskParametersChangeSubmissionWithLogNormalRiskParametersSucceeds)
	t.Run("Submitting a log normal risk parameters change with params fails", testUpdateLogNormalRiskParametersChangeSubmissionWithoutParamsFails)
	t.Run("Submitting a log normal risk parameters change with invalid risk aversion", testUpdateLogNormalRiskParametersChangeSubmissionInvalidRiskAversion)
	t.Run("Submitting a log normal risk parameters change with invalid tau", testUpdateLogNormalRiskParametersChangeSubmissionInvalidTau)
	t.Run("Submitting a log normal risk parameters change with invalid mu", testUpdateLogNormalRiskParametersChangeSubmissionInvalidMu)
	t.Run("Submitting a log normal risk parameters change with invalid sigma", testUpdateLogNormalRiskParametersChangeSubmissionInvalidSigma)
	t.Run("Submitting a log normal risk parameters change with invalid r", testUpdateLogNormalRiskParametersChangeSubmissionInvalidR)
	t.Run("Submitting a update market with a too long reference fails", testUpdateMarketSubmissionWithTooLongReferenceFails)
	t.Run("Submitting a future market with internal time for trade termination succeeds", testUpdateMarketFutureMarketSubmissionWithInternalTimestampForTradingTerminationSucceeds)
	t.Run("Submitting a future market with trade termination from external oracle with no public key fails", testUpdateMarketFutureMarketSubmissionWithExternalTradingTerminationNoPublicKeyFails)
	t.Run("Submitting a market with market ID succeeds", testUpdateMarketWithMarketIDSucceeds)
	t.Run("Submitting a market without market ID fails", testUpdateMarketWithoutMarketIDFails)
	t.Run("Submitting an LP price range change with 'banana' value fails", testUpdateMarketWithLpRangeBananaFails)
	t.Run("Submitting an LP price range change with negative value fails", testUpdateMarketWithLpRangeNegativeFails)
	t.Run("Submitting an LP price range change with zero value fails", testUpdateMarketWithLpRangeZeroFails)
	t.Run("Submitting an LP price range change with value above 100 fails", testUpdateMarketWithLpRangeGreterThan100Fails)
	t.Run("Submitting an LP price range change with value in [0,100] range succeeds", testUpdateMarketWithLpRangePositiveSucceeds)
	t.Run("Submitting a slippage fraction change with 'banana' value fails", tesUpdateMarketChangeSubmissionWithSlippageFactorBananaFails)
	t.Run("Submitting a slippage fraction change with a negative value fails", testUpdateMarketChangeSubmissionWithSlippageFactorNegativeFails)
	t.Run("Submitting a slippage fraction change with a too large value fails", testUpdateMarketChangeSubmissionWithSlippageFactorTooLargeFails)
	t.Run("Submitting a slippage fraction change with an empty string succeeds", testUpdateNewMarketChangeSubmissionWithEmptySlippageFactorPasses)
	t.Run("Submitting a market with external oracle to settle with `vegaprotobol.builtin` key name succeeds", testUpdateMarketWithExternalOracleSettlementUsingBuiltInKeyNameSucceeds)
}

func testUpdateMarketChangeSubmissionWithoutUpdateMarketFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market"), commands.ErrIsRequired)
}

func testUpdateMarketChangeSubmissionWithoutChangesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes"), commands.ErrIsRequired)
}

func testUpdateMarketChangeSubmissionWithoutDecimalPlacesSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.decimal_places"), commands.ErrMustBePositiveOrZero)
}

func testUpdateMarketChangeSubmissionWithoutLiquidityMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.liquidity_monitoring_parameters"), commands.ErrIsRequired)
}

func testUpdateMarketChangeSubmissionWithLiquidityMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.liquidity_monitoring_parameters"), commands.ErrIsRequired)
}

func testUpdateMarketLiquidityMonitoringChangeSubmissionWithWrongTriggeringRatioFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value string
	}{
		{
			msg:   "with probability of -1",
			value: "-1",
		}, {
			msg:   "with probability of 2",
			value: "2",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{
						UpdateMarket: &types.UpdateMarket{
							Changes: &types.UpdateMarketConfiguration{
								LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
									TriggeringRatio: tc.value,
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.liquidity_monitoring_parameters.triggering_ratio"),
				errors.New("should be between 0 (inclusive) and 1 (inclusive)"))
		})
	}
}

func testUpdateMarketLiquidityMonitoringChangeSubmissionWithRightTriggeringRatioSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value string
	}{
		{
			msg:   "with ratio of 0",
			value: "0",
		}, {
			msg:   "with ratio of 0.5",
			value: "0.5",
		}, {
			msg:   "with ratio of 1",
			value: "1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{
						UpdateMarket: &types.UpdateMarket{
							Changes: &types.UpdateMarketConfiguration{
								LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
									TriggeringRatio: tc.value,
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.liquidity_monitoring_parameters"),
				errors.New("should be between 0 (inclusive) and 1 (inclusive)"))
		})
	}
}

func testUpdateMarketLiquidityMonitoringChangeSubmissionWithoutTriggeringRatioParameterFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.liquidity_monitoring_parameters.triggering_ratio"), commands.ErrIsNotValidNumber)
}

func testUpdateMarketLiquidityMonitoringChangeSubmissionWithoutTargetStakeParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
							TriggeringRatio: "1",
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.liquidity_monitoring_parameters.target_stake_parameters"), commands.ErrIsRequired)
}

func testUpdateMarketLiquidityMonitoringChangeSubmissionWithTargetStakeParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
							TriggeringRatio:       "1",
							TargetStakeParameters: &types.TargetStakeParameters{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.liquidity_monitoring_parameters.target_stake_parameters"), commands.ErrIsRequired)
}

func testUpdateMarketLiquidityMonitoringChangeSubmissionWithNonPositiveTimeWindowFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value int64
	}{
		{
			msg:   "with ratio of 0",
			value: 0,
		}, {
			msg:   "with ratio of -1",
			value: RandomNegativeI64(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{
						UpdateMarket: &types.UpdateMarket{
							Changes: &types.UpdateMarketConfiguration{
								LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
									TriggeringRatio: "1",
									TargetStakeParameters: &types.TargetStakeParameters{
										TimeWindow: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.liquidity_monitoring_parameters.target_stake_parameters.time_window"), commands.ErrMustBePositive)
		})
	}
}

func testUpdateMarketLiquidityMonitoringChangeSubmissionWithPositiveTimeWindowSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
							TargetStakeParameters: &types.TargetStakeParameters{
								TimeWindow: RandomPositiveI64(),
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.liquidity_monitoring_parameters.target_stake_parameters.time_window"), commands.ErrMustBePositive)
}

func testUpdateMarketLiquidityMonitoringChangeSubmissionWithNonPositiveScalingFactorFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value float64
	}{
		{
			msg:   "with ratio of 0",
			value: 0,
		}, {
			msg:   "with ratio of -1.5",
			value: -1.5,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{
						UpdateMarket: &types.UpdateMarket{
							Changes: &types.UpdateMarketConfiguration{
								LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
									TriggeringRatio: "1",
									TargetStakeParameters: &types.TargetStakeParameters{
										ScalingFactor: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.liquidity_monitoring_parameters.target_stake_parameters.scaling_factor"), commands.ErrMustBePositive)
		})
	}
}

func testUpdateMarketLiquidityMonitoringChangeSubmissionWithPositiveScalingFactorSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
							TargetStakeParameters: &types.TargetStakeParameters{
								ScalingFactor: 1.5,
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.liquidity_monitoring_parameters.target_stake_parameters.scaling_factor"), commands.ErrMustBePositive)
}

func testUpdateMarketPriceMonitoringChangeSubmissionWithoutTriggersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						PriceMonitoringParameters: &types.PriceMonitoringParameters{
							Triggers: []*types.PriceMonitoringTrigger{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers"), commands.ErrIsRequired)
}

func testUpdateMarketPriceMonitoringChangeSubmissionWithTriggersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						PriceMonitoringParameters: &types.PriceMonitoringParameters{
							Triggers: []*types.PriceMonitoringTrigger{
								{},
								{},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers"), commands.ErrIsRequired)
}

func testUpdateMarketPriceMonitoringChangeSubmissionWithoutTriggerHorizonFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						PriceMonitoringParameters: &types.PriceMonitoringParameters{
							Triggers: []*types.PriceMonitoringTrigger{
								{},
								{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers.0.horizon"), commands.ErrMustBePositive)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers.1.horizon"), commands.ErrMustBePositive)
}

func testUpdateMarketPriceMonitoringChangeSubmissionWithTriggerHorizonSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						PriceMonitoringParameters: &types.PriceMonitoringParameters{
							Triggers: []*types.PriceMonitoringTrigger{
								{
									Horizon: RandomPositiveI64(),
								},
								{
									Horizon: RandomPositiveI64(),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers.0.horizon"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers.1.horizon"), commands.ErrMustBePositive)
}

func testUpdateMarketPriceMonitoringChangeSubmissionWithWrongTriggerProbabilityFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value float64
	}{
		{
			msg:   "with probability of -1",
			value: -1,
		}, {
			msg:   "with probability of 0",
			value: 0,
		}, {
			msg:   "with probability of 1",
			value: 1,
		}, {
			msg:   "with probability of 2",
			value: 2,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{
						UpdateMarket: &types.UpdateMarket{
							Changes: &types.UpdateMarketConfiguration{
								PriceMonitoringParameters: &types.PriceMonitoringParameters{
									Triggers: []*types.PriceMonitoringTrigger{
										{
											Probability: fmt.Sprintf("%f", tc.value),
										},
										{
											Probability: fmt.Sprintf("%f", tc.value),
										},
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers.0.probability"),
				errors.New("should be between 0 (exclusive) and 1 (exclusive)"))
			assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers.1.probability"),
				errors.New("should be between 0 (exclusive) and 1 (exclusive)"))
		})
	}
}

func testUpdateMarketPriceMonitoringChangeSubmissionWithRightTriggerProbabilitySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						PriceMonitoringParameters: &types.PriceMonitoringParameters{
							Triggers: []*types.PriceMonitoringTrigger{
								{
									Probability: "0.01",
								},
								{
									Probability: "0.9",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers.0.probability"),
		errors.New("should be between 0 (exclusive) and 1 (exclusive)"))
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers.1.probability"),
		errors.New("should be between 0 (exclusive) and 1 (exclusive)"))
}

func testUpdateMarketPriceMonitoringChangeSubmissionWithoutTriggerAuctionExtensionFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						PriceMonitoringParameters: &types.PriceMonitoringParameters{
							Triggers: []*types.PriceMonitoringTrigger{
								{},
								{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers.0.auction_extension"), commands.ErrMustBePositive)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers.1.auction_extension"), commands.ErrMustBePositive)
}

func testUpdateMarketPriceMonitoringChangeSubmissionWithTriggerAuctionExtensionSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						PriceMonitoringParameters: &types.PriceMonitoringParameters{
							Triggers: []*types.PriceMonitoringTrigger{
								{
									AuctionExtension: RandomPositiveI64(),
								},
								{
									AuctionExtension: RandomPositiveI64(),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers.0.auction_extension"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters.triggers.1.auction_extension"), commands.ErrMustBePositive)
}

func testUpdateMarketChangeSubmissionWithoutPriceMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters"), commands.ErrIsRequired)
}

func testUpdateMarketChangeSubmissionWithPriceMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						PriceMonitoringParameters: &types.PriceMonitoringParameters{},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.price_monitoring_parameters"), commands.ErrIsRequired)
}

func testUpdateMarketChangeSubmissionWithoutInstrumentCodeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Code: "",
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.code"), commands.ErrIsRequired)
}

func testUpdateMarketChangeSubmissionWithInstrumentCodeSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Code: "My code",
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.code"), commands.ErrIsRequired)
}

func testUpdateMarketChangeSubmissionWithoutProductFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product"), commands.ErrIsRequired)
}

func testUpdateMarketChangeSubmissionWithProductSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithoutFutureFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithFutureSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithoutQuoteNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									QuoteName: "",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.quote_name"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithQuoteNameSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									QuoteName: "BTC",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.quote_name"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithoutOracleSpecFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_trading_termination"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionMissingSingleOracleSpecFails(t *testing.T) {
	testUpdateFutureMarketChangeSubmissionWithoutEitherOracleSpecFails(t, "data_source_spec_for_settlement_data")
	testUpdateFutureMarketChangeSubmissionWithoutEitherOracleSpecFails(t, "data_source_spec_for_trading_termination")
}

func testUpdateFutureMarketChangeSubmissionWithoutEitherOracleSpecFails(t *testing.T, oracleSpecName string) {
	t.Helper()
	future := &types.UpdateFutureProduct{}
	if oracleSpecName == "data_source_spec_for_settlement_data" {
		future.DataSourceSpecForTradingTermination = &vegapb.DataSourceDefinition{}
	} else {
		future.DataSourceSpecForSettlementData = &vegapb.DataSourceDefinition{}
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future."+oracleSpecName), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithOracleSpecSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData:     &vegapb.DataSourceDefinition{},
									DataSourceSpecForTradingTermination: &vegapb.DataSourceDefinition{},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithoutPubKeysFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Signers: []*datapb.Signer{},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithWrongPubKeysFails(t *testing.T) {
	pubKeys := []*datapb.Signer{
		{
			Signer: &datapb.Signer_PubKey{
				PubKey: &datapb.PubKey{
					Key: "0xDEADBEEF",
				},
			},
		},
		{
			Signer: &datapb.Signer_PubKey{
				PubKey: &datapb.PubKey{
					Key: "",
				},
			},
		},
	}

	testCases := []struct {
		msg   string
		value []*datapb.Signer
	}{
		{
			msg:   "with empty signers",
			value: pubKeys,
		}, {
			msg:   "with blank signers",
			value: pubKeys,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{
						UpdateMarket: &types.UpdateMarket{
							Changes: &types.UpdateMarketConfiguration{
								Instrument: &types.UpdateInstrumentConfiguration{
									Product: &types.UpdateInstrumentConfiguration_Future{
										Future: &types.UpdateFutureProduct{
											DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
												vegapb.DataSourceDefinitionTypeExt,
											).SetOracleConfig(
												&vegapb.DataSourceSpecConfiguration{
													Signers: tc.value,
												},
											),
										},
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.1"), commands.ErrIsNotValid)
		})
	}
}

func testUpdateFutureMarketChangeSubmissionWithPubKeysSucceeds(t *testing.T) {
	pubKeys := []*datapb.Signer{
		{
			Signer: &datapb.Signer_PubKey{
				PubKey: &datapb.PubKey{
					Key: "0xDEADBEEF",
				},
			},
		},
		{
			Signer: &datapb.Signer_PubKey{
				PubKey: &datapb.PubKey{
					Key: "0xCAFEDUDE",
				},
			},
		},
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Signers: pubKeys,
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Signers: pubKeys,
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.0"), commands.ErrIsNotValid)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.1"), commands.ErrIsNotValid)
}

func testUpdateFutureMarketChangeSubmissionWithoutFiltersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithFiltersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithFilterWithoutKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{}, {},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.key"), commands.ErrIsNotValid)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.1.key"), commands.ErrIsNotValid)
}

func testUpdateFutureMarketChangeSubmissionWithFilterWithKeySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{
													Key: &datapb.PropertyKey{},
												}, {
													Key: &datapb.PropertyKey{},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{
													Key: &datapb.PropertyKey{},
												}, {
													Key: &datapb.PropertyKey{},
												},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.key"), commands.ErrIsNotValid)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.1.key"), commands.ErrIsNotValid)
}

func testUpdateFutureMarketChangeSubmissionWithFilterWithoutKeyNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{
													Key: &datapb.PropertyKey{
														Name: "",
													},
												}, {
													Key: &datapb.PropertyKey{
														Name: "",
													},
												},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.key.name"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.1.key.name"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithFilterWithKeyNameSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{
													Key: &datapb.PropertyKey{
														Name: "key1",
													},
												}, {
													Key: &datapb.PropertyKey{
														Name: "key2",
													},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{
													Key: &datapb.PropertyKey{
														Name: "key1",
													},
												}, {
													Key: &datapb.PropertyKey{
														Name: "key2",
													},
												},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.key.name"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.1.key.name"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithFilterWithoutKeyTypeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{
													Key: &datapb.PropertyKey{
														Type: datapb.PropertyKey_TYPE_UNSPECIFIED,
													},
												}, {
													Key: &datapb.PropertyKey{},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{
													Key: &datapb.PropertyKey{
														Type: datapb.PropertyKey_TYPE_UNSPECIFIED,
													},
												}, {
													Key: &datapb.PropertyKey{},
												},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.key.type"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.1.key.type"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithFilterWithKeyTypeSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value datapb.PropertyKey_Type
	}{
		{
			msg:   "with EMPTY",
			value: datapb.PropertyKey_TYPE_EMPTY,
		}, {
			msg:   "with INTEGER",
			value: datapb.PropertyKey_TYPE_INTEGER,
		}, {
			msg:   "with STRING",
			value: datapb.PropertyKey_TYPE_STRING,
		}, {
			msg:   "with BOOLEAN",
			value: datapb.PropertyKey_TYPE_BOOLEAN,
		}, {
			msg:   "with DECIMAL",
			value: datapb.PropertyKey_TYPE_DECIMAL,
		}, {
			msg:   "with TIMESTAMP",
			value: datapb.PropertyKey_TYPE_TIMESTAMP,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{
						UpdateMarket: &types.UpdateMarket{
							Changes: &types.UpdateMarketConfiguration{
								Instrument: &types.UpdateInstrumentConfiguration{
									Product: &types.UpdateInstrumentConfiguration_Future{
										Future: &types.UpdateFutureProduct{
											DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
												vegapb.DataSourceDefinitionTypeExt,
											).SetOracleConfig(
												&vegapb.DataSourceSpecConfiguration{
													Filters: []*datapb.Filter{
														{
															Key: &datapb.PropertyKey{
																Type: tc.value,
															},
														}, {
															Key: &datapb.PropertyKey{
																Type: tc.value,
															},
														},
													},
												},
											),
										},
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.0.key.type"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.1.key.type"), commands.ErrIsRequired)
		})
	}
}

func testUpdateFutureMarketChangeSubmissionWithFilterWithoutConditionsSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{
													Conditions: []*datapb.Condition{},
												},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.0.conditions"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithFilterWithoutConditionOperatorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{
													Conditions: []*datapb.Condition{
														{
															Operator: datapb.Condition_OPERATOR_UNSPECIFIED,
														},
														{},
													},
												},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.conditions.0.operator"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.conditions.1.operator"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithFilterWithConditionOperatorSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value datapb.Condition_Operator
	}{
		{
			msg:   "with EQUALS",
			value: datapb.Condition_OPERATOR_EQUALS,
		}, {
			msg:   "with GREATER_THAN",
			value: datapb.Condition_OPERATOR_GREATER_THAN,
		}, {
			msg:   "with GREATER_THAN_OR_EQUAL",
			value: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
		}, {
			msg:   "with LESS_THAN",
			value: datapb.Condition_OPERATOR_LESS_THAN,
		}, {
			msg:   "with LESS_THAN_OR_EQUAL",
			value: datapb.Condition_OPERATOR_LESS_THAN_OR_EQUAL,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{
						UpdateMarket: &types.UpdateMarket{
							Changes: &types.UpdateMarketConfiguration{
								Instrument: &types.UpdateInstrumentConfiguration{
									Product: &types.UpdateInstrumentConfiguration_Future{
										Future: &types.UpdateFutureProduct{
											DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
												vegapb.DataSourceDefinitionTypeExt,
											).SetOracleConfig(
												&vegapb.DataSourceSpecConfiguration{
													Filters: []*datapb.Filter{
														{
															Conditions: []*datapb.Condition{
																{
																	Operator: tc.value,
																},
																{
																	Operator: tc.value,
																},
															},
														},
													},
												},
											),
										},
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.0.conditions.0.operator"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.1.conditions.0.operator"), commands.ErrIsRequired)
		})
	}
}

func testUpdateFutureMarketChangeSubmissionWithFilterWithoutConditionValueFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{
													Conditions: []*datapb.Condition{
														{
															Value: "",
														},
														{
															Value: "",
														},
													},
												},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.conditions.0.value"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.conditions.1.value"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithFilterWithConditionValueSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{
													Conditions: []*datapb.Condition{
														{
															Value: "value 1",
														},
														{
															Value: "value 2",
														},
													},
												},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.0.conditions.0.value"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.0.conditions.1.value"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithoutDataSourceSpecBindingFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_binding"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithDataSourceSpecBindingSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecBinding: &types.DataSourceSpecToFutureBinding{},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_binding"), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionMissingOracleBindingPropertyFails(t *testing.T, property string) {
	t.Helper()
	var binding *types.DataSourceSpecToFutureBinding
	if property == "settlement_data_property" {
		binding = &types.DataSourceSpecToFutureBinding{
			SettlementDataProperty: "",
		}
	} else {
		binding = &types.DataSourceSpecToFutureBinding{
			TradingTerminationProperty: "",
		}
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecBinding: binding,
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_binding."+property), commands.ErrIsRequired)
}

func testUpdateFutureMarketChangeSubmissionWithoutTradingTerminationPropertyFails(t *testing.T) {
	testUpdateFutureMarketChangeSubmissionMissingOracleBindingPropertyFails(t, "trading_termination_property")
}

func testUpdateFutureMarketChangeSubmissionWithoutSettlementDataPropertyFails(t *testing.T) {
	testUpdateFutureMarketChangeSubmissionMissingOracleBindingPropertyFails(t, "settlement_data_property")
}

func testUpdateFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingSucceeds(t *testing.T) {
	testUpdateFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t, &types.DataSourceSpecToFutureBinding{SettlementDataProperty: "key1"}, "settlement_data_property", "key1")
	testUpdateFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t, &types.DataSourceSpecToFutureBinding{TradingTerminationProperty: "key2"}, "settlement_data_property", "key2")
}

func testUpdateFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t *testing.T, binding *types.DataSourceSpecToFutureBinding, bindingName string, bindingKey string) {
	t.Helper()
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Filters: []*datapb.Filter{
												{
													Key: &datapb.PropertyKey{
														Name: bindingKey,
													},
												}, {
													Key: &datapb.PropertyKey{},
												},
											},
										},
									),
									DataSourceSpecBinding: binding,
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_binding."+bindingName), commands.ErrIsMismatching)
}

func testUpdateFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t *testing.T, binding *types.DataSourceSpecToFutureBinding, bindingName string) {
	t.Helper()
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecBinding: binding,
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_binding."+bindingName), commands.ErrIsMismatching)
}

func testUpdateFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingFails(t *testing.T) {
	testUpdateFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t, &types.DataSourceSpecToFutureBinding{SettlementDataProperty: "My property"}, "settlement_data_property")
	testUpdateFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t, &types.DataSourceSpecToFutureBinding{TradingTerminationProperty: "My property"}, "trading_termination_property")
}

func testUpdateFutureMarketChangeSubmissionWithSettlementDataPropertySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecBinding: &types.DataSourceSpecToFutureBinding{
										SettlementDataProperty: "My property",
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_binding.settlement_data_property"), commands.ErrIsRequired)
}

func testUpdateSimpleRiskParametersChangeSubmissionWithoutSimpleRiskParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_Simple{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.simple"), commands.ErrIsRequired)
}

func testUpdateSimpleRiskParametersChangeSubmissionWithSimpleRiskParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_Simple{
							Simple: &types.SimpleModelParams{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.simple"), commands.ErrIsRequired)
}

func testUpdateSimpleRiskParametersChangeSubmissionWithPositiveMinMoveDownFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_Simple{
							Simple: &types.SimpleModelParams{
								MinMoveDown: 1,
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.simple.min_move_down"), commands.ErrMustBeNegativeOrZero)
}

func testUpdateSimpleRiskParametersChangeSubmissionWithNonPositiveMinMoveDownSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value float64
	}{
		{
			msg:   "with min move down of 0",
			value: 0,
		}, {
			msg:   "with min move down of -1",
			value: -1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{
						UpdateMarket: &types.UpdateMarket{
							Changes: &types.UpdateMarketConfiguration{
								RiskParameters: &types.UpdateMarketConfiguration_Simple{
									Simple: &types.SimpleModelParams{
										MinMoveDown: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.simple.min_move_down"), commands.ErrMustBeNegativeOrZero)
		})
	}
}

func testUpdateSimpleRiskParametersChangeSubmissionWithNegativeMaxMoveUpFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_Simple{
							Simple: &types.SimpleModelParams{
								MaxMoveUp: -1,
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.simple.max_move_up"), commands.ErrMustBePositiveOrZero)
}

func testUpdateSimpleRiskParametersChangeSubmissionWithNonNegativeMaxMoveUpSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value float64
	}{
		{
			msg:   "with max move up of 0",
			value: 0,
		}, {
			msg:   "with max move up of 1",
			value: 1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{
						UpdateMarket: &types.UpdateMarket{
							Changes: &types.UpdateMarketConfiguration{
								RiskParameters: &types.UpdateMarketConfiguration_Simple{
									Simple: &types.SimpleModelParams{
										MaxMoveUp: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.simple.max_move_up"), commands.ErrMustBePositiveOrZero)
		})
	}
}

func testUpdateSimpleRiskParametersChangeSubmissionWithWrongProbabilityOfTradingFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value float64
	}{
		{
			msg:   "with probability of trading of -1",
			value: -1,
		}, {
			msg:   "with probability of trading of 2",
			value: 2,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{
						UpdateMarket: &types.UpdateMarket{
							Changes: &types.UpdateMarketConfiguration{
								RiskParameters: &types.UpdateMarketConfiguration_Simple{
									Simple: &types.SimpleModelParams{
										ProbabilityOfTrading: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.simple.probability_of_trading"),
				errors.New("should be between 0 (inclusive) and 1 (inclusive)"))
		})
	}
}

func testUpdateSimpleRiskParametersChangeSubmissionWithRightProbabilityOfTradingSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value float64
	}{
		{
			msg:   "with probability of trading of 0",
			value: 0,
		}, {
			msg:   "with probability of trading of 1",
			value: 1,
		}, {
			msg:   "with probability of trading of 0.5",
			value: 0.5,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{
						UpdateMarket: &types.UpdateMarket{
							Changes: &types.UpdateMarketConfiguration{
								RiskParameters: &types.UpdateMarketConfiguration_Simple{
									Simple: &types.SimpleModelParams{
										ProbabilityOfTrading: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.simple.probability_of_trading"),
				errors.New("should be between 0 (inclusive) and 1 (inclusive)"))
		})
	}
}

func testUpdateLogNormalRiskParametersChangeSubmissionWithoutLogNormalRiskParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_LogNormal{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal"), commands.ErrIsRequired)
}

func testUpdateLogNormalRiskParametersChangeSubmissionWithLogNormalRiskParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_LogNormal{
							LogNormal: &types.LogNormalRiskModel{
								RiskAversionParameter: 1,
								Tau:                   2,
								Params: &types.LogNormalModelParams{
									Mu:    0,
									Sigma: 0.1,
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal"), commands.ErrIsRequired)
}

func testUpdateLogNormalRiskParametersChangeSubmissionWithoutParamsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_LogNormal{
							LogNormal: &types.LogNormalRiskModel{},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.params"), commands.ErrIsRequired)
}

func testUpdateLogNormalRiskParametersChangeSubmissionInvalidRiskAversion(t *testing.T) {
	cZero := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_LogNormal{
							LogNormal: &types.LogNormalRiskModel{
								RiskAversionParameter: 0,
								Tau:                   2,
								Params: &types.LogNormalModelParams{
									Mu:    0,
									Sigma: 0.1,
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	}
	err := checkProposalSubmission(cZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), commands.ErrMustBePositive)

	cNeg := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_LogNormal{
							LogNormal: &types.LogNormalRiskModel{
								RiskAversionParameter: -0.1,
								Tau:                   2,
								Params: &types.LogNormalModelParams{
									Mu:    0,
									Sigma: 0.1,
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	}
	err = checkProposalSubmission(cNeg)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), commands.ErrMustBePositive)
}

func testUpdateLogNormalRiskParametersChangeSubmissionInvalidTau(t *testing.T) {
	cZero := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_LogNormal{
							LogNormal: &types.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0,
								Params: &types.LogNormalModelParams{
									Mu:    0,
									Sigma: 0.1,
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	}
	err := checkProposalSubmission(cZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.tau"), commands.ErrMustBePositive)

	cNeg := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_LogNormal{
							LogNormal: &types.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   -0.2,
								Params: &types.LogNormalModelParams{
									Mu:    0,
									Sigma: 0.1,
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	}
	err = checkProposalSubmission(cNeg)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.tau"), commands.ErrMustBePositive)
}

func testUpdateLogNormalRiskParametersChangeSubmissionInvalidMu(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_LogNormal{
							LogNormal: &types.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &types.LogNormalModelParams{
									Mu:    math.NaN(),
									Sigma: 0.1,
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	}
	err := checkProposalSubmission(cNaN)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.params.mu"), commands.ErrIsNotValidNumber)
}

func testUpdateLogNormalRiskParametersChangeSubmissionInvalidR(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_LogNormal{
							LogNormal: &types.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &types.LogNormalModelParams{
									Mu:    0.2,
									Sigma: 0.1,
									R:     math.NaN(),
								},
							},
						},
					},
				},
			},
		},
	}
	err := checkProposalSubmission(cNaN)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.params.r"), commands.ErrIsNotValidNumber)
}

func testUpdateLogNormalRiskParametersChangeSubmissionInvalidSigma(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_LogNormal{
							LogNormal: &types.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &types.LogNormalModelParams{
									Mu:    0.2,
									Sigma: math.NaN(),
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	}
	err := checkProposalSubmission(cNaN)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.params.sigma"), commands.ErrIsNotValidNumber)

	cNeg := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_LogNormal{
							LogNormal: &types.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &types.LogNormalModelParams{
									Mu:    0.2,
									Sigma: -0.1,
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	}
	err = checkProposalSubmission(cNeg)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.params.sigma"), commands.ErrMustBePositive)

	c0 := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						RiskParameters: &types.UpdateMarketConfiguration_LogNormal{
							LogNormal: &types.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &types.LogNormalModelParams{
									Mu:    0.2,
									Sigma: 0,
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	}
	err = checkProposalSubmission(c0)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.params.sigma"), commands.ErrMustBePositive)
}

func testUpdateMarketSubmissionWithTooLongReferenceFails(t *testing.T) {
	ref := make([]byte, 101)
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Reference: string(ref),
	})
	assert.Contains(t, err.Get("proposal_submission.reference"), commands.ErrReferenceTooLong)
}

func testUpdateMarketFutureMarketSubmissionWithInternalTimestampForTradingTerminationSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Signers: []*datapb.Signer{},
											Filters: []*datapb.Filter{
												{
													Key: &datapb.PropertyKey{
														Name: "vegaprotocol.builtin.timestamp",
														Type: datapb.PropertyKey_TYPE_TIMESTAMP,
													},
													Conditions: []*datapb.Condition{
														{
															Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
															Value:    fmt.Sprintf("%d", time.Now().Add(time.Hour*24*365).UnixNano()),
														},
													},
												},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}

func testUpdateMarketFutureMarketSubmissionWithExternalTradingTerminationNoPublicKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Signers: []*datapb.Signer{},
											Filters: []*datapb.Filter{
												{
													Key: &datapb.PropertyKey{
														Name: "trading.terminated",
														Type: datapb.PropertyKey_TYPE_BOOLEAN,
													},
													Conditions: []*datapb.Condition{
														{
															Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
															Value:    fmt.Sprintf("%d", time.Now().Add(time.Hour*24*365).UnixNano()),
														},
													},
												},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}

func testUpdateMarketWithMarketIDSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					MarketId: "12345",
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.market_id"), commands.ErrIsRequired)
}

func testUpdateMarketWithoutMarketIDFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.market_id"), commands.ErrIsRequired)
}

func testUpdateMarketWithLpRangeBananaFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LpPriceRange: "banana",
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrIsNotValidNumber)
}

func testUpdateMarketWithLpRangeNegativeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LpPriceRange: "-1e-17",
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrMustBePositive)
}

func testUpdateMarketWithLpRangeZeroFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LpPriceRange: "0",
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrMustBePositive)
}

func testUpdateMarketWithLpRangeGreterThan100Fails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LpPriceRange: "100.00000000000001",
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrMustBeAtMost100)
}

func testUpdateMarketWithLpRangePositiveSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LpPriceRange: "1e-17",
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrIsNotValidNumber)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrMustBeAtMost100)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LpPriceRange: "0.95",
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrIsNotValidNumber)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrMustBeAtMost100)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LpPriceRange: "100",
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrIsNotValidNumber)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.lp_price_range"), commands.ErrMustBeAtMost100)
}

func tesUpdateMarketChangeSubmissionWithSlippageFactorBananaFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LinearSlippageFactor: "banana",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.linear_slippage_factor"), commands.ErrIsNotValidNumber)
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						QuadraticSlippageFactor: "banana",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.quadratic_slippage_factor"), commands.ErrIsNotValidNumber)
}

func testUpdateMarketChangeSubmissionWithSlippageFactorNegativeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LinearSlippageFactor: "-0.1",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.linear_slippage_factor"), commands.ErrMustBePositiveOrZero)
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						QuadraticSlippageFactor: "-0.1",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.quadratic_slippage_factor"), commands.ErrMustBePositiveOrZero)
}

func testUpdateMarketChangeSubmissionWithSlippageFactorTooLargeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						LinearSlippageFactor: "1000000.000001",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.linear_slippage_factor"), commands.ErrMustBeAtMost1M)
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						QuadraticSlippageFactor: "1000000.000001",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_market.changes.quadratic_slippage_factor"), commands.ErrMustBeAtMost1M)
}

func testUpdateNewMarketChangeSubmissionWithEmptySlippageFactorPasses(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{},
				},
			},
		},
	})
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.linear_slippage_factor"), commands.ErrIsNotValidNumber)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.quadratic_slippage_factor"), commands.ErrIsNotValidNumber)
}

func testUpdateMarketWithExternalOracleSettlementUsingBuiltInKeyNameSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateMarket{
				UpdateMarket: &types.UpdateMarket{
					Changes: &types.UpdateMarketConfiguration{
						Instrument: &types.UpdateInstrumentConfiguration{
							Product: &types.UpdateInstrumentConfiguration_Future{
								Future: &types.UpdateFutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Signers: []*datapb.Signer{},
											Filters: []*datapb.Filter{
												{
													Key: &datapb.PropertyKey{
														Name: "vegaprotocol.builtin.timestamp",
														Type: datapb.PropertyKey_TYPE_TIMESTAMP,
													},
													Conditions: []*datapb.Condition{
														{
															Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
															Value:    fmt.Sprintf("%d", time.Now().Add(time.Hour*24*365).UnixNano()),
														},
													},
												},
											},
										},
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}
