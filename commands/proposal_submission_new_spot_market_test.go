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

package commands_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/test"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckProposalSubmissionForNewSpotMarket(t *testing.T) {
	t.Run("Submitting a spot market change with a future product", testNewSpotMarketChangeSubmissionWithFutureProductFails)
	t.Run("Submitting a spot market change without new market fails", testNewSpotMarketChangeSubmissionWithoutNewSpotMarketFails)
	t.Run("Submitting a spot market change without changes fails", testNewSpotMarketChangeSubmissionWithoutChangesFails)
	t.Run("Submitting a spot market change without too many pm trigger fails", testNewSpotMarketChangeSubmissionWithTooManyPMTriggersFails)
	t.Run("Submitting a spot market change without decimal places succeeds", testNewSpotMarketChangeSubmissionWithoutDecimalPlacesSucceeds)
	t.Run("Submitting a spot market change with decimal places equal to 0 succeeds", testNewSpotMarketChangeSubmissionWithDecimalPlacesEqualTo0Succeeds)
	t.Run("Submitting a spot market change with decimal places above or equal to 150 fails", testNewSpotMarketChangeSubmissionWithDecimalPlacesAboveOrEqualTo150Fails)
	t.Run("Submitting a spot market change with decimal places below 150 succeeds", testNewSpotMarketChangeSubmissionWithDecimalPlacesBelow150Succeeds)
	t.Run("Submitting a spot market change with position decimal places equal to 0 succeeds", testNewSpotMarketChangeSubmissionWithPositionDecimalPlacesEqualTo0Succeeds)
	t.Run("Submitting a spot market change with position decimal places above or equal to 6 fails", testNewSpotMarketChangeSubmissionWithPositionDecimalPlacesAboveOrEqualTo7Fails)
	t.Run("Submitting a spot market change with position decimal places below 6 succeeds", testNewSpotMarketChangeSubmissionWithPositionDecimalPlacesBelow7Succeeds)
	t.Run("Submitting a new spot market without price monitoring succeeds", testNewSpotMarketChangeSubmissionWithoutPriceMonitoringSucceeds)
	t.Run("Submitting a new spot market with price monitoring succeeds", testNewSpotMarketChangeSubmissionWithPriceMonitoringSucceeds)
	t.Run("Submitting a price monitoring change without triggers succeeds", testSpotPriceMonitoringChangeSubmissionWithoutTriggersSucceeds)
	t.Run("Submitting a price monitoring change with triggers succeeds", testSpotPriceMonitoringChangeSubmissionWithTriggersSucceeds)
	t.Run("Submitting a price monitoring change without trigger horizon fails", testSpotPriceMonitoringChangeSubmissionWithoutTriggerHorizonFails)
	t.Run("Submitting a price monitoring change with trigger horizon succeeds", testSpotPriceMonitoringChangeSubmissionWithTriggerHorizonSucceeds)
	t.Run("Submitting a price monitoring change with wrong trigger probability fails", testSpotPriceMonitoringChangeSubmissionWithWrongTriggerProbabilityFails)
	t.Run("Submitting a price monitoring change with right trigger probability succeeds", testSpotPriceMonitoringChangeSubmissionWithRightTriggerProbabilitySucceeds)
	t.Run("Submitting a price monitoring change without trigger auction extension fails", testSpotPriceMonitoringChangeSubmissionWithoutTriggerAuctionExtensionFails)
	t.Run("Submitting a price monitoring change with trigger auction extension succeeds", testSpotPriceMonitoringChangeSubmissionWithTriggerAuctionExtensionSucceeds)
	t.Run("Submitting a spot market change without instrument name fails", testNewSpotMarketChangeSubmissionWithoutInstrumentNameFails)
	t.Run("Submitting a spot market change with instrument name succeeds", testNewSpotMarketChangeSubmissionWithInstrumentNameSucceeds)
	t.Run("Submitting a spot market change without instrument code fails", testNewSpotMarketChangeSubmissionWithoutInstrumentCodeFails)
	t.Run("Submitting a spot market change with instrument code succeeds", testNewSpotMarketChangeSubmissionWithInstrumentCodeSucceeds)
	t.Run("Submitting a spot market change without product fails", testNewSpotMarketChangeSubmissionWithoutProductFails)
	t.Run("Submitting a spot market change with product succeeds", testNewSpotMarketChangeSubmissionWithProductSucceeds)
	t.Run("Submitting a simple risk parameters change without simple risk parameters fails", testNewSpotSimpleRiskParametersChangeSubmissionWithoutSimpleRiskParametersFails)
	t.Run("Submitting a simple risk parameters change with simple risk parameters succeeds", testNewSpotSimpleRiskParametersChangeSubmissionWithSimpleRiskParametersSucceeds)
	t.Run("Submitting a simple risk parameters change with min move down fails", testNewSpotSimpleRiskParametersChangeSubmissionWithPositiveMinMoveDownFails)
	t.Run("Submitting a simple risk parameters change with min move down succeeds", testNewSpotSimpleRiskParametersChangeSubmissionWithNonPositiveMinMoveDownSucceeds)
	t.Run("Submitting a simple risk parameters change with max move up fails", testNewSpotSimpleRiskParametersChangeSubmissionWithNegativeMaxMoveUpFails)
	t.Run("Submitting a simple risk parameters change with max move up succeeds", testNewSpotSimpleRiskParametersChangeSubmissionWithNonNegativeMaxMoveUpSucceeds)
	t.Run("Submitting a simple risk parameters change with wrong probability of trading fails", testNewSpotSimpleRiskParametersChangeSubmissionWithWrongProbabilityOfTradingFails)
	t.Run("Submitting a simple risk parameters change with right probability of trading succeeds", testNewSpotSimpleRiskParametersChangeSubmissionWithRightProbabilityOfTradingSucceeds)
	t.Run("Submitting a simple risk parameters change with negative max move up fails", testSpotNewSimpleRiskParametersChangeSubmissionWithNegativeMaxMoveUpFails)
	t.Run("Submitting a log normal risk parameters change without log normal risk parameters fails", testNewSpotLogNormalRiskParametersChangeSubmissionWithoutLogNormalRiskParametersFails)
	t.Run("Submitting a log normal risk parameters change with log normal risk parameters succeeds", testNewSpotLogNormalRiskParametersChangeSubmissionWithLogNormalRiskParametersSucceeds)
	t.Run("Submitting a log normal risk parameters change with params fails", testNewSpotLogNormalRiskParametersChangeSubmissionWithoutParamsFails)
	t.Run("Submitting a log normal risk parameters change with invalid risk aversion", testNewSpotLogNormalRiskParametersChangeSubmissionInvalidRiskAversion)
	t.Run("Submitting a log normal risk parameters change with invalid tau", testNewSpotLogNormalRiskParametersChangeSubmissionInvalidTau)
	t.Run("Submitting a log normal risk parameters change with invalid mu", testNewSpotLogNormalRiskParametersChangeSubmissionInvalidMu)
	t.Run("Submitting a log normal risk parameters change with invalid sigma", testNewSpotLogNormalRiskParametersChangeSubmissionInvalidSigma)
	t.Run("Submitting a log normal risk parameters change with invalid r", testNewSpotLogNormalRiskParametersChangeSubmissionInvalidR)
	t.Run("Submitting a new spot market with a too long reference fails", testNewSpotMarketSubmissionWithTooLongReferenceFails)
	t.Run("Submitting an empty target stake parameters succeeds", testSpotTargetStakeParametersSucceeds)
	t.Run("Submitting target stake parameters with a negative time window fails", testSpotTargetStakeWithNonPositiveTimeWindowFails)
	t.Run("Submitting target stake parameters with a positive time window succeeds", testSpotTargetStakeWithPositiveTimeWindowSucceeds)
	t.Run("Submitting target stake parameters with non positive scaling factor fails ", testSpotTargetStakeWithNonPositiveScalingFactorFails)
	t.Run("Submitting a target stake parameters with positive scaling factor succeeds", testSpotTargetStakeWithPositiveScalingFactorSucceeds)
	t.Run("Submitting a new spot market without target stake parameters succeeds", testNewSpotMarketChangeSubmissionWithoutTargetStakeParamSucceeds)
	t.Run("Submitting a new spot market without spot product definition fails", testNewSpotMarketMarketChangeSubmissionWithoutSpotFails)
	t.Run("Submitting a new spot market with spot product definition succeeds", testNewSpotMarketMarketChangeSubmissionWithSpotSucceeds)
	t.Run("Submitting a new spot market without base or quote asset fails", testNewSpotMarketMarketChangeSubmissionWithoutEitherAssetFails)
	t.Run("Submitting a new spot market with base and quote asset succeeds", testNewSpotMarketMarketChangeSubmissionWithBaseAndQuoteAssetsSucceeds)
	t.Run("Submitting a new spot market without a name fails ", testNewSpotMarketMarketChangeSubmissionWithoutNameFails)
	t.Run("Submitting a new spot market with a name succeeds", testNewSpotMarketMarketChangeSubmissionWithQuoteNameSucceeds)
	t.Run("Submitting a new spot market with price monitoring without triggers succeeds", testSpotPriceMonitoringChangeSubmissionWithoutTriggersSucceeds)
	t.Run("Submitting a new spot market with price monitoring with triggers succeeds", testSpotPriceMonitoringChangeSubmissionWithTriggersSucceeds)
	t.Run("Submitting a new spot market with price monitoring with probability succeeds", testSpotMarketPriceMonitoringChangeSubmissionWithRightTriggerProbabilitySucceeds)
	t.Run("Submitting a new spot market with price monitoring without trigger auction extension fails", testSpotPriceMonitoringChangeSubmissionWithoutTriggerAuctionExtensionFails)
	t.Run("Submitting a new spot market with price monitoring with trigger auction extension succeeds", testSpotMarketPriceMonitoringChangeSubmissionWithTriggerAuctionExtensionSucceeds)
	t.Run("Submitting a new spot market with invalid SLA price range fails", testNewSpotMarketChangeSubmissionWithInvalidLpRangeFails)
	t.Run("Submitting a new spot market with valid SLA price range succeeds", testNewSpotMarketChangeSubmissionWithValidLpRangeSucceeds)
	t.Run("Submitting a new spot market with invalid min time fraction fails", testNewSpotMarketChangeSubmissionWithInvalidMinTimeFractionFails)
	t.Run("Submitting a new spot market with valid min time fraction succeeds", testNewSpotMarketChangeSubmissionWithValidMinTimeFractionSucceeds)
	t.Run("Submitting a new spot market with invalid competition factor fails", testNewSpotMarketChangeSubmissionWithInvalidCompetitionFactorFails)
	t.Run("Submitting a new spot market with valid competition factor succeeds", testNewSpotMarketChangeSubmissionWithValidCompetitionFactorSucceeds)
	t.Run("Submitting a new spot market with invalid hysteresis epochs fails", testNewSpotMarketChangeSubmissionWithInvalidPerformanceHysteresisEpochsFails)
	t.Run("Submitting a new spot market with valid hysteresis epochs succeeds", testNewSpotMarketChangeSubmissionWithValidPerformanceHysteresisEpochsSucceeds)
}

func testNewSpotMarketChangeSubmissionWithoutNewSpotMarketFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market"), commands.ErrIsRequired)
}

func testNewSpotMarketChangeSubmissionWithoutChangesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes"), commands.ErrIsRequired)
}

func testNewSpotMarketChangeSubmissionWithoutDecimalPlacesSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.decimal_places"), commands.ErrMustBePositiveOrZero)
}

func testNewSpotMarketChangeSubmissionWithDecimalPlacesEqualTo0Succeeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						DecimalPlaces: 0,
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.decimal_places"), commands.ErrMustBePositiveOrZero)
}

func testNewSpotMarketChangeSubmissionWithDecimalPlacesAboveOrEqualTo150Fails(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_NewSpotMarket{
						NewSpotMarket: &protoTypes.NewSpotMarket{
							Changes: &protoTypes.NewSpotMarketConfiguration{
								Instrument: &protoTypes.InstrumentConfiguration{
									Product: &protoTypes.InstrumentConfiguration_Spot{},
								},
								DecimalPlaces: tc.value,
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.decimal_places"), commands.ErrMustBeLessThan150)
		})
	}
}

func testNewSpotMarketChangeSubmissionWithDecimalPlacesBelow150Succeeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						DecimalPlaces: test.RandomPositiveU64Before(150),
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.decimal_places"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.decimal_places"), commands.ErrMustBeLessThan150)
}

func testNewSpotMarketChangeSubmissionWithPositionDecimalPlacesEqualTo0Succeeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						PositionDecimalPlaces: 0,
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.position_decimal_places"), commands.ErrMustBePositiveOrZero)
}

func testNewSpotMarketChangeSubmissionWithPositionDecimalPlacesAboveOrEqualTo7Fails(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_NewSpotMarket{
						NewSpotMarket: &protoTypes.NewSpotMarket{
							Changes: &protoTypes.NewSpotMarketConfiguration{
								Instrument: &protoTypes.InstrumentConfiguration{
									Product: &protoTypes.InstrumentConfiguration_Spot{},
								},
								PositionDecimalPlaces: tc.value,
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.position_decimal_places"), commands.ErrMustBeWithinRange7)
		})
	}
}

func testNewSpotMarketChangeSubmissionWithPositionDecimalPlacesBelow7Succeeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						PositionDecimalPlaces: test.RandomPositiveI64Before(7),
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.position_decimal_places"), commands.ErrMustBeWithinRange7)
}

func testNewSpotMarketChangeSubmissionWithoutTargetStakeParamSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.target_stake_parameters"), commands.ErrIsRequired)
}

func testSpotTargetStakeParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						TargetStakeParameters: &protoTypes.TargetStakeParameters{},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.target_stake_parameters"), commands.ErrIsRequired)
}

func testSpotTargetStakeWithNonPositiveTimeWindowFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value int64
	}{
		{
			msg:   "with ratio of 0",
			value: 0,
		}, {
			msg:   "with ratio of -1",
			value: test.RandomNegativeI64(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &protoTypes.ProposalTerms{
					Change: &protoTypes.ProposalTerms_NewSpotMarket{
						NewSpotMarket: &protoTypes.NewSpotMarket{
							Changes: &protoTypes.NewSpotMarketConfiguration{
								Instrument: &protoTypes.InstrumentConfiguration{
									Product: &protoTypes.InstrumentConfiguration_Spot{},
								},
								TargetStakeParameters: &protoTypes.TargetStakeParameters{
									TimeWindow: tc.value,
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.target_stake_parameters.time_window"), commands.ErrMustBePositive)
		})
	}
}

func testSpotTargetStakeWithPositiveTimeWindowSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						TargetStakeParameters: &protoTypes.TargetStakeParameters{
							TimeWindow: test.RandomPositiveI64(),
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.target_stake_parameters.time_window"), commands.ErrMustBePositive)
}

func testSpotTargetStakeWithNonPositiveScalingFactorFails(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_NewSpotMarket{
						NewSpotMarket: &protoTypes.NewSpotMarket{
							Changes: &protoTypes.NewSpotMarketConfiguration{
								Instrument: &protoTypes.InstrumentConfiguration{
									Product: &protoTypes.InstrumentConfiguration_Spot{},
								},
								TargetStakeParameters: &protoTypes.TargetStakeParameters{
									ScalingFactor: tc.value,
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.target_stake_parameters.scaling_factor"), commands.ErrMustBePositive)
		})
	}
}

func testSpotTargetStakeWithPositiveScalingFactorSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						TargetStakeParameters: &protoTypes.TargetStakeParameters{
							ScalingFactor: 1.5,
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.target_stake_parameters.scaling_factor"), commands.ErrMustBePositive)
}

func testSpotPriceMonitoringChangeSubmissionWithoutTriggersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
							Triggers: []*protoTypes.PriceMonitoringTrigger{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers"), commands.ErrIsRequired)
}

func testSpotPriceMonitoringChangeSubmissionWithTriggersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers"), commands.ErrIsRequired)
}

func testNewSpotMarketChangeSubmissionWithTooManyPMTriggersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers"), errors.New("maximum 5 triggers allowed"))
}

func testSpotPriceMonitoringChangeSubmissionWithoutTriggerHorizonFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.0.horizon"), commands.ErrMustBePositive)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.1.horizon"), commands.ErrMustBePositive)
}

func testSpotPriceMonitoringChangeSubmissionWithTriggerHorizonSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
							Triggers: []*protoTypes.PriceMonitoringTrigger{
								{
									Horizon: test.RandomPositiveI64(),
								},
								{
									Horizon: test.RandomPositiveI64(),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.0.horizon"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.1.horizon"), commands.ErrMustBePositive)
}

func testSpotPriceMonitoringChangeSubmissionWithWrongTriggerProbabilityFails(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_NewSpotMarket{
						NewSpotMarket: &protoTypes.NewSpotMarket{
							Changes: &protoTypes.NewSpotMarketConfiguration{
								Instrument: &protoTypes.InstrumentConfiguration{
									Product: &protoTypes.InstrumentConfiguration_Spot{},
								},
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

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.0.probability"),
				errors.New("should be between 0.9 (exclusive) and 1 (exclusive)"))
			assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.1.probability"),
				errors.New("should be between 0.9 (exclusive) and 1 (exclusive)"))
		})
	}
}

func testSpotPriceMonitoringChangeSubmissionWithRightTriggerProbabilitySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &vegapb.NewSpotMarket{
					Changes: &vegapb.NewSpotMarketConfiguration{
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

func testSpotPriceMonitoringChangeSubmissionWithTriggerAuctionExtensionSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &vegapb.ProposalTerms{
			Change: &vegapb.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &vegapb.NewSpotMarket{
					Changes: &vegapb.NewSpotMarketConfiguration{
						PriceMonitoringParameters: &vegapb.PriceMonitoringParameters{
							Triggers: []*vegapb.PriceMonitoringTrigger{
								{
									AuctionExtension: test.RandomPositiveI64(),
								},
								{
									AuctionExtension: test.RandomPositiveI64(),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.0.auction_extension"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.1.auction_extension"), commands.ErrMustBePositive)
}

func testSpotMarketPriceMonitoringChangeSubmissionWithRightTriggerProbabilitySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.0.probability"),
		errors.New("should be between 0 (exclusive) and 1 (exclusive)"))
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.1.probability"),
		errors.New("should be between 0 (exclusive) and 1 (exclusive)"))
}

func testSpotPriceMonitoringChangeSubmissionWithoutTriggerAuctionExtensionFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.0.auction_extension"), commands.ErrMustBePositive)
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.1.auction_extension"), commands.ErrMustBePositive)
}

func testSpotMarketPriceMonitoringChangeSubmissionWithTriggerAuctionExtensionSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
							Triggers: []*protoTypes.PriceMonitoringTrigger{
								{
									AuctionExtension: test.RandomPositiveI64(),
								},
								{
									AuctionExtension: test.RandomPositiveI64(),
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.0.auction_extension"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters.triggers.1.auction_extension"), commands.ErrMustBePositive)
}

func testNewSpotMarketChangeSubmissionWithoutPriceMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters"), commands.ErrIsRequired)
}

func testNewSpotMarketChangeSubmissionWithPriceMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.price_monitoring_parameters"), commands.ErrIsRequired)
}

func testNewSpotMarketChangeSubmissionWithoutInstrumentNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
							Name:    "",
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.name"), commands.ErrIsRequired)
}

func testNewSpotMarketChangeSubmissionWithInstrumentNameSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
							Name:    "My name",
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.name"), commands.ErrIsRequired)
}

func testNewSpotMarketChangeSubmissionWithoutInstrumentCodeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
							Code:    "",
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.code"), commands.ErrIsRequired)
}

func testNewSpotMarketChangeSubmissionWithInstrumentCodeSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Code: "My code",
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.code"), commands.ErrIsRequired)
}

func testNewSpotMarketChangeSubmissionWithFutureProductFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Future{},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.product"), commands.ErrIsMismatching)
}

func testNewSpotMarketChangeSubmissionWithoutProductFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.product"), commands.ErrIsRequired)
}

func testNewSpotMarketChangeSubmissionWithProductSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.product"), commands.ErrIsRequired)
}

func testNewSpotMarketMarketChangeSubmissionWithoutSpotFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot"), commands.ErrIsRequired)
}

func testNewSpotMarketMarketChangeSubmissionWithSpotSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{
								Spot: &protoTypes.SpotProduct{},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot"), commands.ErrIsRequired)
}

func testNewSpotMarketMarketChangeSubmissionWithoutEitherAssetFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{
								Spot: &protoTypes.SpotProduct{
									BaseAsset: "BTC",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot.quote_asset"), commands.ErrIsRequired)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{
								Spot: &protoTypes.SpotProduct{
									QuoteAsset: "USDT",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot.base_asset"), commands.ErrIsRequired)
}

func testNewSpotMarketMarketChangeSubmissionWithBaseAndQuoteAssetsSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{
								Spot: &protoTypes.SpotProduct{
									BaseAsset:  "BTC",
									QuoteAsset: "USDT",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot.base_asset"), commands.ErrIsRequired)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot.quote_asset"), commands.ErrIsRequired)
}

func testNewSpotMarketMarketChangeSubmissionWithoutNameFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{
								Spot: &protoTypes.SpotProduct{
									Name: "",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot.name"), commands.ErrIsRequired)
}

func testNewSpotMarketMarketChangeSubmissionWithQuoteNameSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{
								Spot: &protoTypes.SpotProduct{
									Name: "BTC/USDT",
								},
							},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot.quote_name"), commands.ErrIsRequired)
}

func testNewSpotSimpleRiskParametersChangeSubmissionWithoutSimpleRiskParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_Simple{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.simple"), commands.ErrIsRequired)
}

func testNewSpotSimpleRiskParametersChangeSubmissionWithSimpleRiskParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_Simple{
							Simple: &protoTypes.SimpleModelParams{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.simple"), commands.ErrIsRequired)
}

func testNewSpotSimpleRiskParametersChangeSubmissionWithPositiveMinMoveDownFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_Simple{
							Simple: &protoTypes.SimpleModelParams{
								MinMoveDown: 1,
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.simple.min_move_down"), commands.ErrMustBeNegativeOrZero)
}

func testNewSpotSimpleRiskParametersChangeSubmissionWithNonPositiveMinMoveDownSucceeds(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_NewSpotMarket{
						NewSpotMarket: &protoTypes.NewSpotMarket{
							Changes: &protoTypes.NewSpotMarketConfiguration{
								Instrument: &protoTypes.InstrumentConfiguration{
									Product: &protoTypes.InstrumentConfiguration_Spot{},
								},
								RiskParameters: &protoTypes.NewSpotMarketConfiguration_Simple{
									Simple: &protoTypes.SimpleModelParams{
										MinMoveDown: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.simple.min_move_down"), commands.ErrMustBeNegativeOrZero)
		})
	}
}

func testSpotNewSimpleRiskParametersChangeSubmissionWithNegativeMaxMoveUpFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_Simple{
							Simple: &protoTypes.SimpleModelParams{
								MaxMoveUp: -1,
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.simple.max_move_up"), commands.ErrMustBePositiveOrZero)
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

func testNewSpotSimpleRiskParametersChangeSubmissionWithNonNegativeMaxMoveUpSucceeds(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_NewSpotMarket{
						NewSpotMarket: &protoTypes.NewSpotMarket{
							Changes: &protoTypes.NewSpotMarketConfiguration{
								Instrument: &protoTypes.InstrumentConfiguration{
									Product: &protoTypes.InstrumentConfiguration_Spot{},
								},
								RiskParameters: &protoTypes.NewSpotMarketConfiguration_Simple{
									Simple: &protoTypes.SimpleModelParams{
										MaxMoveUp: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.simple.max_move_up"), commands.ErrMustBePositiveOrZero)
		})
	}
}

func testNewSpotSimpleRiskParametersChangeSubmissionWithWrongProbabilityOfTradingFails(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_NewSpotMarket{
						NewSpotMarket: &protoTypes.NewSpotMarket{
							Changes: &protoTypes.NewSpotMarketConfiguration{
								Instrument: &protoTypes.InstrumentConfiguration{
									Product: &protoTypes.InstrumentConfiguration_Spot{},
								},
								RiskParameters: &protoTypes.NewSpotMarketConfiguration_Simple{
									Simple: &protoTypes.SimpleModelParams{
										ProbabilityOfTrading: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.simple.probability_of_trading"),
				errors.New("should be between 0 (inclusive) and 1 (inclusive)"))
		})
	}
}

func testNewSpotSimpleRiskParametersChangeSubmissionWithRightProbabilityOfTradingSucceeds(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_NewSpotMarket{
						NewSpotMarket: &protoTypes.NewSpotMarket{
							Changes: &protoTypes.NewSpotMarketConfiguration{
								Instrument: &protoTypes.InstrumentConfiguration{
									Product: &protoTypes.InstrumentConfiguration_Spot{},
								},
								RiskParameters: &protoTypes.NewSpotMarketConfiguration_Simple{
									Simple: &protoTypes.SimpleModelParams{
										ProbabilityOfTrading: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.simple.probability_of_trading"),
				errors.New("should be between 0 (inclusive) and 1 (inclusive)"))
		})
	}
}

func testNewSpotLogNormalRiskParametersChangeSubmissionWithoutLogNormalRiskParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal"), commands.ErrIsRequired)
}

func testNewSpotLogNormalRiskParametersChangeSubmissionWithLogNormalRiskParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal"), commands.ErrIsRequired)
}

func testNewSpotLogNormalRiskParametersChangeSubmissionWithoutParamsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params"), commands.ErrIsRequired)
}

func testNewSpotLogNormalRiskParametersChangeSubmissionInvalidRiskAversion(t *testing.T) {
	cTooSmall := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), errors.New("must be between [1e-8, 0.1]"))

	cNeg := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), errors.New("must be between [1e-8, 0.1]"))

	cTooBig := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), errors.New("must be between [1e-8, 0.1]"))

	cJustAboutRight1 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), errors.New("must be between [1e-8, 1)"))

	cJustAboutRight2 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), errors.New("must be between [1e-8, 1)"))
}

func testNewSpotLogNormalRiskParametersChangeSubmissionInvalidTau(t *testing.T) {
	cZero := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.tau"), errors.New("must be between [1e-8, 1]"))

	cNeg := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.tau"), errors.New("must be between [1e-8, 1]"))

	cTooLarge := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.tau"), errors.New("must be between [1e-8, 1]"))

	cJustAboutRight1 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.tau"), errors.New("must be between (0, 1]"))

	cJustAboutRight2 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.tau"), errors.New("must be between (0, 1]"))
}

func testNewSpotLogNormalRiskParametersChangeSubmissionInvalidMu(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.mu"), commands.ErrIsNotValidNumber)

	cTooSmall := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.mu"), errors.New("must be between [-1e-6,1e-6]"))

	cTooLarge := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.mu"), errors.New("must be between [-1e-6,1e-6]"))

	cJustAboutRight1 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.mu"), errors.New("must be between [-20,20]"))

	cJustAboutRight2 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.mu"), errors.New("must be between [-20,20]"))
}

func testNewSpotLogNormalRiskParametersChangeSubmissionInvalidR(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.r"), commands.ErrIsNotValidNumber)

	cTooSmall := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.r"), errors.New("must be between [-1,1]"))

	cTooLarge := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.r"), errors.New("must be between [-1,1]"))

	cJustAboutRight1 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.r"), errors.New("must be between [-20,20]"))

	cJustAboutRight2 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.r"), errors.New("must be between [-20,20]"))
}

func testNewSpotLogNormalRiskParametersChangeSubmissionInvalidSigma(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.sigma"), commands.ErrIsNotValidNumber)

	cNeg := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.sigma"), errors.New("must be between [1e-3,50]"))

	cTooSmall := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.sigma"), errors.New("must be between [1e-3,50]"))

	cTooLarge := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.sigma"), errors.New("must be between [1e-3,50]"))

	cJustAboutRight1 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.sigma"), errors.New("must be between [1e-4,100]"))

	cJustAboutRight2 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						RiskParameters: &protoTypes.NewSpotMarketConfiguration_LogNormal{
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
	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.sigma"), errors.New("must be between [1e-4,100]"))
}

func testNewSpotMarketSubmissionWithTooLongReferenceFails(t *testing.T) {
	ref := make([]byte, 101)
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Reference: string(ref),
	})
	assert.Contains(t, err.Get("proposal_submission.reference"), commands.ErrReferenceTooLong)
}

func testNewSpotMarketChangeSubmissionWithValidLpRangeSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						SlaParams: &protoTypes.LiquiditySLAParameters{
							PriceRange: "50",
						},
					},
				},
			},
		},
	})
	errors := []error{commands.ErrIsNotValidNumber, commands.ErrMustBePositive, commands.ErrMustBePositive, commands.ErrMustBeAtMost100}
	for _, e := range errors {
		assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.sla_params.price_range"), e)
	}
}

func testNewSpotMarketChangeSubmissionWithInvalidLpRangeFails(t *testing.T) {
	priceRanges := []string{"banana", "-1", "0", "101"}
	errors := []error{commands.ErrIsNotValidNumber, commands.ErrMustBeWithinRangeGT0LT20, commands.ErrMustBeWithinRangeGT0LT20, commands.ErrMustBeWithinRangeGT0LT20}

	for i, v := range priceRanges {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &protoTypes.ProposalTerms{
				Change: &protoTypes.ProposalTerms_NewSpotMarket{
					NewSpotMarket: &protoTypes.NewSpotMarket{
						Changes: &protoTypes.NewSpotMarketConfiguration{
							Instrument: &protoTypes.InstrumentConfiguration{
								Product: &protoTypes.InstrumentConfiguration_Spot{},
							},
							SlaParams: &protoTypes.LiquiditySLAParameters{
								PriceRange: v,
							},
						},
					},
				},
			},
		})
		assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.sla_params.price_range"), errors[i])
	}
}

func testNewSpotMarketChangeSubmissionWithInvalidMinTimeFractionFails(t *testing.T) {
	minTimeFraction := []string{"banana", "-1", "-1.1", "1.1", "100"}
	errors := []error{commands.ErrIsNotValidNumber, commands.ErrMustBeWithinRange01, commands.ErrMustBeWithinRange01, commands.ErrMustBeWithinRange01, commands.ErrMustBeWithinRange01}

	for i, v := range minTimeFraction {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &protoTypes.ProposalTerms{
				Change: &protoTypes.ProposalTerms_NewSpotMarket{
					NewSpotMarket: &protoTypes.NewSpotMarket{
						Changes: &protoTypes.NewSpotMarketConfiguration{
							Instrument: &protoTypes.InstrumentConfiguration{
								Product: &protoTypes.InstrumentConfiguration_Spot{},
							},
							SlaParams: &protoTypes.LiquiditySLAParameters{
								CommitmentMinTimeFraction: v,
							},
						},
					},
				},
			},
		})
		assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.sla_params.commitment_min_time_fraction"), errors[i])
	}
}

func testNewSpotMarketChangeSubmissionWithValidMinTimeFractionSucceeds(t *testing.T) {
	minTimeFraction := []string{"0", "0.1", "0.99", "1"}

	for _, v := range minTimeFraction {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &protoTypes.ProposalTerms{
				Change: &protoTypes.ProposalTerms_NewSpotMarket{
					NewSpotMarket: &protoTypes.NewSpotMarket{
						Changes: &protoTypes.NewSpotMarketConfiguration{
							Instrument: &protoTypes.InstrumentConfiguration{
								Product: &protoTypes.InstrumentConfiguration_Spot{},
							},
							SlaParams: &protoTypes.LiquiditySLAParameters{
								CommitmentMinTimeFraction: v,
							},
						},
					},
				},
			},
		})

		errors := []error{commands.ErrIsNotValidNumber, commands.ErrMustBeWithinRange01}
		for _, e := range errors {
			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.sla_params.commitment_min_time_fraction"), e)
		}
	}
}

func testNewSpotMarketChangeSubmissionWithInvalidCompetitionFactorFails(t *testing.T) {
	competitionFactors := []string{"banana", "-1", "-1.1", "1.1", "100"}
	errors := []error{commands.ErrIsNotValidNumber, commands.ErrMustBeWithinRange01, commands.ErrMustBeWithinRange01, commands.ErrMustBeWithinRange01, commands.ErrMustBeWithinRange01}

	for i, v := range competitionFactors {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &protoTypes.ProposalTerms{
				Change: &protoTypes.ProposalTerms_NewSpotMarket{
					NewSpotMarket: &protoTypes.NewSpotMarket{
						Changes: &protoTypes.NewSpotMarketConfiguration{
							Instrument: &protoTypes.InstrumentConfiguration{
								Product: &protoTypes.InstrumentConfiguration_Spot{},
							},
							SlaParams: &protoTypes.LiquiditySLAParameters{
								SlaCompetitionFactor: v,
							},
						},
					},
				},
			},
		})
		assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.sla_params.sla_competition_factor"), errors[i])
	}
}

func testNewSpotMarketChangeSubmissionWithValidCompetitionFactorSucceeds(t *testing.T) {
	minTimeFraction := []string{"0", "0.1", "0.99", "1"}

	for _, v := range minTimeFraction {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &protoTypes.ProposalTerms{
				Change: &protoTypes.ProposalTerms_NewSpotMarket{
					NewSpotMarket: &protoTypes.NewSpotMarket{
						Changes: &protoTypes.NewSpotMarketConfiguration{
							Instrument: &protoTypes.InstrumentConfiguration{
								Product: &protoTypes.InstrumentConfiguration_Spot{},
							},
							SlaParams: &protoTypes.LiquiditySLAParameters{
								SlaCompetitionFactor: v,
							},
						},
					},
				},
			},
		})

		errors := []error{commands.ErrIsNotValidNumber, commands.ErrMustBeWithinRange01}
		for _, e := range errors {
			assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.sla_params.sla_competition_factor"), e)
		}
	}
}

func testNewSpotMarketChangeSubmissionWithInvalidPerformanceHysteresisEpochsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						SlaParams: &protoTypes.LiquiditySLAParameters{
							PerformanceHysteresisEpochs: 367,
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.sla_params.performance_hysteresis_epochs"), commands.ErrMustBeLessThen366)
}

func testNewSpotMarketChangeSubmissionWithValidPerformanceHysteresisEpochsSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewSpotMarket{
				NewSpotMarket: &protoTypes.NewSpotMarket{
					Changes: &protoTypes.NewSpotMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{
							Product: &protoTypes.InstrumentConfiguration_Spot{},
						},
						SlaParams: &protoTypes.LiquiditySLAParameters{
							PerformanceHysteresisEpochs: 1,
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.new_spot_market.changes.sla_params.performance_hysteresis_epochs"), commands.ErrMustBePositive)
}
