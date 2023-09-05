// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gql_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/datanode/gateway"
	gql "code.vegaprotocol.io/vega/datanode/gateway/graphql"
	"code.vegaprotocol.io/vega/datanode/gateway/graphql/mocks"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	vega "code.vegaprotocol.io/vega/protos/vega"
	datav1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

func TestNewResolverRoot_ConstructAndResolve(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
	assert.NotNil(t, root)

	partyResolver := root.Party()
	assert.NotNil(t, partyResolver)

	marketResolver := root.Market()
	assert.NotNil(t, marketResolver)

	depthResolver := root.MarketDepth()
	assert.NotNil(t, depthResolver)

	candleResolver := root.Candle()
	assert.NotNil(t, candleResolver)

	orderResolver := root.Order()
	assert.NotNil(t, orderResolver)

	tradeResolver := root.Trade()
	assert.NotNil(t, tradeResolver)

	priceLevelResolver := root.PriceLevel()
	assert.NotNil(t, priceLevelResolver)

	positionResolver := root.Position()
	assert.NotNil(t, positionResolver)

	queryResolver := root.Query()
	assert.NotNil(t, queryResolver)

	subsResolver := root.Subscription()
	assert.NotNil(t, subsResolver)

	epochResolver := root.Epoch()
	assert.NotNil(t, epochResolver)

	perpetualResolver := root.Perpetual()
	assert.NotNil(t, perpetualResolver)

	perpetualProductResolver := root.PerpetualProduct()
	assert.NotNil(t, perpetualProductResolver)
}

func TestNewResolverRoot_QueryResolver(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
	assert.NotNil(t, root)

	queryResolver := root.Query()
	assert.NotNil(t, queryResolver)
}

func getTestFutureMarket(termType protoTypes.DataSourceContentType) *protoTypes.Market {
	pk := dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)
	term := &protoTypes.DataSourceSpec{}
	switch termType {
	case protoTypes.DataSourceContentTypeOracle:
		term = &protoTypes.DataSourceSpec{
			Data: protoTypes.NewDataSourceDefinition(
				protoTypes.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&vega.DataSourceDefinitionExternal_Oracle{
					Oracle: &protoTypes.DataSourceSpecConfiguration{
						Signers: []*datav1.Signer{pk.IntoProto()},
						Filters: []*datav1.Filter{
							{
								Key: &datav1.PropertyKey{
									Name: "trading.terminated",
									Type: datav1.PropertyKey_TYPE_BOOLEAN,
								},
								Conditions: []*datav1.Condition{},
							},
						},
					},
				},
			),
		}

	case protoTypes.DataSourceContentTypeEthOracle:
		term = &protoTypes.DataSourceSpec{
			Data: protoTypes.NewDataSourceDefinition(
				protoTypes.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&vega.DataSourceDefinitionExternal_EthOracle{
					EthOracle: &protoTypes.EthCallSpec{
						Address:               "test-address",
						Abi:                   "null",
						Method:                "stake",
						RequiredConfirmations: uint64(9),
						Trigger: &protoTypes.EthCallTrigger{
							Trigger: &protoTypes.EthCallTrigger_TimeTrigger{
								TimeTrigger: &protoTypes.EthTimeTrigger{},
							},
						},
					},
				},
			),
		}

	case protoTypes.DataSourceContentTypeInternalTimeTermination:
		term = &protoTypes.DataSourceSpec{
			Data: protoTypes.NewDataSourceDefinition(
				protoTypes.DataSourceContentTypeInternalTimeTermination,
			).SetTimeTriggerConditionConfig(
				[]*datav1.Condition{
					{
						Operator: datav1.Condition_OPERATOR_GREATER_THAN,
						Value:    "test-value",
					},
				},
			),
		}
	}
	market := getTestMarket()
	market.TradableInstrument.Instrument.Product = &protoTypes.Instrument_Future{
		Future: &protoTypes.Future{
			SettlementAsset: "Ethereum/Ether",
			DataSourceSpecForSettlementData: &protoTypes.DataSourceSpec{
				Data: protoTypes.NewDataSourceDefinition(
					protoTypes.DataSourceContentTypeOracle,
				).SetOracleConfig(
					&vega.DataSourceDefinitionExternal_Oracle{
						Oracle: &protoTypes.DataSourceSpecConfiguration{
							Signers: []*datav1.Signer{pk.IntoProto()},
							Filters: []*datav1.Filter{
								{
									Key: &datav1.PropertyKey{
										Name: "prices.ETH.value",
										Type: datav1.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*datav1.Condition{},
								},
							},
						},
					},
				),
			},
			DataSourceSpecForTradingTermination: term,
			DataSourceSpecBinding: &protoTypes.DataSourceSpecToFutureBinding{
				SettlementDataProperty:     "prices.ETH.value",
				TradingTerminationProperty: "trading.terminated",
			},
		},
	}
	return market
}

func getTestSpotMarket() *protoTypes.Market {
	mkt := getTestMarket()

	mkt.TradableInstrument.Instrument.Product = &protoTypes.Instrument_Spot{
		Spot: &protoTypes.Spot{
			BaseAsset:  "Ethereum",
			QuoteAsset: "USD",
			Name:       "ETH/USD",
		},
	}

	return mkt
}

func getTestPerpetualMarket() *protoTypes.Market {
	pk := dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)
	mkt := getTestMarket()
	mkt.TradableInstrument.Instrument.Product = &protoTypes.Instrument_Perpetual{
		Perpetual: &protoTypes.Perpetual{
			SettlementAsset:     "Ethereum/Ether",
			QuoteName:           "ETH-230929",
			MarginFundingFactor: "0.5",
			InterestRate:        "0.012",
			ClampLowerBound:     "0.2",
			ClampUpperBound:     "0.8",
			DataSourceSpecForSettlementSchedule: &protoTypes.DataSourceSpec{
				Id:        "test-settlement-schedule",
				CreatedAt: time.Now().UnixNano(),
				UpdatedAt: time.Now().UnixNano(),
				Data: protoTypes.NewDataSourceDefinition(
					protoTypes.DataSourceContentTypeOracle,
				).SetOracleConfig(
					&vega.DataSourceDefinitionExternal_Oracle{
						Oracle: &protoTypes.DataSourceSpecConfiguration{
							Signers: []*datav1.Signer{pk.IntoProto()},
							Filters: []*datav1.Filter{
								{
									Key: &datav1.PropertyKey{
										Name: "prices.ETH.value",
										Type: datav1.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*datav1.Condition{},
								},
							},
						},
					},
				),
				Status: protoTypes.DataSourceSpec_STATUS_ACTIVE,
			},
			DataSourceSpecForSettlementData: &protoTypes.DataSourceSpec{
				Id:        "test-settlement-data",
				CreatedAt: time.Now().UnixNano(),
				UpdatedAt: time.Now().UnixNano(),
				Data: protoTypes.NewDataSourceDefinition(
					protoTypes.DataSourceContentTypeOracle,
				).SetOracleConfig(
					&vega.DataSourceDefinitionExternal_Oracle{
						Oracle: &protoTypes.DataSourceSpecConfiguration{
							Signers: []*datav1.Signer{pk.IntoProto()},
							Filters: []*datav1.Filter{
								{
									Key: &datav1.PropertyKey{
										Name: "prices.ETH.value",
										Type: datav1.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*datav1.Condition{},
								},
							},
						},
					},
				),
				Status: protoTypes.DataSourceSpec_STATUS_ACTIVE,
			},
			DataSourceSpecBinding: &protoTypes.DataSourceSpecToPerpetualBinding{
				SettlementDataProperty:     "prices.ETH.value",
				SettlementScheduleProperty: "2023-09-29T00:00:00.000000000Z",
			},
		},
	}
	return mkt
}

func getTestMarket() *protoTypes.Market {
	return &protoTypes.Market{
		Id: "BTC/DEC19",
		TradableInstrument: &protoTypes.TradableInstrument{
			Instrument: &protoTypes.Instrument{
				Id:   "Crypto/BTCUSD/Futures/Dec19",
				Code: "FX:BTCUSD/DEC19",
				Name: "December 2019 BTC vs USD future",
				Metadata: &protoTypes.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
			},
			MarginCalculator: &protoTypes.MarginCalculator{
				ScalingFactors: &protoTypes.ScalingFactors{
					SearchLevel:       1.1,
					InitialMargin:     1.2,
					CollateralRelease: 1.4,
				},
			},
			RiskModel: &protoTypes.TradableInstrument_LogNormalRiskModel{
				LogNormalRiskModel: &protoTypes.LogNormalRiskModel{
					RiskAversionParameter: 0.01,
					Tau:                   1.0 / 365.25 / 24,
					Params: &protoTypes.LogNormalModelParams{
						Mu:    0,
						R:     0.016,
						Sigma: 0.09,
					},
				},
			},
		},
		LiquidityMonitoringParameters: &protoTypes.LiquidityMonitoringParameters{
			TriggeringRatio: "0.3",
		},
	}
}

func getNewProposal() *protoTypes.Proposal {
	return &protoTypes.Proposal{
		Id:        "ETH/DEC23",
		Reference: "TestNewMarket",
		PartyId:   "DEADBEEF01",
		State:     protoTypes.Proposal_STATE_OPEN,
		Timestamp: time.Now().UnixNano(),
		Terms: &protoTypes.ProposalTerms{
			Change: &protoTypes.ProposalTerms_NewMarket{
				NewMarket: &protoTypes.NewMarket{
					Changes: &protoTypes.NewMarketConfiguration{
						Instrument: &protoTypes.InstrumentConfiguration{},
					},
				},
			},
		},
	}
}

func getNewFutureMarketProposal() *protoTypes.Proposal {
	pk := dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)
	proposal := getNewProposal()

	proposal.Terms.Change = &protoTypes.ProposalTerms_NewMarket{
		NewMarket: &protoTypes.NewMarket{
			Changes: &protoTypes.NewMarketConfiguration{
				Instrument: &protoTypes.InstrumentConfiguration{
					Product: &protoTypes.InstrumentConfiguration_Future{
						Future: &protoTypes.FutureProduct{
							SettlementAsset: "Ethereum/Ether",
							QuoteName:       "ETH/DEC23",
							DataSourceSpecForSettlementData: &protoTypes.DataSourceDefinition{
								SourceType: &protoTypes.DataSourceDefinition_External{
									External: &protoTypes.DataSourceDefinitionExternal{
										SourceType: &protoTypes.DataSourceDefinitionExternal_Oracle{
											Oracle: &protoTypes.DataSourceSpecConfiguration{
												Signers: []*datav1.Signer{pk.IntoProto()},
												Filters: []*datav1.Filter{
													{
														Key: &datav1.PropertyKey{
															Name: "prices.ETH.value",
															Type: datav1.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datav1.Condition{},
													},
												},
											},
										},
									},
								},
							},
							DataSourceSpecForTradingTermination: &protoTypes.DataSourceDefinition{
								SourceType: &protoTypes.DataSourceDefinition_Internal{
									Internal: &protoTypes.DataSourceDefinitionInternal{
										SourceType: &protoTypes.DataSourceDefinitionInternal_Time{
											Time: &protoTypes.DataSourceSpecConfigurationTime{
												Conditions: []*datav1.Condition{
													{
														Operator: datav1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
														Value:    "2023-09-29T00:00:00.000000000Z",
													},
												},
											},
										},
									},
								},
							},
							DataSourceSpecBinding: &protoTypes.DataSourceSpecToFutureBinding{
								SettlementDataProperty:     "prices.ETH.value",
								TradingTerminationProperty: "trading.terminated",
							},
						},
					},
				},
			},
		},
	}
	return proposal
}

func getFutureMarketUpdateProposal() *protoTypes.Proposal {
	pk := dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)
	proposal := getNewProposal()
	proposal.Terms.Change = &protoTypes.ProposalTerms_UpdateMarket{
		UpdateMarket: &protoTypes.UpdateMarket{
			MarketId: "ETH/DEC23",
			Changes: &protoTypes.UpdateMarketConfiguration{
				Instrument: &protoTypes.UpdateInstrumentConfiguration{
					Code: "",
					Product: &protoTypes.UpdateInstrumentConfiguration_Future{
						Future: &protoTypes.UpdateFutureProduct{
							QuoteName: "ETH/DEC23",
							DataSourceSpecForSettlementData: &protoTypes.DataSourceDefinition{
								SourceType: &protoTypes.DataSourceDefinition_External{
									External: &protoTypes.DataSourceDefinitionExternal{
										SourceType: &protoTypes.DataSourceDefinitionExternal_Oracle{
											Oracle: &protoTypes.DataSourceSpecConfiguration{
												Signers: []*datav1.Signer{pk.IntoProto()},
												Filters: []*datav1.Filter{
													{
														Key: &datav1.PropertyKey{
															Name: "prices.ETH.value",
															Type: datav1.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datav1.Condition{},
													},
												},
											},
										},
									},
								},
							},
							DataSourceSpecForTradingTermination: &protoTypes.DataSourceDefinition{
								SourceType: &protoTypes.DataSourceDefinition_Internal{
									Internal: &protoTypes.DataSourceDefinitionInternal{
										SourceType: &protoTypes.DataSourceDefinitionInternal_Time{
											Time: &protoTypes.DataSourceSpecConfigurationTime{
												Conditions: []*datav1.Condition{
													{
														Operator: datav1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
														Value:    "2023-09-28T00:00:00.000000000Z",
													},
												},
											},
										},
									},
								},
							},
							DataSourceSpecBinding: &protoTypes.DataSourceSpecToFutureBinding{
								SettlementDataProperty:     "prices.ETH.value",
								TradingTerminationProperty: "trading.terminated",
							},
						},
					},
				},
			},
		},
	}

	return proposal
}

func getNewSpotMarketProposal() *protoTypes.Proposal {
	proposal := getNewProposal()

	proposal.Terms.Change = &protoTypes.ProposalTerms_NewSpotMarket{
		NewSpotMarket: &protoTypes.NewSpotMarket{
			Changes: &protoTypes.NewSpotMarketConfiguration{
				Instrument: &protoTypes.InstrumentConfiguration{
					Product: &protoTypes.InstrumentConfiguration_Spot{
						Spot: &protoTypes.SpotProduct{
							BaseAsset:  "USD",
							QuoteAsset: "ETH",
							Name:       "ETH/USD",
						},
					},
				},
			},
		},
	}
	return proposal
}

func getSpotMarketUpdateProposal() *protoTypes.Proposal {
	proposal := getNewProposal()
	proposal.Terms.Change = &protoTypes.ProposalTerms_UpdateSpotMarket{
		UpdateSpotMarket: &protoTypes.UpdateSpotMarket{
			MarketId: "USD/ETH",
			Changes: &protoTypes.UpdateSpotMarketConfiguration{
				Metadata: []string{"ETH", "USD"},
				PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
					Triggers: []*protoTypes.PriceMonitoringTrigger{
						{
							Horizon:          1,
							Probability:      "0.5",
							AuctionExtension: 0,
						},
					},
				},
				TargetStakeParameters: &protoTypes.TargetStakeParameters{
					TimeWindow:    1,
					ScalingFactor: 1,
				},
				RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_Simple{
					Simple: &protoTypes.SimpleModelParams{
						FactorLong:           1,
						FactorShort:          1,
						MaxMoveUp:            1,
						MinMoveDown:          1,
						ProbabilityOfTrading: 1,
					},
				},
				SlaParams: &protoTypes.LiquiditySLAParameters{
					PriceRange:                  "",
					CommitmentMinTimeFraction:   "0.5",
					PerformanceHysteresisEpochs: 2,
					SlaCompetitionFactor:        "0.75",
				},
			},
		},
	}
	return proposal
}

func getNewPerpetualMarketProposal() *protoTypes.Proposal {
	pk := dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)
	proposal := getNewProposal()

	proposal.Terms.Change = &protoTypes.ProposalTerms_NewMarket{
		NewMarket: &protoTypes.NewMarket{
			Changes: &protoTypes.NewMarketConfiguration{
				Instrument: &protoTypes.InstrumentConfiguration{
					Product: &protoTypes.InstrumentConfiguration_Perpetual{
						Perpetual: &protoTypes.PerpetualProduct{
							SettlementAsset:     "Ethereum/Ether",
							QuoteName:           "ETH-230929",
							MarginFundingFactor: "0.5",
							InterestRate:        "0.0125",
							ClampLowerBound:     "0.2",
							ClampUpperBound:     "0.8",
							DataSourceSpecForSettlementSchedule: &protoTypes.DataSourceDefinition{
								SourceType: &protoTypes.DataSourceDefinition_External{
									External: &protoTypes.DataSourceDefinitionExternal{
										SourceType: &protoTypes.DataSourceDefinitionExternal_Oracle{
											Oracle: &protoTypes.DataSourceSpecConfiguration{
												Signers: []*datav1.Signer{pk.IntoProto()},
												Filters: []*datav1.Filter{
													{
														Key: &datav1.PropertyKey{
															Name: "prices.ETH.value",
															Type: datav1.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datav1.Condition{},
													},
												},
											},
										},
									},
								},
							},
							DataSourceSpecForSettlementData: &protoTypes.DataSourceDefinition{
								SourceType: &protoTypes.DataSourceDefinition_Internal{
									Internal: &protoTypes.DataSourceDefinitionInternal{
										SourceType: &protoTypes.DataSourceDefinitionInternal_Time{
											Time: &protoTypes.DataSourceSpecConfigurationTime{
												Conditions: []*datav1.Condition{
													{
														Operator: datav1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
														Value:    "2023-09-29T00:00:00.000000000Z",
													},
												},
											},
										},
									},
								},
							},
							DataSourceSpecBinding: &protoTypes.DataSourceSpecToPerpetualBinding{
								SettlementDataProperty:     "prices.ETH.value",
								SettlementScheduleProperty: "2023-09-29T00:00:00.000000000Z",
							},
						},
					},
				},
			},
		},
	}
	return proposal
}

func getPerpetualMarketUpdateProposal() *protoTypes.Proposal {
	pk := dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)
	proposal := getNewProposal()

	proposal.Terms.Change = &protoTypes.ProposalTerms_UpdateMarket{
		UpdateMarket: &protoTypes.UpdateMarket{
			Changes: &protoTypes.UpdateMarketConfiguration{
				Instrument: &protoTypes.UpdateInstrumentConfiguration{
					Product: &protoTypes.UpdateInstrumentConfiguration_Perpetual{
						Perpetual: &protoTypes.UpdatePerpetualProduct{
							QuoteName:           "ETH-230929",
							MarginFundingFactor: "0.6",
							InterestRate:        "0.015",
							ClampLowerBound:     "0.1",
							ClampUpperBound:     "0.9",
							DataSourceSpecForSettlementSchedule: &protoTypes.DataSourceDefinition{
								SourceType: &protoTypes.DataSourceDefinition_External{
									External: &protoTypes.DataSourceDefinitionExternal{
										SourceType: &protoTypes.DataSourceDefinitionExternal_Oracle{
											Oracle: &protoTypes.DataSourceSpecConfiguration{
												Signers: []*datav1.Signer{pk.IntoProto()},
												Filters: []*datav1.Filter{
													{
														Key: &datav1.PropertyKey{
															Name: "prices.ETH.value",
															Type: datav1.PropertyKey_TYPE_INTEGER,
														},
														Conditions: []*datav1.Condition{},
													},
												},
											},
										},
									},
								},
							},
							DataSourceSpecForSettlementData: &protoTypes.DataSourceDefinition{
								SourceType: &protoTypes.DataSourceDefinition_Internal{
									Internal: &protoTypes.DataSourceDefinitionInternal{
										SourceType: &protoTypes.DataSourceDefinitionInternal_Time{
											Time: &protoTypes.DataSourceSpecConfigurationTime{
												Conditions: []*datav1.Condition{
													{
														Operator: datav1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
														Value:    "2023-09-29T00:00:00.000000000Z",
													},
												},
											},
										},
									},
								},
							},
							DataSourceSpecBinding: &protoTypes.DataSourceSpecToPerpetualBinding{
								SettlementDataProperty:     "prices.ETH.value",
								SettlementScheduleProperty: "2023-09-29T00:00:00.000000000Z",
							},
						},
					},
				},
			},
		},
	}

	return proposal
}

func TestNewResolverRoot_Proposals(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proposals := map[string]*protoTypes.Proposal{
		"NewFutureMarket":       getNewFutureMarketProposal(),
		"NewSpotMarket":         getNewSpotMarketProposal(),
		"NewPerpetualMarket":    getNewPerpetualMarketProposal(),
		"UpdateFutureMarket":    getFutureMarketUpdateProposal(),
		"UpdateSpotMarket":      getSpotMarketUpdateProposal(),
		"UpdatePerpetualMarket": getPerpetualMarketUpdateProposal(),
	}

	t.Run("GraphQL should support new futures market proposals", func(t *testing.T) {
		id := "NewFutureMarket"
		root.tradingDataClient.EXPECT().GetGovernanceData(gomock.Any(), gomock.Any()).Return(
			&v2.GetGovernanceDataResponse{
				Data: &protoTypes.GovernanceData{
					Proposal: proposals[id],
				},
			}, nil,
		)

		var (
			p         *protoTypes.GovernanceData
			terms     *protoTypes.ProposalTerms
			newMarket *protoTypes.ProposalTerms_NewMarket
			asset     *protoTypes.Asset
			product   *protoTypes.InstrumentConfiguration_Future
			err       error
		)

		p, err = root.Query().Proposal(ctx, &id, nil)

		t.Run("Proposal terms should be for a new market", func(t *testing.T) {
			terms, err = root.Proposal().Terms(ctx, p)
			require.NoError(t, err)
			want := proposals[id].Terms
			assert.Equal(t, want, terms)
			assert.IsType(t, &protoTypes.ProposalTerms_NewMarket{}, terms.Change)
		})

		t.Run("New market should be for a futures market", func(t *testing.T) {
			newMarket = terms.Change.(*protoTypes.ProposalTerms_NewMarket)
			assert.IsType(t, &protoTypes.InstrumentConfiguration_Future{}, newMarket.NewMarket.Changes.Instrument.Product)
		})

		t.Run("The product and asset should be a future", func(t *testing.T) {
			product = newMarket.NewMarket.Changes.Instrument.Product.(*protoTypes.InstrumentConfiguration_Future)
			assert.IsType(t, &protoTypes.FutureProduct{}, product.Future)
		})

		t.Run("The future resolver should retrieve the settlement asset using the data node API", func(t *testing.T) {
			wantAsset := &protoTypes.Asset{
				Id: "TestFuture",
				Details: &protoTypes.AssetDetails{
					Name:   "TestFuture",
					Symbol: "Test",
				},
				Status: protoTypes.Asset_STATUS_PROPOSED,
			}

			root.tradingDataClient.EXPECT().GetAsset(gomock.Any(), gomock.Any()).Return(
				&v2.GetAssetResponse{
					Asset: wantAsset,
				}, nil,
			).Times(1)

			asset, err = root.FutureProduct().SettlementAsset(ctx, product.Future)
			assert.Equal(t, wantAsset, asset)
		})
	})

	t.Run("GraphQL should support new spot market proposals", func(t *testing.T) {
		id := "NewSpotMarket"
		root.tradingDataClient.EXPECT().GetGovernanceData(gomock.Any(), gomock.Any()).Return(
			&v2.GetGovernanceDataResponse{
				Data: &protoTypes.GovernanceData{
					Proposal: proposals[id],
				},
			}, nil,
		)

		var (
			p         *protoTypes.GovernanceData
			terms     *protoTypes.ProposalTerms
			newMarket *protoTypes.ProposalTerms_NewSpotMarket
			asset     *protoTypes.Asset
			product   *protoTypes.InstrumentConfiguration_Spot
			err       error
		)

		p, err = root.Query().Proposal(ctx, &id, nil)

		t.Run("Proposal should be for a new spot market", func(t *testing.T) {
			terms, err = root.Proposal().Terms(ctx, p)
			require.NoError(t, err)
			want := proposals[id].Terms
			assert.Equal(t, want, terms)
			assert.IsType(t, &protoTypes.ProposalTerms_NewSpotMarket{}, terms.Change)
		})

		t.Run("Product should be a spot product", func(t *testing.T) {
			newMarket = terms.Change.(*protoTypes.ProposalTerms_NewSpotMarket)
			assert.IsType(t, &protoTypes.InstrumentConfiguration_Spot{}, newMarket.NewSpotMarket.Changes.Instrument.Product)
		})

		t.Run("Spot product resolver should retrieve the asset data using the data node API", func(t *testing.T) {
			wantQuote := &protoTypes.Asset{
				Id: "ETH",
				Details: &protoTypes.AssetDetails{
					Name:   "Ethereum/Ether",
					Symbol: "ETH",
				},
				Status: protoTypes.Asset_STATUS_ENABLED,
			}

			wantBase := &protoTypes.Asset{
				Id: "USD",
				Details: &protoTypes.AssetDetails{
					Name:   "US Dollar",
					Symbol: "USD",
				},
				Status: protoTypes.Asset_STATUS_ENABLED,
			}

			callCount := 0
			root.tradingDataClient.EXPECT().GetAsset(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *v2.GetAssetRequest, _ ...grpc.CallOption) (*v2.GetAssetResponse, error) {
				defer func() {
					callCount++
				}()

				if callCount%2 == 1 {
					return &v2.GetAssetResponse{
						Asset: wantBase,
					}, nil
				}

				return &v2.GetAssetResponse{
					Asset: wantQuote,
				}, nil
			}).Times(2)

			product = newMarket.NewSpotMarket.Changes.Instrument.Product.(*protoTypes.InstrumentConfiguration_Spot)
			assert.IsType(t, &protoTypes.SpotProduct{}, product.Spot)
			asset, err = root.SpotProduct().QuoteAsset(ctx, product.Spot)
			assert.Equal(t, wantQuote, asset)
			asset, err = root.SpotProduct().BaseAsset(ctx, product.Spot)
			assert.Equal(t, wantBase, asset)
		})
	})

	t.Run("GraphQL should support new perpetual market proposals", func(t *testing.T) {
		id := "NewPerpetualMarket"
		root.tradingDataClient.EXPECT().GetGovernanceData(gomock.Any(), gomock.Any()).Return(
			&v2.GetGovernanceDataResponse{
				Data: &protoTypes.GovernanceData{
					Proposal: proposals[id],
				},
			}, nil,
		)

		var (
			p         *protoTypes.GovernanceData
			terms     *protoTypes.ProposalTerms
			newMarket *protoTypes.ProposalTerms_NewMarket
			asset     *protoTypes.Asset
			product   *protoTypes.InstrumentConfiguration_Perpetual
			err       error
		)

		p, err = root.Query().Proposal(ctx, &id, nil)

		t.Run("Proposal terms should be for a new market", func(t *testing.T) {
			terms, err = root.Proposal().Terms(ctx, p)
			require.NoError(t, err)
			want := proposals[id].Terms
			assert.Equal(t, want, terms)
			assert.IsType(t, &protoTypes.ProposalTerms_NewMarket{}, terms.Change)
		})

		t.Run("New market should be for a perpetual market", func(t *testing.T) {
			newMarket = terms.Change.(*protoTypes.ProposalTerms_NewMarket)
			assert.IsType(t, &protoTypes.InstrumentConfiguration_Perpetual{}, newMarket.NewMarket.Changes.Instrument.Product)
		})

		t.Run("The product and asset should be a perpetual", func(t *testing.T) {
			product = newMarket.NewMarket.Changes.Instrument.Product.(*protoTypes.InstrumentConfiguration_Perpetual)
			assert.IsType(t, &protoTypes.PerpetualProduct{}, product.Perpetual)
		})

		t.Run("The perpetual product resolver should retrieve the settlement asset using the data node API", func(t *testing.T) {
			wantAsset := &protoTypes.Asset{
				Id: "TestPerpetual",
				Details: &protoTypes.AssetDetails{
					Name:   "TestPerpetual",
					Symbol: "Test",
				},
				Status: protoTypes.Asset_STATUS_PROPOSED,
			}

			root.tradingDataClient.EXPECT().GetAsset(gomock.Any(), gomock.Any()).Return(
				&v2.GetAssetResponse{
					Asset: wantAsset,
				}, nil,
			).Times(1)

			asset, err = root.PerpetualProduct().SettlementAsset(ctx, product.Perpetual)
			assert.Equal(t, wantAsset, asset)
		})
	})

	t.Run("GraohQL should support update futures market proposals", func(t *testing.T) {
		id := "UpdateFutureMarket"
		root.tradingDataClient.EXPECT().GetGovernanceData(gomock.Any(), gomock.Any()).Return(
			&v2.GetGovernanceDataResponse{
				Data: &protoTypes.GovernanceData{
					Proposal: proposals[id],
				},
			}, nil,
		)

		var (
			p         *protoTypes.GovernanceData
			terms     *protoTypes.ProposalTerms
			newMarket *protoTypes.ProposalTerms_UpdateMarket
			product   *protoTypes.UpdateInstrumentConfiguration_Future
			err       error
		)

		p, err = root.Query().Proposal(ctx, &id, nil)

		t.Run("Proposal terms should be to update market", func(t *testing.T) {
			terms, err = root.Proposal().Terms(ctx, p)
			require.NoError(t, err)
			want := proposals[id].Terms
			assert.Equal(t, want, terms)
			assert.IsType(t, &protoTypes.ProposalTerms_UpdateMarket{}, terms.Change)
		})

		t.Run("Update market should be for a futures market", func(t *testing.T) {
			newMarket = terms.Change.(*protoTypes.ProposalTerms_UpdateMarket)
			assert.IsType(t, &protoTypes.UpdateInstrumentConfiguration_Future{}, newMarket.UpdateMarket.Changes.Instrument.Product)
		})

		t.Run("The product and asset should be a future", func(t *testing.T) {
			pk := dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)
			product = newMarket.UpdateMarket.Changes.Instrument.Product.(*protoTypes.UpdateInstrumentConfiguration_Future)
			assert.IsType(t, &protoTypes.UpdateFutureProduct{}, product.Future)
			want := &protoTypes.UpdateFutureProduct{
				QuoteName: "ETH/DEC23",
				DataSourceSpecForSettlementData: &protoTypes.DataSourceDefinition{
					SourceType: &protoTypes.DataSourceDefinition_External{
						External: &protoTypes.DataSourceDefinitionExternal{
							SourceType: &protoTypes.DataSourceDefinitionExternal_Oracle{
								Oracle: &protoTypes.DataSourceSpecConfiguration{
									Signers: []*datav1.Signer{pk.IntoProto()},
									Filters: []*datav1.Filter{
										{
											Key: &datav1.PropertyKey{
												Name: "prices.ETH.value",
												Type: datav1.PropertyKey_TYPE_INTEGER,
											},
											Conditions: []*datav1.Condition{},
										},
									},
								},
							},
						},
					},
				},
				DataSourceSpecForTradingTermination: &protoTypes.DataSourceDefinition{
					SourceType: &protoTypes.DataSourceDefinition_Internal{
						Internal: &protoTypes.DataSourceDefinitionInternal{
							SourceType: &protoTypes.DataSourceDefinitionInternal_Time{
								Time: &protoTypes.DataSourceSpecConfigurationTime{
									Conditions: []*datav1.Condition{
										{
											Operator: datav1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
											Value:    "2023-09-28T00:00:00.000000000Z",
										},
									},
								},
							},
						},
					},
				},
				DataSourceSpecBinding: &protoTypes.DataSourceSpecToFutureBinding{
					SettlementDataProperty:     "prices.ETH.value",
					TradingTerminationProperty: "trading.terminated",
				},
			}
			assert.Equal(t, want, product.Future)
		})
	})

	t.Run("GraphQL should support update spot market proposals", func(t *testing.T) {
		id := "UpdateSpotMarket"
		root.tradingDataClient.EXPECT().GetGovernanceData(gomock.Any(), gomock.Any()).Return(
			&v2.GetGovernanceDataResponse{
				Data: &protoTypes.GovernanceData{
					Proposal: proposals[id],
				},
			}, nil,
		)

		var (
			p         *protoTypes.GovernanceData
			terms     *protoTypes.ProposalTerms
			newMarket *protoTypes.ProposalTerms_UpdateSpotMarket
			err       error
		)

		p, err = root.Query().Proposal(ctx, &id, nil)

		t.Run("Proposal should be to update a spot market", func(t *testing.T) {
			terms, err = root.Proposal().Terms(ctx, p)
			require.NoError(t, err)
			want := proposals[id].Terms
			assert.Equal(t, want, terms)
			assert.IsType(t, &protoTypes.ProposalTerms_UpdateSpotMarket{}, terms.Change)
		})

		t.Run("Product should be a spot product", func(t *testing.T) {
			newMarket = terms.Change.(*protoTypes.ProposalTerms_UpdateSpotMarket)
			assert.IsType(t, &protoTypes.UpdateSpotMarketConfiguration{}, newMarket.UpdateSpotMarket.Changes)
			want := &protoTypes.UpdateSpotMarketConfiguration{
				Metadata: []string{"ETH", "USD"},
				PriceMonitoringParameters: &protoTypes.PriceMonitoringParameters{
					Triggers: []*protoTypes.PriceMonitoringTrigger{
						{
							Horizon:          1,
							Probability:      "0.5",
							AuctionExtension: 0,
						},
					},
				},
				TargetStakeParameters: &protoTypes.TargetStakeParameters{
					TimeWindow:    1,
					ScalingFactor: 1,
				},
				RiskParameters: &protoTypes.UpdateSpotMarketConfiguration_Simple{
					Simple: &protoTypes.SimpleModelParams{
						FactorLong:           1,
						FactorShort:          1,
						MaxMoveUp:            1,
						MinMoveDown:          1,
						ProbabilityOfTrading: 1,
					},
				},
				SlaParams: &protoTypes.LiquiditySLAParameters{
					PriceRange:                  "",
					CommitmentMinTimeFraction:   "0.5",
					PerformanceHysteresisEpochs: 2,
					SlaCompetitionFactor:        "0.75",
				},
			}
			assert.Equal(t, want, newMarket.UpdateSpotMarket.Changes)
		})
	})

	t.Run("GraphQL should support update perpetual market proposals", func(t *testing.T) {
		id := "UpdatePerpetualMarket"
		root.tradingDataClient.EXPECT().GetGovernanceData(gomock.Any(), gomock.Any()).Return(
			&v2.GetGovernanceDataResponse{
				Data: &protoTypes.GovernanceData{
					Proposal: proposals[id],
				},
			}, nil,
		)

		var (
			p         *protoTypes.GovernanceData
			terms     *protoTypes.ProposalTerms
			newMarket *protoTypes.ProposalTerms_UpdateMarket
			product   *protoTypes.UpdateInstrumentConfiguration_Perpetual
			err       error
		)

		p, err = root.Query().Proposal(ctx, &id, nil)

		t.Run("Proposal terms should be to update market", func(t *testing.T) {
			// Test the proposal resolver to make sure the terms and underlying changes are correct
			terms, err = root.Proposal().Terms(ctx, p)
			require.NoError(t, err)
			want := proposals[id].Terms
			assert.Equal(t, want, terms)
			assert.IsType(t, &protoTypes.ProposalTerms_UpdateMarket{}, terms.Change)
		})

		t.Run("Update market should be for a perpetual market", func(t *testing.T) {
			newMarket = terms.Change.(*protoTypes.ProposalTerms_UpdateMarket)
			assert.IsType(t, &protoTypes.UpdateInstrumentConfiguration_Perpetual{}, newMarket.UpdateMarket.Changes.Instrument.Product)
		})

		t.Run("The product and asset should be a future", func(t *testing.T) {
			pk := dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)
			product = newMarket.UpdateMarket.Changes.Instrument.Product.(*protoTypes.UpdateInstrumentConfiguration_Perpetual)
			assert.IsType(t, &protoTypes.UpdatePerpetualProduct{}, product.Perpetual)
			want := &protoTypes.UpdatePerpetualProduct{
				QuoteName:           "ETH-230929",
				MarginFundingFactor: "0.6",
				InterestRate:        "0.015",
				ClampLowerBound:     "0.1",
				ClampUpperBound:     "0.9",
				DataSourceSpecForSettlementSchedule: &protoTypes.DataSourceDefinition{
					SourceType: &protoTypes.DataSourceDefinition_External{
						External: &protoTypes.DataSourceDefinitionExternal{
							SourceType: &protoTypes.DataSourceDefinitionExternal_Oracle{
								Oracle: &protoTypes.DataSourceSpecConfiguration{
									Signers: []*datav1.Signer{pk.IntoProto()},
									Filters: []*datav1.Filter{
										{
											Key: &datav1.PropertyKey{
												Name: "prices.ETH.value",
												Type: datav1.PropertyKey_TYPE_INTEGER,
											},
											Conditions: []*datav1.Condition{},
										},
									},
								},
							},
						},
					},
				},
				DataSourceSpecForSettlementData: &protoTypes.DataSourceDefinition{
					SourceType: &protoTypes.DataSourceDefinition_Internal{
						Internal: &protoTypes.DataSourceDefinitionInternal{
							SourceType: &protoTypes.DataSourceDefinitionInternal_Time{
								Time: &protoTypes.DataSourceSpecConfigurationTime{
									Conditions: []*datav1.Condition{
										{
											Operator: datav1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
											Value:    "2023-09-29T00:00:00.000000000Z",
										},
									},
								},
							},
						},
					},
				},
				DataSourceSpecBinding: &protoTypes.DataSourceSpecToPerpetualBinding{
					SettlementDataProperty:     "prices.ETH.value",
					SettlementScheduleProperty: "2023-09-29T00:00:00.000000000Z",
				},
			}
			assert.Equal(t, want, product.Perpetual)
		})
	})
}

func TestNewResolverRoot_SpotResolver(t *testing.T) {
	ctx := context.Background()
	root := buildTestResolverRoot(t)
	defer root.Finish()

	spotMarket := getTestSpotMarket()
	root.tradingDataClient.EXPECT().GetMarket(gomock.Any(), gomock.Any()).Return(&v2.GetMarketResponse{Market: spotMarket}, nil)
	wantAsset1 := &protoTypes.Asset{
		Id:      "Asset1",
		Details: nil,
		Status:  0,
	}

	wantAsset2 := &protoTypes.Asset{
		Id:      "Asset2",
		Details: nil,
		Status:  0,
	}
	call := 0
	root.tradingDataClient.EXPECT().GetAsset(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *v2.GetAssetRequest, opts ...grpc.CallOption) (*v2.GetAssetResponse, error) {
			defer func() { call++ }()
			if call%2 == 0 {
				return &v2.GetAssetResponse{Asset: wantAsset1}, nil
			}

			return &v2.GetAssetResponse{Asset: wantAsset2}, nil
		},
	).Times(2)

	mkt, err := root.tradingDataClient.GetMarket(ctx, &v2.GetMarketRequest{
		MarketId: spotMarket.Id,
	})
	require.NoError(t, err)

	asset, err := root.Spot().BaseAsset(ctx, mkt.GetMarket().TradableInstrument.Instrument.GetSpot())
	require.NoError(t, err)
	assert.Equal(t, wantAsset1, asset)

	asset, err = root.Spot().QuoteAsset(ctx, mkt.GetMarket().TradableInstrument.Instrument.GetSpot())
	require.NoError(t, err)
	assert.Equal(t, wantAsset2, asset)
}

func TestNewResolverRoot_PerpetualResolver(t *testing.T) {
	ctx := context.Background()
	root := buildTestResolverRoot(t)
	defer root.Finish()

	perpsMarket := getTestPerpetualMarket()
	want := perpsMarket.TradableInstrument.Instrument.GetPerpetual()

	root.tradingDataClient.EXPECT().GetMarket(gomock.Any(), gomock.Any()).Return(&v2.GetMarketResponse{Market: perpsMarket}, nil)
	wantAsset := &protoTypes.Asset{
		Id: "Asset1",
	}
	root.tradingDataClient.EXPECT().GetAsset(gomock.Any(), gomock.Any(), gomock.Any()).Return(&v2.GetAssetResponse{Asset: wantAsset}, nil)

	mkt, err := root.tradingDataClient.GetMarket(ctx, &v2.GetMarketRequest{
		MarketId: perpsMarket.Id,
	})
	require.NoError(t, err)
	perps := mkt.GetMarket().TradableInstrument.Instrument.GetPerpetual()
	asset, err := root.Perpetual().SettlementAsset(ctx, perps)
	require.NoError(t, err)
	assert.Equal(t, wantAsset, asset)

	gotSchedule, err := root.Perpetual().DataSourceSpecForSettlementSchedule(ctx, perps)
	require.NoError(t, err)
	assert.Equal(t, want.DataSourceSpecForSettlementSchedule.Id, gotSchedule.ID)
	assert.Equal(t, want.DataSourceSpecForSettlementSchedule.CreatedAt, gotSchedule.CreatedAt)
	assert.NotNil(t, gotSchedule.UpdatedAt)
	assert.Equal(t, want.DataSourceSpecForSettlementSchedule.UpdatedAt, *gotSchedule.UpdatedAt)
	assert.Equal(t, want.DataSourceSpecForSettlementSchedule.Data, gotSchedule.Data)
	assert.Equal(t, want.DataSourceSpecForSettlementSchedule.Status.String(), gotSchedule.Status.String())

	gotData, err := root.Perpetual().DataSourceSpecForSettlementData(ctx, perps)
	require.NoError(t, err)
	assert.Equal(t, want.DataSourceSpecForSettlementData.Id, gotData.ID)
	assert.Equal(t, want.DataSourceSpecForSettlementData.CreatedAt, gotData.CreatedAt)
	assert.NotNil(t, gotData.UpdatedAt)
	assert.Equal(t, want.DataSourceSpecForSettlementData.UpdatedAt, *gotData.UpdatedAt)
	assert.Equal(t, want.DataSourceSpecForSettlementData.Data, gotData.Data)
	assert.Equal(t, want.DataSourceSpecForSettlementData.Status.String(), gotData.Status.String())

	wantBinding, err := root.Perpetual().DataSourceSpecBinding(ctx, perps)
	require.NoError(t, err)
	assert.Equal(t, want.DataSourceSpecBinding.SettlementScheduleProperty, wantBinding.SettlementScheduleProperty)
	assert.Equal(t, want.DataSourceSpecBinding.SettlementDataProperty, wantBinding.SettlementDataProperty)
}

func TestNewResolverRoot_Resolver(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
	ctx := context.Background()

	marketNotExistsErr := errors.New("market does not exist")
	markets := map[string]*protoTypes.Market{
		"BTC/DEC19":  getTestFutureMarket(protoTypes.DataSourceContentTypeInternalTimeTermination),
		"ETH/USD18":  nil,
		"ETH/USD":    getTestSpotMarket(),
		"ETH-230929": getTestPerpetualMarket(),
	}

	root.tradingDataClient.EXPECT().GetAsset(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&v2.GetAssetResponse{Asset: &protoTypes.Asset{}}, nil)

	root.tradingDataClient.EXPECT().GetMarket(gomock.Any(), gomock.Any()).Times(len(markets)).DoAndReturn(func(_ context.Context, req *v2.GetMarketRequest, _ ...grpc.CallOption) (*v2.GetMarketResponse, error) {
		m, ok := markets[req.MarketId]
		assert.True(t, ok)
		if m == nil {
			return nil, marketNotExistsErr
		}
		return &v2.GetMarketResponse{Market: m}, nil
	})

	name := "BTC/DEC19"
	vMarkets, err := root.Query().MarketsConnection(ctx, &name, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, vMarkets)
	assert.Len(t, vMarkets.Edges, 1)

	name = "ETH/USD18"
	vMarkets, err = root.Query().MarketsConnection(ctx, &name, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, vMarkets)

	name = "ETH/USD"
	vMarkets, err = root.Query().MarketsConnection(ctx, &name, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, vMarkets)
	assert.Len(t, vMarkets.Edges, 1)

	name = "ETH-230929"
	vMarkets, err = root.Query().MarketsConnection(ctx, &name, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, vMarkets)
	assert.Len(t, vMarkets.Edges, 1)

	name = "barney"
	root.tradingDataClient.EXPECT().ListParties(gomock.Any(), gomock.Any()).Times(1).Return(&v2.ListPartiesResponse{
		Parties: &v2.PartyConnection{
			Edges: []*v2.PartyEdge{
				{
					Node:   &protoTypes.Party{Id: name},
					Cursor: name,
				},
			},
		},
	}, nil)
	vParties, err := root.Query().PartiesConnection(ctx, &name, nil)
	assert.Nil(t, err)
	assert.NotNil(t, vParties)
	assert.Len(t, vParties.Edges, 1)

	root.tradingDataClient.EXPECT().ListParties(gomock.Any(), gomock.Any()).Times(1).Return(&v2.ListPartiesResponse{Parties: &v2.PartyConnection{
		Edges:    nil,
		PageInfo: &v2.PageInfo{},
	}}, nil)
	vParties, err = root.Query().PartiesConnection(ctx, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, vParties)
	assert.Equal(t, len(vParties.Edges), 0)
}

func TestNewResolverRoot_MarketResolver(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
	ctx := context.Background()

	marketID := "BTC/DEC19"
	market := &protoTypes.Market{
		Id: marketID,
	}

	root.tradingDataClient.EXPECT().ListOrders(gomock.Any(), gomock.Any()).Times(1).Return(&v2.ListOrdersResponse{Orders: &v2.OrderConnection{
		Edges: []*v2.OrderEdge{
			{
				Node: &protoTypes.Order{
					Id:        "order-id-1",
					Price:     "1000",
					CreatedAt: 1,
				},
				Cursor: "1",
			},
			{
				Node: &protoTypes.Order{
					Id:        "order-id-2",
					Price:     "2000",
					CreatedAt: 2,
				},
				Cursor: "2",
			},
		},
	}}, nil)

	marketResolver := root.Market()
	assert.NotNil(t, marketResolver)

	orders, err := marketResolver.OrdersConnection(ctx, market, nil, nil)
	assert.NotNil(t, orders)
	assert.Nil(t, err)
	assert.Len(t, orders.Edges, 2)
}

func TestRewardsResolver(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
	ctx := context.Background()
	partyResolver := root.Party()
	root.tradingDataClient.EXPECT().ListRewardSummaries(gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("some error"))
	assetID := "asset"
	r, e := partyResolver.RewardSummaries(ctx, &protoTypes.Party{Id: "some"}, &assetID)
	require.Nil(t, r)
	require.NotNil(t, e)
}

func TestNewResolverRoot_EpochResolver(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
	ctx := context.Background()

	now := time.Now()
	epochResp := &v2.GetEpochResponse{Epoch: &protoTypes.Epoch{
		Seq: 10,
		Timestamps: &protoTypes.EpochTimestamps{
			StartTime:  now.Unix(),
			ExpiryTime: now.Add(time.Hour).Unix(),
			EndTime:    now.Add(time.Hour * 2).Unix(),
			FirstBlock: 100,
			LastBlock:  110,
		},
	}}
	root.tradingDataClient.EXPECT().GetEpoch(gomock.Any(), gomock.Any()).Times(1).Return(epochResp, nil)

	epochResolver := root.Epoch()
	assert.NotNil(t, epochResolver)

	block := uint64(100)
	got, err := root.tradingDataClient.GetEpoch(ctx, &v2.GetEpochRequest{Block: &block})
	assert.Nil(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, got.Epoch, epochResp.Epoch)

	id, err := epochResolver.ID(ctx, got.Epoch)
	assert.Nil(t, err)
	assert.Equal(t, id, fmt.Sprint(got.Epoch.Seq))
}

//nolint:interfacebloat
type resolverRoot interface {
	Query() gql.QueryResolver
	Candle() gql.CandleResolver
	MarketDepth() gql.MarketDepthResolver
	MarketDepthUpdate() gql.MarketDepthUpdateResolver
	PriceLevel() gql.PriceLevelResolver
	Market() gql.MarketResolver
	Order() gql.OrderResolver
	Trade() gql.TradeResolver
	Position() gql.PositionResolver
	Party() gql.PartyResolver
	Subscription() gql.SubscriptionResolver
	Epoch() gql.EpochResolver
	Future() gql.FutureResolver
	FutureProduct() gql.FutureProductResolver
	Perpetual() gql.PerpetualResolver
	PerpetualProduct() gql.PerpetualProductResolver
	Proposal() gql.ProposalResolver
	Spot() gql.SpotResolver
	SpotProduct() gql.SpotProductResolver
}

type testResolver struct {
	resolverRoot
	log               *logging.Logger
	ctrl              *gomock.Controller
	coreProxyClient   *mocks.MockCoreProxyServiceClient
	tradingDataClient *mocks.MockTradingDataServiceClientV2
}

func buildTestResolverRoot(t *testing.T) *testResolver {
	t.Helper()
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	conf := gateway.NewDefaultConfig()
	coreProxyClient := mocks.NewMockCoreProxyServiceClient(ctrl)
	tradingDataClientV2 := mocks.NewMockTradingDataServiceClientV2(ctrl)
	resolver := gql.NewResolverRoot(
		log,
		conf,
		coreProxyClient,
		tradingDataClientV2,
	)
	return &testResolver{
		resolverRoot:      resolver,
		log:               log,
		ctrl:              ctrl,
		coreProxyClient:   coreProxyClient,
		tradingDataClient: tradingDataClientV2,
	}
}

func (t *testResolver) Finish() {
	_ = t.log.Sync()
	t.ctrl.Finish()
}
