package commands_test

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	oraclespb "code.vegaprotocol.io/vega/protos/vega/oracles/v1"
	"github.com/stretchr/testify/assert"
)

func TestCheckProposalSubmissionForNewMarket(t *testing.T) {
	t.Run("Submitting a market change without new market fails", testNewMarketChangeSubmissionWithoutNewMarketFails)
	t.Run("Submitting a market change without changes fails", testNewMarketChangeSubmissionWithoutChangesFails)
	t.Run("Submitting a market change without too many pm trigger fails", testNewMarketChangeSubmissionWithTooManyPMTriggersFails)
	t.Run("Submitting a market change without decimal places succeeds", testNewMarketChangeSubmissionWithoutDecimalPlacesSucceeds)
	t.Run("Submitting a market change with decimal places equal to 0 succeeds", testNewMarketChangeSubmissionWithDecimalPlacesEqualTo0Succeeds)
	t.Run("Submitting a market change with decimal places above or equal to 150 fails", testNewMarketChangeSubmissionWithDecimalPlacesAboveOrEqualTo150Fails)
	t.Run("Submitting a market change with decimal places below 150 succeeds", testNewMarketChangeSubmissionWithDecimalPlacesBelow150Succeeds)
	t.Run("Submitting a market change without decimal places succeeds", testNewMarketChangeSubmissionWithoutDecimalPlacesSucceeds)
	t.Run("Submitting a market change with position decimal places equal to 0 succeeds", testNewMarketChangeSubmissionWithPositionDecimalPlacesEqualTo0Succeeds)
	t.Run("Submitting a market change with position decimal places above or equal to 6 fails", testNewMarketChangeSubmissionWithPositionDecimalPlacesAboveOrEqualTo7Fails)
	t.Run("Submitting a market change with position decimal places below 6 succeeds", testNewMarketChangeSubmissionWithPositionDecimalPlacesBelow7Succeeds)
	t.Run("Submitting a new market without price monitoring succeeds", testNewMarketChangeSubmissionWithoutPriceMonitoringSucceeds)
	t.Run("Submitting a new market with price monitoring succeeds", testNewMarketChangeSubmissionWithPriceMonitoringSucceeds)
	t.Run("Submitting a price monitoring change without triggers succeeds", testPriceMonitoringChangeSubmissionWithoutTriggersSucceeds)
	t.Run("Submitting a price monitoring change with triggers succeeds", testPriceMonitoringChangeSubmissionWithTriggersSucceeds)
	t.Run("Submitting a price monitoring change without trigger horizon fails", testPriceMonitoringChangeSubmissionWithoutTriggerHorizonFails)
	t.Run("Submitting a price monitoring change with trigger horizon succeeds", testPriceMonitoringChangeSubmissionWithTriggerHorizonSucceeds)
	t.Run("Submitting a price monitoring change with wrong trigger probability fails", testPriceMonitoringChangeSubmissionWithWrongTriggerProbabilityFails)
	t.Run("Submitting a price monitoring change with right trigger probability succeeds", testPriceMonitoringChangeSubmissionWithRightTriggerProbabilitySucceeds)
	t.Run("Submitting a price monitoring change without trigger auction extension fails", testPriceMonitoringChangeSubmissionWithoutTriggerAuctionExtensionFails)
	t.Run("Submitting a price monitoring change with trigger auction extension succeeds", testPriceMonitoringChangeSubmissionWithTriggerAuctionExtensionSucceeds)
	t.Run("Submitting a new market without liquidity monitoring succeeds", testNewMarketChangeSubmissionWithoutLiquidityMonitoringSucceeds)
	t.Run("Submitting a new market with liquidity monitoring succeeds", testNewMarketChangeSubmissionWithLiquidityMonitoringSucceeds)
	t.Run("Submitting a liquidity monitoring change with wrong triggering ratio fails", testLiquidityMonitoringChangeSubmissionWithWrongTriggeringRatioFails)
	t.Run("Submitting a liquidity monitoring change with right triggering ratio succeeds", testLiquidityMonitoringChangeSubmissionWithRightTriggeringRatioSucceeds)
	t.Run("Submitting a liquidity monitoring change without target stake parameters fails", testLiquidityMonitoringChangeSubmissionWithoutTargetStakeParametersFails)
	t.Run("Submitting a liquidity monitoring change with target stake parameters succeeds", testLiquidityMonitoringChangeSubmissionWithTargetStakeParametersSucceeds)
	t.Run("Submitting a liquidity monitoring change with non-positive time window fails", testLiquidityMonitoringChangeSubmissionWithNonPositiveTimeWindowFails)
	t.Run("Submitting a liquidity monitoring change with positive time window succeeds", testLiquidityMonitoringChangeSubmissionWithPositiveTimeWindowSucceeds)
	t.Run("Submitting a liquidity monitoring change with non-positive scaling factor fails", testLiquidityMonitoringChangeSubmissionWithNonPositiveScalingFactorFails)
	t.Run("Submitting a liquidity monitoring change with positive scaling factor succeeds", testLiquidityMonitoringChangeSubmissionWithPositiveScalingFactorSucceeds)
	t.Run("Submitting a market change without instrument name fails", testNewMarketChangeSubmissionWithoutInstrumentNameFails)
	t.Run("Submitting a market change with instrument name succeeds", testNewMarketChangeSubmissionWithInstrumentNameSucceeds)
	t.Run("Submitting a market change without instrument code fails", testNewMarketChangeSubmissionWithoutInstrumentCodeFails)
	t.Run("Submitting a market change with instrument code succeeds", testNewMarketChangeSubmissionWithInstrumentCodeSucceeds)
	t.Run("Submitting a market change without product fails", testNewMarketChangeSubmissionWithoutProductFails)
	t.Run("Submitting a market change with product succeeds", testNewMarketChangeSubmissionWithProductSucceeds)
	t.Run("Submitting a future market change without future fails", testNewFutureMarketChangeSubmissionWithoutFutureFails)
	t.Run("Submitting a future market change with future succeeds", testNewFutureMarketChangeSubmissionWithFutureSucceeds)
	t.Run("Submitting a future market change without settlement asset fails", testNewFutureMarketChangeSubmissionWithoutSettlementAssetFails)
	t.Run("Submitting a future market change with settlement asset succeeds", testNewFutureMarketChangeSubmissionWithSettlementAssetSucceeds)
	t.Run("Submitting a future market change without quote name fails", testNewFutureMarketChangeSubmissionWithoutQuoteNameFails)
	t.Run("Submitting a future market change with quote name succeeds", testNewFutureMarketChangeSubmissionWithQuoteNameSucceeds)
	t.Run("Submitting a future market change without oracle spec fails", testNewFutureMarketChangeSubmissionWithoutOracleSpecFails)
	t.Run("Submitting a future market change without either of the required oracle spec fails", testNewFutureMarketChangeSubmissionMissingSingleOracleSpecFails)
	t.Run("Submitting a future market change with oracle spec succeeds", testNewFutureMarketChangeSubmissionWithOracleSpecSucceeds)
	t.Run("Submitting a future market change without pub-keys fails", testNewFutureMarketChangeSubmissionWithoutPubKeysFails)
	t.Run("Submitting a future market change with wrong pub-keys fails", testNewFutureMarketChangeSubmissionWithWrongPubKeysFails)
	t.Run("Submitting a future market change with pub-keys succeeds", testNewFutureMarketChangeSubmissionWithPubKeysSucceeds)
	t.Run("Submitting a future market change without filters fails", testNewFutureMarketChangeSubmissionWithoutFiltersFails)
	t.Run("Submitting a future market change with filters succeeds", testNewFutureMarketChangeSubmissionWithFiltersSucceeds)
	t.Run("Submitting a future market change with filter without key fails", testNewFutureMarketChangeSubmissionWithFilterWithoutKeyFails)
	t.Run("Submitting a future market change with filter with key succeeds", testNewFutureMarketChangeSubmissionWithFilterWithKeySucceeds)
	t.Run("Submitting a future market change with filter without key name fails", testNewFutureMarketChangeSubmissionWithFilterWithoutKeyNameFails)
	t.Run("Submitting a future market change with filter with key name succeeds", testNewFutureMarketChangeSubmissionWithFilterWithKeyNameSucceeds)
	t.Run("Submitting a future market change with filter without key type fails", testNewFutureMarketChangeSubmissionWithFilterWithoutKeyTypeFails)
	t.Run("Submitting a future market change with filter with key type succeeds", testNewFutureMarketChangeSubmissionWithFilterWithKeyTypeSucceeds)
	t.Run("Submitting a future market change with filter without condition succeeds", testNewFutureMarketChangeSubmissionWithFilterWithoutConditionsSucceeds)
	t.Run("Submitting a future market change with filter without condition operator fails", testNewFutureMarketChangeSubmissionWithFilterWithoutConditionOperatorFails)
	t.Run("Submitting a future market change with filter with condition operator succeeds", testNewFutureMarketChangeSubmissionWithFilterWithConditionOperatorSucceeds)
	t.Run("Submitting a future market change with filter without condition value fails", testNewFutureMarketChangeSubmissionWithFilterWithoutConditionValueFails)
	t.Run("Submitting a future market change with filter with condition value succeeds", testNewFutureMarketChangeSubmissionWithFilterWithConditionValueSucceeds)
	t.Run("Submitting a future market change without oracle spec bindings fails", testNewFutureMarketChangeSubmissionWithoutOracleSpecBindingFails)
	t.Run("Submitting a future market change with oracle spec binding succeeds", testNewFutureMarketChangeSubmissionWithOracleSpecBindingSucceeds)
	t.Run("Submitting a future market change without settlement price property fails", testNewFutureMarketChangeSubmissionWithoutSettlementPricePropertyFails)
	t.Run("Submitting a future market change without trading termination property fails", testNewFutureMarketChangeSubmissionWithoutTradingTerminationPropertyFails)
	t.Run("Submitting a future market change with a mismatch between binding property name and filter fails", testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingFails)
	t.Run("Submitting a future market change with match between binding property name and filter succeeds", testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingSucceeds)
	t.Run("Submitting a future market change with settlement price and trading termination properties succeeds", testNewFutureMarketChangeSubmissionWithSettlementPricePropertySucceeds)
	t.Run("Submitting a simple risk parameters change without simple risk parameters fails", testNewSimpleRiskParametersChangeSubmissionWithoutSimpleRiskParametersFails)
	t.Run("Submitting a simple risk parameters change with simple risk parameters succeeds", testNewSimpleRiskParametersChangeSubmissionWithSimpleRiskParametersSucceeds)
	t.Run("Submitting a simple risk parameters change with min move down fails", testNewSimpleRiskParametersChangeSubmissionWithPositiveMinMoveDownFails)
	t.Run("Submitting a simple risk parameters change with min move down succeeds", testNewSimpleRiskParametersChangeSubmissionWithNonPositiveMinMoveDownSucceeds)
	t.Run("Submitting a simple risk parameters change with max move up fails", testNewSimpleRiskParametersChangeSubmissionWithNegativeMaxMoveUpFails)
	t.Run("Submitting a simple risk parameters change with max move up succeeds", testNewSimpleRiskParametersChangeSubmissionWithNonNegativeMaxMoveUpSucceeds)
	t.Run("Submitting a simple risk parameters change with wrong probability of trading fails", testNewSimpleRiskParametersChangeSubmissionWithWrongProbabilityOfTradingFails)
	t.Run("Submitting a simple risk parameters change with right probability of trading succeeds", testNewSimpleRiskParametersChangeSubmissionWithRightProbabilityOfTradingSucceeds)
	t.Run("Submitting a log normal risk parameters change without log normal risk parameters fails", testNewLogNormalRiskParametersChangeSubmissionWithoutLogNormalRiskParametersFails)
	t.Run("Submitting a log normal risk parameters change with log normal risk parameters succeeds", testNewLogNormalRiskParametersChangeSubmissionWithLogNormalRiskParametersSucceeds)
	t.Run("Submitting a log normal risk parameters change with params fails", testNewLogNormalRiskParametersChangeSubmissionWithoutParamsFails)
	t.Run("Submitting a log normal risk parameters change with invalid risk aversion", testNewLogNormalRiskParametersChangeSubmissionInvalidRiskAversion)
	t.Run("Submitting a log normal risk parameters change with invalid tau", testNewLogNormalRiskParametersChangeSubmissionInvalidTau)
	t.Run("Submitting a log normal risk parameters change with invalid mu", testNewLogNormalRiskParametersChangeSubmissionInvalidMu)
	t.Run("Submitting a log normal risk parameters change with invalid sigma", testNewLogNormalRiskParametersChangeSubmissionInvalidSigma)
	t.Run("Submitting a log normal risk parameters change with invalid r", testNewLogNormalRiskParametersChangeSubmissionInvalidR)
	t.Run("Submitting a new market with a too long reference fails", testNewMarketSubmissionWithTooLongReferenceFails)
	t.Run("Submitting a future market with internal time for trade termination succeeds", testFutureMarketSubmissionWithInternalTimestampForTradingTerminationSucceeds)
	t.Run("Submitting a future market with trade termination from external oracle with no public key fails", testFutureMarketSubmissionWithExternalTradingTerminationNoPublicKeyFails)
}

func testNewMarketChangeSubmissionWithoutNewMarketFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithoutChangesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithoutDecimalPlacesSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.decimal_places"), commands.ErrMustBePositiveOrZero)
}

func testNewMarketChangeSubmissionWithDecimalPlacesEqualTo0Succeeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						DecimalPlaces: 0,
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.decimal_places"), commands.ErrMustBePositiveOrZero)
}

func testNewMarketChangeSubmissionWithDecimalPlacesAboveOrEqualTo150Fails(t *testing.T) {
	testCases := []struct {
		msg   string
		value uint64
	}{
		{
			msg:   "equal to 150",
			value: 150,
		}, {
			msg:   "above 150",
			value: 1000,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								DecimalPlaces: tc.value,
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.decimal_places"), commands.ErrMustBeLessThan150)
		})
	}
}

func testNewMarketChangeSubmissionWithDecimalPlacesBelow150Succeeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						DecimalPlaces: RandomPositiveU64Before(150),
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.decimal_places"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.decimal_places"), commands.ErrMustBeLessThan150)
}

func testNewMarketChangeSubmissionWithPositionDecimalPlacesEqualTo0Succeeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						PositionDecimalPlaces: 0,
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.position_decimal_places"), commands.ErrMustBePositiveOrZero)
}

func testNewMarketChangeSubmissionWithPositionDecimalPlacesAboveOrEqualTo7Fails(t *testing.T) {
	testCases := []struct {
		msg   string
		value uint64
	}{
		{
			msg:   "equal to 7",
			value: 7,
		}, {
			msg:   "above 7",
			value: 1000,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								PositionDecimalPlaces: tc.value,
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.position_decimal_places"), commands.ErrMustBeLessThan7)
		})
	}
}

func testNewMarketChangeSubmissionWithPositionDecimalPlacesBelow7Succeeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						PositionDecimalPlaces: RandomPositiveU64Before(7),
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.position_decimal_places"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.position_decimal_places"), commands.ErrMustBeLessThan7)
}

func testNewMarketChangeSubmissionWithoutLiquidityMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithLiquidityMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters"), commands.ErrIsRequired)
}

func testLiquidityMonitoringChangeSubmissionWithWrongTriggeringRatioFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value float64
	}{
		{
			msg:   "with probability of -1",
			value: -1,
		}, {
			msg:   "with probability of 2",
			value: 2,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
									TriggeringRatio: tc.value,
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.triggering_ratio"),
				errors.New("should be between 0 (inclusive) and 1 (inclusive)"))
		})
	}
}

func testLiquidityMonitoringChangeSubmissionWithRightTriggeringRatioSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value float64
	}{
		{
			msg:   "with ratio of 0",
			value: 0,
		}, {
			msg:   "with ratio of 0.5",
			value: 0.5,
		}, {
			msg:   "with ratio of 1",
			value: 1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
									TriggeringRatio: tc.value,
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters"),
				errors.New("should be between 0 (inclusive) and 1 (inclusive)"))
		})
	}
}

func testLiquidityMonitoringChangeSubmissionWithoutTargetStakeParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.target_stake_parameters"), commands.ErrIsRequired)
}

func testLiquidityMonitoringChangeSubmissionWithTargetStakeParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
							TargetStakeParameters: &types.TargetStakeParameters{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.target_stake_parameters"), commands.ErrIsRequired)
}

func testLiquidityMonitoringChangeSubmissionWithNonPositiveTimeWindowFails(t *testing.T) {
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
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
									TargetStakeParameters: &types.TargetStakeParameters{
										TimeWindow: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.target_stake_parameters.time_window"), commands.ErrMustBePositive)
		})
	}
}

func testLiquidityMonitoringChangeSubmissionWithPositiveTimeWindowSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.target_stake_parameters.time_window"), commands.ErrMustBePositive)
}

func testLiquidityMonitoringChangeSubmissionWithNonPositiveScalingFactorFails(t *testing.T) {
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
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
									TargetStakeParameters: &types.TargetStakeParameters{
										ScalingFactor: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.target_stake_parameters.scaling_factor"), commands.ErrMustBePositive)
		})
	}
}

func testLiquidityMonitoringChangeSubmissionWithPositiveScalingFactorSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.target_stake_parameters.scaling_factor"), commands.ErrMustBePositive)
}

func testPriceMonitoringChangeSubmissionWithoutTriggersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						PriceMonitoringParameters: &types.PriceMonitoringParameters{
							Triggers: []*types.PriceMonitoringTrigger{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers"), commands.ErrIsRequired)
}

func testPriceMonitoringChangeSubmissionWithTriggersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithTooManyPMTriggersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						PriceMonitoringParameters: &types.PriceMonitoringParameters{
							Triggers: []*types.PriceMonitoringTrigger{
								{},
								{},
								{},
								{},
								{},
								{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers"), errors.New("maximum 5 triggers allowed"))
}

func testPriceMonitoringChangeSubmissionWithoutTriggerHorizonFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.0.horizon"), commands.ErrMustBePositive)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.1.horizon"), commands.ErrMustBePositive)
}

func testPriceMonitoringChangeSubmissionWithTriggerHorizonSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.0.horizon"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.1.horizon"), commands.ErrMustBePositive)
}

func testPriceMonitoringChangeSubmissionWithWrongTriggerProbabilityFails(t *testing.T) {
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
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
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

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.0.probability"),
				errors.New("should be between 0 (exclusive) and 1 (exclusive)"))
			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.1.probability"),
				errors.New("should be between 0 (exclusive) and 1 (exclusive)"))
		})
	}
}

func testPriceMonitoringChangeSubmissionWithRightTriggerProbabilitySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.0.probability"),
		errors.New("should be between 0 (exclusive) and 1 (exclusive)"))
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.1.probability"),
		errors.New("should be between 0 (exclusive) and 1 (exclusive)"))
}

func testPriceMonitoringChangeSubmissionWithoutTriggerAuctionExtensionFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.0.auction_extension"), commands.ErrMustBePositive)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.1.auction_extension"), commands.ErrMustBePositive)
}

func testPriceMonitoringChangeSubmissionWithTriggerAuctionExtensionSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.0.auction_extension"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.1.auction_extension"), commands.ErrMustBePositive)
}

func testNewMarketChangeSubmissionWithoutPriceMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithPriceMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						PriceMonitoringParameters: &types.PriceMonitoringParameters{},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithoutInstrumentNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Name: "",
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.name"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithInstrumentNameSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Name: "My name",
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.name"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithoutInstrumentCodeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Code: "",
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.code"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithInstrumentCodeSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Code: "My code",
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.code"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithoutProductFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithProductSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithoutFutureFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFutureSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithoutSettlementAssetFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									SettlementAsset: "",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.settlement_asset"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithSettlementAssetSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									SettlementAsset: "BTC",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.settlement_asset"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithoutQuoteNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									QuoteName: "",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.quote_name"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithQuoteNameSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									QuoteName: "BTC",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.quote_name"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithoutOracleSpecFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{},
							},
						},
					},
				},
			},
		},
	})

	fmt.Println(err)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_trading_termination"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionMissingSingleOracleSpecFails(t *testing.T) {
	testNewFutureMarketChangeSubmissionWithoutEitherOracleSpecFails(t, "oracle_spec_for_settlement_data")
	testNewFutureMarketChangeSubmissionWithoutEitherOracleSpecFails(t, "oracle_spec_for_trading_termination")
}

func testNewFutureMarketChangeSubmissionWithoutEitherOracleSpecFails(t *testing.T, oracleSpecName string) {
	t.Helper()
	future := &types.FutureProduct{}
	if oracleSpecName == "oracle_spec_for_settlement_data" {
		future.OracleSpecForTradingTermination = &oraclespb.OracleSpecConfiguration{}
	} else {
		future.OracleSpecForSettlementData = &oraclespb.OracleSpecConfiguration{}
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future."+oracleSpecName), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithOracleSpecSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData:     &oraclespb.OracleSpecConfiguration{},
									OracleSpecForTradingTermination: &oraclespb.OracleSpecConfiguration{},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithoutPubKeysFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										PubKeys: []string{},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.pub_keys"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithWrongPubKeysFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value []string
	}{
		{
			msg:   "with empty pub-keys",
			value: []string{"0xDEADBEEF", ""},
		}, {
			msg:   "with blank pub-keys",
			value: []string{"0xDEADBEEF", " "},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								Instrument: &types.InstrumentConfiguration{
									Product: &types.InstrumentConfiguration_Future{
										Future: &types.FutureProduct{
											OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
												PubKeys: tc.value,
											},
										},
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.pub_keys.1"), commands.ErrIsNotValid)
		})
	}
}

func testNewFutureMarketChangeSubmissionWithPubKeysSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										PubKeys: []string{"0xDEADBEEF", "0xCAFEDUDE"},
									},
									OracleSpecForTradingTermination: &oraclespb.OracleSpecConfiguration{
										PubKeys: []string{"0xDEADBEEF", "0xCAFEDUDE"},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.pub_keys"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.pub_keys.0"), commands.ErrIsNotValid)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.pub_keys.1"), commands.ErrIsNotValid)
}

func testNewFutureMarketChangeSubmissionWithoutFiltersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFiltersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{},
										},
									},
									OracleSpecForTradingTermination: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{}, {},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.0.key"), commands.ErrIsNotValid)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.1.key"), commands.ErrIsNotValid)
}

func testNewFutureMarketChangeSubmissionWithFilterWithKeySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{
												Key: &oraclespb.PropertyKey{},
											}, {
												Key: &oraclespb.PropertyKey{},
											},
										},
									},
									OracleSpecForTradingTermination: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{
												Key: &oraclespb.PropertyKey{},
											}, {
												Key: &oraclespb.PropertyKey{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.0.key"), commands.ErrIsNotValid)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.1.key"), commands.ErrIsNotValid)
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutKeyNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{
												Key: &oraclespb.PropertyKey{
													Name: "",
												},
											}, {
												Key: &oraclespb.PropertyKey{
													Name: "",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.0.key.name"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.1.key.name"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithKeyNameSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{
												Key: &oraclespb.PropertyKey{
													Name: "key1",
												},
											}, {
												Key: &oraclespb.PropertyKey{
													Name: "key2",
												},
											},
										},
									},
									OracleSpecForTradingTermination: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{
												Key: &oraclespb.PropertyKey{
													Name: "key1",
												},
											}, {
												Key: &oraclespb.PropertyKey{
													Name: "key2",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.0.key.name"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.1.key.name"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutKeyTypeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{
												Key: &oraclespb.PropertyKey{
													Type: oraclespb.PropertyKey_TYPE_UNSPECIFIED,
												},
											}, {
												Key: &oraclespb.PropertyKey{},
											},
										},
									},
									OracleSpecForTradingTermination: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{
												Key: &oraclespb.PropertyKey{
													Type: oraclespb.PropertyKey_TYPE_UNSPECIFIED,
												},
											}, {
												Key: &oraclespb.PropertyKey{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.0.key.type"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.1.key.type"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithKeyTypeSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value oraclespb.PropertyKey_Type
	}{
		{
			msg:   "with EMPTY",
			value: oraclespb.PropertyKey_TYPE_EMPTY,
		}, {
			msg:   "with INTEGER",
			value: oraclespb.PropertyKey_TYPE_INTEGER,
		}, {
			msg:   "with STRING",
			value: oraclespb.PropertyKey_TYPE_STRING,
		}, {
			msg:   "with BOOLEAN",
			value: oraclespb.PropertyKey_TYPE_BOOLEAN,
		}, {
			msg:   "with DECIMAL",
			value: oraclespb.PropertyKey_TYPE_DECIMAL,
		}, {
			msg:   "with TIMESTAMP",
			value: oraclespb.PropertyKey_TYPE_TIMESTAMP,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								Instrument: &types.InstrumentConfiguration{
									Product: &types.InstrumentConfiguration_Future{
										Future: &types.FutureProduct{
											OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
												Filters: []*oraclespb.Filter{
													{
														Key: &oraclespb.PropertyKey{
															Type: tc.value,
														},
													}, {
														Key: &oraclespb.PropertyKey{
															Type: tc.value,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters.0.key.type"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters.1.key.type"), commands.ErrIsRequired)
		})
	}
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutConditionsSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{
												Conditions: []*oraclespb.Condition{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters.0.conditions"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutConditionOperatorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{
												Conditions: []*oraclespb.Condition{
													{
														Operator: oraclespb.Condition_OPERATOR_UNSPECIFIED,
													},
													{},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.0.conditions.0.operator"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.0.conditions.1.operator"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithConditionOperatorSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value oraclespb.Condition_Operator
	}{
		{
			msg:   "with EQUALS",
			value: oraclespb.Condition_OPERATOR_EQUALS,
		}, {
			msg:   "with GREATER_THAN",
			value: oraclespb.Condition_OPERATOR_GREATER_THAN,
		}, {
			msg:   "with GREATER_THAN_OR_EQUAL",
			value: oraclespb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
		}, {
			msg:   "with LESS_THAN",
			value: oraclespb.Condition_OPERATOR_LESS_THAN,
		}, {
			msg:   "with LESS_THAN_OR_EQUAL",
			value: oraclespb.Condition_OPERATOR_LESS_THAN_OR_EQUAL,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								Instrument: &types.InstrumentConfiguration{
									Product: &types.InstrumentConfiguration_Future{
										Future: &types.FutureProduct{
											OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
												Filters: []*oraclespb.Filter{
													{
														Conditions: []*oraclespb.Condition{
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
										},
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters.0.conditions.0.operator"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters.1.conditions.0.operator"), commands.ErrIsRequired)
		})
	}
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutConditionValueFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{
												Conditions: []*oraclespb.Condition{
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
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.0.conditions.0.value"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_settlement_data.filters.0.conditions.1.value"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithConditionValueSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{
												Conditions: []*oraclespb.Condition{
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
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters.0.conditions.0.value"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters.0.conditions.1.value"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithoutOracleSpecBindingFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_binding"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithOracleSpecBindingSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecBinding: &types.OracleSpecToFutureBinding{},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_binding"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionMissingOracleBindingPropertyFails(t *testing.T, property string) {
	t.Helper()
	var binding *types.OracleSpecToFutureBinding
	if property == "settlement_price_property" {
		binding = &types.OracleSpecToFutureBinding{
			SettlementPriceProperty: "",
		}
	} else {
		binding = &types.OracleSpecToFutureBinding{
			TradingTerminationProperty: "",
		}
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecBinding: binding,
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_binding."+property), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithoutTradingTerminationPropertyFails(t *testing.T) {
	testNewFutureMarketChangeSubmissionMissingOracleBindingPropertyFails(t, "trading_termination_property")
}

func testNewFutureMarketChangeSubmissionWithoutSettlementPricePropertyFails(t *testing.T) {
	testNewFutureMarketChangeSubmissionMissingOracleBindingPropertyFails(t, "settlement_price_property")
}

func testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingSucceeds(t *testing.T) {
	testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t, &types.OracleSpecToFutureBinding{SettlementPriceProperty: "key1"}, "settlement_price_property", "key1")
	testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t, &types.OracleSpecToFutureBinding{TradingTerminationProperty: "key2"}, "settlement_price_property", "key2")
}

func testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t *testing.T, binding *types.OracleSpecToFutureBinding, bindingName string, bindingKey string) {
	t.Helper()
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForSettlementData: &oraclespb.OracleSpecConfiguration{
										Filters: []*oraclespb.Filter{
											{
												Key: &oraclespb.PropertyKey{
													Name: bindingKey,
												},
											}, {
												Key: &oraclespb.PropertyKey{},
											},
										},
									},
									OracleSpecBinding: binding,
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_binding."+bindingName), commands.ErrIsMismatching)
}

func testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t *testing.T, binding *types.OracleSpecToFutureBinding, bindingName string) {
	t.Helper()
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecBinding: binding,
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_binding."+bindingName), commands.ErrIsMismatching)
}

func testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingFails(t *testing.T) {
	testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t, &types.OracleSpecToFutureBinding{SettlementPriceProperty: "My property"}, "settlement_price_property")
	testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t, &types.OracleSpecToFutureBinding{TradingTerminationProperty: "My property"}, "trading_termination_property")
}

func testNewFutureMarketChangeSubmissionWithSettlementPricePropertySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecBinding: &types.OracleSpecToFutureBinding{
										SettlementPriceProperty: "My property",
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_binding.settlement_price_property"), commands.ErrIsRequired)
}

func testNewSimpleRiskParametersChangeSubmissionWithoutSimpleRiskParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_Simple{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.simple"), commands.ErrIsRequired)
}

func testNewSimpleRiskParametersChangeSubmissionWithSimpleRiskParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_Simple{
							Simple: &types.SimpleModelParams{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.simple"), commands.ErrIsRequired)
}

func testNewSimpleRiskParametersChangeSubmissionWithPositiveMinMoveDownFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_Simple{
							Simple: &types.SimpleModelParams{
								MinMoveDown: 1,
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.simple.min_move_down"), commands.ErrMustBeNegativeOrZero)
}

func testNewSimpleRiskParametersChangeSubmissionWithNonPositiveMinMoveDownSucceeds(t *testing.T) {
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
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								RiskParameters: &types.NewMarketConfiguration_Simple{
									Simple: &types.SimpleModelParams{
										MinMoveDown: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.simple.min_move_down"), commands.ErrMustBeNegativeOrZero)
		})
	}
}

func testNewSimpleRiskParametersChangeSubmissionWithNegativeMaxMoveUpFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_Simple{
							Simple: &types.SimpleModelParams{
								MaxMoveUp: -1,
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.simple.max_move_up"), commands.ErrMustBePositiveOrZero)
}

func testNewSimpleRiskParametersChangeSubmissionWithNonNegativeMaxMoveUpSucceeds(t *testing.T) {
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
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								RiskParameters: &types.NewMarketConfiguration_Simple{
									Simple: &types.SimpleModelParams{
										MaxMoveUp: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.simple.max_move_up"), commands.ErrMustBePositiveOrZero)
		})
	}
}

func testNewSimpleRiskParametersChangeSubmissionWithWrongProbabilityOfTradingFails(t *testing.T) {
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
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								RiskParameters: &types.NewMarketConfiguration_Simple{
									Simple: &types.SimpleModelParams{
										ProbabilityOfTrading: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.simple.probability_of_trading"),
				errors.New("should be between 0 (inclusive) and 1 (inclusive)"))
		})
	}
}

func testNewSimpleRiskParametersChangeSubmissionWithRightProbabilityOfTradingSucceeds(t *testing.T) {
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
					Change: &types.ProposalTerms_NewMarket{
						NewMarket: &types.NewMarket{
							Changes: &types.NewMarketConfiguration{
								RiskParameters: &types.NewMarketConfiguration_Simple{
									Simple: &types.SimpleModelParams{
										ProbabilityOfTrading: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.simple.probability_of_trading"),
				errors.New("should be between 0 (inclusive) and 1 (inclusive)"))
		})
	}
}

func testNewLogNormalRiskParametersChangeSubmissionWithoutLogNormalRiskParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_LogNormal{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal"), commands.ErrIsRequired)
}

func testNewLogNormalRiskParametersChangeSubmissionWithLogNormalRiskParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_LogNormal{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal"), commands.ErrIsRequired)
}

func testNewLogNormalRiskParametersChangeSubmissionWithoutParamsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_LogNormal{
							LogNormal: &types.LogNormalRiskModel{},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params"), commands.ErrIsRequired)
}

func testNewLogNormalRiskParametersChangeSubmissionInvalidRiskAversion(t *testing.T) {
	cZero := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), commands.ErrMustBePositive)

	cNeg := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), commands.ErrMustBePositive)
}

func testNewLogNormalRiskParametersChangeSubmissionInvalidTau(t *testing.T) {
	cZero := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.tau"), commands.ErrMustBePositive)

	cNeg := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.tau"), commands.ErrMustBePositive)
}

func testNewLogNormalRiskParametersChangeSubmissionInvalidMu(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.mu"), commands.ErrIsNotValidNumber)
}

func testNewLogNormalRiskParametersChangeSubmissionInvalidR(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.r"), commands.ErrIsNotValidNumber)
}

func testNewLogNormalRiskParametersChangeSubmissionInvalidSigma(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.sigma"), commands.ErrIsNotValidNumber)

	cNeg := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.sigma"), commands.ErrMustBePositive)

	c0 := &commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						RiskParameters: &types.NewMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.sigma"), commands.ErrMustBePositive)
}

func testNewMarketSubmissionWithTooLongReferenceFails(t *testing.T) {
	ref := make([]byte, 101)
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Reference: string(ref),
	})
	assert.Contains(t, err.Get("proposal_submission.reference"), commands.ErrReferenceTooLong)
}

func testFutureMarketSubmissionWithInternalTimestampForTradingTerminationSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForTradingTermination: &oraclespb.OracleSpecConfiguration{
										PubKeys: []string{},
										Filters: []*oraclespb.Filter{
											{
												Key: &oraclespb.PropertyKey{
													Name: "vegaprotocol.builtin.timestamp",
													Type: oraclespb.PropertyKey_TYPE_TIMESTAMP,
												},
												Conditions: []*oraclespb.Condition{
													{
														Operator: oraclespb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
														Value:    fmt.Sprintf("%d", time.Now().Add(time.Hour*24*365).UnixNano()),
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_trading_termination.pub_keys"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithExternalTradingTerminationNoPublicKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{
				NewMarket: &types.NewMarket{
					Changes: &types.NewMarketConfiguration{
						Instrument: &types.InstrumentConfiguration{
							Product: &types.InstrumentConfiguration_Future{
								Future: &types.FutureProduct{
									OracleSpecForTradingTermination: &oraclespb.OracleSpecConfiguration{
										PubKeys: []string{},
										Filters: []*oraclespb.Filter{
											{
												Key: &oraclespb.PropertyKey{
													Name: "trading.terminated",
													Type: oraclespb.PropertyKey_TYPE_BOOLEAN,
												},
												Conditions: []*oraclespb.Condition{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_for_trading_termination.pub_keys"), commands.ErrIsRequired)
}
