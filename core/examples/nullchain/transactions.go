// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package nullchain

import (
	"encoding/json"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/examples/nullchain/config"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	oraclesv1 "code.vegaprotocol.io/vega/protos/vega/oracles/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
)

func MarketProposalTxn(now time.Time, oraclePubkey string) (*walletpb.SubmitTransactionRequest, string) {
	buys := []*vega.LiquidityOrder{
		{Reference: vega.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: "1600", Proportion: 25},
	}
	sells := []*vega.LiquidityOrder{
		{Reference: vega.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Offset: "1600", Proportion: 25},
	}

	reference := "ref-" + vgrand.RandomStr(10)
	asset := config.NormalAsset

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
										OracleSpecForSettlementPrice: &oraclesv1.OracleSpecConfiguration{
											PubKeys: []string{oraclePubkey},
											Filters: []*oraclesv1.Filter{
												{
													Key: &oraclesv1.PropertyKey{
														Name: "prices." + asset + ".value",
														Type: oraclesv1.PropertyKey_TYPE_INTEGER,
													},
													Conditions: []*oraclesv1.Condition{},
												},
											},
										},
										OracleSpecForTradingTermination: &oraclesv1.OracleSpecConfiguration{
											PubKeys: []string{oraclePubkey},
											Filters: []*oraclesv1.Filter{
												{
													Key: &oraclesv1.PropertyKey{
														Name: "trading.termination",
														Type: oraclesv1.PropertyKey_TYPE_BOOLEAN,
													},
													Conditions: []*oraclesv1.Condition{},
												},
											},
										},
										OracleSpecBinding: &vega.OracleSpecToFutureBinding{
											SettlementPriceProperty:    "prices." + asset + ".value",
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
						},
						LiquidityCommitment: &vega.NewMarketCommitment{
							Fee:              "0.01",
							CommitmentAmount: "50000000",
							Buys:             buys,
							Sells:            sells,
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
