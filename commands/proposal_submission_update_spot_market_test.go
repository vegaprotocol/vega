package commands_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestCheckProposalSubmissionForUpdateSpotMarket(t *testing.T) {
	t.Run("Submitting a market change without update market fails", testUpdateSpotMarketChangeSubmissionWithoutUpdateMarketFails)
	t.Run("Submitting a market change without changes fails", testUpdateSpotMarketChangeSubmissionWithoutChangesFails)
	t.Run("Submitting a market change without decimal places succeeds", testUpdateSpotMarketChangeSubmissionWithoutDecimalPlacesSucceeds)
	t.Run("Submitting a update market without price monitoring succeeds", testUpdateSpotMarketChangeSubmissionWithoutPriceMonitoringSucceeds)
	t.Run("Submitting a update market with price monitoring succeeds", testUpdateSpotMarketChangeSubmissionWithPriceMonitoringSucceeds)
	t.Run("Submitting a price monitoring change without triggers succeeds", testUpdateSpotMarketPriceMonitoringChangeSubmissionWithoutTriggersSucceeds)
	t.Run("Submitting a price monitoring change with triggers succeeds", testUpdateSpotMarketPriceMonitoringChangeSubmissionWithTriggersSucceeds)
	t.Run("Submitting a price monitoring change without trigger horizon fails", testUpdateSpotMarketPriceMonitoringChangeSubmissionWithoutTriggerHorizonFails)
	t.Run("Submitting a price monitoring change with trigger horizon succeeds", testUpdateSpotMarketPriceMonitoringChangeSubmissionWithTriggerHorizonSucceeds)
	t.Run("Submitting a price monitoring change with wrong trigger probability fails", testUpdateSpotMarketPriceMonitoringChangeSubmissionWithWrongTriggerProbabilityFails)
	t.Run("Submitting a price monitoring change with right trigger probability succeeds", testUpdateSpotMarketPriceMonitoringChangeSubmissionWithRightTriggerProbabilitySucceeds)
	t.Run("Submitting a price monitoring change without trigger auction extension fails", testUpdateSpotMarketPriceMonitoringChangeSubmissionWithoutTriggerAuctionExtensionFails)
	t.Run("Submitting a price monitoring change with trigger auction extension succeeds", testUpdateSpotMarketPriceMonitoringChangeSubmissionWithTriggerAuctionExtensionSucceeds)
	t.Run("Submitting a update market without liquidity monitoring succeeds", testUpdateSpotMarketChangeSubmissionWithoutLiquidityMonitoringSucceeds)
	t.Run("Submitting a simple risk parameters change without simple risk parameters fails", testUpdateSpotSimpleRiskParametersChangeSubmissionWithoutSimpleRiskParametersFails)
	t.Run("Submitting a simple risk parameters change with simple risk parameters succeeds", testUpdateSpotSimpleRiskParametersChangeSubmissionWithSimpleRiskParametersSucceeds)
	t.Run("Submitting a simple risk parameters change with min move down fails", testUpdateSpotSimpleRiskParametersChangeSubmissionWithPositiveMinMoveDownFails)
	t.Run("Submitting a simple risk parameters change with min move down succeeds", testUpdateSpotSimpleRiskParametersChangeSubmissionWithNonPositiveMinMoveDownSucceeds)
	t.Run("Submitting a simple risk parameters change with max move up fails", testUpdateSpotSimpleRiskParametersChangeSubmissionWithNegativeMaxMoveUpFails)
	t.Run("Submitting a simple risk parameters change with max move up succeeds", testUpdateSpotSimpleRiskParametersChangeSubmissionWithNonNegativeMaxMoveUpSucceeds)
	t.Run("Submitting a simple risk parameters change with wrong probability of trading fails", testUpdateSpotSimpleRiskParametersChangeSubmissionWithWrongProbabilityOfTradingFails)
	t.Run("Submitting a simple risk parameters change with right probability of trading succeeds", testUpdateSpotSimpleRiskParametersChangeSubmissionWithRightProbabilityOfTradingSucceeds)
	t.Run("Submitting a log normal risk parameters change without log normal risk parameters fails", testUpdateSpotLogNormalRiskParametersChangeSubmissionWithoutLogNormalRiskParametersFails)
	t.Run("Submitting a log normal risk parameters change with log normal risk parameters succeeds", testUpdateSpotLogNormalRiskParametersChangeSubmissionWithLogNormalRiskParametersSucceeds)
	t.Run("Submitting a log normal risk parameters change with params fails", testUpdateSpotLogNormalRiskParametersChangeSubmissionWithoutParamsFails)
	t.Run("Submitting a log normal risk parameters change with invalid risk aversion", testUpdateSpotLogNormalRiskParametersChangeSubmissionInvalidRiskAversion)
	t.Run("Submitting a log normal risk parameters change with invalid tau", testUpdateSpotLogNormalRiskParametersChangeSubmissionInvalidTau)
	t.Run("Submitting a log normal risk parameters change with invalid mu", testUpdateSpotLogNormalRiskParametersChangeSubmissionInvalidMu)
	t.Run("Submitting a log normal risk parameters change with invalid sigma", testUpdateSpotLogNormalRiskParametersChangeSubmissionInvalidSigma)
	t.Run("Submitting a log normal risk parameters change with invalid r", testUpdateSpotLogNormalRiskParametersChangeSubmissionInvalidR)
	t.Run("Submitting a spot market update with a too long reference fails", testUpdateSpotMarketSubmissionWithTooLongReferenceFails)
	t.Run("Submitting a spot market update with market ID succeeds", testUpdateSpotMarketWithMarketIDSucceeds)
	t.Run("Submitting a spot market update without market ID fails", testUpdateSpotMarketWithoutMarketIDFails)
	t.Run("Submitting a spot market update without target stake parameters fails", testUpdateSpotMarketWithoutTargetStakeParametersFails)
	t.Run("Submitting a spot market update with a target stake parameters with non positive time window fails", testUpdateSpotMarketTargetStakeParamsWithNonPositiveTimeWindowFails)
	t.Run("Submitting a spot market update with a target stake parameters with positive time window succeeds", testUpdateSpotMarketWithTargetStakeParamsPositiveTimeWindowSucceeds)
	t.Run("Submitting a spot market update with a target stake parameters with non positive scaling factor fails", testUpdateSpotMarketWithTargetStakeParamsNonPositiveScalingFactorFails)
	t.Run("Submitting a spot market update with a target stake parameters with positive time window succeeds", testUpdateSpotMarketWithTargetStakeParamsPositiveScalingFactorSucceeds)
	t.Run("Submitting a spot market update with a target stake parameters succeeds", testUpdateSpotMarketWithTargetStakeParametersSucceeds)
	t.Run("Submitting a spot market update with invalid SLA price range fails", testUpdateSpotMarketChangeSubmissionWithInvalidLpRangeFails)
	t.Run("Submitting a spot market update with valid SLA price range succeeds", testUpdateSpotMarketChangeSubmissionWithValidLpRangeSucceeds)
	t.Run("Submitting a spot market update with invalid min time fraction fails", testUpdateSpotMarketChangeSubmissionWithInvalidMinTimeFractionFails)
	t.Run("Submitting a spot market update with valid min time fraction succeeds", testUpdateSpotMarketChangeSubmissionWithValidMinTimeFractionSucceeds)
	t.Run("Submitting a spot market update with invalid fee calculation time step fails", testUpdateSpotMarketChangeSubmissionWithInvalidCalculationTimeStepFails)
	t.Run("Submitting a spot market update with valid fee calculation time step succeeds", testUpdateSpotMarketChangeSubmissionWithValidCalculationTimeStepSucceeds)
	t.Run("Submitting a spot market update with invalid competition factor fails", testUpdateSpotMarketChangeSubmissionWithInvalidCompetitionFactorFails)
	t.Run("Submitting a spot market update with valid competition factor succeeds", testUpdateSpotMarketChangeSubmissionWithValidCompetitionFactorSucceeds)
	t.Run("Submitting a spot market update with valid hysteresis epochs succeeds", testUpdateSpotMarketChangeSubmissionWithValidPerformanceHysteresisEpochsSucceeds)
}

func testUpdateSpotMarketChangeSubmissionWithoutUpdateMarketFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market"), commands.ErrIsRequired)
}

func testUpdateSpotMarketChangeSubmissionWithoutChangesFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes"), commands.ErrIsRequired)
}

func testUpdateSpotMarketChangeSubmissionWithoutDecimalPlacesSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.decimal_places"), commands.ErrMustBePositiveOrZero)
}

func testUpdateSpotMarketChangeSubmissionWithoutLiquidityMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.liquidity_monitoring_parameters"), commands.ErrIsRequired)
}

func testUpdateSpotMarketWithoutTargetStakeParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.target_stake_parameters"), commands.ErrIsRequired)
}

func testUpdateSpotMarketWithTargetStakeParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						TargetStakeParameters: &protoTypes.TargetStakeParameters{},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.target_stake_parameters"), commands.ErrIsRequired)
}

func testUpdateSpotMarketTargetStakeParamsWithNonPositiveTimeWindowFails(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
						UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
							Changes: &protoTypes.UpdateSpotMarketConfiguration{
								TargetStakeParameters: &protoTypes.TargetStakeParameters{
									TimeWindow: tc.value,
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.target_stake_parameters.time_window"), commands.ErrMustBePositive)
		})
	}
}

func testUpdateSpotMarketWithTargetStakeParamsPositiveTimeWindowSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						TargetStakeParameters: &protoTypes.TargetStakeParameters{
							TimeWindow: RandomPositiveI64(),
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.liquidity_monitoring_parameters.target_stake_parameters.time_window"), commands.ErrMustBePositive)
}

func testUpdateSpotMarketWithTargetStakeParamsNonPositiveScalingFactorFails(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
						UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
							Changes: &protoTypes.UpdateSpotMarketConfiguration{
								TargetStakeParameters: &protoTypes.TargetStakeParameters{
									ScalingFactor: tc.value,
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.target_stake_parameters.scaling_factor"), commands.ErrMustBePositive)
		})
	}
}

func testUpdateSpotMarketWithTargetStakeParamsPositiveScalingFactorSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						TargetStakeParameters: &protoTypes.TargetStakeParameters{
							ScalingFactor: 1.5,
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.target_stake_parameters.scaling_factor"), commands.ErrMustBePositive)
}

func testUpdateSpotMarketPriceMonitoringChangeSubmissionWithoutTriggersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
							Triggers: []*protoTypes.PriceMonitoringTrigger{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers"), commands.ErrIsRequired)
}

func testUpdateSpotMarketPriceMonitoringChangeSubmissionWithTriggersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers"), commands.ErrIsRequired)
}

func testUpdateSpotMarketPriceMonitoringChangeSubmissionWithoutTriggerHorizonFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers.0.horizon"), commands.ErrMustBePositive)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers.1.horizon"), commands.ErrMustBePositive)
}

func testUpdateSpotMarketPriceMonitoringChangeSubmissionWithTriggerHorizonSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers.0.horizon"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers.1.horizon"), commands.ErrMustBePositive)
}

func testUpdateSpotMarketPriceMonitoringChangeSubmissionWithWrongTriggerProbabilityFails(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
						UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
							Changes: &protoTypes.UpdateSpotMarketConfiguration{
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

			assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers.0.probability"),
				errors.New("should be between 0.9 (exclusive) and 1 (exclusive)"))
			assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers.1.probability"),
				errors.New("should be between 0.9 (exclusive) and 1 (exclusive)"))
		})
	}
}

func testUpdateSpotMarketPriceMonitoringChangeSubmissionWithRightTriggerProbabilitySucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers.0.probability"),
		errors.New("should be between 0 (exclusive) and 1 (exclusive)"))
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers.1.probability"),
		errors.New("should be between 0 (exclusive) and 1 (exclusive)"))
}

func testUpdateSpotMarketPriceMonitoringChangeSubmissionWithoutTriggerAuctionExtensionFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
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

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers.0.auction_extension"), commands.ErrMustBePositive)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers.1.auction_extension"), commands.ErrMustBePositive)
}

func testUpdateSpotMarketPriceMonitoringChangeSubmissionWithTriggerAuctionExtensionSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers.0.auction_extension"), commands.ErrMustBePositive)
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters.triggers.1.auction_extension"), commands.ErrMustBePositive)
}

func testUpdateSpotMarketChangeSubmissionWithoutPriceMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters"), commands.ErrIsRequired)
}

func testUpdateSpotMarketChangeSubmissionWithPriceMonitoringSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.price_monitoring_parameters"), commands.ErrIsRequired)
}

func testUpdateSpotSimpleRiskParametersChangeSubmissionWithoutSimpleRiskParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_Simple{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.simple"), commands.ErrIsRequired)
}

func testUpdateSpotSimpleRiskParametersChangeSubmissionWithSimpleRiskParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_Simple{
							Simple: &protoTypes.SimpleModelParams{},
						},
					},
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.simple"), commands.ErrIsRequired)
}

func testUpdateSpotSimpleRiskParametersChangeSubmissionWithPositiveMinMoveDownFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_Simple{
							Simple: &protoTypes.SimpleModelParams{
								MinMoveDown: 1,
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.simple.min_move_down"), commands.ErrMustBeNegativeOrZero)
}

func testUpdateSpotSimpleRiskParametersChangeSubmissionWithNonPositiveMinMoveDownSucceeds(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
						UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
							Changes: &protoTypes.UpdateSpotMarketConfiguration{
								RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_Simple{
									Simple: &protoTypes.SimpleModelParams{
										MinMoveDown: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.simple.min_move_down"), commands.ErrMustBeNegativeOrZero)
		})
	}
}

func testUpdateSpotSimpleRiskParametersChangeSubmissionWithNegativeMaxMoveUpFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_Simple{
							Simple: &protoTypes.SimpleModelParams{
								MaxMoveUp: -1,
							},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.simple.max_move_up"), commands.ErrMustBePositiveOrZero)
}

func testUpdateSpotSimpleRiskParametersChangeSubmissionWithNonNegativeMaxMoveUpSucceeds(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
						UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
							Changes: &protoTypes.UpdateSpotMarketConfiguration{
								RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_Simple{
									Simple: &protoTypes.SimpleModelParams{
										MaxMoveUp: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.simple.max_move_up"), commands.ErrMustBePositiveOrZero)
		})
	}
}

func testUpdateSpotSimpleRiskParametersChangeSubmissionWithWrongProbabilityOfTradingFails(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
						UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
							Changes: &protoTypes.UpdateSpotMarketConfiguration{
								RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_Simple{
									Simple: &protoTypes.SimpleModelParams{
										ProbabilityOfTrading: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.simple.probability_of_trading"),
				errors.New("should be between 0 (inclusive) and 1 (inclusive)"))
		})
	}
}

func testUpdateSpotSimpleRiskParametersChangeSubmissionWithRightProbabilityOfTradingSucceeds(t *testing.T) {
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
					Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
						UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
							Changes: &protoTypes.UpdateSpotMarketConfiguration{
								RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_Simple{
									Simple: &protoTypes.SimpleModelParams{
										ProbabilityOfTrading: tc.value,
									},
								},
							},
						},
					},
				},
			})

			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.simple.probability_of_trading"),
				errors.New("should be between 0 (inclusive) and 1 (inclusive)"))
		})
	}
}

func testUpdateSpotLogNormalRiskParametersChangeSubmissionWithoutLogNormalRiskParametersFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_LogNormal{},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal"), commands.ErrIsRequired)
}

func testUpdateSpotLogNormalRiskParametersChangeSubmissionWithLogNormalRiskParametersSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_LogNormal{
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

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal"), commands.ErrIsRequired)
}

func testUpdateSpotLogNormalRiskParametersChangeSubmissionWithoutParamsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.params"), commands.ErrIsRequired)
}

func testUpdateSpotLogNormalRiskParametersChangeSubmissionInvalidRiskAversion(t *testing.T) {
	cZero := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0,
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
	err := checkProposalSubmission(cZero)
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), commands.ErrMustBePositive)

	cNeg := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: -0.1,
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.risk_aversion_parameter"), commands.ErrMustBePositive)
}

func testUpdateSpotLogNormalRiskParametersChangeSubmissionInvalidTau(t *testing.T) {
	cZero := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.tau"), commands.ErrMustBePositive)

	cNeg := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   -0.2,
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.tau"), commands.ErrMustBePositive)
}

func testUpdateSpotLogNormalRiskParametersChangeSubmissionInvalidMu(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_LogNormal{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.params.mu"), commands.ErrIsNotValidNumber)
}

func testUpdateSpotLogNormalRiskParametersChangeSubmissionInvalidR(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.params.r"), commands.ErrIsNotValidNumber)
}

func testUpdateSpotLogNormalRiskParametersChangeSubmissionInvalidSigma(t *testing.T) {
	cNaN := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.params.sigma"), commands.ErrIsNotValidNumber)

	cNeg := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.params.sigma"), commands.ErrMustBePositive)

	c0 := &commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_LogNormal{
							LogNormal: &protoTypes.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &protoTypes.LogNormalModelParams{
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
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.params.sigma"), commands.ErrMustBePositive)
}

func testUpdateSpotMarketSubmissionWithTooLongReferenceFails(t *testing.T) {
	ref := make([]byte, 101)
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Reference: string(ref),
	})
	assert.Contains(t, err.Get("proposal_submission.reference"), commands.ErrReferenceTooLong)
}

func testUpdateSpotMarketWithMarketIDSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					MarketId: "12345",
				},
			},
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.market_id"), commands.ErrIsRequired)
}

func testUpdateSpotMarketWithoutMarketIDFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.market_id"), commands.ErrIsRequired)
}

func testUpdateSpotMarketChangeSubmissionWithValidLpRangeSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
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
		assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.sla_params.price_range"), e)
	}
}

func testUpdateSpotMarketChangeSubmissionWithInvalidLpRangeFails(t *testing.T) {
	priceRanges := []string{"banana", "-1", "0", "101"}
	errors := []error{commands.ErrIsNotValidNumber, commands.ErrMustBePositive, commands.ErrMustBePositive, commands.ErrMustBeAtMost100}

	for i, v := range priceRanges {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &protoTypes.ProposalTerms{
				Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
					UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
						Changes: &protoTypes.UpdateSpotMarketConfiguration{
							SlaParams: &protoTypes.LiquiditySLAParameters{
								PriceRange: v,
							},
						},
					},
				},
			},
		})
		assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.sla_params.price_range"), errors[i])
	}
}

func testUpdateSpotMarketChangeSubmissionWithInvalidMinTimeFractionFails(t *testing.T) {
	minTimeFraction := []string{"banana", "-1", "-1.1", "1.1", "100"}
	errors := []error{commands.ErrIsNotValidNumber, commands.ErrMustBeWithinRange01, commands.ErrMustBeWithinRange01, commands.ErrMustBeWithinRange01, commands.ErrMustBeWithinRange01}

	for i, v := range minTimeFraction {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &protoTypes.ProposalTerms{
				Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
					UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
						Changes: &protoTypes.UpdateSpotMarketConfiguration{
							SlaParams: &protoTypes.LiquiditySLAParameters{
								CommitmentMinTimeFraction: v,
							},
						},
					},
				},
			},
		})
		assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.sla_params.commitment_min_time_fraction"), errors[i])
	}
}

func testUpdateSpotMarketChangeSubmissionWithValidMinTimeFractionSucceeds(t *testing.T) {
	minTimeFraction := []string{"0", "0.1", "0.99", "1"}

	for _, v := range minTimeFraction {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &protoTypes.ProposalTerms{
				Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
					UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
						Changes: &protoTypes.UpdateSpotMarketConfiguration{
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
			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.sla_params.commitment_min_time_fraction"), e)
		}
	}
}

func testUpdateSpotMarketChangeSubmissionWithInvalidCalculationTimeStepFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						SlaParams: &protoTypes.LiquiditySLAParameters{
							ProvidersFeeCalculationTimeStep: 0,
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.sla_params.providers.fee.calculation_time_step"), commands.ErrMustBePositive)
}

func testUpdateSpotMarketChangeSubmissionWithValidCalculationTimeStepSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						SlaParams: &protoTypes.LiquiditySLAParameters{
							ProvidersFeeCalculationTimeStep: 1,
						},
					},
				},
			},
		},
	})
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.sla_params.providers.fee.calculation_time_step"), commands.ErrMustBePositive)
}

func testUpdateSpotMarketChangeSubmissionWithInvalidCompetitionFactorFails(t *testing.T) {
	competitionFactors := []string{"banana", "-1", "-1.1", "1.1", "100"}
	errors := []error{commands.ErrIsNotValidNumber, commands.ErrMustBeWithinRange01, commands.ErrMustBeWithinRange01, commands.ErrMustBeWithinRange01, commands.ErrMustBeWithinRange01}

	for i, v := range competitionFactors {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &protoTypes.ProposalTerms{
				Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
					UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
						Changes: &protoTypes.UpdateSpotMarketConfiguration{
							SlaParams: &protoTypes.LiquiditySLAParameters{
								SlaCompetitionFactor: v,
							},
						},
					},
				},
			},
		})
		assert.Contains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.sla_params.sla_competition_factor"), errors[i])
	}
}

func testUpdateSpotMarketChangeSubmissionWithValidCompetitionFactorSucceeds(t *testing.T) {
	minTimeFraction := []string{"0", "0.1", "0.99", "1"}

	for _, v := range minTimeFraction {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &protoTypes.ProposalTerms{
				Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
					UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
						Changes: &protoTypes.UpdateSpotMarketConfiguration{
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
			assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.sla_params.sla_competition_factor"), e)
		}
	}
}

func testUpdateSpotMarketChangeSubmissionWithValidPerformanceHysteresisEpochsSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_UpdateSpotMarket{
				UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
					Changes: &protoTypes.UpdateSpotMarketConfiguration{
						SlaParams: &protoTypes.LiquiditySLAParameters{
							PerformanceHysteresisEpochs: 1,
						},
					},
				},
			},
		},
	})
	assert.NotContains(t, err.Get("proposal_submission.terms.change.update_spot_market.changes.sla_params.performance_hysteresis_epochs"), commands.ErrMustBePositive)
}
