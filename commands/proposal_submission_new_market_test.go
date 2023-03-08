package commands_test

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/core/types"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
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
	t.Run("Submitting a market change with lp price range 'banana' fails", testNewMarketChangeSubmissionWithLpRangeBananaFails)
	t.Run("Submitting a market change with slippage factor 'banana' fails", testNewMarketChangeSubmissionWithSlippageFactorBananaFails)
	t.Run("Submitting a market change with negative slippage factor fails", testNewMarketChangeSubmissionWithSlippageFactorNegativeFails)
	t.Run("Submitting a market change with empty max slippage factor succeeds", testNewMarketChangeSubmissionWithEmptySlippageFactorPasses)
	t.Run("Submitting a market change with too large slippage factor fails", testNewMarketChangeSubmissionWithSlippageFactorTooLargeFails)
	t.Run("Submitting a market change with lp price range below 0 fails", testNewMarketChangeSubmissionWithLpRangeNegativeFails)
	t.Run("Submitting a market change with lp price range equal to 0 fails", testNewMarketChangeSubmissionWithLpRangeZeroFails)
	t.Run("Submitting a market change with lp price range in [0,100] range succeeds", testNewMarketChangeSubmissionWithLpRangeGreaterThan100)
	t.Run("Submitting a market change with lp price range above 100 fails", testNewMarketChangeSubmissionWithLpRangePositiveSucceeds)
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
	t.Run("Submitting a liquidity monitoring change without triggering ratio parameter fails", testLiquidityMonitoringChangeSubmissionWithoutTriggeringRatioFails)
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
	t.Run("Submitting a future market change with bad pubkey or address fails", testNewFutureMarketChangeSubmissionWithBadPubKeysOrderAddressFail)
	t.Run("Submitting a future market change with good pubkey or address succeed", testNewFutureMarketChangeSubmissionWithGoodPubKeysOrderAddressSucceed)
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
	t.Run("Submitting a future market change without oracle spec bindings fails", testNewFutureMarketChangeSubmissionWithoutDataSourceSpecBindingFails)
	t.Run("Submitting a future market change with oracle spec binding succeeds", testNewFutureMarketChangeSubmissionWithDataSourceSpecBindingSucceeds)
	t.Run("Submitting a future market change without settlement data property fails", testNewFutureMarketChangeSubmissionWithoutSettlementDataPropertyFails)
	t.Run("Submitting a future market change without trading termination property fails", testNewFutureMarketChangeSubmissionWithoutTradingTerminationPropertyFails)
	t.Run("Submitting a future market change with a mismatch between binding property name and filter fails", testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingFails)
	t.Run("Submitting a future market change with match between binding property name and filter succeeds", testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingSucceeds)
	t.Run("Submitting a future market change with settlement data and trading termination properties succeeds", testNewFutureMarketChangeSubmissionWithSettlementDataPropertySucceeds)
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
	t.Run("Submitting a future market with trading termination from internal source with no public keys succeeds", testFutureMarketSubmissionWithInternalTradingTerminationSucceeds)
	t.Run("Submitting a future market with trading termination from internal source with invalid operator fails", testFutureMarketSubmissionWithInternalTradingInvalidOperatorTerminationFails)
	t.Run("Submitting a future market with trading termination from external source with `vegaprotocol.builtin` key and no public keys fails", testFutureMarketSubmissionWithExternalTradingTerminationBuiltInKeyNoPublicKeyFails)
	t.Run("Submitting a future market with trading settlement from external source with `vegaprotocol.builtin` key and no public keys fails", testFutureMarketSubmissionWithExternalTradingSettlementBuiltInKeyNoPublicKeyFails)

	t.Run("Submitting a future market with trade termination from oracle with no public key fails", testFutureMarketSubmissionWithExternalTradingTerminationNoPublicKeyFails)
}

func testNewMarketChangeSubmissionWithoutNewMarketFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithoutChangesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithoutDecimalPlacesSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.decimal_places"), commands.ErrMustBePositiveOrZero)
}

func testNewMarketChangeSubmissionWithDecimalPlacesEqualTo0Succeeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
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
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
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
		value int64
	}{
		{
			msg:   "equal to 7",
			value: 7,
		},
		{
			msg:   "greater than 7",
			value: 8,
		},
		{
			msg:   "equal to -7",
			value: -7,
		},
		{
			msg:   "less than -7",
			value: -8,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								PositionDecimalPlaces: tc.value,
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.position_decimal_places"), commands.ErrMustBeWithinRange7)
		})
	}
}

func testNewMarketChangeSubmissionWithPositionDecimalPlacesBelow7Succeeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						PositionDecimalPlaces: RandomPositiveI64Before(7),
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.position_decimal_places"), commands.ErrMustBeWithinRange7)
}

func testNewMarketChangeSubmissionWithLpRangeBananaFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LpPriceRange: "banana",
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrIsNotValidNumber)
}

func testNewMarketChangeSubmissionWithSlippageFactorBananaFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LinearSlippageFactor: "banana",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.linear_slippage_factor"), commands.ErrIsNotValidNumber)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						QuadraticSlippageFactor: "banana",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.quadratic_slippage_factor"), commands.ErrIsNotValidNumber)
}

func testNewMarketChangeSubmissionWithSlippageFactorNegativeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LinearSlippageFactor: "-0.1",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.linear_slippage_factor"), commands.ErrMustBePositiveOrZero)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						QuadraticSlippageFactor: "-0.1",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.quadratic_slippage_factor"), commands.ErrMustBePositiveOrZero)
}

func testNewMarketChangeSubmissionWithSlippageFactorTooLargeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LinearSlippageFactor: "1000000.000001",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.linear_slippage_factor"), commands.ErrMustBeAtMost1M)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						QuadraticSlippageFactor: "1000000.000001",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.quadratic_slippage_factor"), commands.ErrMustBeAtMost1M)
}

func testNewMarketChangeSubmissionWithEmptySlippageFactorPasses(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{},
				},
			},
		},
	})
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.linear_slippage_factor"), commands.ErrIsNotValidNumber)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.quadratic_slippage_factor"), commands.ErrIsNotValidNumber)
}

func testNewMarketChangeSubmissionWithLpRangeNegativeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LpPriceRange: "-1e-17",
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrMustBePositive)
}

func testNewMarketChangeSubmissionWithLpRangeGreaterThan100(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LpPriceRange: "100.0000000000001",
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrMustBeAtMost100)
}

func testNewMarketChangeSubmissionWithLpRangeZeroFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LpPriceRange: "0",
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrMustBePositive)
}

func testNewMarketChangeSubmissionWithLpRangePositiveSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LpPriceRange: "1e-17",
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrIsNotValidNumber)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrMustBeAtMost100)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LpPriceRange: "0.95",
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrIsNotValidNumber)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrMustBeAtMost100)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LpPriceRange: "100",
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrIsNotValidNumber)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.lp_price_range"), commands.ErrMustBeAtMost100)
}

func testNewMarketChangeSubmissionWithoutLiquidityMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithLiquidityMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LiquidityMonitoringParameters: &protoTypes.LiquidityMonitoringParameters{},
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
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								LiquidityMonitoringParameters: &protoTypes.LiquidityMonitoringParameters{
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
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								LiquidityMonitoringParameters: &protoTypes.LiquidityMonitoringParameters{
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

func testLiquidityMonitoringChangeSubmissionWithoutTriggeringRatioFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LiquidityMonitoringParameters: &protoTypes.LiquidityMonitoringParameters{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.triggering_ratio"), commands.ErrIsNotValidNumber)
}

func testLiquidityMonitoringChangeSubmissionWithoutTargetStakeParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LiquidityMonitoringParameters: &protoTypes.LiquidityMonitoringParameters{
							TriggeringRatio: "1",
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.target_stake_parameters"), commands.ErrIsRequired)
}

func testLiquidityMonitoringChangeSubmissionWithTargetStakeParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LiquidityMonitoringParameters: &protoTypes.LiquidityMonitoringParameters{
							TargetStakeParameters: &protoTypes.TargetStakeParameters{},
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
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								LiquidityMonitoringParameters: &protoTypes.LiquidityMonitoringParameters{
									TriggeringRatio: "1",
									TargetStakeParameters: &protoTypes.TargetStakeParameters{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LiquidityMonitoringParameters: &protoTypes.LiquidityMonitoringParameters{
							TargetStakeParameters: &protoTypes.TargetStakeParameters{
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
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								LiquidityMonitoringParameters: &protoTypes.LiquidityMonitoringParameters{
									TriggeringRatio: "1",
									TargetStakeParameters: &protoTypes.TargetStakeParameters{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						LiquidityMonitoringParameters: &protoTypes.LiquidityMonitoringParameters{
							TargetStakeParameters: &protoTypes.TargetStakeParameters{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
							Triggers: []*protoTypes.PriceMonitoringTrigger{},
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
							Triggers: []*protoTypes.PriceMonitoringTrigger{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
							Triggers: []*protoTypes.PriceMonitoringTrigger{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
							Triggers: []*protoTypes.PriceMonitoringTrigger{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
							Triggers: []*protoTypes.PriceMonitoringTrigger{
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
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
									Triggers: []*protoTypes.PriceMonitoringTrigger{
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
				errors.New("should be between 0.9 (exclusive) and 1 (exclusive)"))
			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.1.probability"),
				errors.New("should be between 0.9 (exclusive) and 1 (exclusive)"))
		})
	}
}

func testPriceMonitoringChangeSubmissionWithRightTriggerProbabilitySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
							Triggers: []*protoTypes.PriceMonitoringTrigger{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
							Triggers: []*protoTypes.PriceMonitoringTrigger{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
							Triggers: []*protoTypes.PriceMonitoringTrigger{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithPriceMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithoutInstrumentNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithProductSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{},
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{},
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{},
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination"), commands.ErrIsRequired)
}

// func testNewFutureMarketChangeSubmissionWithInvalidOracleSpecVegaPubkeySignerFails(t *testing.T) {
// 	err := checkProposalSubmission(&commandspb.ProposalSubmission{
// 		Terms: &protoTypes.ProposalTerms{
// 			Change: &protoTypes.ProposalTerms_NewMarket{
// 				NewMarket: &protoTypes.NewMarket{
// 					Changes: &protoTypes.NewMarketConfiguration{
// 						Instrument: &protoTypes.InstrumentConfiguration{
// 							Product: &protoTypes.InstrumentConfiguration_Future{
// 								Future: &protoTypes.FutureProduct{
// 									DataSourceSpecForSettlementData: &protoTypes.DataSourceDefinition{
// 										SourceType: &protoTypes.DataSourceDefinitionExternal_Oracle{

// 										}
// 									}
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	})

// 	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data"), commands.ErrIsRequired)
// 	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination"), commands.ErrIsRequired)
// }

func testNewFutureMarketChangeSubmissionMissingSingleOracleSpecFails(t *testing.T) {
	testNewFutureMarketChangeSubmissionWithoutEitherOracleSpecFails(t, "data_source_spec_for_settlement_data")
	testNewFutureMarketChangeSubmissionWithoutEitherOracleSpecFails(t, "data_source_spec_for_trading_termination")
}

func testNewFutureMarketChangeSubmissionWithoutEitherOracleSpecFails(t *testing.T, oracleSpecName string) {
	t.Helper()
	future := &protoTypes.FutureProduct{}
	if oracleSpecName == "data_source_spec_for_settlement_data" {
		future.DataSourceSpecForTradingTermination = &vegapb.DataSourceDefinition{}
	} else {
		future.DataSourceSpecForSettlementData = &vegapb.DataSourceDefinition{}
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{},
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithoutPubKeysFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(vegapb.DataSourceDefinitionTypeExt),
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithWrongPubKeysFails(t *testing.T) {
	pubKeys := []*types.Signer{
		types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey),
		types.CreateSignerFromString("", types.DataSignerTypePubKey),
	}

	testCases := []struct {
		msg   string
		value []*datapb.Signer
	}{
		{
			msg:   "with empty signers",
			value: types.SignersIntoProto(pubKeys),
		}, {
			msg:   "with blank signers",
			value: types.SignersIntoProto(pubKeys),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								Instrument: &protoTypes.InstrumentConfiguration{
									Product: &protoTypes.InstrumentConfiguration_Future{
										Future: &protoTypes.FutureProduct{
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

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.1"), commands.ErrIsNotValid)
		})
	}
}

func testNewFutureMarketChangeSubmissionWithBadPubKeysOrderAddressFail(t *testing.T) {
	pubKeys := []*types.Signer{
		types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey),
		types.CreateSignerFromString("0xCAFEDUDE", types.DataSignerTypePubKey),
		types.CreateSignerFromString("0xCAFEDUDE", types.DataSignerTypeEthAddress),
		types.CreateSignerFromString("36393436346533356263623865386132393030636130663837616361663235326435306366326162326663373336393438343561313662376338613064633666", types.DataSignerTypePubKey),
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Signers: types.SignersIntoProto(pubKeys),
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Signers: types.SignersIntoProto(pubKeys),
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.0"), commands.ErrIsNotValid)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.1"), commands.ErrIsNotValid)

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.3"), commands.ErrIsNotValidVegaPubkey)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.2"), commands.ErrIsNotValidEthereumAddress)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.1"), commands.ErrIsNotValidVegaPubkey)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.0"), commands.ErrIsNotValidVegaPubkey)
}

func testNewFutureMarketChangeSubmissionWithGoodPubKeysOrderAddressSucceed(t *testing.T) {
	pubKeys := []*types.Signer{
		types.CreateSignerFromString("0x8565a19c49bcD6Fa7b6EB0221a50606F9c9cC683", types.DataSignerTypeEthAddress),
		types.CreateSignerFromString("bd069246503a57271375f1995c46e03db88c4e1a564077b33a9872f905650dc4", types.DataSignerTypePubKey),
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Signers: types.SignersIntoProto(pubKeys),
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Signers: types.SignersIntoProto(pubKeys),
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.0"), commands.ErrIsNotValid)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.1"), commands.ErrIsNotValid)

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.1"), commands.ErrIsNotValidEthereumAddress)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.0"), commands.ErrIsNotValidVegaPubkey)
}

func testNewFutureMarketChangeSubmissionWithoutFiltersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeExt,
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFiltersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.key"), commands.ErrIsNotValid)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.1.key"), commands.ErrIsNotValid)
}

func testNewFutureMarketChangeSubmissionWithFilterWithKeySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.key"), commands.ErrIsNotValid)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.1.key"), commands.ErrIsNotValid)
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutKeyNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.key.name"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.1.key.name"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithKeyNameSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.key.name"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.1.key.name"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutKeyTypeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.key.type"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.1.key.type"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithKeyTypeSucceeds(t *testing.T) {
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
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								Instrument: &protoTypes.InstrumentConfiguration{
									Product: &protoTypes.InstrumentConfiguration_Future{
										Future: &protoTypes.FutureProduct{
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

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.0.key.type"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.1.key.type"), commands.ErrIsRequired)
		})
	}
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutConditionsSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.0.conditions"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutConditionOperatorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.conditions.0.operator"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.conditions.1.operator"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithConditionOperatorSucceeds(t *testing.T) {
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
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								Instrument: &protoTypes.InstrumentConfiguration{
									Product: &protoTypes.InstrumentConfiguration_Future{
										Future: &protoTypes.FutureProduct{
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

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.0.conditions.0.operator"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.1.conditions.0.operator"), commands.ErrIsRequired)
		})
	}
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutConditionValueFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.conditions.0.value"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.conditions.1.value"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithConditionValueSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.0.conditions.0.value"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.0.conditions.1.value"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithoutDataSourceSpecBindingFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_binding"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithDataSourceSpecBindingSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
									DataSourceSpecBinding: &protoTypes.DataSourceSpecToFutureBinding{},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_binding"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionMissingOracleBindingPropertyFails(t *testing.T, property string) {
	t.Helper()
	var binding *protoTypes.DataSourceSpecToFutureBinding
	if property == "settlement_data_property" {
		binding = &protoTypes.DataSourceSpecToFutureBinding{
			SettlementDataProperty: "",
		}
	} else {
		binding = &protoTypes.DataSourceSpecToFutureBinding{
			TradingTerminationProperty: "",
		}
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
									DataSourceSpecBinding: binding,
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_binding."+property), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithoutTradingTerminationPropertyFails(t *testing.T) {
	testNewFutureMarketChangeSubmissionMissingOracleBindingPropertyFails(t, "trading_termination_property")
}

func testNewFutureMarketChangeSubmissionWithoutSettlementDataPropertyFails(t *testing.T) {
	testNewFutureMarketChangeSubmissionMissingOracleBindingPropertyFails(t, "settlement_data_property")
}

func testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingSucceeds(t *testing.T) {
	testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t, &protoTypes.DataSourceSpecToFutureBinding{SettlementDataProperty: "key1"}, "settlement_data_property", "key1")
	testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t, &protoTypes.DataSourceSpecToFutureBinding{TradingTerminationProperty: "key2"}, "settlement_data_property", "key2")
}

func testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t *testing.T, binding *protoTypes.DataSourceSpecToFutureBinding, bindingName string, bindingKey string) {
	t.Helper()
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_binding."+bindingName), commands.ErrIsMismatching)
}

func testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t *testing.T, binding *protoTypes.DataSourceSpecToFutureBinding, bindingName string) {
	t.Helper()
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
									DataSourceSpecBinding: binding,
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_binding."+bindingName), commands.ErrIsMismatching)
}

func testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingFails(t *testing.T) {
	testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t, &protoTypes.DataSourceSpecToFutureBinding{SettlementDataProperty: "My property"}, "settlement_data_property")
	testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t, &protoTypes.DataSourceSpecToFutureBinding{TradingTerminationProperty: "My property"}, "trading_termination_property")
}

func testNewFutureMarketChangeSubmissionWithSettlementDataPropertySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
									DataSourceSpecBinding: &protoTypes.DataSourceSpecToFutureBinding{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_binding.settlement_data_property"), commands.ErrIsRequired)
}

func testNewSimpleRiskParametersChangeSubmissionWithoutSimpleRiskParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_Simple{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.simple"), commands.ErrIsRequired)
}

func testNewSimpleRiskParametersChangeSubmissionWithSimpleRiskParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_Simple{
							Simple: &protoTypes.SimpleModelParams{},
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_Simple{
							Simple: &protoTypes.SimpleModelParams{
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
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								RiskParameters: &protoTypes.NewMarketConfiguration_Simple{
									Simple: &protoTypes.SimpleModelParams{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_Simple{
							Simple: &protoTypes.SimpleModelParams{
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
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								RiskParameters: &protoTypes.NewMarketConfiguration_Simple{
									Simple: &protoTypes.SimpleModelParams{
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
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								RiskParameters: &protoTypes.NewMarketConfiguration_Simple{
									Simple: &protoTypes.SimpleModelParams{
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
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewMarket{
						NewMarket: &protoTypes.NewMarket{
							Changes: &protoTypes.NewMarketConfiguration{
								RiskParameters: &protoTypes.NewMarketConfiguration_Simple{
									Simple: &protoTypes.SimpleModelParams{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal"), commands.ErrIsRequired)
}

func testNewLogNormalRiskParametersChangeSubmissionWithLogNormalRiskParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 1,
								Tau:                   2,
								Params: &protoTypes.LogNormalModelParams{
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params"), commands.ErrIsRequired)
}

func testNewLogNormalRiskParametersChangeSubmissionInvalidRiskAversion(t *testing.T) {
	cTooSmall := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 5e-9,
								Tau:                   1.0,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    0.0,
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
	err := checkProposalSubmission(cTooSmall)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), errors.New("must be between [1e-8, 0.1]"))

	cNeg := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 1e-9,
								Tau:                   2,
								Params: &protoTypes.LogNormalModelParams{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), errors.New("must be between [1e-8, 0.1]"))

	cTooBig := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1 + 1e-8,
								Tau:                   2,
								Params: &protoTypes.LogNormalModelParams{
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
	err = checkProposalSubmission(cTooBig)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), errors.New("must be between [1e-8, 0.1]"))

	cJustAboutRight1 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 1e-8,
								Tau:                   2,
								Params: &protoTypes.LogNormalModelParams{
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
	err = checkProposalSubmission(cJustAboutRight1)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), errors.New("must be between [1e-8, 1)"))

	cJustAboutRight2 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 1 - 1e-12,
								Tau:                   2,
								Params: &protoTypes.LogNormalModelParams{
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
	err = checkProposalSubmission(cJustAboutRight2)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), errors.New("must be between [1e-8, 1)"))
}

func testNewLogNormalRiskParametersChangeSubmissionInvalidTau(t *testing.T) {
	cZero := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0,
								Params: &protoTypes.LogNormalModelParams{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.tau"), errors.New("must be between [1e-8, 1]"))

	cNeg := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   1e-9,
								Params: &protoTypes.LogNormalModelParams{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.tau"), errors.New("must be between [1e-8, 1]"))

	cTooLarge := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   1 + 1e-12,
								Params: &protoTypes.LogNormalModelParams{
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
	err = checkProposalSubmission(cTooLarge)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.tau"), errors.New("must be between [1e-8, 1]"))

	cJustAboutRight1 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   1e-12,
								Params: &protoTypes.LogNormalModelParams{
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
	err = checkProposalSubmission(cJustAboutRight1)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.tau"), errors.New("must be between (0, 1]"))

	cJustAboutRight2 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   1,
								Params: &protoTypes.LogNormalModelParams{
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
	err = checkProposalSubmission(cJustAboutRight2)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.tau"), errors.New("must be between (0, 1]"))
}

func testNewLogNormalRiskParametersChangeSubmissionInvalidMu(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
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

	cTooSmall := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    -1e-6 - 1e-12,
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
	err = checkProposalSubmission(cTooSmall)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.mu"), errors.New("must be between [-1e-6,1e-6]"))

	cTooLarge := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    1e-6 + 1e-12,
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
	err = checkProposalSubmission(cTooLarge)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.mu"), errors.New("must be between [-1e-6,1e-6]"))

	cJustAboutRight1 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    -20,
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
	err = checkProposalSubmission(cJustAboutRight1)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.mu"), errors.New("must be between [-20,20]"))

	cJustAboutRight2 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    20,
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
	err = checkProposalSubmission(cJustAboutRight2)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.mu"), errors.New("must be between [-20,20]"))
}

func testNewLogNormalRiskParametersChangeSubmissionInvalidR(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    0.0,
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

	cTooSmall := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    0.0,
									Sigma: 0.1,
									R:     -1 - 1e-12,
								},
							},
						},
					},
				},
			},
		},
	}
	err = checkProposalSubmission(cTooSmall)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.r"), errors.New("must be between [-1,1]"))

	cTooLarge := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    0.0,
									Sigma: 0.1,
									R:     1 + 1e-12,
								},
							},
						},
					},
				},
			},
		},
	}
	err = checkProposalSubmission(cTooLarge)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.r"), errors.New("must be between [-1,1]"))

	cJustAboutRight1 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    0.0,
									Sigma: 0.1,
									R:     -20,
								},
							},
						},
					},
				},
			},
		},
	}
	err = checkProposalSubmission(cJustAboutRight1)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.r"), errors.New("must be between [-20,20]"))

	cJustAboutRight2 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    0.0,
									Sigma: 0.1,
									R:     20,
								},
							},
						},
					},
				},
			},
		},
	}
	err = checkProposalSubmission(cJustAboutRight2)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.r"), errors.New("must be between [-20,20]"))
}

func testNewLogNormalRiskParametersChangeSubmissionInvalidSigma(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    0.0,
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    0.0,
									Sigma: 1e-4,
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.sigma"), errors.New("must be between [1e-3,50]"))

	cTooSmall := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    0.0,
									Sigma: 1e-3 - 1e-12,
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	}
	err = checkProposalSubmission(cTooSmall)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.sigma"), errors.New("must be between [1e-3,50]"))

	cTooLarge := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    0.0,
									Sigma: 50 + 1e-12,
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	}
	err = checkProposalSubmission(cTooLarge)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.sigma"), errors.New("must be between [1e-3,50]"))

	cJustAboutRight1 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    0.0,
									Sigma: 1e-4,
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	}
	err = checkProposalSubmission(cJustAboutRight1)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.sigma"), errors.New("must be between [1e-4,100]"))

	cJustAboutRight2 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						RiskParameters: &protoTypes.NewMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
									Mu:    0.0,
									Sigma: 50,
									R:     0,
								},
							},
						},
					},
				},
			},
		},
	}
	err = checkProposalSubmission(cJustAboutRight2)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.sigma"), errors.New("must be between [1e-4,100]"))
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
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
									DataSourceSpecForTradingTermination: &vegapb.DataSourceDefinition{
										SourceType: &vegapb.DataSourceDefinition_Internal{
											Internal: &vegapb.DataSourceDefinitionInternal{
												SourceType: &vegapb.DataSourceDefinitionInternal_Time{
													Time: &vegapb.DataSourceSpecConfigurationTime{
														Conditions: []*datapb.Condition{
															{
																Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
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
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithExternalTradingTerminationNoPublicKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithInternalTradingTerminationSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeInt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Signers: []*datapb.Signer{},
											Filters: []*datapb.Filter{
												{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithInternalTradingInvalidOperatorTerminationFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceDefinitionTypeInt,
									).SetOracleConfig(
										&vegapb.DataSourceSpecConfiguration{
											Signers: []*datapb.Signer{},
											Filters: []*datapb.Filter{
												{
													Conditions: []*datapb.Condition{
														{
															Operator: datapb.Condition_OPERATOR_UNSPECIFIED,
															Value:    "value 1",
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.internal.time.conditions[0].operator"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithExternalTradingTerminationBuiltInKeyNoPublicKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithExternalTradingSettlementBuiltInKeyNoPublicKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{
								Future: &protoTypes.FutureProduct{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers"), commands.ErrIsRequired)
}
