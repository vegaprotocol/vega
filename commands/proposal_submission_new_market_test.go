package commands_test

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/commands"
	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/protos/vega"
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
	t.Run("Submitting a future market change with empty oracle spec fails", testNewFutureMarketChangeSubmissionWithEmptyOracleSpecFails)
	t.Run("Submitting a future market change with empty oracle spec type fails", testNewFutureMarketChangeSubmissionWithEmptyOracleSpecTypeFails)
	t.Run("Submitting a future market change with empty internal spec type fails", testNewFutureMarketChangeSubmissionWithEmptyInternalSpecTypeFails)
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
	t.Run("Submitting a future market change without pub-keys fails", testNewFutureMarketChangeSubmissionWithoutPubKeysFails)
	t.Run("Submitting a future market change without trading termination property fails", testNewFutureMarketChangeSubmissionWithoutTradingTerminationPropertyFails)
	t.Run("Submitting a future market change with a mismatch between binding property name and filter fails", testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingFails)
	t.Run("Submitting a future market change with match between binding property name and filter succeeds", testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingSucceeds)
	t.Run("Submitting a future market change with settlement data and trading termination properties succeeds", testNewFutureMarketChangeSubmissionWithSettlementDataPropertySucceeds)
	t.Run("Submitting a simple risk parameters change without simple risk parameters fails", testNewSimpleRiskParametersChangeSubmissionWithoutSimpleRiskParametersFails)
	t.Run("Submitting a simple risk parameters change with simple risk parameters succeeds", testNewSimpleRiskParametersChangeSubmissionWithSimpleRiskParametersSucceeds)
	t.Run("Submitting a simple risk parameters change with min move down fails", testNewSimpleRiskParametersChangeSubmissionWithPositiveMinMoveDownFails)
	t.Run("Submitting a simple risk parameters change with min move down succeeds", testNewSimpleRiskParametersChangeSubmissionWithNonPositiveMinMoveDownSucceeds)
	t.Run("Submitting a simple risk parameters change with max move up fails", testNewSpotSimpleRiskParametersChangeSubmissionWithNegativeMaxMoveUpFails)
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
	t.Run("Submitting a future market with trading settlement from external source with one timestamp filter succeeds", testFutureMarketSubmissionWithExternalTradingSettlementTimestampKeySucceeds)
	t.Run("Submitting a future market with trade termination from oracle with no public key fails", testFutureMarketSubmissionWithExternalTradingTerminationNoPublicKeyFails)
	t.Run("Submitting a future market with invalid oracle condition or type", testFutureMarketSubmissionWithInvalidOracleConditionOrType)
	t.Run("Submitting a future market with external data source for termination succeeds", testFutureMarketSubmissionWithExternalTradingTerminationBuiltInKeySucceeds)
	t.Run("Submitting a future market with external data source for termination without signers fails", testFutureMarketSubmissionWithExternalTradingTerminationNoSignerFails)
	t.Run("Submitting a future market with external data source for termination with signers and external settlement data without signers fails", testFutureMarketSubmissionWithExternalSettlementDataNoSignerFails)
	t.Run("Submitting a future market with internal data for settlement fails", testFutureMarketSubmissionWithInternalSettlementDataFails)
	t.Run("Submitting a future market with external data sources for settlement and termination with empty signers fail", testFutureMarketSubmissionWithExternalSettlementDataAndTerminationEmptySignerFails)
	t.Run("Submitting a future market with external data sources for settlement and termination with empty pubKey signer fail", testFutureMarketSubmissionWithExternalSettlementDataAndTerminationEmptyPubKeySignerFails)
	t.Run("Submitting a future market with external data sources for settlement and termination with empty eth address signer fail", testFutureMarketSubmissionWithExternalSettlementDataAndTerminationEmptyEthAddressSignerFails)
	t.Run("Submitting a future market with external data sources for settlement and termination with no signers fail", testFutureMarketSubmissionWithExternalSettlementDataAndTerminationNoSignerFails)
	t.Run("Submitting a future market with internal time trigger termination data fails", testFutureMarketSubmissionWithInternalTimeTriggerTerminationDataFails)
	t.Run("Submitting a future market with internal time trigger settlement data fails", testFutureMarketSubmissionWithInternalTimeTriggerSettlementDataFails)

	t.Run("Submitting a perps market change without perps fails", testNewPerpsMarketChangeSubmissionWithoutPerpsFails)
	t.Run("Submitting a perps market change with perps succeeds", testNewPerpsMarketChangeSubmissionWithPerpsSucceeds)
	t.Run("Submitting a perps market change without settlement asset fails", testNewPerpsMarketChangeSubmissionWithoutSettlementAssetFails)
	t.Run("Submitting a perps market change with settlement asset succeeds", testNewPerpsMarketChangeSubmissionWithSettlementAssetSucceeds)
	t.Run("Submitting a perps market change without quote name fails", testNewPerpsMarketChangeSubmissionWithoutQuoteNameFails)
	t.Run("Submitting a perps market change with quote name succeeds", testNewPerpsMarketChangeSubmissionWithQuoteNameSucceeds)
	t.Run("Submitting a perps market change without oracle spec fails", testNewPerpsMarketChangeSubmissionWithoutOracleSpecFails)
	t.Run("Submitting a perps market change with oracle spec succeeds", testNewPerpsMarketChangeSubmissionWithOracleSpecSucceeds)
	t.Run("Submitting a perps market change without filters fails", testNewPerpsMarketChangeSubmissionWithoutFiltersFails)
	t.Run("Submitting a perps market change with filters succeeds", testNewPerpsMarketChangeSubmissionWithFiltersSucceeds)
	t.Run("Submitting a perps market change with filter without key fails", testNewPerpsMarketChangeSubmissionWithFilterWithoutKeyFails)
	t.Run("Submitting a perps market change with filter with key succeeds", testNewPerpsMarketChangeSubmissionWithFilterWithKeySucceeds)
	t.Run("Submitting a perps market change with filter without key name fails", testNewPerpsMarketChangeSubmissionWithFilterWithoutKeyNameFails)
	t.Run("Submitting a perps market change with filter with key name succeeds", testNewPerpsMarketChangeSubmissionWithFilterWithKeyNameSucceeds)
	t.Run("Submitting a perps market change with filter without key type fails", testNewPerpsMarketChangeSubmissionWithFilterWithoutKeyTypeFails)
	t.Run("Submitting a perps market change with filter with key type succeeds", testNewPerpsMarketChangeSubmissionWithFilterWithKeyTypeSucceeds)
	t.Run("Submitting a perps market change with filter without condition succeeds", testNewPerpsMarketChangeSubmissionWithFilterWithoutConditionsSucceeds)
	t.Run("Submitting a perps market change with filter without condition operator fails", testNewPerpsMarketChangeSubmissionWithFilterWithoutConditionOperatorFails)
	t.Run("Submitting a perps market change with filter with condition operator succeeds", testNewPerpsMarketChangeSubmissionWithFilterWithConditionOperatorSucceeds)
	t.Run("Submitting a perps market change with filter without condition value fails", testNewPerpsMarketChangeSubmissionWithFilterWithoutConditionValueFails)
	t.Run("Submitting a perps market change with filter with condition value succeeds", testNewPerpsMarketChangeSubmissionWithFilterWithConditionValueSucceeds)
	t.Run("Submitting a perps market change without oracle spec bindings fails", testNewPerpsMarketChangeSubmissionWithoutDataSourceSpecBindingFails)
	t.Run("Submitting a perps market change with oracle spec binding succeeds", testNewPerpsMarketChangeSubmissionWithDataSourceSpecBindingSucceeds)
	t.Run("Submitting a perps market change with a mismatch between binding property name and filter fails", testNewPerpsMarketChangeSubmissionWithMismatchBetweenFilterAndBindingFails)
	t.Run("Submitting a perps market change with match between binding property name and filter succeeds", testNewPerpsMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingSucceeds)
	t.Run("Submitting a perps market change with settlement data and trading termination properties succeeds", testNewPerpsMarketChangeSubmissionWithSettlementDataPropertySucceeds)
}

func testNewMarketChangeSubmissionWithoutNewMarketFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithoutChangesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithoutDecimalPlacesSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.decimal_places"), commands.ErrMustBePositiveOrZero)
}

func testNewMarketChangeSubmissionWithDecimalPlacesEqualTo0Succeeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						LinearSlippageFactor: "banana",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.linear_slippage_factor"), commands.ErrIsNotValidNumber)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						LinearSlippageFactor: "-0.1",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.linear_slippage_factor"), commands.ErrMustBePositiveOrZero)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						LinearSlippageFactor: "1000000.000001",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.linear_slippage_factor"), commands.ErrMustBeAtMost1M)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{},
				},
			},
		},
	})
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.linear_slippage_factor"), commands.ErrIsNotValidNumber)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.quadratic_slippage_factor"), commands.ErrIsNotValidNumber)
}

func testNewMarketChangeSubmissionWithLpRangeNegativeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithLiquidityMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{},
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.triggering_ratio"), commands.ErrIsNotValidNumber)
}

func testLiquidityMonitoringChangeSubmissionWithoutTargetStakeParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{
							TargetStakeParameters: &vegapb.TargetStakeParameters{},
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{
									TriggeringRatio: "1",
									TargetStakeParameters: &vegapb.TargetStakeParameters{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{
							TargetStakeParameters: &vegapb.TargetStakeParameters{
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{
									TriggeringRatio: "1",
									TargetStakeParameters: &vegapb.TargetStakeParameters{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{
							TargetStakeParameters: &vegapb.TargetStakeParameters{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						PriceMonitoringParameters: &vegapb.PriceMonitoringParameters{
							Triggers: []*vegapb.PriceMonitoringTrigger{},
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						PriceMonitoringParameters: &vegapb.PriceMonitoringParameters{
							Triggers: []*vegapb.PriceMonitoringTrigger{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						PriceMonitoringParameters: &vegapb.PriceMonitoringParameters{
							Triggers: []*vegapb.PriceMonitoringTrigger{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						PriceMonitoringParameters: &vegapb.PriceMonitoringParameters{
							Triggers: []*vegapb.PriceMonitoringTrigger{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						PriceMonitoringParameters: &vegapb.PriceMonitoringParameters{
							Triggers: []*vegapb.PriceMonitoringTrigger{
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								PriceMonitoringParameters: &vegapb.PriceMonitoringParameters{
									Triggers: []*vegapb.PriceMonitoringTrigger{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						PriceMonitoringParameters: &vegapb.PriceMonitoringParameters{
							Triggers: []*vegapb.PriceMonitoringTrigger{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						PriceMonitoringParameters: &vegapb.PriceMonitoringParameters{
							Triggers: []*vegapb.PriceMonitoringTrigger{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						PriceMonitoringParameters: &vegapb.PriceMonitoringParameters{
							Triggers: []*vegapb.PriceMonitoringTrigger{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithPriceMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						PriceMonitoringParameters: &vegapb.PriceMonitoringParameters{},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithoutInstrumentNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product"), commands.ErrIsRequired)
}

func testNewMarketChangeSubmissionWithProductSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{},
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{},
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{},
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{},
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
// 		Terms: &vegapb.ProposalTerms{
// 			Change: &vegapb.ProposalTerms_NewMarket{
// 				NewMarket: &vegapb.NewMarket{
// 					Changes: &vegapb.NewMarketConfiguration{
// 						Instrument: &vegapb.InstrumentConfiguration{
// 							Product: &vegapb.InstrumentConfiguration_Future{
// 								Future: &vegapb.FutureProduct{
// 									DataSourceSpecForSettlementData: &vegapb.DataSourceDefinition{
// 										SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{

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
	future := &vegapb.FutureProduct{}
	if oracleSpecName == "data_source_spec_for_settlement_data" {
		future.DataSourceSpecForTradingTermination = &vegapb.DataSourceDefinition{}
	} else {
		future.DataSourceSpecForSettlementData = &vegapb.DataSourceDefinition{}
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future."+oracleSpecName), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithEmptyOracleSpecFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.source_type"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.source_type"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithEmptyOracleSpecTypeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: &vegapb.DataSourceDefinition{
										SourceType: &vegapb.DataSourceDefinition_External{
											External: &vegapb.DataSourceDefinitionExternal{
												SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
													Oracle: nil,
												},
											},
										},
									},
									DataSourceSpecForTradingTermination: &vegapb.DataSourceDefinition{
										SourceType: &vegapb.DataSourceDefinition_External{
											External: &vegapb.DataSourceDefinitionExternal{
												SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
													Oracle: nil,
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithEmptyInternalSpecTypeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: &vegapb.DataSourceDefinition{
										SourceType: &vegapb.DataSourceDefinition_Internal{
											Internal: &vegapb.DataSourceDefinitionInternal{
												SourceType: &vegapb.DataSourceDefinitionInternal_Time{
													Time: nil,
												},
											},
										},
									},
									DataSourceSpecForTradingTermination: &vegapb.DataSourceDefinition{
										SourceType: &vegapb.DataSourceDefinition_Internal{
											Internal: &vegapb.DataSourceDefinitionInternal{
												SourceType: &vegapb.DataSourceDefinitionInternal_Time{
													Time: nil,
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data"), commands.ErrIsNotValid)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.internal.time"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithoutPubKeysFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(vegapb.DataSourceContentTypeOracle),
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
	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
		dstypes.CreateSignerFromString("", dstypes.SignerTypePubKey),
	}

	testCases := []struct {
		msg   string
		value []*datapb.Signer
	}{
		{
			msg:   "with empty signers",
			value: dstypes.SignersIntoProto(pubKeys),
		}, {
			msg:   "with blank signers",
			value: dstypes.SignersIntoProto(pubKeys),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								Instrument: &vegapb.InstrumentConfiguration{
									Product: &vegapb.InstrumentConfiguration_Future{
										Future: &vegapb.FutureProduct{
											DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
												vegapb.DataSourceContentTypeOracle,
											).SetOracleConfig(
												&vegapb.DataSourceDefinitionExternal_Oracle{
													Oracle: &vegapb.DataSourceSpecConfiguration{
														Signers: tc.value,
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

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.1"), commands.ErrIsNotValid)
		})
	}
}

func testNewFutureMarketChangeSubmissionWithBadPubKeysOrderAddressFail(t *testing.T) {
	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
		dstypes.CreateSignerFromString("0xCAFEDUDE", dstypes.SignerTypePubKey),
		dstypes.CreateSignerFromString("0xCAFEDUDE", dstypes.SignerTypeEthAddress),
		dstypes.CreateSignerFromString("36393436346533356263623865386132393030636130663837616361663235326435306366326162326663373336393438343561313662376338613064633666", dstypes.SignerTypePubKey),
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: dstypes.SignersIntoProto(pubKeys),
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: dstypes.SignersIntoProto(pubKeys),
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.0"), commands.ErrIsNotValid)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.1"), commands.ErrIsNotValid)

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.3"), commands.ErrIsNotValidVegaPubkey)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.2"), commands.ErrIsNotValidEthereumAddress)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.1"), commands.ErrIsNotValidVegaPubkey)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.0"), commands.ErrIsNotValidVegaPubkey)
}

func testNewFutureMarketChangeSubmissionWithGoodPubKeysOrderAddressSucceed(t *testing.T) {
	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0x8565a19c49bcD6Fa7b6EB0221a50606F9c9cC683", dstypes.SignerTypeEthAddress),
		dstypes.CreateSignerFromString("bd069246503a57271375f1995c46e03db88c4e1a564077b33a9872f905650dc4", dstypes.SignerTypePubKey),
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: dstypes.SignersIntoProto(pubKeys),
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: dstypes.SignersIntoProto(pubKeys),
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.0"), commands.ErrIsNotValid)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.1"), commands.ErrIsNotValid)

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.1"), commands.ErrIsNotValidEthereumAddress)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.0"), commands.ErrIsNotValidVegaPubkey)
}

func testNewFutureMarketChangeSubmissionWithoutFiltersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Filters: []*datapb.Filter{
													{},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Filters: []*datapb.Filter{
													{},
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Filters: []*datapb.Filter{
													{}, {},
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.key"), commands.ErrIsNotValid)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.1.key"), commands.ErrIsNotValid)
}

func testNewFutureMarketChangeSubmissionWithFilterWithKeySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{},
													}, {
														Key: &datapb.PropertyKey{},
													},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{},
													}, {
														Key: &datapb.PropertyKey{},
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.0.key"), commands.ErrIsNotValid)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.filters.1.key"), commands.ErrIsNotValid)
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutKeyNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								Instrument: &vegapb.InstrumentConfiguration{
									Product: &vegapb.InstrumentConfiguration_Future{
										Future: &vegapb.FutureProduct{
											DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
												vegapb.DataSourceContentTypeOracle,
											).SetOracleConfig(
												&vegapb.DataSourceDefinitionExternal_Oracle{
													Oracle: &vegapb.DataSourceSpecConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Filters: []*datapb.Filter{
													{
														Conditions: []*datapb.Condition{},
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec.external.oracle.filters.0.conditions"), commands.ErrIsRequired)
}

func testNewFutureMarketChangeSubmissionWithFilterWithoutConditionOperatorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								Instrument: &vegapb.InstrumentConfiguration{
									Product: &vegapb.InstrumentConfiguration_Future{
										Future: &vegapb.FutureProduct{
											DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
												vegapb.DataSourceContentTypeOracle,
											).SetOracleConfig(
												&vegapb.DataSourceDefinitionExternal_Oracle{
													Oracle: &vegapb.DataSourceSpecConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{},
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecBinding: &vegapb.DataSourceSpecToFutureBinding{},
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
	var binding *vegapb.DataSourceSpecToFutureBinding
	if property == "settlement_data_property" {
		binding = &vegapb.DataSourceSpecToFutureBinding{
			SettlementDataProperty: "",
		}
	} else {
		binding = &vegapb.DataSourceSpecToFutureBinding{
			TradingTerminationProperty: "",
		}
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
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
	testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t, &vegapb.DataSourceSpecToFutureBinding{SettlementDataProperty: "key1"}, "settlement_data_property", "key1")
	testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t, &vegapb.DataSourceSpecToFutureBinding{TradingTerminationProperty: "key2"}, "settlement_data_property", "key2")
}

func testNewFutureMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t *testing.T, binding *vegapb.DataSourceSpecToFutureBinding, bindingName string, bindingKey string) {
	t.Helper()
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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

func testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t *testing.T, binding *vegapb.DataSourceSpecToFutureBinding, bindingName string) {
	t.Helper()
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
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
	testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t, &vegapb.DataSourceSpecToFutureBinding{SettlementDataProperty: "My property"}, "settlement_data_property")
	testNewFutureMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t, &vegapb.DataSourceSpecToFutureBinding{TradingTerminationProperty: "My property"}, "trading_termination_property")
}

func testNewFutureMarketChangeSubmissionWithSettlementDataPropertySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecBinding: &vegapb.DataSourceSpecToFutureBinding{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_Simple{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.simple"), commands.ErrIsRequired)
}

func testNewSimpleRiskParametersChangeSubmissionWithSimpleRiskParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_Simple{
							Simple: &vegapb.SimpleModelParams{},
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_Simple{
							Simple: &vegapb.SimpleModelParams{
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								RiskParameters: &vegapb.NewMarketConfiguration_Simple{
									Simple: &vegapb.SimpleModelParams{
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

func testNewSpotSimpleRiskParametersChangeSubmissionWithNegativeMaxMoveUpFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_Simple{
							Simple: &vegapb.SimpleModelParams{
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								RiskParameters: &vegapb.NewMarketConfiguration_Simple{
									Simple: &vegapb.SimpleModelParams{
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								RiskParameters: &vegapb.NewMarketConfiguration_Simple{
									Simple: &vegapb.SimpleModelParams{
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								RiskParameters: &vegapb.NewMarketConfiguration_Simple{
									Simple: &vegapb.SimpleModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal"), commands.ErrIsRequired)
}

func testNewLogNormalRiskParametersChangeSubmissionWithLogNormalRiskParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 1,
								Tau:                   2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{},
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 5e-9,
								Tau:                   1.0,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 1e-9,
								Tau:                   2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1 + 1e-8,
								Tau:                   2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 1e-8,
								Tau:                   2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 1 - 1e-12,
								Tau:                   2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   1e-9,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   1 + 1e-12,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   1e-12,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   1,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						RiskParameters: &vegapb.NewMarketConfiguration_LogNormal{
							LogNormal: &vegapb.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vegapb.LogNormalModelParams{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
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

func testFutureMarketSubmissionWithInvalidOracleConditionOrType(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: []*datapb.Signer{},
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{
															Name: "trading.terminated",
															Type: datapb.PropertyKey_Type(10000),
														},
														Conditions: []*datapb.Condition{
															{
																Operator: datapb.Condition_Operator(10000),
															},
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.filters.0.conditions.0.operator"), commands.ErrIsNotValid)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.filters.0.key.type"), commands.ErrIsNotValid)
}

func testFutureMarketSubmissionWithExternalTradingTerminationNoPublicKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeInternalTimeTermination,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeInternalTimeTermination,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.internal.time.conditions"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data"), commands.ErrIsRequired)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeInternalTimeTermination,
									).SetTimeTriggerConditionConfig(
										[]*datapb.Condition{
											{
												Operator: datapb.Condition_OPERATOR_UNSPECIFIED,
												Value:    "value 1",
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.internal.time.conditions.0.operator"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithExternalTradingTerminationBuiltInKeyNoPublicKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data"), commands.ErrIsNotValid)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithExternalTradingSettlementTimestampKeySucceeds(t *testing.T) {
	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
		dstypes.CreateSignerFromString("", dstypes.SignerTypePubKey),
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: dstypes.SignersIntoProto(pubKeys),
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{
															Name: "price.BTC.value",
															Type: datapb.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datapb.Condition{
															{
																Operator: datapb.Condition_OPERATOR_EQUALS,
																Value:    "15",
															},
														},
													},
													{
														Key: &datapb.PropertyKey{
															Name: "price.BTC.timestamp",
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data"), commands.ErrIsNotValid)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithExternalTradingTerminationBuiltInKeySucceeds(t *testing.T) {
	pubKey := []*dstypes.Signer{
		dstypes.CreateSignerFromString("bd069246503a57271375f1995c46e03db88c4e1a564077b33a9872f905650dc4", dstypes.SignerTypePubKey),
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: dstypes.SignersIntoProto(pubKey),
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{
															Name: "prices.ETH.value",
															Type: datapb.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datapb.Condition{
															{
																Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
															},
														},
													},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: dstypes.SignersIntoProto(pubKey),
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
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithExternalTradingTerminationNoSignerFails(t *testing.T) {
	pubKey := []*dstypes.Signer{
		dstypes.CreateSignerFromString("bd069246503a57271375f1995c46e03db88c4e1a564077b33a9872f905650dc4", dstypes.SignerTypePubKey),
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: dstypes.SignersIntoProto(pubKey),
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{
															Name: "vegaprotocol.builtin.prices.ETH.value",
															Type: datapb.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datapb.Condition{
															{
																Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
															},
														},
													},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithExternalSettlementDataNoSignerFails(t *testing.T) {
	pubKey := []*dstypes.Signer{
		dstypes.CreateSignerFromString("bd069246503a57271375f1995c46e03db88c4e1a564077b33a9872f905650dc4", dstypes.SignerTypePubKey),
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{
															Name: "vegaprotocol.builtin.prices.ETH.value",
															Type: datapb.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datapb.Condition{
															{
																Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
															},
														},
													},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: dstypes.SignersIntoProto(pubKey),
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
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithInternalSettlementDataFails(t *testing.T) {
	pubKey := []*dstypes.Signer{
		dstypes.CreateSignerFromString("bd069246503a57271375f1995c46e03db88c4e1a564077b33a9872f905650dc4", dstypes.SignerTypePubKey),
	}

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeInternalTimeTermination,
									).SetTimeTriggerConditionConfig(
										[]*datapb.Condition{
											{
												// It does not matter what conditions are set here
												Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: dstypes.SignersIntoProto(pubKey),
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data"), commands.ErrIsNotValid)
}

func testFutureMarketSubmissionWithExternalSettlementDataAndTerminationEmptySignerFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: []*datapb.Signer{},
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{
															Name: "vegaprotocol.builtin.prices.ETH.value",
															Type: datapb.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datapb.Condition{
															{
																Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
															},
														},
													},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithExternalSettlementDataAndTerminationNoSignerFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{
															Name: "vegaprotocol.builtin.prices.ETH.value",
															Type: datapb.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datapb.Condition{
															{
																Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
															},
														},
													},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers"), commands.ErrIsRequired)
}

func testFutureMarketSubmissionWithExternalSettlementDataAndTerminationEmptyPubKeySignerFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: []*datapb.Signer{
													{
														Signer: &datapb.Signer_PubKey{
															PubKey: &datapb.PubKey{
																Key: "",
															},
														},
													},
												},
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{
															Name: "vegaprotocol.builtin.prices.ETH.value",
															Type: datapb.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datapb.Condition{
															{
																Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
															},
														},
													},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: []*datapb.Signer{
													{
														Signer: &datapb.Signer_PubKey{
															PubKey: &datapb.PubKey{},
														},
													},
												},
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.0"), commands.ErrIsNotValid)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers.0"), commands.ErrIsNotValid)
}

func testFutureMarketSubmissionWithExternalSettlementDataAndTerminationEmptyEthAddressSignerFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: []*datapb.Signer{
													{
														Signer: &datapb.Signer_EthAddress{
															EthAddress: &datapb.ETHAddress{
																Address: "",
															},
														},
													},
												},
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{
															Name: "vegaprotocol.builtin.prices.ETH.value",
															Type: datapb.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datapb.Condition{
															{
																Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
															},
														},
													},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: []*datapb.Signer{
													{
														Signer: &datapb.Signer_EthAddress{
															EthAddress: &datapb.ETHAddress{},
														},
													},
												},
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.external.oracle.signers.0"), commands.ErrIsNotValid)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.external.oracle.signers.0"), commands.ErrIsNotValid)
}

func testNewPerpsMarketChangeSubmissionWithoutPerpsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithPerpsSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithoutSettlementAssetFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									SettlementAsset: "",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.settlement_asset"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithSettlementAssetSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									SettlementAsset: "BTC",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.settlement_asset"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithoutQuoteNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									QuoteName: "",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.quote_name"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithQuoteNameSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									QuoteName: "BTC",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.quote_name"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithoutOracleSpecFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithOracleSpecSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: &vegapb.DataSourceDefinition{},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithoutFiltersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeEthOracle,
									),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithFiltersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeEthOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_EthOracle{
											EthOracle: &vegapb.EthCallSpec{
												Filters: []*datapb.Filter{
													{},
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithFilterWithoutKeyFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeEthOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_EthOracle{
											EthOracle: &vegapb.EthCallSpec{
												Filters: []*datapb.Filter{
													{}, {},
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.0.key"), commands.ErrIsNotValid)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.1.key"), commands.ErrIsNotValid)
}

func testNewPerpsMarketChangeSubmissionWithFilterWithKeySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeEthOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_EthOracle{
											EthOracle: &vegapb.EthCallSpec{
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{},
													}, {
														Key: &datapb.PropertyKey{},
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.0.key"), commands.ErrIsNotValid)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.1.key"), commands.ErrIsNotValid)
}

func testNewPerpsMarketChangeSubmissionWithFilterWithoutKeyNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeEthOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_EthOracle{
											EthOracle: &vegapb.EthCallSpec{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.0.key.name"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.1.key.name"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithFilterWithKeyNameSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeEthOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_EthOracle{
											EthOracle: &vegapb.EthCallSpec{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.0.key.name"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.1.key.name"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithFilterWithoutKeyTypeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeEthOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_EthOracle{
											EthOracle: &vegapb.EthCallSpec{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.0.key.type"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.1.key.type"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithFilterWithKeyTypeSucceeds(t *testing.T) {
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								Instrument: &vegapb.InstrumentConfiguration{
									Product: &vegapb.InstrumentConfiguration_Perps{
										Perps: &vegapb.PerpsProduct{
											DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
												vegapb.DataSourceContentTypeEthOracle,
											).SetOracleConfig(
												&vegapb.DataSourceDefinitionExternal_EthOracle{
													EthOracle: &vegapb.EthCallSpec{
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

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec.external.ethoracle.filters.0.key.type"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec.external.ethoracle.filters.1.key.type"), commands.ErrIsRequired)
		})
	}
}

func testNewPerpsMarketChangeSubmissionWithFilterWithoutConditionsSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeEthOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_EthOracle{
											EthOracle: &vegapb.EthCallSpec{
												Filters: []*datapb.Filter{
													{
														Conditions: []*datapb.Condition{},
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec.external.ethoracle.filters.0.conditions"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithFilterWithoutConditionOperatorFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeEthOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_EthOracle{
											EthOracle: &vegapb.EthCallSpec{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.0.conditions.0.operator"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.0.conditions.1.operator"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithFilterWithConditionOperatorSucceeds(t *testing.T) {
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
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								Instrument: &vegapb.InstrumentConfiguration{
									Product: &vegapb.InstrumentConfiguration_Perps{
										Perps: &vegapb.PerpsProduct{
											DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
												vegapb.DataSourceContentTypeEthOracle,
											).SetOracleConfig(
												&vegapb.DataSourceDefinitionExternal_EthOracle{
													EthOracle: &vegapb.EthCallSpec{
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

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec.external.ethoracle.filters.0.conditions.0.operator"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec.external.ethoracle.filters.1.conditions.0.operator"), commands.ErrIsRequired)
		})
	}
}

func testNewPerpsMarketChangeSubmissionWithFilterWithoutConditionValueFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeEthOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_EthOracle{
											EthOracle: &vegapb.EthCallSpec{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.0.conditions.0.value"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_data.external.ethoracle.filters.0.conditions.1.value"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithFilterWithConditionValueSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeEthOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_EthOracle{
											EthOracle: &vegapb.EthCallSpec{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec.external.ethoracle.filters.0.conditions.0.value"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec.external.ethoracle.filters.0.conditions.1.value"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithoutDataSourceSpecBindingFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_binding"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithDataSourceSpecBindingSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecBinding: &vegapb.DataSourceSpecToPerpsBinding{},
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_binding"), commands.ErrIsRequired)
}

func testNewPerpsMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingSucceeds(t *testing.T) {
	testNewPerpsMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t, &vegapb.DataSourceSpecToPerpsBinding{SettlementDataProperty: "key1"}, "settlement_data_property", "key1")
}

func testNewPerpsMarketChangeSubmissionWithNoMismatchBetweenFilterAndBindingForSpecSucceeds(t *testing.T, binding *vegapb.DataSourceSpecToPerpsBinding, bindingName string, bindingKey string) {
	t.Helper()
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeEthOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_EthOracle{
											EthOracle: &vegapb.EthCallSpec{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_binding."+bindingName), commands.ErrIsMismatching)
}

func testNewPerpsMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t *testing.T, binding *vegapb.DataSourceSpecToPerpsBinding, bindingName string) {
	t.Helper()
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecBinding: binding,
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_binding."+bindingName), commands.ErrIsMismatching)
}

func testNewPerpsMarketChangeSubmissionWithMismatchBetweenFilterAndBindingFails(t *testing.T) {
	testNewPerpsMarketChangeSubmissionWithMismatchBetweenFilterAndBindingForSpecFails(t, &vegapb.DataSourceSpecToPerpsBinding{SettlementDataProperty: "My property"}, "settlement_data_property")
}

func testNewPerpsMarketChangeSubmissionWithSettlementDataPropertySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Perps{
								Perps: &vegapb.PerpsProduct{
									DataSourceSpecBinding: &vegapb.DataSourceSpecToPerpsBinding{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_binding.settlement_data_property"), commands.ErrIsRequired)
}

func TestNewPerpsMarketChangeSubmissionProductParameters(t *testing.T) {
	cases := []struct {
		product vegapb.PerpsProduct
		err     error
		path    string
		desc    string
	}{
		// margin_funding_factor
		{
			product: vegapb.PerpsProduct{
				MarginFundingFactor: "",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.margin_funding_factor",
			err:  commands.ErrIsRequired,
			desc: "margin_funding_factor is empty",
		},
		{
			product: vegapb.PerpsProduct{
				MarginFundingFactor: "nope",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.margin_funding_factor",
			err:  commands.ErrIsNotValidNumber,
			desc: "margin_funding_factor is not a valid number",
		},
		{
			product: vegapb.PerpsProduct{
				MarginFundingFactor: "-10",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.margin_funding_factor",
			err:  commands.ErrMustBeWithinRange01,
			desc: "margin_funding_factor is not within range (< 0)",
		},
		{
			product: vegapb.PerpsProduct{
				MarginFundingFactor: "10",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.margin_funding_factor",
			err:  commands.ErrMustBeWithinRange01,
			desc: "margin_funding_factor is not within range (> 1)",
		},
		{
			product: vegapb.PerpsProduct{
				MarginFundingFactor: "0.5",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.margin_funding_factor",
			desc: "margin_funding_factor is valid",
		},
		// interest_rate
		{
			product: vegapb.PerpsProduct{
				InterestRate: "",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.interest_rate",
			err:  commands.ErrIsRequired,
			desc: "interest_rate is empty",
		},
		{
			product: vegapb.PerpsProduct{
				InterestRate: "nope",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.interest_rate",
			err:  commands.ErrIsNotValidNumber,
			desc: "interest_rate is not a valid number",
		},
		{
			product: vegapb.PerpsProduct{
				InterestRate: "-10",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.interest_rate",
			err:  commands.ErrMustBeWithinRange11,
			desc: "interest_rate is not within range (< -1)",
		},
		{
			product: vegapb.PerpsProduct{
				InterestRate: "10",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.interest_rate",
			err:  commands.ErrMustBeWithinRange11,
			desc: "interest_rate is not within range (> 1)",
		},
		{
			product: vegapb.PerpsProduct{
				InterestRate: "0.5",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.interest_rate",
			desc: "interest_rate is valid",
		},
		{
			product: vegapb.PerpsProduct{
				InterestRate: "-0.5",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.interest_rate",
			desc: "interest_rate is valid",
		},
		// clamp_lower_bound
		{
			product: vegapb.PerpsProduct{
				ClampLowerBound: "",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_lower_bound",
			err:  commands.ErrIsRequired,
			desc: "clamp_lower_bound is empty",
		},
		{
			product: vegapb.PerpsProduct{
				ClampLowerBound: "nope",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_lower_bound",
			err:  commands.ErrIsNotValidNumber,
			desc: "clamp_lower_bound is not a valid number",
		},
		{
			product: vegapb.PerpsProduct{
				ClampLowerBound: "-10",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_lower_bound",
			err:  commands.ErrMustBeWithinRange11,
			desc: "clamp_lower_bound is not within range (< -1)",
		},
		{
			product: vegapb.PerpsProduct{
				ClampLowerBound: "10",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_lower_bound",
			err:  commands.ErrMustBeWithinRange11,
			desc: "clamp_lower_bound is not within range (> 1)",
		},
		{
			product: vegapb.PerpsProduct{
				ClampLowerBound: "0.5",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_lower_bound",
			desc: "clamp_lower_bound is valid",
		},
		{
			product: vegapb.PerpsProduct{
				ClampLowerBound: "-0.5",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_lower_bound",
			desc: "clamp_lower_bound is valid",
		},
		// clamp_upper_bound
		{
			product: vegapb.PerpsProduct{
				ClampUpperBound: "",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_upper_bound",
			err:  commands.ErrIsRequired,
			desc: "clamp_upper_bound is empty",
		},
		{
			product: vegapb.PerpsProduct{
				ClampUpperBound: "nope",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_upper_bound",
			err:  commands.ErrIsNotValidNumber,
			desc: "clamp_upper_bound is not a valid number",
		},
		{
			product: vegapb.PerpsProduct{
				ClampUpperBound: "-10",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_upper_bound",
			err:  commands.ErrMustBeWithinRange11,
			desc: "clamp_upper_bound is not within range (< -1)",
		},
		{
			product: vegapb.PerpsProduct{
				ClampUpperBound: "10",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_upper_bound",
			err:  commands.ErrMustBeWithinRange11,
			desc: "clamp_upper_bound is not within range (> 1)",
		},
		{
			product: vegapb.PerpsProduct{
				ClampUpperBound: "0.5",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_upper_bound",
			desc: "clamp_upper_bound is valid",
		},
		{
			product: vegapb.PerpsProduct{
				ClampUpperBound: "-0.5",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_upper_bound",
			desc: "clamp_upper_bound is valid",
		},
		// clamp lower and upper
		{
			product: vegapb.PerpsProduct{
				ClampLowerBound: "0.5",
				ClampUpperBound: "0.5",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_upper_bound",
			desc: "clamp_upper_bound == clamp_lower_bound is valid",
		},
		{
			product: vegapb.PerpsProduct{
				ClampLowerBound: "0.4",
				ClampUpperBound: "0.5",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_upper_bound",
			desc: "clamp_upper_bound > clamp_lower_bound is valid",
		},
		{
			product: vegapb.PerpsProduct{
				ClampLowerBound: "0.5",
				ClampUpperBound: "0.4",
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.clamp_upper_bound",
			err:  commands.ErrMustBeSuperiorOrEqualToClampLowerBound,
			desc: "clamp_upper_bound < clamp_lower_bound is invalid",
		},
	}

	for _, v := range cases {
		t.Run(v.desc, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								Instrument: &vegapb.InstrumentConfiguration{
									Product: &vegapb.InstrumentConfiguration_Perps{
										Perps: &v.product,
									},
								},
							},
						},
					},
				},
			})

			errs := err.Get(v.path)

			// no errors expected
			if v.err == nil {
				assert.Len(t, errs, 0, v.desc)
				return
			}

			assert.Contains(t, errs, v.err, v.desc)
		})
	}
}

func TestNewPerpsMarketChangeSubmissionSettlementSchedule(t *testing.T) {
	cases := []struct {
		product vegapb.PerpsProduct
		err     error
		path    string
		desc    string
	}{
		{
			product: vegapb.PerpsProduct{
				DataSourceSpecForSettlementSchedule: &vegapb.DataSourceDefinition{
					SourceType: &vegapb.DataSourceDefinition_Internal{
						Internal: &vega.DataSourceDefinitionInternal{
							SourceType: &vegapb.DataSourceDefinitionInternal_TimeTrigger{
								TimeTrigger: &vegapb.DataSourceSpecConfigurationTimeTrigger{
									Triggers: []*datapb.InternalTimeTrigger{
										{
											Initial: nil,
											Every:   0,
										},
									},
								},
							},
						},
					},
				},
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_schedule.internal.timetrigger.triggers.0.every",
			err:  commands.ErrIsNotValid,
			desc: "not a valid every",
		},
		{
			product: vegapb.PerpsProduct{
				DataSourceSpecForSettlementSchedule: &vegapb.DataSourceDefinition{
					SourceType: &vegapb.DataSourceDefinition_Internal{
						Internal: &vega.DataSourceDefinitionInternal{
							SourceType: &vegapb.DataSourceDefinitionInternal_TimeTrigger{
								TimeTrigger: &vegapb.DataSourceSpecConfigurationTimeTrigger{
									Triggers: []*datapb.InternalTimeTrigger{
										{
											Initial: nil,
											Every:   -1,
										},
									},
								},
							},
						},
					},
				},
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_schedule.internal.timetrigger.triggers.0.every",
			err:  commands.ErrIsNotValid,
			desc: "not a valid every",
		},
		{
			product: vegapb.PerpsProduct{
				DataSourceSpecForSettlementSchedule: &vegapb.DataSourceDefinition{
					SourceType: &vegapb.DataSourceDefinition_Internal{
						Internal: &vega.DataSourceDefinitionInternal{
							SourceType: &vegapb.DataSourceDefinitionInternal_TimeTrigger{
								TimeTrigger: &vegapb.DataSourceSpecConfigurationTimeTrigger{
									Triggers: []*datapb.InternalTimeTrigger{
										{
											Initial: ptr.From(int64(-1)),
											Every:   100,
										},
									},
								},
							},
						},
					},
				},
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_schedule.internal.timetrigger.triggers.0.initial",
			err:  commands.ErrIsNotValid,
			desc: "not a valid every",
		},
		{
			product: vegapb.PerpsProduct{
				DataSourceSpecForSettlementSchedule: &vegapb.DataSourceDefinition{
					SourceType: &vegapb.DataSourceDefinition_Internal{
						Internal: &vega.DataSourceDefinitionInternal{
							SourceType: &vegapb.DataSourceDefinitionInternal_TimeTrigger{
								TimeTrigger: &vegapb.DataSourceSpecConfigurationTimeTrigger{
									Triggers: []*datapb.InternalTimeTrigger{
										{
											Initial: nil,
											Every:   100,
										},
									},
								},
							},
						},
					},
				},
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_schedule.internal.timetrigger",
			desc: "valid with initial nil",
		},
		{
			product: vegapb.PerpsProduct{
				DataSourceSpecForSettlementSchedule: &vegapb.DataSourceDefinition{
					SourceType: &vegapb.DataSourceDefinition_Internal{
						Internal: &vega.DataSourceDefinitionInternal{
							SourceType: &vegapb.DataSourceDefinitionInternal_TimeTrigger{
								TimeTrigger: &vegapb.DataSourceSpecConfigurationTimeTrigger{
									Triggers: []*datapb.InternalTimeTrigger{
										{
											Initial: ptr.From(int64(100)),
											Every:   100,
										},
									},
								},
							},
						},
					},
				},
			},
			path: "proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_for_settlement_schedule.internal.timetrigger",
			desc: "valid",
		},
	}

	for _, v := range cases {
		t.Run(v.desc, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &vegapb.ProposalTerms{
					Change: &vegapb.ProposalTerms_NewMarket{
						NewMarket: &vegapb.NewMarket{
							Changes: &vegapb.NewMarketConfiguration{
								Instrument: &vegapb.InstrumentConfiguration{
									Product: &vegapb.InstrumentConfiguration_Perps{
										Perps: &v.product,
									},
								},
							},
						},
					},
				},
			})

			errs := err.Get(v.path)

			// no errors expected
			if v.err == nil {
				assert.Len(t, errs, 0, v.desc)
				return
			}

			assert.Contains(t, errs, v.err, v.desc)
		})
	}
}

func testFutureMarketSubmissionWithInternalTimeTriggerTerminationDataFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeOracle,
									).SetOracleConfig(
										&vegapb.DataSourceDefinitionExternal_Oracle{
											Oracle: &vegapb.DataSourceSpecConfiguration{
												Signers: []*datapb.Signer{
													{
														Signer: &datapb.Signer_EthAddress{
															EthAddress: &datapb.ETHAddress{
																Address: "",
															},
														},
													},
												},
												Filters: []*datapb.Filter{
													{
														Key: &datapb.PropertyKey{
															Name: "vegaprotocol.builtin.prices.ETH.value",
															Type: datapb.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datapb.Condition{
															{
																Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
															},
														},
													},
												},
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeInternalTimeTriggerTermination,
									).SetTimeTriggerConditionConfig(
										[]*datapb.Condition{
											{
												// It does not matter what conditions are set here
												Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_trading_termination.internal.timetrigger"), commands.ErrIsNotValid)
}

func testFutureMarketSubmissionWithInternalTimeTriggerSettlementDataFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									DataSourceSpecForSettlementData: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeInternalTimeTriggerTermination,
									).SetTimeTriggerConditionConfig(
										[]*datapb.Condition{
											{
												// It does not matter what conditions are set here
												Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
											},
										},
									),
									DataSourceSpecForTradingTermination: vegapb.NewDataSourceDefinition(
										vegapb.DataSourceContentTypeInternalTimeTriggerTermination,
									).SetTimeTriggerConditionConfig(
										[]*datapb.Condition{
											{
												// It does not matter what conditions are set here
												Operator: datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_for_settlement_data.internal.timetrigger"), commands.ErrIsNotValid)
}
