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

package nullchain

import (
	"encoding/json"
	"strconv"
	"time"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/examples/nullchain/config"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	datav1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
)

func MarketProposalTxn(now time.Time, oraclePubkey string) (*walletpb.SubmitTransactionRequest, string) {
	reference := "ref-" + vgrand.RandomStr(10)
	asset := config.NormalAsset

	pubKey := dstypes.CreateSignerFromString(oraclePubkey, dstypes.SignerTypePubKey)
	cmd := &walletpb.SubmitTransactionRequest_ProposalSubmission{
		ProposalSubmission: &v1.ProposalSubmission{
			Reference: reference,
			Terms: &vega.ProposalTerms{
				ValidationTimestamp: now.Add(2 * time.Second).Unix(),
				ClosingTimestamp:    now.Add(10 * time.Second).Unix(),
				EnactmentTimestamp:  now.Add(15 * time.Second).Unix(),
				Change: &vega.ProposalTerms_NewMarket{
					NewMarket: &vega.NewMarket{
						Changes: &vega.NewMarketConfiguration{
							Instrument: &vega.InstrumentConfiguration{
								Code: "CRYPTO:BTCUSD/NOV21",
								Name: "NOV 2021 BTC vs USD future",
								Product: &vega.InstrumentConfiguration_Future{
									Future: &vega.FutureProduct{
										SettlementAsset: asset,
										QuoteName:       "BTCUSD",
										DataSourceSpecForSettlementData: &vega.DataSourceDefinition{
											SourceType: &vega.DataSourceDefinition_External{
												External: &vega.DataSourceDefinitionExternal{
													SourceType: &vega.DataSourceDefinitionExternal_Oracle{
														Oracle: &vega.DataSourceSpecConfiguration{
															Signers: []*datav1.Signer{pubKey.IntoProto()},
															Filters: []*datav1.Filter{
																{
																	Key: &datav1.PropertyKey{
																		Name: "prices." + asset + ".value",
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
										DataSourceSpecForTradingTermination: &vega.DataSourceDefinition{
											SourceType: &vega.DataSourceDefinition_External{
												External: &vega.DataSourceDefinitionExternal{
													SourceType: &vega.DataSourceDefinitionExternal_Oracle{
														Oracle: &vega.DataSourceSpecConfiguration{
															Signers: []*datav1.Signer{pubKey.IntoProto()},
															Filters: []*datav1.Filter{
																{
																	Key: &datav1.PropertyKey{
																		Name: "trading.termination",
																		Type: datav1.PropertyKey_TYPE_BOOLEAN,
																	},
																	Conditions: []*datav1.Condition{},
																},
															},
														},
													},
												},
											},
										},
										DataSourceSpecBinding: &vega.DataSourceSpecToFutureBinding{
											SettlementDataProperty:     "prices." + asset + ".value",
											TradingTerminationProperty: "trading.termination",
										},
									},
								},
							},
							DecimalPlaces: 5,
							Metadata:      []string{"base:BTC", "quote:USD", "class:fx/crypto", "monthly", "sector:crypto"},
							RiskParameters: &vega.NewMarketConfiguration_Simple{
								Simple: &vega.SimpleModelParams{
									FactorLong:           0.15,
									FactorShort:          0.25,
									MaxMoveUp:            10,
									MinMoveDown:          -5,
									ProbabilityOfTrading: 0.1,
								},
							},
							LiquiditySlaParameters: &vega.LiquiditySLAParameters{
								PriceRange:                  "0.95",
								CommitmentMinTimeFraction:   "0.5",
								PerformanceHysteresisEpochs: 4,
								SlaCompetitionFactor:        "0.5",
							},
							LinearSlippageFactor: "0.1",
						},
					},
				},
			},
		},
	}

	return &walletpb.SubmitTransactionRequest{
		Command: cmd,
	}, reference
}

func VoteTxn(proposalID string, vote vega.Vote_Value) *walletpb.SubmitTransactionRequest {
	return &walletpb.SubmitTransactionRequest{
		Command: &walletpb.SubmitTransactionRequest_VoteSubmission{
			VoteSubmission: &v1.VoteSubmission{
				ProposalId: proposalID,
				Value:      vote,
			},
		},
	}
}

func OrderTxn(
	marketId string,
	price, size uint64,
	side vega.Side,
	orderT vega.Order_Type,
	expiresAt time.Time,
) *walletpb.SubmitTransactionRequest {
	cmd := &walletpb.SubmitTransactionRequest_OrderSubmission{
		OrderSubmission: &v1.OrderSubmission{
			MarketId:    marketId,
			Price:       strconv.FormatUint(price, 10),
			Size:        size,
			Side:        side,
			Type:        orderT,
			TimeInForce: vega.Order_TIME_IN_FORCE_GTT,
			ExpiresAt:   expiresAt.UnixNano(),
		},
	}

	return &walletpb.SubmitTransactionRequest{
		Command: cmd,
	}
}

func OracleTxn(key, value string) *walletpb.SubmitTransactionRequest {
	data := map[string]string{
		key: value,
	}

	b, _ := json.Marshal(data)

	cmd := &walletpb.SubmitTransactionRequest_OracleDataSubmission{
		OracleDataSubmission: &v1.OracleDataSubmission{
			Source:  v1.OracleDataSubmission_ORACLE_SOURCE_JSON,
			Payload: b,
		},
	}

	return &walletpb.SubmitTransactionRequest{
		Command: cmd,
	}
}
