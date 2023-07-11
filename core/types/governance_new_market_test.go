package types_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	v1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/require"
)

func TestNewMarketProposalMapping(t *testing.T) {
	now := time.Now()
	closeDelta, enactDelta := time.Second, 2*time.Second
	parentID := "parent ID"
	insFraction := num.DecimalFromFloat(.8)
	cmd := &commandspb.ProposalSubmission{
		Reference: "proposal reference",
		Terms: &vegapb.ProposalTerms{
			ClosingTimestamp:   now.Add(closeDelta).Unix(),
			EnactmentTimestamp: now.Add(enactDelta).Unix(),
			Change: &vegapb.ProposalTerms_NewMarket{
				NewMarket: &vegapb.NewMarket{
					Changes: &vegapb.NewMarketConfiguration{
						Instrument: &vegapb.InstrumentConfiguration{
							Name: "test instrument",
							Code: "TI",
							Product: &vegapb.InstrumentConfiguration_Future{
								Future: &vegapb.FutureProduct{
									SettlementAsset: "ETH",
									QuoteName:       "ETH",
									DataSourceSpecForSettlementData: &vegapb.DataSourceDefinition{
										SourceType: &vegapb.DataSourceDefinition_External{
											External: &vegapb.DataSourceDefinitionExternal{
												SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
													Oracle: &vegapb.DataSourceSpecConfiguration{
														Signers: []*v1.Signer{
															{
																Signer: &v1.Signer_PubKey{
																	PubKey: &v1.PubKey{
																		Key: "0xiBADC0FFEE0DDF00D",
																	},
																},
															},
														},
														Filters: []*v1.Filter{
															{
																Key: &v1.PropertyKey{
																	Name:                "settlekey",
																	Type:                v1.PropertyKey_TYPE_INTEGER,
																	NumberDecimalPlaces: ptr[uint64](5),
																},
																Conditions: []*v1.Condition{
																	{
																		Operator: v1.Condition_OPERATOR_UNSPECIFIED,
																	},
																},
															},
														},
													},
												},
											},
										},
									},
									DataSourceSpecForTradingTermination: &vegapb.DataSourceDefinition{
										SourceType: &vegapb.DataSourceDefinition_External{
											External: &vegapb.DataSourceDefinitionExternal{
												SourceType: &vegapb.DataSourceDefinitionExternal_Oracle{
													Oracle: &vegapb.DataSourceSpecConfiguration{
														Signers: []*v1.Signer{
															{
																Signer: &v1.Signer_PubKey{
																	PubKey: &v1.PubKey{
																		Key: "0xiBADC0FFEE0DDF00D",
																	},
																},
															},
														},
														Filters: []*v1.Filter{
															{
																Key: &v1.PropertyKey{
																	Name: "termkey",
																	Type: v1.PropertyKey_TYPE_BOOLEAN,
																},
																Conditions: []*v1.Condition{
																	{
																		Operator: v1.Condition_OPERATOR_EQUALS,
																		Value:    "true",
																	},
																},
															},
														},
													},
												},
											},
										},
									},
									DataSourceSpecBinding: &vegapb.DataSourceSpecToFutureBinding{
										SettlementDataProperty:     "settlekey",
										TradingTerminationProperty: "termkey",
									},
								},
							},
						},
						DecimalPlaces: 5,
						Metadata:      []string{"foo"},
						PriceMonitoringParameters: &vegapb.PriceMonitoringParameters{
							Triggers: []*vegapb.PriceMonitoringTrigger{
								{
									Horizon:          5,
									Probability:      "0.95",
									AuctionExtension: 3,
								},
							},
						},
						LiquidityMonitoringParameters: &vegapb.LiquidityMonitoringParameters{
							TargetStakeParameters: &vegapb.TargetStakeParameters{
								TimeWindow:    10,
								ScalingFactor: 0.1,
							},
							TriggeringRatio:  "0.01",
							AuctionExtension: 1,
						},
						RiskParameters: &vegapb.NewMarketConfiguration_Simple{
							Simple: &vegapb.SimpleModelParams{
								FactorLong:           1.0,
								FactorShort:          1.1,
								MaxMoveUp:            2.0,
								MinMoveDown:          3.0,
								ProbabilityOfTrading: 0.96,
							},
						},
						PositionDecimalPlaces:   1,
						LinearSlippageFactor:    "0.1",
						QuadraticSlippageFactor: "0.0001",
						Successor: &vegapb.SuccessorConfiguration{
							ParentMarketId:        parentID,
							InsurancePoolFraction: insFraction.String(),
						},
						LiquiditySlaParameters: &vegapb.LiquiditySLAParameters{
							PriceRange:                      "0.95",
							CommitmentMinTimeFraction:       "0.5",
							ProvidersFeeCalculationTimeStep: time.Duration(time.Second * 5).Nanoseconds(),
							PerformanceHysteresisEpochs:     4,
							SlaCompetitionFactor:            "0.5",
						},
					},
				},
			},
		},
		Rationale: &vegapb.ProposalRationale{
			Description: "test a successor market proposal",
			Title:       "proposal mapping",
		},
	}
	// convert to internal proposal type
	submission, err := types.NewProposalSubmissionFromProto(cmd)
	require.NoError(t, err)
	// convert back
	s2proto := submission.IntoProto()
	require.EqualValues(t, cmd, s2proto)
	// make sure successor fields are mapped as expected
	nm := submission.Terms.GetNewMarket()
	require.NotNil(t, nm)
	suc := nm.Successor()
	require.NotNil(t, suc)
	parent, ok := nm.ParentMarketID()
	require.True(t, ok)
	require.Equal(t, parentID, parent)
	require.EqualValues(t, insFraction, suc.InsurancePoolFraction)
}

func ptr[T any](v T) *T {
	return &v
}
