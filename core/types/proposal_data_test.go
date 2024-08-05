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
package types_test

import (
	"fmt"
	"log"
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func unmarshalGovernanceEnacted(t *testing.T, jsonStr string) (*snapshotpb.GovernanceEnacted, error) {
	t.Helper()
	pb := snapshotpb.Payload{}
	unmarshaler := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}
	err := unmarshaler.Unmarshal([]byte(jsonStr), &pb)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to proto: %w", err)
	}
	return pb.GetGovernanceEnacted(), nil
}

func TestMarshalling(t *testing.T) {
	jsonStr := `{
		"governanceEnacted": {
			"proposals": [
				{
					"proposal": {
						"id": "b844aacfe0c6a5db17c1a65164a6c4418f9cc4c5d0e29eed28811487befd296b",
						"reference": "SyW1TPPG2tGWkWV8Msr4b7W2l67b152zO7DuiMwH",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1688142755690838874",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689767313",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ETHDAI Monthly (Jul 2023)",
										"code": "ETHDAI.MF21",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "DAI",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.ETH.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.ETH.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.ETH.value",
												"tradingTerminationProperty": "termination.ETH.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:4BC6D2154BE74E1F",
										"base:ETH",
										"quote:DAI",
										"class:fx/crypto",
										"monthly",
										"sector:defi",
										"auto:ethdai"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.01,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.3
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New ETHDAI market",
							"title": "New ETHDAI market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "b66cd4be223dfd900a4750bb5175e17d8f678996877d262be4c749a99e22a970",
						"reference": "dE6VwMjuVW9SV6Mj5bp2HtJ5Tw3FMHjYHezLH7Jk",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1688142755690838874",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689767313",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Tesla Quarterly (Sep 2023)",
										"code": "TSLA.QM21",
										"future": {
											"settlementAsset": "177e8f6c25a955bd18475084b99b2b1d37f28f3dec393fab7755a7e69c3d8c3b",
											"quoteName": "EURO",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.TSLA.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.TSLA.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.TSLA.value",
												"tradingTerminationProperty": "termination.TSLA.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:5A86B190C384997F",
										"quote:EURO",
										"ticker:TSLA",
										"class:equities/single-stock-futures",
										"sector:tech",
										"listing_venue:NASDAQ",
										"country:US",
										"auto:tsla"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.01,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.8
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New EURO market",
							"title": "New EURO market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "24df2940a2617b78b4e9f64c6508e4d5d509151a6321a3244a11e1d9859e7cb1",
						"reference": "Z0BxjyvsX7nOeItoHc0uYZ4igGXXPDBFUW08Ogxf",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1688142755690838874",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689767313",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Apple Monthly (Jul 2023)",
										"code": "AAPL.MF21",
										"future": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.AAPL.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.AAPL.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.AAPL.value",
												"tradingTerminationProperty": "termination.AAPL.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:4899E01009F1A721",
										"quote:USD",
										"ticker:AAPL",
										"class:equities/single-stock-futures",
										"sector:tech",
										"listing_venue:NASDAQ",
										"country:US",
										"auto:aapl"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.1,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.3
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New USD market",
							"title": "New USD market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "d780eeb28a3a6e3a047019ca68e510dbe53f57b13eb33c9385221d128e218fce",
						"reference": "j4cdF8jMYstOfpFUKlj607A34ZdsDWOhFLgMqs0w",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1688142755690838874",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689767313",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "UNIDAI Monthly (Jul 2023)",
										"code": "UNIDAI.MF21",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "DAI",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.UNI.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.UNI.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.UNI.value",
												"tradingTerminationProperty": "termination.UNI.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:3C58ED2A4A6C5D7E",
										"base:UNI",
										"quote:DAI",
										"class:fx/crypto",
										"monthly",
										"sector:defi",
										"auto:unidai"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.01,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.5
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New DAI market",
							"title": "New DAI market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "38a4aa08cc0ff0da92ab7d0951de999b0ee0f81029f452b8cf77fb4c300dbd41",
						"reference": "A3rN5PTedEe7ydRU2uoayABRjoOgC9G2IusMTnEo",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1688142755690838874",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689767313",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ETHBTC Quarterly (Sep 2023)",
										"code": "ETHBTC.QM21",
										"future": {
											"settlementAsset": "cee709223217281d7893b650850ae8ee8a18b7539b5658f9b4cc24de95dd18ad",
											"quoteName": "BTC",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.ETH.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.ETH.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.ETH.value",
												"tradingTerminationProperty": "termination.ETH.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:1F0BB6EB5703B099",
										"base:ETH",
										"quote:BTC",
										"class:fx/crypto",
										"quarterly",
										"sector:crypto",
										"auto:ethbtc"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.01,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.3
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New BTC market",
							"title": "New BTC market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "c1cb93afe500f2ce3f68ab8a083cd440bd15f037fa5a64a1e65be40975b09f4d",
						"reference": "yl2383kpvQxyuAH7HJLBaHzg3bTFwf44rXfMeilr",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1688142755690838874",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689767313",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "AAVEDAI Monthly (Jul 2023)",
										"code": "AAVEDAI.MF21",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "DAI",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.AAVE.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.AAVE.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.AAVE.value",
												"tradingTerminationProperty": "termination.AAVE.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:2839D9B2329C9E70",
										"base:AAVE",
										"quote:DAI",
										"class:fx/crypto",
										"monthly",
										"sector:defi",
										"auto:aavedai"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.01,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.5
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New DAI market",
							"title": "New DAI market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "0776fbdef4ee2fd13784875ae61b48f1b3ab554b65e3304828bb926d7f19922f",
						"reference": "SCGFvul0tJbShyq1UsnPxS7LZ5OyMmFnFLxpAAvy",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1688142755690838874",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689767313",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "BTCUSD Monthly (Jul 2023)",
										"code": "BTCUSD.MF21",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.BTC.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC.value",
												"tradingTerminationProperty": "termination.BTC.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:076BB86A5AA41E3E",
										"base:BTC",
										"quote:USD",
										"class:fx/crypto",
										"monthly",
										"sector:crypto",
										"auto:btcusd"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											},
											{
												"horizon": "300",
												"probability": "0.9999",
												"auctionExtension": "60"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.0001,
										"tau": 0.0000190129,
										"params": {
											"r": 0.016,
											"sigma": 1.25
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New BTCUSD Market",
							"title": "New BTCUSD market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "4bd471659ae3e63e6671f10778c4cf801533b146dbb33c7aa3ab9ea90fd05320",
						"reference": "Test Successsor Market 2",
						"partyId": "2e1ef32e5804e14232406aebaad719087d326afa5c648b7824d0823d8a46c8d1",
						"state": "STATE_ENACTED",
						"timestamp": "1689230479296530529",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689768198",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Test Successsor Market 2",
										"code": "Test Successsor Market 2",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "DAI",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"ethAddress": {
																	"address": "0xfCEAdAFab14d46e20144F48824d0C09B1a03F2BC"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.ETH.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "6"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															},
															{
																"key": {
																	"name": "prices.ETH.timestamp",
																	"type": "TYPE_TIMESTAMP"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "1693382400"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"internal": {
													"time": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "1693382400"
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.ETH.value",
												"tradingTerminationProperty": "vegaprotocol.builtin.timestamp"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:BTC",
										"quote:DAI",
										"class:fx/crypto",
										"quarterly",
										"sector:defi",
										"enactment:2023-08-23T14:00:00Z",
										"settlement:2023-08-30T08:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "3600",
												"probability": "0.9999",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.9999",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.9999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0001140771161,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"successor": {
										"parentMarketId": "d780eeb28a3a6e3a047019ca68e510dbe53f57b13eb33c9385221d128e218fce",
										"insurancePoolFraction": "1"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Success success success!!!",
							"title": "Test successor market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "707c17dd2817521233e77b109f8f8aa65b8bd1148ea6a252e736677ea2e0f35c",
						"reference": "injected_at_runtime",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1689253863379357172",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689767358",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "XX tDAI TERM",
										"code": "XX/tDAI TERM",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "5"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "trading.terminated.BTC-term",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC.value",
												"tradingTerminationProperty": "trading.terminated.BTC-term"
											}
										}
									},
									"decimalPlaces": "2",
									"metadata": [
										"base:BTC",
										"quote:USDT",
										"class:fx/crypto",
										"monthly",
										"sector:defi",
										"enactment:2023-05-15T13:25:00Z",
										"settlement:2023-05-15T14:25:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "720",
												"probability": "0.99",
												"auctionExtension": "60"
											},
											{
												"horizon": "3600",
												"probability": "0.99",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.99",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.99",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.5
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.001,
										"tau": 0.00001901285269,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "# Summary\n\nThis proposal requests to list XX as a market with ($XX) as a settlement asset on the Vega Network as discussed in: https://community.vega.xyz/.\n\n# Rationale\n\n- XX is the largest Crypto asset with the highest volume and Marketcap.\n- Given the price, 2 decimal places will be used for price\n- Position decimal places will be set to 4 considering the value per contract\n- The settlement asset chosen is the largest by trading volume",
							"title": "XX tDAI TERM"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "749e5924010353243dc3423bf3bf7edcca312239d91a4bbe2e987646154470d0",
						"partyId": "2e1ef32e5804e14232406aebaad719087d326afa5c648b7824d0823d8a46c8d1",
						"state": "STATE_ENACTED",
						"timestamp": "1689255668155592706",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689767598",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Test Successor Market 4",
										"code": "Test Successor Market 4",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "DAI",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"ethAddress": {
																	"address": "0x973cB2a51F83a707509fe7cBafB9206982E1c3ad"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.ETH.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "6"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															},
															{
																"key": {
																	"name": "prices.ETH.timestamp",
																	"type": "TYPE_TIMESTAMP"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "1688738871"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"internal": {
													"time": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "1691572666"
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.ETH.value",
												"tradingTerminationProperty": "vegaprotocol.builtin.timestamp"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"enactment:2023-07-30T11:48:47Z",
										"settlement:2023-07-30T11:48:47Z",
										"source:docs.vega.xyz"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.01,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.5
										}
									},
									"positionDecimalPlaces": "5",
									"linearSlippageFactor": "0.001",
									"successor": {
										"parentMarketId": "4bd471659ae3e63e6671f10778c4cf801533b146dbb33c7aa3ab9ea90fd05320",
										"insurancePoolFraction": "1"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "An example proposal to add Lorem Ipsum market",
							"title": "Add Lorem Ipsum market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "370751c99dee8d4ef0f2481f767800d9f9e0a0bfd466ae32b0e97a47e53dd860",
						"reference": "YY SUCCESSOR Market C4",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1689264962341903110",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689767358",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Market YY SUCCESSOR",
										"code": "Market YY SUCCESSOR",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC2.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "5"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "trading.terminated.BTC2",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC2.value",
												"tradingTerminationProperty": "trading.terminated.BTC2"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:ETH",
										"quote:USDT",
										"class:fx/crypto",
										"quarterly",
										"sector:defi",
										"enactment:2023-05-23T14:00:00Z",
										"settlement:2023-06-30T08:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "3600",
												"probability": "0.9999",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.9999",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.9999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0001140771161,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"successor": {
										"parentMarketId": "707c17dd2817521233e77b109f8f8aa65b8bd1148ea6a252e736677ea2e0f35c",
										"insurancePoolFraction": "1"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\n YY SUCCESSOR MARKET C4 This proposal requests to list BTC/USDT-230630 as a market with USDT as a settlement asset on the Vega Network as discussed in: https://community.vega.xyz/.\n\n## Rationale\n\n- BTC is the largest Crypto asset with the highest volume and Marketcap.\n- Given the price, 1 decimal places will be used for price due to the number of valid digits in asset price. \n- Position decimal places will be set to 4 considering the value per contract\n- USDT is chosen as settlement asset due to its stability.-success",
							"title": "YY SUCCESSOR -001 - Create market - Market C4 Future - 2023/06/30 -success"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "c4fb2668061f0e660009413d1a7073cd73ccbbe5e821a3597df2db51fea4189e",
						"reference": "YY SUCCESSOR Enact Last voting end 1st",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_PASSED",
						"timestamp": "1689266245054453580",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689770358",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "YY SUCCESSOR Enact Last voting end 1st",
										"code": "YY SUCCESSOR Enact Last voting end 1st",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC3.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "5"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "trading.terminated.BTC3",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC3.value",
												"tradingTerminationProperty": "trading.terminated.BTC3"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:ETH",
										"quote:USDT",
										"class:fx/crypto",
										"quarterly",
										"sector:defi",
										"enactment:2023-05-23T14:00:00Z",
										"settlement:2023-06-30T08:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "3600",
												"probability": "0.9999",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.9999",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.9999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0001140771161,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"successor": {
										"parentMarketId": "370751c99dee8d4ef0f2481f767800d9f9e0a0bfd466ae32b0e97a47e53dd860",
										"insurancePoolFraction": "1"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\n YY SUCCESSOR  lasst enactment first vote end This proposal requests to list BTC/USDT-230630 as a market with USDT as a settlement asset on the Vega Network as discussed in: https://community.vega.xyz/.\n\n## Rationale\n\n- BTC is the largest Crypto asset with the highest volume and Marketcap.\n- Given the price, 1 decimal places will be used for price due to the number of valid digits in asset price. \n- Position decimal places will be set to 4 considering the value per contract\n- USDT is chosen as settlement asset due to its stability.-success",
							"title": "YY SUCCESSOR  last enact vote end 1st -001 - Create market - Market C4 Future - 2023/06/30 -success"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "f7062e14d1199e026b390f44fcd3ed590d65954dad92f14ac24dedba63a75729",
						"reference": "YY SUCCESSOR Enacting 1st",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_PASSED",
						"timestamp": "1689266260574252071",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689767538",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "YY SUCCESSOR 1st enactment",
										"code": "YY SUCCESSOR 1st enactment",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC3.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "5"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "trading.terminated.BTC3",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC3.value",
												"tradingTerminationProperty": "trading.terminated.BTC3"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:ETH",
										"quote:USDT",
										"class:fx/crypto",
										"quarterly",
										"sector:defi",
										"enactment:2023-05-23T14:00:00Z",
										"settlement:2023-06-30T08:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "3600",
												"probability": "0.9999",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.9999",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.9999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0001140771161,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"successor": {
										"parentMarketId": "370751c99dee8d4ef0f2481f767800d9f9e0a0bfd466ae32b0e97a47e53dd860",
										"insurancePoolFraction": "1"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\n YY SUCCESSOR  1st enactment This proposal requests to list BTC/USDT-230630 as a market with USDT as a settlement asset on the Vega Network as discussed in: https://community.vega.xyz/.\n\n## Rationale\n\n- BTC is the largest Crypto asset with the highest volume and Marketcap.\n- Given the price, 1 decimal places will be used for price due to the number of valid digits in asset price. \n- Position decimal places will be set to 4 considering the value per contract\n- USDT is chosen as settlement asset due to its stability.-success",
							"title": "YY SUCCESSOR  1st enactment -001 - Create market - Market C4 Future - 2023/06/30 -success"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "c97084bde81e07fdcb3b5c67cf9c9457567ba57f5bf4ee84d675c11e9fb0c3ce",
						"reference": "YY SUCCESSOR Enacting second",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_PASSED",
						"timestamp": "1689266342785990755",
						"terms": {
							"closingTimestamp": "1689767298",
							"enactmentTimestamp": "1689768018",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "YY SUCCESSOR 2nd enactment",
										"code": "YY SUCCESSOR 2nd enactment",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC3.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "5"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "trading.terminated.BTC3",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC3.value",
												"tradingTerminationProperty": "trading.terminated.BTC3"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:ETH",
										"quote:USDT",
										"class:fx/crypto",
										"quarterly",
										"sector:defi",
										"enactment:2023-05-23T14:00:00Z",
										"settlement:2023-06-30T08:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "3600",
												"probability": "0.9999",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.9999",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.9999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0001140771161,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"successor": {
										"parentMarketId": "370751c99dee8d4ef0f2481f767800d9f9e0a0bfd466ae32b0e97a47e53dd860",
										"insurancePoolFraction": "1"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\n YY SUCCESSOR  2nd enactment This proposal requests to list BTC/USDT-230630 as a market with USDT as a settlement asset on the Vega Network as discussed in: https://community.vega.xyz/.\n\n## Rationale\n\n- BTC is the largest Crypto asset with the highest volume and Marketcap.\n- Given the price, 1 decimal places will be used for price due to the number of valid digits in asset price. \n- Position decimal places will be set to 4 considering the value per contract\n- USDT is chosen as settlement asset due to its stability.-success",
							"title": "YY SUCCESSOR  2nd enactment -001 - Create market - Market C4 Future - 2023/06/30 -success"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					}
				},
				{
					"proposal": {
						"id": "b37d4ad5cd7561cbf16859325f8baf58d96a3cb3c9cb6726621e2a1dc0f30801",
						"reference": "injected_at_runtime",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1689775672681753985",
						"terms": {
							"closingTimestamp": "1689776065",
							"enactmentTimestamp": "1689776125",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ZZ1 tDAI TERM",
										"code": "ZZ1/tDAI TERM",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "5"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "trading.terminated.ZZ1",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC.value",
												"tradingTerminationProperty": "trading.terminated.ZZ1"
											}
										}
									},
									"decimalPlaces": "2",
									"metadata": [
										"base:BTC",
										"quote:USDT",
										"class:fx/crypto",
										"monthly",
										"sector:defi",
										"enactment:2023-05-15T13:25:00Z",
										"settlement:2023-05-15T14:25:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "720",
												"probability": "0.99",
												"auctionExtension": "60"
											},
											{
												"horizon": "3600",
												"probability": "0.99",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.99",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.99",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.5
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.001,
										"tau": 0.00001901285269,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "# Summary\n\nThis proposal requests to list ZZ1 as a market with ($ZZ1) as a settlement asset on the Vega Network as discussed in: https://community.vega.xyz/.\n\n# Rationale\n\n- XX is the largest Crypto asset with the highest volume and Marketcap.\n- Given the price, 2 decimal places will be used for price\n- Position decimal places will be set to 4 considering the value per contract\n- The settlement asset chosen is the largest by trading volume",
							"title": "ZZ1 tDAI TERM"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "b37d4ad5cd7561cbf16859325f8baf58d96a3cb3c9cb6726621e2a1dc0f30801",
							"timestamp": "1689775712592319057",
							"totalGovernanceTokenBalance": "5000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "b6755a98db5b7376a3ed836d65e75b5bb99bbb4a8f3b2b88edbc93cb942ee6a4",
						"reference": "ZZ3 SUCCESSOR Enacting 1st",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1689776134227266289",
						"terms": {
							"closingTimestamp": "1689776699",
							"enactmentTimestamp": "1689776759",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ZZ3 SUCCESSOR 1st enactment",
										"code": "ZZ3 SUCCESSOR 1st enactment",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC3.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "5"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "trading.terminated.ZZ3",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC3.value",
												"tradingTerminationProperty": "trading.terminated.ZZ3"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:ETH",
										"quote:USDT",
										"class:fx/crypto",
										"quarterly",
										"sector:defi",
										"enactment:2023-05-23T14:00:00Z",
										"settlement:2023-06-30T08:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "3600",
												"probability": "0.9999",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.9999",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.9999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0001140771161,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"successor": {
										"parentMarketId": "b37d4ad5cd7561cbf16859325f8baf58d96a3cb3c9cb6726621e2a1dc0f30801",
										"insurancePoolFraction": "1"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\n ZZ3 SUCCESSOR  1st enactment This proposal requests to list BTC/USDT-230630 as a market with USDT as a settlement asset on the Vega Network as discussed in: https://community.vega.xyz/.\n\n## Rationale\n\n- BTC is the largest Crypto asset with the highest volume and Marketcap.\n- Given the price, 1 decimal places will be used for price due to the number of valid digits in asset price. \n- Position decimal places will be set to 4 considering the value per contract\n- USDT is chosen as settlement asset due to its stability.-success",
							"title": "ZZ3 SUCCESSOR  1st enactment -001 - Create market - Market C4 Future - 2023/06/30 -success"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "b6755a98db5b7376a3ed836d65e75b5bb99bbb4a8f3b2b88edbc93cb942ee6a4",
							"timestamp": "1689776221222639218",
							"totalGovernanceTokenBalance": "5000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "bbbf50bdf026e40e3663cbe9e9df25b0dae9f43b2ef000ccb3b3c63c8d3a9280",
						"reference": "injected_at_runtime",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1689780899487249272",
						"terms": {
							"closingTimestamp": "1689781222",
							"enactmentTimestamp": "1689784822",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ZZ5 tDAI TERM",
										"code": "ZZ5/tDAI TERM",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "5"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "trading.terminated.ZZ5",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC.value",
												"tradingTerminationProperty": "trading.terminated.ZZ5"
											}
										}
									},
									"decimalPlaces": "2",
									"metadata": [
										"base:BTC",
										"quote:USDT",
										"class:fx/crypto",
										"monthly",
										"sector:defi",
										"enactment:2023-05-15T13:25:00Z",
										"settlement:2023-05-15T14:25:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "720",
												"probability": "0.99",
												"auctionExtension": "60"
											},
											{
												"horizon": "3600",
												"probability": "0.99",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.99",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.99",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.5
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.001,
										"tau": 0.00001901285269,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "# Summary\n\nThis proposal requests to list ZZ5 as a market with ($ZZ1) as a settlement asset on the Vega Network as discussed in: https://community.vega.xyz/.\n\n# Rationale\n\n- XX is the largest Crypto asset with the highest volume and Marketcap.\n- Given the price, 2 decimal places will be used for price\n- Position decimal places will be set to 4 considering the value per contract\n- The settlement asset chosen is the largest by trading volume",
							"title": "ZZ5 tDAI TERM"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "bbbf50bdf026e40e3663cbe9e9df25b0dae9f43b2ef000ccb3b3c63c8d3a9280",
							"timestamp": "1689781055163365268",
							"totalGovernanceTokenBalance": "5000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "edf9aa407a79da28f612637131d03d2264cdd16f4fe33bb8e0479b7185756d06",
						"reference": "ZZ6 SUCCESSOR Enacting second",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1689785219703502195",
						"terms": {
							"closingTimestamp": "1689785735",
							"enactmentTimestamp": "1689787535",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ZZ6 SUCCESSOR 2nd enactment",
										"code": "ZZ6 SUCCESSOR 2nd enactment",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC3.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "5"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "trading.terminated.ZZ6",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC3.value",
												"tradingTerminationProperty": "trading.terminated.ZZ6"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:ETH",
										"quote:USDT",
										"class:fx/crypto",
										"quarterly",
										"sector:defi",
										"enactment:2023-05-23T14:00:00Z",
										"settlement:2023-06-30T08:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "3600",
												"probability": "0.9999",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.9999",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.9999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0001140771161,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"successor": {
										"parentMarketId": "bbbf50bdf026e40e3663cbe9e9df25b0dae9f43b2ef000ccb3b3c63c8d3a9280",
										"insurancePoolFraction": "1"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\n ZZ6 SUCCESSOR  2nd enactment This proposal requests to list BTC/USDT-230630 as a market with USDT as a settlement asset on the Vega Network as discussed in: https://community.vega.xyz/.\n\n## Rationale\n\n- BTC is the largest Crypto asset with the highest volume and Marketcap.\n- Given the price, 1 decimal places will be used for price due to the number of valid digits in asset price. \n- Position decimal places will be set to 4 considering the value per contract\n- USDT is chosen as settlement asset due to its stability.-success",
							"title": "ZZ6 SUCCESSOR  2nd enactment -001 - Create market - Market C4 Future - 2023/06/30 -success"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "edf9aa407a79da28f612637131d03d2264cdd16f4fe33bb8e0479b7185756d06",
							"timestamp": "1689785290665319867",
							"totalGovernanceTokenBalance": "5000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "f89edff9a4aca0088e28363e4a20a9d0fd9ea4cb89bf9dc816f2cb5ca4189453",
						"reference": "ZZ7 SUCCESSOR Enacting 1st",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1689785252934860194",
						"terms": {
							"closingTimestamp": "1689785735",
							"enactmentTimestamp": "1689786155",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ZZ7 SUCCESSOR 1st enactment",
										"code": "ZZ7 SUCCESSOR 1st enactment",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC3.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "5"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "trading.terminated.ZZ7",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC3.value",
												"tradingTerminationProperty": "trading.terminated.ZZ7"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:ETH",
										"quote:USDT",
										"class:fx/crypto",
										"quarterly",
										"sector:defi",
										"enactment:2023-05-23T14:00:00Z",
										"settlement:2023-06-30T08:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "3600",
												"probability": "0.9999",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.9999",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.9999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0001140771161,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"successor": {
										"parentMarketId": "b37d4ad5cd7561cbf16859325f8baf58d96a3cb3c9cb6726621e2a1dc0f30801",
										"insurancePoolFraction": "1"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\n ZZ7 SUCCESSOR  1st enactment This proposal requests to list BTC/USDT-230630 as a market with USDT as a settlement asset on the Vega Network as discussed in: https://community.vega.xyz/.\n\n## Rationale\n\n- BTC is the largest Crypto asset with the highest volume and Marketcap.\n- Given the price, 1 decimal places will be used for price due to the number of valid digits in asset price. \n- Position decimal places will be set to 4 considering the value per contract\n- USDT is chosen as settlement asset due to its stability.-success",
							"title": "ZZ7 SUCCESSOR  1st enactment -001 - Create market - Market C4 Future - 2023/06/30 -success"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "f89edff9a4aca0088e28363e4a20a9d0fd9ea4cb89bf9dc816f2cb5ca4189453",
							"timestamp": "1689785313676544470",
							"totalGovernanceTokenBalance": "5000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "896bc6593dd4ab4cb81b2eb7b1b4b208861fa96b66a7d59fb16bcaef057cd882",
						"reference": "injected_at_runtime",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1689786582538900712",
						"terms": {
							"closingTimestamp": "1689786764",
							"enactmentTimestamp": "1689786824",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ZY1 tDAI TERM",
										"code": "ZY1/tDAI TERM",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "5"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "trading.terminated.ZY1",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC.value",
												"tradingTerminationProperty": "trading.terminated.ZY1"
											}
										}
									},
									"decimalPlaces": "2",
									"metadata": [
										"base:BTC",
										"quote:USDT",
										"class:fx/crypto",
										"monthly",
										"sector:defi",
										"enactment:2023-05-15T13:25:00Z",
										"settlement:2023-05-15T14:25:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "720",
												"probability": "0.99",
												"auctionExtension": "60"
											},
											{
												"horizon": "3600",
												"probability": "0.99",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.99",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.99",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.5
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.001,
										"tau": 0.00001901285269,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "# Summary\n\nThis proposal requests to list ZY1 as a market with ($ZZ1) as a settlement asset on the Vega Network as discussed in: https://community.vega.xyz/.\n\n# Rationale\n\n- XX is the largest Crypto asset with the highest volume and Marketcap.\n- Given the price, 2 decimal places will be used for price\n- Position decimal places will be set to 4 considering the value per contract\n- The settlement asset chosen is the largest by trading volume",
							"title": "ZY1 tDAI TERM"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "896bc6593dd4ab4cb81b2eb7b1b4b208861fa96b66a7d59fb16bcaef057cd882",
							"timestamp": "1689786626771020341",
							"totalGovernanceTokenBalance": "5000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "9bd440f8ec1c7639ef4988a2938180236d996796ec03ffa20974d5bf6f294354",
						"reference": "ZY3 SUCCESSOR Enacting 1st",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1689787059699515263",
						"terms": {
							"closingTimestamp": "1689787325",
							"enactmentTimestamp": "1689787385",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ZY3 SUCCESSOR 1st enactment",
										"code": "ZY3 SUCCESSOR 1st enactment",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC3.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "5"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "trading.terminated.ZY3",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC3.value",
												"tradingTerminationProperty": "trading.terminated.ZY3"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:ETH",
										"quote:USDT",
										"class:fx/crypto",
										"quarterly",
										"sector:defi",
										"enactment:2023-05-23T14:00:00Z",
										"settlement:2023-06-30T08:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "3600",
												"probability": "0.9999",
												"auctionExtension": "120"
											},
											{
												"horizon": "14400",
												"probability": "0.9999",
												"auctionExtension": "180"
											},
											{
												"horizon": "43200",
												"probability": "0.9999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0001140771161,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"successor": {
										"parentMarketId": "896bc6593dd4ab4cb81b2eb7b1b4b208861fa96b66a7d59fb16bcaef057cd882",
										"insurancePoolFraction": "1"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\n ZY3 SUCCESSOR  1st enactment This proposal requests to list BTC/USDT-230630 as a market with USDT as a settlement asset on the Vega Network as discussed in: https://community.vega.xyz/.\n\n## Rationale\n\n- BTC is the largest Crypto asset with the highest volume and Marketcap.\n- Given the price, 1 decimal places will be used for price due to the number of valid digits in asset price. \n- Position decimal places will be set to 4 considering the value per contract\n- USDT is chosen as settlement asset due to its stability.-success",
							"title": "ZY3 SUCCESSOR  1st enactment -001 - Create market - Market C4 Future - 2023/06/30 -success"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "9bd440f8ec1c7639ef4988a2938180236d996796ec03ffa20974d5bf6f294354",
							"timestamp": "1689787112499280816",
							"totalGovernanceTokenBalance": "5000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "728e68e99e926ec1fe125cf303d18fc1a52e956b5a35f92e84177b7fa639fbe3",
						"reference": "injected_at_runtime",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1690454777674769965",
						"terms": {
							"closingTimestamp": "1690455303",
							"enactmentTimestamp": "1690455363",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "New York temperature 3 day average (°C)",
										"code": "NY Celsius/USDC-CE",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USDC-CE",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.NY.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "4"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	},
																	{
																		"operator": "OPERATOR_LESS_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"internal": {
													"time": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "1691232149"
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.NY.value",
												"tradingTerminationProperty": "vegaprotocol.builtin.timestamp"
											}
										}
									},
									"decimalPlaces": "1",
									"metadata": [
										"base:BTC",
										"quote:USDT",
										"class:fx/crypto",
										"monthly",
										"sector:defi",
										"enactment:2023-07-20T17:00:00Z",
										"settlement:2023-07-25T15:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "14400",
												"probability": "0.999",
												"auctionExtension": "30"
											},
											{
												"horizon": "43200",
												"probability": "0.9999",
												"auctionExtension": "60"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0001140771161,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "# Summary\n\nThis proposal requests to list the average 3 day temperature (°C) in New York as a market with ($USD-CE) as a settlement asset on the Vega Network\n\n# Rationale\n\n- The underying of hte market is the temperature in celsius in New York (latitude=40.7128, longitude=-74.0060) \n- Source used for settling data will be the 3 day average using https://api.open-meteo.com/v1/forecast?latitude=40.7128\u0026longitude=-74.0060\u0026hourly=temperature_2m\n- After the termination of the market the market will update with a vote for the settlement value of the 3 day average temeperature\n- Leaderboard will only use public key so feel free to check what your fellow iceberg orders competitors are doing\n- Remember to checkout the bughunts since finding bugs are rewarded\n- Browser wallet and Windows CLI wallet are now available for Iceberg orders so please try and break them",
							"title": "Avg 3 day temperature (°C) New York market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "728e68e99e926ec1fe125cf303d18fc1a52e956b5a35f92e84177b7fa639fbe3",
							"timestamp": "1690455138915113618",
							"totalGovernanceTokenBalance": "5000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "c663f1e0adfeeaa692bb4cb1d17363a214d83666be2573b7b9462433f80757f0",
						"reference": "injected_at_runtime",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1690457567059687881",
						"terms": {
							"closingTimestamp": "1690457758",
							"enactmentTimestamp": "1690457818",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "New York temperature 3 day average (°C) v2",
										"code": "NY Celsius/USDC-CE v2",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USDC-CE",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.NY.value",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "4"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	},
																	{
																		"operator": "OPERATOR_LESS_THAN",
																		"value": "0"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"internal": {
													"time": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "1691232149"
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.NY.value",
												"tradingTerminationProperty": "vegaprotocol.builtin.timestamp"
											}
										}
									},
									"decimalPlaces": "3",
									"metadata": [
										"base:BTC",
										"quote:USDT",
										"class:fx/crypto",
										"monthly",
										"sector:defi",
										"enactment:2023-07-20T17:00:00Z",
										"settlement:2023-07-25T15:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "14400",
												"probability": "0.999",
												"auctionExtension": "30"
											},
											{
												"horizon": "43200",
												"probability": "0.9999",
												"auctionExtension": "60"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0001140771161,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "1",
									"linearSlippageFactor": "0.001",
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "# Summary\n\nThis proposal requests to list the average 3 day temperature (°C) in New York as a market with ($USD-CE) as a settlement asset on the Vega Network\n\n# Rationale\n\n- The underying of hte market is the temperature in celsius in New York (latitude=40.7128, longitude=-74.0060) \n- Source used for settling data will be the 3 day average using https://api.open-meteo.com/v1/forecast?latitude=40.7128\u0026longitude=-74.0060\u0026hourly=temperature_2m\n- After the termination of the market the market will update with a vote for the settlement value of the 3 day average temeperature\n- Leaderboard will only use public key so feel free to check what your fellow iceberg orders competitors are doing\n- Remember to checkout the bughunts since finding bugs are rewarded\n- Browser wallet and Windows CLI wallet are now available for Iceberg orders so please try and break them",
							"title": "Avg 3 day temperature (°C) New York market v2"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "c663f1e0adfeeaa692bb4cb1d17363a214d83666be2573b7b9462433f80757f0",
							"timestamp": "1690457582581164302",
							"totalGovernanceTokenBalance": "5000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "03d8a113c3dbe1cf601e4351178b1b728e50a1798dc2ecb1d696bfb1e35459b6",
						"reference": "aBh86rj3YvGo6fdfM216YpqelbH8WYvgW2X9xUTW",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891791249715900",
						"terms": {
							"closingTimestamp": "1699891815",
							"enactmentTimestamp": "1699891830",
							"updateMarketState": {
								"changes": {
									"marketId": "0776fbdef4ee2fd13784875ae61b48f1b3ab554b65e3304828bb926d7f19922f",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate BTCUSD Monthly (Jul 2023) market",
							"title": "Terminate BTCUSD Monthly (Jul 2023) market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "03d8a113c3dbe1cf601e4351178b1b728e50a1798dc2ecb1d696bfb1e35459b6",
							"timestamp": "1699891801423732939",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "c67a287d69c9482f2cc9c141e2ffce3625555a7625240db9117b85adccdfba4e",
						"reference": "ZJEQiTTSt7Vc9KIIgLLzRvlpPCFqFbS05efZqiCs",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891801423732939",
						"terms": {
							"closingTimestamp": "1699891825",
							"enactmentTimestamp": "1699891840",
							"updateMarketState": {
								"changes": {
									"marketId": "24df2940a2617b78b4e9f64c6508e4d5d509151a6321a3244a11e1d9859e7cb1",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate Apple Monthly (Jul 2023) market",
							"title": "Terminate Apple Monthly (Jul 2023) market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "c67a287d69c9482f2cc9c141e2ffce3625555a7625240db9117b85adccdfba4e",
							"timestamp": "1699891810653434355",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "7a8bce665a36ab6ab3d8e175ab2928356b99aba38005ffb1b008159d298f5a11",
						"reference": "kdBHqkxCOumYSd8F0bzCaUe7iDK1GmnUF3BEn1Ff",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891810653434355",
						"terms": {
							"closingTimestamp": "1699891835",
							"enactmentTimestamp": "1699891850",
							"updateMarketState": {
								"changes": {
									"marketId": "38a4aa08cc0ff0da92ab7d0951de999b0ee0f81029f452b8cf77fb4c300dbd41",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate ETHBTC Quarterly (Sep 2023) market",
							"title": "Terminate ETHBTC Quarterly (Sep 2023) market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "7a8bce665a36ab6ab3d8e175ab2928356b99aba38005ffb1b008159d298f5a11",
							"timestamp": "1699891821691014906",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "dca04dee0546cc003ee4b341d35c4590d93a5cfdd91c7cbc2975d6d4a463459f",
						"reference": "3ZLeSwA33tx7X0FYuFwsfxSNw8DsTlKn8UHVwT3a",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891821691014906",
						"terms": {
							"closingTimestamp": "1699891845",
							"enactmentTimestamp": "1699891860",
							"updateMarketState": {
								"changes": {
									"marketId": "b66cd4be223dfd900a4750bb5175e17d8f678996877d262be4c749a99e22a970",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate Tesla Quarterly (Sep 2023) market",
							"title": "Terminate Tesla Quarterly (Sep 2023) market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "dca04dee0546cc003ee4b341d35c4590d93a5cfdd91c7cbc2975d6d4a463459f",
							"timestamp": "1699891832082127741",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "0fad7395ef3bf1beb8ddef760ba7b5a96505bec27336319322c5e1b11fac5d6a",
						"reference": "Jnm9gdocpZi5qCi35Dd8qrzxm9z7W7h5G4ZQQYlf",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891832082127741",
						"terms": {
							"closingTimestamp": "1699891856",
							"enactmentTimestamp": "1699891871",
							"updateMarketState": {
								"changes": {
									"marketId": "b844aacfe0c6a5db17c1a65164a6c4418f9cc4c5d0e29eed28811487befd296b",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate ETHDAI Monthly (Jul 2023) market",
							"title": "Terminate ETHDAI Monthly (Jul 2023) market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "0fad7395ef3bf1beb8ddef760ba7b5a96505bec27336319322c5e1b11fac5d6a",
							"timestamp": "1699891841358501240",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "0eb63efe42df03dca4430c1074936c59957f5ec49524a171c56fda9f575fd774",
						"reference": "IFwv40hQixounHCE3CgRM5OMCRjv0W248o6eatwQ",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891841358501240",
						"terms": {
							"closingTimestamp": "1699891866",
							"enactmentTimestamp": "1699891881",
							"updateMarketState": {
								"changes": {
									"marketId": "bbbf50bdf026e40e3663cbe9e9df25b0dae9f43b2ef000ccb3b3c63c8d3a9280",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate ZZ5 tDAI TERM market",
							"title": "Terminate ZZ5 tDAI TERM market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "0eb63efe42df03dca4430c1074936c59957f5ec49524a171c56fda9f575fd774",
							"timestamp": "1699891852386861499",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "22f360b44dd3943e54f45fe5b40a7ec4106f86e479a0dc16b1ffae000f683470",
						"reference": "B6VbPrXPDKEu6NtIQ7rGaMPsydDqqPkVkN8Zna18",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891852386861499",
						"terms": {
							"closingTimestamp": "1699891876",
							"enactmentTimestamp": "1699891891",
							"updateMarketState": {
								"changes": {
									"marketId": "c1cb93afe500f2ce3f68ab8a083cd440bd15f037fa5a64a1e65be40975b09f4d",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate AAVEDAI Monthly (Jul 2023) market",
							"title": "Terminate AAVEDAI Monthly (Jul 2023) market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "22f360b44dd3943e54f45fe5b40a7ec4106f86e479a0dc16b1ffae000f683470",
							"timestamp": "1699891861375351935",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "1164590407beb942fc4e920472f394fb500229bdd0aba83d53ba410c57a7a7c4",
						"reference": "tZHkt7eLhMBQXhHnaNRTtRIcYDnVeVTg81JvDF7N",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891861375351935",
						"terms": {
							"closingTimestamp": "1699891886",
							"enactmentTimestamp": "1699891901",
							"updateMarketState": {
								"changes": {
									"marketId": "d780eeb28a3a6e3a047019ca68e510dbe53f57b13eb33c9385221d128e218fce",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate UNIDAI Monthly (Jul 2023) market",
							"title": "Terminate UNIDAI Monthly (Jul 2023) market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "1164590407beb942fc4e920472f394fb500229bdd0aba83d53ba410c57a7a7c4",
							"timestamp": "1699891871767055221",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "e1a28abfdfec6bead77921cc8537f34f040c90c9256524d08c3f6529623b2f40",
						"reference": "mS1Ioavr5EN40YHr3G4rtjNJuRKBzj1TPxlbUDBM",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891871767055221",
						"terms": {
							"closingTimestamp": "1699891897",
							"enactmentTimestamp": "1699891912",
							"updateMarketState": {
								"changes": {
									"marketId": "896bc6593dd4ab4cb81b2eb7b1b4b208861fa96b66a7d59fb16bcaef057cd882",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate ZY1 tDAI TERM market",
							"title": "Terminate ZY1 tDAI TERM market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "e1a28abfdfec6bead77921cc8537f34f040c90c9256524d08c3f6529623b2f40",
							"timestamp": "1699891882701545556",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "988be8259bf01f15729e3e1e1fd499036104b5c6daf46b9cd63a2fa7fd215d1b",
						"reference": "RYkIYPUzgaKBD8OfI33ALm3Vahpgd0yDmv5kv792",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891882701545556",
						"terms": {
							"closingTimestamp": "1699891907",
							"enactmentTimestamp": "1699891922",
							"updateMarketState": {
								"changes": {
									"marketId": "9bd440f8ec1c7639ef4988a2938180236d996796ec03ffa20974d5bf6f294354",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate ZY3 SUCCESSOR 1st enactment market",
							"title": "Terminate ZY3 SUCCESSOR 1st enactment market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "988be8259bf01f15729e3e1e1fd499036104b5c6daf46b9cd63a2fa7fd215d1b",
							"timestamp": "1699891893136290128",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "c6df0cb9484482763ccac2cc5e65613076e589849e55f4af89d914c1e87e092e",
						"reference": "ypnEOeeFIgfS63bIKPN9D6c1YOcVoIJ62SwZZZH1",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891903827513949",
						"terms": {
							"closingTimestamp": "1699891927",
							"enactmentTimestamp": "1699891942",
							"updateMarketState": {
								"changes": {
									"marketId": "f89edff9a4aca0088e28363e4a20a9d0fd9ea4cb89bf9dc816f2cb5ca4189453",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate ZZ7 SUCCESSOR 1st enactment market",
							"title": "Terminate ZZ7 SUCCESSOR 1st enactment market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "c6df0cb9484482763ccac2cc5e65613076e589849e55f4af89d914c1e87e092e",
							"timestamp": "1699891913528299989",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "75ea731815d61e87f5cc0e657eff39a220cb65b2dd3d48bf483732167f714b63",
						"reference": "f8SThQ2tMyhE2VwcCgEa6n3KSVhs8zEWIlBhv8pP",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891914125915089",
						"terms": {
							"closingTimestamp": "1699891937",
							"enactmentTimestamp": "1699891952",
							"updateMarketState": {
								"changes": {
									"marketId": "edf9aa407a79da28f612637131d03d2264cdd16f4fe33bb8e0479b7185756d06",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate ZZ6 SUCCESSOR 2nd enactment market",
							"title": "Terminate ZZ6 SUCCESSOR 2nd enactment market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "75ea731815d61e87f5cc0e657eff39a220cb65b2dd3d48bf483732167f714b63",
							"timestamp": "1699891923025595200",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "ad640a7f55832c4330a75939f3926bc9ea3448bb00d8dc5bb899b1535ede5bb2",
						"reference": "JEGqc8B3AP4OBOJDOxP4W3HIYY2rJgJCRee5dFKr",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891923025595200",
						"terms": {
							"closingTimestamp": "1699891948",
							"enactmentTimestamp": "1699891963",
							"updateMarketState": {
								"changes": {
									"marketId": "b6755a98db5b7376a3ed836d65e75b5bb99bbb4a8f3b2b88edbc93cb942ee6a4",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate ZZ3 SUCCESSOR 1st enactment market",
							"title": "Terminate ZZ3 SUCCESSOR 1st enactment market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "ad640a7f55832c4330a75939f3926bc9ea3448bb00d8dc5bb899b1535ede5bb2",
							"timestamp": "1699891933175448740",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "aa1a91f08994aa924ef82e44c5e5c610d22d138bc848690285108c6708501c72",
						"reference": "GzLJJ0ENBGR7JcCS5IKuaHwU9XGqSJozNaSiRNNI",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891953575329379",
						"terms": {
							"closingTimestamp": "1699891978",
							"enactmentTimestamp": "1699891993",
							"updateMarketState": {
								"changes": {
									"marketId": "707c17dd2817521233e77b109f8f8aa65b8bd1148ea6a252e736677ea2e0f35c",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate XX tDAI TERM market",
							"title": "Terminate XX tDAI TERM market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "aa1a91f08994aa924ef82e44c5e5c610d22d138bc848690285108c6708501c72",
							"timestamp": "1699891963877033931",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "37b2ec935c3f8503e1690013962e0821bd467ae2523bebccd9cda45e446e4692",
						"reference": "j0sr8CEfaGJqk4f9om7ziHsP2jpSTy2LtA6prTla",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891963877033931",
						"terms": {
							"closingTimestamp": "1699891988",
							"enactmentTimestamp": "1699892003",
							"updateMarketState": {
								"changes": {
									"marketId": "370751c99dee8d4ef0f2481f767800d9f9e0a0bfd466ae32b0e97a47e53dd860",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "10"
								}
							}
						},
						"rationale": {
							"description": "Terminate Market YY SUCCESSOR market",
							"title": "Terminate Market YY SUCCESSOR market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "37b2ec935c3f8503e1690013962e0821bd467ae2523bebccd9cda45e446e4692",
							"timestamp": "1699891974780239571",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "d6839f728d0bf1c5394015e6e3a9483796911d1b212a18144a0a64d2bb3e893a",
						"reference": "WaQjAAOoYSQIydCxuadhin5PAC2QS6auK7nvkASM",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699891998712904740",
						"terms": {
							"closingTimestamp": "1699892023",
							"enactmentTimestamp": "1699892038",
							"updateNetworkParameter": {
								"changes": {
									"key": "limits.markets.proposePerpetualEnabled",
									"value": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of limits.markets.proposePerpetualEnabled to 1 from the previous value",
							"title": "Update limits.markets.proposePerpetualEnabled"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "d6839f728d0bf1c5394015e6e3a9483796911d1b212a18144a0a64d2bb3e893a",
							"timestamp": "1699892009334370206",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "1b0d91c8422f6dfb3037f0d2beea9b41a6045e42341b6b9deeefa956cead18fb",
						"reference": "z5O5t9jpnWYeQZ9NTswuUlGGKB5jA4X0nK38eGGU",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892048754483549",
						"terms": {
							"closingTimestamp": "1699892073",
							"enactmentTimestamp": "1699892088",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ETHDAI Monthly (Dec 2023)",
										"code": "ETHDAI.MF21",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "DAI",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.ETH.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.ETH.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.ETH.value",
												"tradingTerminationProperty": "termination.ETH.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:4BC6D2154BE74E1F",
										"base:ETH",
										"quote:DAI",
										"class:fx/crypto",
										"monthly",
										"managed:vega/ops",
										"sector:defi",
										"auto:ethdai"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.01,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.3
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"liquiditySlaParameters": {
										"priceRange": "0.05",
										"commitmentMinTimeFraction": "0.95",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.9"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New ETHDAI market",
							"title": "New ETHDAI market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "1b0d91c8422f6dfb3037f0d2beea9b41a6045e42341b6b9deeefa956cead18fb",
							"timestamp": "1699892059028287988",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "498a96130727446e28efb6df6b2da9590b369142216a31f9a844bee1ced2d3bd",
						"reference": "j67oV0vCxemil88Lng7W5ydqitQtpCGoXcqLYtHg",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892048754483549",
						"terms": {
							"closingTimestamp": "1699892073",
							"enactmentTimestamp": "1699892088",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "AAVEDAI Monthly (Dec 2023)",
										"code": "AAVEDAI.MF21",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "DAI",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.AAVE.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.AAVE.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.AAVE.value",
												"tradingTerminationProperty": "termination.AAVE.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:2839D9B2329C9E70",
										"base:AAVE",
										"quote:DAI",
										"class:fx/crypto",
										"monthly",
										"managed:vega/ops",
										"sector:defi",
										"auto:aavedai"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.01,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.5
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"liquiditySlaParameters": {
										"priceRange": "0.05",
										"commitmentMinTimeFraction": "0.95",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.9"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New DAI market",
							"title": "New DAI market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "498a96130727446e28efb6df6b2da9590b369142216a31f9a844bee1ced2d3bd",
							"timestamp": "1699892059821916752",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "7cf3abfe9c3310de14dd613523caf8d52cd27a7bd7befb958693b4ec9abee6ae",
						"reference": "cvhiQ4vOGw0q770VaPnOb11EXxAJd5GZsfbXC3Rv",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892049382487724",
						"terms": {
							"closingTimestamp": "1699892073",
							"enactmentTimestamp": "1699892088",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Apple Monthly (Dec 2023)",
										"code": "AAPL.MF21",
										"future": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.AAPL.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.AAPL.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.AAPL.value",
												"tradingTerminationProperty": "termination.AAPL.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:4899E01009F1A721",
										"quote:USD",
										"ticker:AAPL",
										"class:equities/single-stock-futures",
										"sector:tech",
										"managed:vega/ops",
										"listing_venue:NASDAQ",
										"country:US",
										"auto:aapl"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.1,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.3
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"liquiditySlaParameters": {
										"priceRange": "0.05",
										"commitmentMinTimeFraction": "0.95",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.9"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New USD market",
							"title": "New USD market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "7cf3abfe9c3310de14dd613523caf8d52cd27a7bd7befb958693b4ec9abee6ae",
							"timestamp": "1699892059821916752",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "66aa425859126386468555ee489d9ef657e6d70061e01f70913500c9cb74a237",
						"reference": "jsUmP9iKBtavDAeHMHx73myeWhSl78NNgjrzkTBZ",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892049382487724",
						"terms": {
							"closingTimestamp": "1699892073",
							"enactmentTimestamp": "1699892088",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "EURUSD Perpetual",
										"code": "EURUSD.PERP",
										"perpetual": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USD",
											"marginFundingFactor": "0.1",
											"interestRate": "0",
											"clampLowerBound": "0",
											"clampUpperBound": "0",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1699892088",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x1a81afB8146aeFfCFc5E50e8479e826E7D55b910",
														"abi": "[{\"inputs\":[],\"name\":\"latestAnswer\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
														"method": "latestAnswer",
														"trigger": {
															"timeTrigger": {
																"initial": "1699892088",
																"every": "300"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eur.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "8"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eur.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "11155111"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "eur.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:70657270657572757364",
										"base:EUR",
										"quote:USD",
										"class:fx/crypto",
										"monthly",
										"managed:vega/ops",
										"sector:crypto",
										"auto:perpetual_eur_usd"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "2160",
												"probability": "0.99",
												"auctionExtension": "300"
											},
											{
												"horizon": "720",
												"probability": "0.99",
												"auctionExtension": "180"
											},
											{
												"horizon": "180",
												"probability": "0.99",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.0001,
										"tau": 0.00000380258,
										"params": {
											"r": 0.016,
											"sigma": 0.7
										}
									},
									"positionDecimalPlaces": "1",
									"linearSlippageFactor": "0.01",
									"liquiditySlaParameters": {
										"priceRange": "0.05",
										"commitmentMinTimeFraction": "0.95",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.9"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New EURUSD perpetual market",
							"title": "New EURUSD perpetual market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "66aa425859126386468555ee489d9ef657e6d70061e01f70913500c9cb74a237",
							"timestamp": "1699892059028287988",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "d5f5b8d3bdb4f918b86bc3e6609b3ded911b8b47ea4ca7170869dddee2eb2639",
						"reference": "PMJhGomRnZq7NWwCCBqXg0LJNlNGfB41KSScthAO",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892049382487724",
						"terms": {
							"closingTimestamp": "1699892073",
							"enactmentTimestamp": "1699892088",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "UNIDAI Monthly (Dec 2023)",
										"code": "UNIDAI.MF21",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "DAI",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.UNI.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.UNI.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.UNI.value",
												"tradingTerminationProperty": "termination.UNI.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:3C58ED2A4A6C5D7E",
										"base:UNI",
										"quote:DAI",
										"class:fx/crypto",
										"monthly",
										"managed:vega/ops",
										"sector:defi",
										"auto:unidai"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.01,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.5
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"liquiditySlaParameters": {
										"priceRange": "0.05",
										"commitmentMinTimeFraction": "0.95",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.9"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New DAI market",
							"title": "New DAI market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "d5f5b8d3bdb4f918b86bc3e6609b3ded911b8b47ea4ca7170869dddee2eb2639",
							"timestamp": "1699892059821916752",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "74ea9b5f3441189dcd7b1fbff42f200bb77653cac76f14d21123a80267c2730b",
						"reference": "tCB9tKovPWzitQSq0BeukZYMap6X1cnrHEFpCVSW",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892049382487724",
						"terms": {
							"closingTimestamp": "1699892073",
							"enactmentTimestamp": "1699892088",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "LINKUSD Perpetual",
										"code": "LINKUSD.PERP",
										"perpetual": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USD",
											"marginFundingFactor": "0.1",
											"interestRate": "0",
											"clampLowerBound": "0",
											"clampUpperBound": "0",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1699892088",
																"every": "14400"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0xc59E3633BAAC79493d908e63626716e204A45EdF",
														"abi": "[{\"inputs\":[],\"name\":\"latestAnswer\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
														"method": "latestAnswer",
														"trigger": {
															"timeTrigger": {
																"initial": "1699892088",
																"every": "120"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "link.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "8"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "link.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "11155111"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "link.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:706572706c696e6b757364",
										"base:LINK",
										"quote:USD",
										"class:fx/crypto",
										"monthly",
										"managed:vega/ops",
										"sector:crypto",
										"auto:perpetual_link_usd"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "4320",
												"probability": "0.99",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.99",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.99",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.0001,
										"tau": 0.00000380258,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "1",
									"linearSlippageFactor": "0.01",
									"liquiditySlaParameters": {
										"priceRange": "0.05",
										"commitmentMinTimeFraction": "0.95",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.9"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New LINKUSD perpetual market",
							"title": "New LINKUSD perpetual market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "74ea9b5f3441189dcd7b1fbff42f200bb77653cac76f14d21123a80267c2730b",
							"timestamp": "1699892059821916752",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "4ea4c96f3a498c2384fa5b3362fc97da157f5bc17260627c7d7c5ea92392da1c",
						"reference": "Lr6qknL3nbQvQvzqAIlyjrAQIg8ecvIbYDZE4Ura",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892049382487724",
						"terms": {
							"closingTimestamp": "1699892073",
							"enactmentTimestamp": "1699892088",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Tesla Quarterly (Feb 2024)",
										"code": "TSLA.QM21",
										"future": {
											"settlementAsset": "177e8f6c25a955bd18475084b99b2b1d37f28f3dec393fab7755a7e69c3d8c3b",
											"quoteName": "EURO",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.TSLA.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.TSLA.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.TSLA.value",
												"tradingTerminationProperty": "termination.TSLA.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:5A86B190C384997F",
										"quote:EURO",
										"ticker:TSLA",
										"class:equities/single-stock-futures",
										"sector:tech",
										"managed:vega/ops",
										"listing_venue:NASDAQ",
										"country:US",
										"auto:tsla"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.01,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.8
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"liquiditySlaParameters": {
										"priceRange": "0.05",
										"commitmentMinTimeFraction": "0.95",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.9"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New EURO market",
							"title": "New EURO market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "4ea4c96f3a498c2384fa5b3362fc97da157f5bc17260627c7d7c5ea92392da1c",
							"timestamp": "1699892059821916752",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "7314703a4424f74e88869196e27beaac5bac5f4da86cdfe41b036bf7a66e9cfb",
						"reference": "a4sLBZICz6690hTsTvTu52kAPuJru6AVjwKKyBN1",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892049382487724",
						"terms": {
							"closingTimestamp": "1699892073",
							"enactmentTimestamp": "1699892088",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "BTCUSD Monthly (Dec 2023)",
										"code": "BTCUSD.MF21",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "USD",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.BTC.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.BTC.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.BTC.value",
												"tradingTerminationProperty": "termination.BTC.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:076BB86A5AA41E3E",
										"base:BTC",
										"quote:USD",
										"class:fx/crypto",
										"monthly",
										"managed:vega/ops",
										"sector:crypto",
										"auto:btcusd"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											},
											{
												"horizon": "300",
												"probability": "0.9999",
												"auctionExtension": "60"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.0001,
										"tau": 0.0000190129,
										"params": {
											"r": 0.016,
											"sigma": 1.25
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"liquiditySlaParameters": {
										"priceRange": "0.05",
										"commitmentMinTimeFraction": "0.95",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.9"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New BTCUSD Market",
							"title": "New BTCUSD market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "7314703a4424f74e88869196e27beaac5bac5f4da86cdfe41b036bf7a66e9cfb",
							"timestamp": "1699892059821916752",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "40f22f1012e971d3532df04f4c9c15bbda43715dd46304612eeb6bd17d5eb3a4",
						"reference": "Z24iNgZlVDzWLJ7iresz86zsYofHWzI8MFl8QhBy",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892049382487724",
						"terms": {
							"closingTimestamp": "1699892073",
							"enactmentTimestamp": "1699892088",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ETHBTC Quarterly (Feb 2024)",
										"code": "ETHBTC.QM21",
										"future": {
											"settlementAsset": "cee709223217281d7893b650850ae8ee8a18b7539b5658f9b4cc24de95dd18ad",
											"quoteName": "BTC",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.ETH.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.ETH.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.ETH.value",
												"tradingTerminationProperty": "termination.ETH.value"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:1F0BB6EB5703B099",
										"base:ETH",
										"quote:BTC",
										"class:fx/crypto",
										"quarterly",
										"managed:vega/ops",
										"sector:crypto",
										"auto:ethbtc"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.01,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.3
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.1",
									"liquiditySlaParameters": {
										"priceRange": "0.05",
										"commitmentMinTimeFraction": "0.95",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.9"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New BTC market",
							"title": "New BTC market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "40f22f1012e971d3532df04f4c9c15bbda43715dd46304612eeb6bd17d5eb3a4",
							"timestamp": "1699892059821916752",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "c2697de7929f37adf10b4e91a99fe4133f2f1d0209c6089ff8a41b6519214238",
						"reference": "1pyILXAJGEl8ScLcz5S1xjsCG9TY5lboX4EoUN1C",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892049382487724",
						"terms": {
							"closingTimestamp": "1699892073",
							"enactmentTimestamp": "1699892088",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "DAIUSD Perpetual",
										"code": "DAIUSD.PERP",
										"perpetual": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USD",
											"marginFundingFactor": "0.1",
											"interestRate": "0",
											"clampLowerBound": "0",
											"clampUpperBound": "0",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1699892088",
																"every": "3600"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x14866185B1962B63C3Ea9E03Bc1da838bab34C19",
														"abi": "[{\"inputs\":[],\"name\":\"latestAnswer\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
														"method": "latestAnswer",
														"trigger": {
															"timeTrigger": {
																"initial": "1699892088",
																"every": "30"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "dai.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "8"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "dai.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "11155111"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "dai.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"formerly:70657270646169757364",
										"base:DAI",
										"quote:USD",
										"class:fx/crypto",
										"monthly",
										"managed:vega/ops",
										"sector:crypto",
										"auto:perpetual_dai_usd"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "720",
												"probability": "0.99",
												"auctionExtension": "300"
											},
											{
												"horizon": "240",
												"probability": "0.99",
												"auctionExtension": "180"
											},
											{
												"horizon": "60",
												"probability": "0.99",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.0001,
										"tau": 0.00000380258,
										"params": {
											"sigma": 0.6
										}
									},
									"positionDecimalPlaces": "1",
									"linearSlippageFactor": "0.01",
									"liquiditySlaParameters": {
										"priceRange": "0.05",
										"commitmentMinTimeFraction": "0.95",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.9"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New DAIUSD perpetual market",
							"title": "New DAIUSD perpetual market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "c2697de7929f37adf10b4e91a99fe4133f2f1d0209c6089ff8a41b6519214238",
							"timestamp": "1699892059821916752",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "0738f83c527b3ac9df8bcf399017b5f74c781c7a3e41db7e18dfc473e75e1256",
						"reference": "PtSNgkbk9SFkQ96IO4rM8q4JD6a7xEA1lS3jCNzG",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892161313763986",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.VolumeDiscountProgram.minEnact",
									"value": "48h0m0s"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.VolumeDiscountProgram.minEnact to 48h0m0s from the previous value",
							"title": "Update governance.proposal.VolumeDiscountProgram.minEnact"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "0738f83c527b3ac9df8bcf399017b5f74c781c7a3e41db7e18dfc473e75e1256",
							"timestamp": "1699892175353756695",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "2d2c64fcf4abd68c4377b60f11484208613445668050365fd7ebb341a5422706",
						"reference": "djlWZOQ1HIwkspowu7zq06djmphpd5LfAAMJrPKV",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892161313763986",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "rewards.vesting.benefitTiers",
									"value": "{\"tiers\": [{\"minimum_quantum_balance\": \"10\", \"reward_multiplier\": \"1.05\"}, {\"minimum_quantum_balance\": \"100\", \"reward_multiplier\": \"1.10\"},{\"minimum_quantum_balance\": \"1000\", \"reward_multiplier\": \"1.15\"},{\"minimum_quantum_balance\": \"10000\", \"reward_multiplier\": \"1.20\"}]}"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of rewards.vesting.benefitTiers to {\"tiers\": [{\"minimum_quantum_balance\": \"10\", \"reward_multiplier\": \"1.05\"}, {\"minimum_quantum_balance\": \"100\", \"reward_multiplier\": \"1.10\"},{\"minimum_quantum_balance\": \"1000\", \"reward_multiplier\": \"1.15\"},{\"minimum_quantum_balance\": \"10000\", \"reward_multiplier\": \"1.20\"}]} from the previous value",
							"title": "Update rewards.vesting.benefitTiers"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "2d2c64fcf4abd68c4377b60f11484208613445668050365fd7ebb341a5422706",
							"timestamp": "1699892176699434033",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "e4f3bbd99f0bc0ea0075a27f8455adaea43cebabdde116f80b86de9e45d0ad32",
						"reference": "4drdr2fE6Bemsvbm7etbrnjiubgJGZ0MyinDsT5s",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892162840856585",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.transfer.minClose",
									"value": "1m"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.transfer.minClose to 1m from the previous value",
							"title": "Update governance.proposal.transfer.minClose"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "e4f3bbd99f0bc0ea0075a27f8455adaea43cebabdde116f80b86de9e45d0ad32",
							"timestamp": "1699892175988510843",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "6df8add0a3fcf6e96c64be1292c4ec1f789e0930b7124a58cd1dfeb48a16b9e9",
						"reference": "48JNP5mrSVJOmuAUZjb7Mi18vN0gizUjlPqMglVI",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892162840856585",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.referralProgram.requiredParticipation",
									"value": "0.00001"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.referralProgram.requiredParticipation to 0.00001 from the previous value",
							"title": "Update governance.proposal.referralProgram.requiredParticipation"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "6df8add0a3fcf6e96c64be1292c4ec1f789e0930b7124a58cd1dfeb48a16b9e9",
							"timestamp": "1699892176699434033",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "b826da73dcd90551915d41bb7c5f48790badee5f09e196224e47df9c00a1cb78",
						"reference": "0i2LXYNTLejQssvF7fh2AxjaJetKmSXNJ5ICruhb",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892162840856585",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.VolumeDiscountProgram.minClose",
									"value": "48h0m0s"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.VolumeDiscountProgram.minClose to 48h0m0s from the previous value",
							"title": "Update governance.proposal.VolumeDiscountProgram.minClose"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "b826da73dcd90551915d41bb7c5f48790badee5f09e196224e47df9c00a1cb78",
							"timestamp": "1699892175353756695",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "31a448202808c9fa5d09797014de8602f54920b01f8c0fad41693bf4134118b2",
						"reference": "UrvP7EIvGx5zRvtEc8MsoRiN1rJqPe4NoOzePL9R",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892162840856585",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.referralProgram.minClose",
									"value": "48h0m0s"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.referralProgram.minClose to 48h0m0s from the previous value",
							"title": "Update governance.proposal.referralProgram.minClose"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "31a448202808c9fa5d09797014de8602f54920b01f8c0fad41693bf4134118b2",
							"timestamp": "1699892175988510843",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "df4164186ac1af3fc7092e9ace5a6a761f938c8c61abdefc18a406e52de03462",
						"reference": "3vfYGjJOiWslfQPYRzk1AMzOBOObxIGgPngy5jrS",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892162840856585",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "rewards.activityStreak.minQuantumOpenVolume",
									"value": "100"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of rewards.activityStreak.minQuantumOpenVolume to 100 from the previous value",
							"title": "Update rewards.activityStreak.minQuantumOpenVolume"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "df4164186ac1af3fc7092e9ace5a6a761f938c8c61abdefc18a406e52de03462",
							"timestamp": "1699892175988510843",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "ad72ce243157f8c6368552fdc3a7b8dd9c2b02cce0d2f09341db87eeff654b2e",
						"reference": "5FoIA0wbfFPQBIQRTqcjmxxTNACe0froZ0R4ZxGM",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892162840856585",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.VolumeDiscountProgram.minVoterBalance",
									"value": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.VolumeDiscountProgram.minVoterBalance to 1 from the previous value",
							"title": "Update governance.proposal.VolumeDiscountProgram.minVoterBalance"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "ad72ce243157f8c6368552fdc3a7b8dd9c2b02cce0d2f09341db87eeff654b2e",
							"timestamp": "1699892176699434033",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "992e86bea7461f6163c1fa3337eff20e6701a32487bc86093132f70a476b9723",
						"reference": "5HImlJVXcLdo7Kw2O6ODM78dpYFVIfoSQMB0nTtk",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892162840856585",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.referralProgram.minEnact",
									"value": "48h0m0s"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.referralProgram.minEnact to 48h0m0s from the previous value",
							"title": "Update governance.proposal.referralProgram.minEnact"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "992e86bea7461f6163c1fa3337eff20e6701a32487bc86093132f70a476b9723",
							"timestamp": "1699892175988510843",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "46e5057ab8a855d72bea141c07baa671703c9838dcb54cf061a2b04d61e94a90",
						"reference": "Nmvl8ccyEXSKn6p2mATpaBPDmg1bUddn6lJlPiMQ",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892163467156559",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "referralProgram.maxReferralRewardFactor",
									"value": "0.2"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of referralProgram.maxReferralRewardFactor to 0.2 from the previous value",
							"title": "Update referralProgram.maxReferralRewardFactor"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "46e5057ab8a855d72bea141c07baa671703c9838dcb54cf061a2b04d61e94a90",
							"timestamp": "1699892175988510843",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "a9467c50b180e59360078ceabcdc6062fe43ce4144038da26cb6b491fff9380b",
						"reference": "80DB4Q2TiFh0jo7Gzp1uLG36tq0WEjnbTIJmOFDk",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892163467156559",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "validators.epoch.length",
									"value": "30m"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of validators.epoch.length to 30m from the previous value",
							"title": "Update validators.epoch.length"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "a9467c50b180e59360078ceabcdc6062fe43ce4144038da26cb6b491fff9380b",
							"timestamp": "1699892175988510843",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "75e3e2cf9c94e6b5a51a894d115f9e6b85a2d57be1a10baa1a2653b079c6b702",
						"reference": "pzgqsuUNQ0AsFagChkUPtZfE0DCtSBuRATWpEILb",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892163467156559",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "rewards.activityStreak.minQuantumTradeVolume",
									"value": "100"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of rewards.activityStreak.minQuantumTradeVolume to 100 from the previous value",
							"title": "Update rewards.activityStreak.minQuantumTradeVolume"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "75e3e2cf9c94e6b5a51a894d115f9e6b85a2d57be1a10baa1a2653b079c6b702",
							"timestamp": "1699892176699434033",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "dfc7f0d34e37e95578d787fb03d8d5960c7e7cbef2782631979805aba5e7df4c",
						"reference": "u344kspSYjSQTPCdpRDEsG9xULUkQJVcdAWFdiBw",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892163467156559",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "volumeDiscountProgram.maxVolumeDiscountFactor",
									"value": "0.4"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of volumeDiscountProgram.maxVolumeDiscountFactor to 0.4 from the previous value",
							"title": "Update volumeDiscountProgram.maxVolumeDiscountFactor"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "dfc7f0d34e37e95578d787fb03d8d5960c7e7cbef2782631979805aba5e7df4c",
							"timestamp": "1699892175353756695",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "4b83d728f58ff527cd58428cdd44678fd1e5bf1679e55bfa6579f6c1bc8915d2",
						"reference": "sPB4BmstH8aVJPsZVCo3meR0HKBqS7dJUWk29GLc",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892163467156559",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "rewards.activityStreak.inactivityLimit",
									"value": "24"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of rewards.activityStreak.inactivityLimit to 24 from the previous value",
							"title": "Update rewards.activityStreak.inactivityLimit"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "4b83d728f58ff527cd58428cdd44678fd1e5bf1679e55bfa6579f6c1bc8915d2",
							"timestamp": "1699892175988510843",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "7dc7ee42ff749fc3aa6c4f503a1e6d4cd99a8f7f083c8b15549da46187d015dc",
						"reference": "BEKGfJ3j3glZuIhEHWgRJcNwkjE4T0bfxj9zTdmV",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892163467156559",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.VolumeDiscountProgram.requiredParticipation",
									"value": "0.00001"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.VolumeDiscountProgram.requiredParticipation to 0.00001 from the previous value",
							"title": "Update governance.proposal.VolumeDiscountProgram.requiredParticipation"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "7dc7ee42ff749fc3aa6c4f503a1e6d4cd99a8f7f083c8b15549da46187d015dc",
							"timestamp": "1699892175988510843",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "772d25c22405a27d12357b35b2072e3c7b74d1991d1104663462c8e38d3e36f0",
						"reference": "3hWpk9Eu4Kdo7fXHbf7ZxLS0UaObC8qkOO102qIq",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892164118791476",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.VolumeDiscountProgram.minProposerBalance",
									"value": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.VolumeDiscountProgram.minProposerBalance to 1 from the previous value",
							"title": "Update governance.proposal.VolumeDiscountProgram.minProposerBalance"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "772d25c22405a27d12357b35b2072e3c7b74d1991d1104663462c8e38d3e36f0",
							"timestamp": "1699892175988510843",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "518a46fc477c053472d8bc92177735cf8b99cd6e2d1057329c4bf18f542626c6",
						"reference": "uNNT49qksASAEglHJIrRmIysX0suQivaHIYlNt0a",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892164118791476",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "rewards.vesting.baseRate",
									"value": "0.0055"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of rewards.vesting.baseRate to 0.0055 from the previous value",
							"title": "Update rewards.vesting.baseRate"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "518a46fc477c053472d8bc92177735cf8b99cd6e2d1057329c4bf18f542626c6",
							"timestamp": "1699892175988510843",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "dc493dbdbbdf356e904808cfb3f7006275be6e99694ebb6f4754266d5d711b03",
						"reference": "UxPt6YPWqsRnNwGliJPLVTtuvJIUp9VCYlvZC3gI",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892164118791476",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.transfer.minEnact",
									"value": "1m"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.transfer.minEnact to 1m from the previous value",
							"title": "Update governance.proposal.transfer.minEnact"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "dc493dbdbbdf356e904808cfb3f7006275be6e99694ebb6f4754266d5d711b03",
							"timestamp": "1699892176699434033",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "476c8849287be022cf2eb25786a7af72074fc80ad77a76af8ac1d1ddd8a4b1ca",
						"reference": "Hjcx9hV9tQUbUNlUOdVqHYH1z4I5LjZ5XqOnjiZF",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892164118791476",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.referralProgram.minVoterBalance",
									"value": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.referralProgram.minVoterBalance to 1 from the previous value",
							"title": "Update governance.proposal.referralProgram.minVoterBalance"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "476c8849287be022cf2eb25786a7af72074fc80ad77a76af8ac1d1ddd8a4b1ca",
							"timestamp": "1699892175353756695",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "c94d6e4ef33fd8d3948f0d8d79cf148f8522c9a9555d53e2febec00b3a7e3eba",
						"reference": "FaG2D75mvFiHNykpAKkqZlr8owDb998ZPPZSondD",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892164118791476",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "referralProgram.maxReferralDiscountFactor",
									"value": "0.1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of referralProgram.maxReferralDiscountFactor to 0.1 from the previous value",
							"title": "Update referralProgram.maxReferralDiscountFactor"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "c94d6e4ef33fd8d3948f0d8d79cf148f8522c9a9555d53e2febec00b3a7e3eba",
							"timestamp": "1699892175353756695",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "6821facc8060996b542f80dd94f1aead62ddfe42b31cf071fca087283bfd2dfb",
						"reference": "CRG9Bw7mpPbeejW9IBzzAX8DJFUpKSm5MqDY1CzQ",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892164118791476",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "rewards.activityStreak.benefitTiers",
									"value": "{\"tiers\": [{\"minimum_activity_streak\": 1, \"reward_multiplier\": \"1.05\", \"vesting_multiplier\": \"1.05\"}, {\"minimum_activity_streak\": 6, \"reward_multiplier\": \"1.10\", \"vesting_multiplier\": \"1.10\"}, {\"minimum_activity_streak\": 24, \"reward_multiplier\": \"1.10\", \"vesting_multiplier\": \"1.15\"}, {\"minimum_activity_streak\": 72, \"reward_multiplier\": \"1.20\", \"vesting_multiplier\": \"1.20\"}]}"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of rewards.activityStreak.benefitTiers to {\"tiers\": [{\"minimum_activity_streak\": 1, \"reward_multiplier\": \"1.05\", \"vesting_multiplier\": \"1.05\"}, {\"minimum_activity_streak\": 6, \"reward_multiplier\": \"1.10\", \"vesting_multiplier\": \"1.10\"}, {\"minimum_activity_streak\": 24, \"reward_multiplier\": \"1.10\", \"vesting_multiplier\": \"1.15\"}, {\"minimum_activity_streak\": 72, \"reward_multiplier\": \"1.20\", \"vesting_multiplier\": \"1.20\"}]} from the previous value",
							"title": "Update rewards.activityStreak.benefitTiers"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "6821facc8060996b542f80dd94f1aead62ddfe42b31cf071fca087283bfd2dfb",
							"timestamp": "1699892175988510843",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "51993a4cfc2f2d7a56c661c211d72e82da1a22ac6ef1e7f02df13a605083c3b2",
						"reference": "OTAQ8lCOSbCfPE5dmkYJNtpEEIbGsENmJ631eMuA",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699892164118791476",
						"terms": {
							"closingTimestamp": "1699892186",
							"enactmentTimestamp": "1699892201",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.referralProgram.minProposerBalance",
									"value": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.referralProgram.minProposerBalance to 1 from the previous value",
							"title": "Update governance.proposal.referralProgram.minProposerBalance"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "51993a4cfc2f2d7a56c661c211d72e82da1a22ac6ef1e7f02df13a605083c3b2",
							"timestamp": "1699892176699434033",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "a0b23dd16ccf749487b1c6cc2a3356edbd6638b9691250038a4805b0f15c0dc4",
						"reference": "p78e6hq3mRoAPY3LQSyiCO2mEYt3oAXAnEk3dkRu",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894298461410992",
						"terms": {
							"closingTimestamp": "1699894322",
							"enactmentTimestamp": "1699894337",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ETH/USD Perpetual",
										"code": "ETHEREUM.PERP",
										"perpetual": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDT",
											"marginFundingFactor": "0.95",
											"interestRate": "0",
											"clampLowerBound": "0",
											"clampUpperBound": "0",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1699894337",
																"every": "1800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x694AA1769357215DE4FAC081bf1f309aDC325306",
														"abi": "[{\"inputs\":[],\"name\":\"latestAnswer\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
														"method": "latestAnswer",
														"trigger": {
															"timeTrigger": {
																"initial": "1699894337",
																"every": "30"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "8"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "11155111"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "eth.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											}
										}
									},
									"decimalPlaces": "2",
									"metadata": [
										"formerly:70657270657468757364",
										"base:ETH",
										"quote:USD",
										"class:fx/crypto",
										"monthly",
										"managed:vega/ops",
										"sector:crypto",
										"auto:perpetual_eth_usd"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.000009506426342,
										"params": {
											"r": 0.016,
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "An Ethereum (ETH) Perpetual Market denominated in USD and settled in USDT",
							"title": "ETH/USD Perpetual"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "a0b23dd16ccf749487b1c6cc2a3356edbd6638b9691250038a4805b0f15c0dc4",
							"timestamp": "1699894307934515827",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "1c49476272bd5a5a3516bfecf5e12b48a121ce1061f34ba6a316397214056ab5",
						"reference": "hBknyl6SKXSjy2lhtYpuEVQEw7X4QDhudnOSg8k0",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894298461410992",
						"terms": {
							"closingTimestamp": "1699894322",
							"enactmentTimestamp": "1699894337",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "BTCUSDT Perp",
										"code": "BTCUSDT.PERP",
										"perpetual": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDT",
											"marginFundingFactor": "0.95",
											"interestRate": "0",
											"clampLowerBound": "0",
											"clampUpperBound": "0",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1699894337",
																"every": "1800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x1b44F3514812d835EB1BDB0acB33d3fA3351Ee43",
														"abi": "[{\"inputs\":[],\"name\":\"latestAnswer\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
														"method": "latestAnswer",
														"trigger": {
															"timeTrigger": {
																"initial": "1699894337",
																"every": "30"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "8"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "11155111"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "btc.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											}
										}
									},
									"decimalPlaces": "1",
									"metadata": [
										"formerly:50657270657475616c",
										"base:BTC",
										"quote:USD",
										"class:fx/crypto",
										"perpetual",
										"managed:vega/ops",
										"sector:crypto",
										"auto:perpetual_btc_usd"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "4320",
												"probability": "0.99",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.99",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.99",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.00000380258,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.015",
										"commitmentMinTimeFraction": "0.6",
										"slaCompetitionFactor": "0.2"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nThis proposal requests to list BTCUSDT Perp as a market with USDT as a settlement asset",
							"title": "BTCUSDT Perp"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "1c49476272bd5a5a3516bfecf5e12b48a121ce1061f34ba6a316397214056ab5",
							"timestamp": "1699894307934515827",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "5a9b394e734f849b90e650915259a47bbf15ef735df5dd6fa33d9636905f4ff1",
						"reference": "yF0o3B8PJE94IOfSzmhRPX5dDHFFIDiSCrxkMTYu",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894395621957908",
						"terms": {
							"closingTimestamp": "1699894420",
							"enactmentTimestamp": "1699894435",
							"updateNetworkParameter": {
								"changes": {
									"key": "referralProgram.maxReferralRewardFactor",
									"value": "0.02"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of referralProgram.maxReferralRewardFactor to 0.02 from the previous value",
							"title": "Update referralProgram.maxReferralRewardFactor"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "5a9b394e734f849b90e650915259a47bbf15ef735df5dd6fa33d9636905f4ff1",
							"timestamp": "1699894406894073428",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "bc8b242877e45feb11e821905d5247a8dd8962ab7bb2dab1f359cc01e96b6218",
						"reference": "Qhhl8FWfiTVLkx635kFTnfABNMeYKGqNokPXh6Aq",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894395621957908",
						"terms": {
							"closingTimestamp": "1699894420",
							"enactmentTimestamp": "1699894435",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.referralProgram.minEnact",
									"value": "5s"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.referralProgram.minEnact to 5s from the previous value",
							"title": "Update governance.proposal.referralProgram.minEnact"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "bc8b242877e45feb11e821905d5247a8dd8962ab7bb2dab1f359cc01e96b6218",
							"timestamp": "1699894406894073428",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "738249eebc12b958c8efb328eb02fd88a1f762626a1c50760a705381d67a15d2",
						"reference": "DPzNkU7BMQlJa3FLuYp3cWHNUQ5qzEv6YtcthghB",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894396401092407",
						"terms": {
							"closingTimestamp": "1699894420",
							"enactmentTimestamp": "1699894435",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.referralProgram.minClose",
									"value": "5s"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.referralProgram.minClose to 5s from the previous value",
							"title": "Update governance.proposal.referralProgram.minClose"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "738249eebc12b958c8efb328eb02fd88a1f762626a1c50760a705381d67a15d2",
							"timestamp": "1699894405373814084",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "d9c9961389cb39f83709b56532bdc69e1cd6bcc83cd4fa568f5ced25ed312e03",
						"reference": "HWitDArJwWJ24fdfPL1zO1di1AcRg3XA8UnO8Td8",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894396401092407",
						"terms": {
							"closingTimestamp": "1699894420",
							"enactmentTimestamp": "1699894435",
							"updateNetworkParameter": {
								"changes": {
									"key": "referralProgram.maxReferralDiscountFactor",
									"value": "0.02"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of referralProgram.maxReferralDiscountFactor to 0.02 from the previous value",
							"title": "Update referralProgram.maxReferralDiscountFactor"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "d9c9961389cb39f83709b56532bdc69e1cd6bcc83cd4fa568f5ced25ed312e03",
							"timestamp": "1699894405373814084",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "70c76b55efd728655105a9abaaf8c1ec85cdb66e22572b5cd23dd172c01099f2",
						"reference": "8buXIBeSgZKZNPBdRcCGFmrEGAYbw8jA4BZnanUj",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894440391086263",
						"terms": {
							"closingTimestamp": "1699894465",
							"enactmentTimestamp": "1699894480",
							"updateReferralProgram": {
								"changes": {
									"benefitTiers": [
										{
											"minimumRunningNotionalTakerVolume": "10000",
											"minimumEpochs": "1",
											"referralRewardFactor": "0.001",
											"referralDiscountFactor": "0.001"
										},
										{
											"minimumRunningNotionalTakerVolume": "500000",
											"minimumEpochs": "6",
											"referralRewardFactor": "0.005",
											"referralDiscountFactor": "0.005"
										},
										{
											"minimumRunningNotionalTakerVolume": "1000000",
											"minimumEpochs": "24",
											"referralRewardFactor": "0.01",
											"referralDiscountFactor": "0.01"
										}
									],
									"endOfProgramTimestamp": "1794502440",
									"windowLength": "3",
									"stakingTiers": [
										{
											"minimumStakedTokens": "1",
											"referralRewardMultiplier": "1"
										},
										{
											"minimumStakedTokens": "2",
											"referralRewardMultiplier": "2"
										},
										{
											"minimumStakedTokens": "5",
											"referralRewardMultiplier": "3"
										}
									]
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nUpdates the referral program",
							"title": "Update the referral program"
						},
						"requiredParticipation": "0.00001",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "70c76b55efd728655105a9abaaf8c1ec85cdb66e22572b5cd23dd172c01099f2",
							"timestamp": "1699894451614488957",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "a932359ba060962a60648adf487e6b9377c160703f8aeb733c9f9f7e35a488d6",
						"reference": "oadRFe9CFR35VChkOt5tEynMne4busu6vhwNcvnw",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894507060017198",
						"terms": {
							"closingTimestamp": "1699894530",
							"enactmentTimestamp": "1699894545",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.VolumeDiscountProgram.minEnact",
									"value": "5s"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.VolumeDiscountProgram.minEnact to 5s from the previous value",
							"title": "Update governance.proposal.VolumeDiscountProgram.minEnact"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "a932359ba060962a60648adf487e6b9377c160703f8aeb733c9f9f7e35a488d6",
							"timestamp": "1699894515972972592",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "10ffc707dc5d6e33a91f7681d4ab3d5f0e8b07411fb50e4d72f3e1800ede7b2c",
						"reference": "Y2z0lzkdcSg8rbAYBcwD9Zbmc1CiJdZaVPbJccyz",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894507060017198",
						"terms": {
							"closingTimestamp": "1699894530",
							"enactmentTimestamp": "1699894545",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.VolumeDiscountProgram.minClose",
									"value": "5s"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.VolumeDiscountProgram.minClose to 5s from the previous value",
							"title": "Update governance.proposal.VolumeDiscountProgram.minClose"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "10ffc707dc5d6e33a91f7681d4ab3d5f0e8b07411fb50e4d72f3e1800ede7b2c",
							"timestamp": "1699894515972972592",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "fac1ccd328acb1024a426e38a31b24ccfdf67f555aaff1ffc403c17ce814aa1c",
						"reference": "aWTLpzJBRh0eJjkKo8pqeYVGXTTrPqnqklGYgVhz",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894551310113508",
						"terms": {
							"closingTimestamp": "1699894575",
							"enactmentTimestamp": "1699894590",
							"updateVolumeDiscountProgram": {
								"changes": {
									"benefitTiers": [
										{
											"minimumRunningNotionalTakerVolume": "10000",
											"volumeDiscountFactor": "0.05"
										},
										{
											"minimumRunningNotionalTakerVolume": "50000",
											"volumeDiscountFactor": "0.1"
										},
										{
											"minimumRunningNotionalTakerVolume": "100000",
											"volumeDiscountFactor": "0.15"
										},
										{
											"minimumRunningNotionalTakerVolume": "250000",
											"volumeDiscountFactor": "0.2"
										},
										{
											"minimumRunningNotionalTakerVolume": "500000",
											"volumeDiscountFactor": "0.25"
										},
										{
											"minimumRunningNotionalTakerVolume": "1000000",
											"volumeDiscountFactor": "0.3"
										},
										{
											"minimumRunningNotionalTakerVolume": "1500000",
											"volumeDiscountFactor": "0.35"
										},
										{
											"minimumRunningNotionalTakerVolume": "2000000",
											"volumeDiscountFactor": "0.4"
										}
									],
									"endOfProgramTimestamp": "1794502550",
									"windowLength": "7"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nUpdates the volume discount program",
							"title": "Update the Volume Discount program"
						},
						"requiredParticipation": "0.00001",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "fac1ccd328acb1024a426e38a31b24ccfdf67f555aaff1ffc403c17ce814aa1c",
							"timestamp": "1699894561744104950",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "c44759cb46f687a62bbdcb4814e3bdbe2dec581ef1b1138d460864b1c7945f07",
						"reference": "qR21An6M0RLzrNLjdRPGC8rExCKGWrQMkmCSuSkQ",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894702341510856",
						"terms": {
							"closingTimestamp": "1699894726",
							"enactmentTimestamp": "1699894741",
							"updateNetworkParameter": {
								"changes": {
									"key": "referralProgram.maxReferralDiscountFactor",
									"value": "0.1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of referralProgram.maxReferralDiscountFactor to 0.1 from the previous value",
							"title": "Update referralProgram.maxReferralDiscountFactor"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "c44759cb46f687a62bbdcb4814e3bdbe2dec581ef1b1138d460864b1c7945f07",
							"timestamp": "1699894712494706185",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "6385a98c953d1ef325a0615af2e9100ed498e75787a59dd8e2c24ba085888461",
						"reference": "Ygga4vC0lMLuqvqLo31REBeZEiUgWl8VLgkksprE",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894702341510856",
						"terms": {
							"closingTimestamp": "1699894726",
							"enactmentTimestamp": "1699894741",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.referralProgram.minClose",
									"value": "48h0m0s"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.referralProgram.minClose to 48h0m0s from the previous value",
							"title": "Update governance.proposal.referralProgram.minClose"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "6385a98c953d1ef325a0615af2e9100ed498e75787a59dd8e2c24ba085888461",
							"timestamp": "1699894713356497682",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "8863ec9708190714729eee4931347ce921f295feda4a0bf92e346ee35b987f1e",
						"reference": "rEU72aNTZAmv7WFR30qAbGWF8p5WvYFGvBqQnXK3",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894702341510856",
						"terms": {
							"closingTimestamp": "1699894726",
							"enactmentTimestamp": "1699894741",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.referralProgram.minEnact",
									"value": "48h0m0s"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.referralProgram.minEnact to 48h0m0s from the previous value",
							"title": "Update governance.proposal.referralProgram.minEnact"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "8863ec9708190714729eee4931347ce921f295feda4a0bf92e346ee35b987f1e",
							"timestamp": "1699894713356497682",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "87f6f15eb0c213cfc1dd792835c453fc31b558ea4866e53d59cfb62f2b571e4e",
						"reference": "tOvM4q9rwJPJsJAOmvGoG02qERHYsGknn3BocOgp",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894702341510856",
						"terms": {
							"closingTimestamp": "1699894726",
							"enactmentTimestamp": "1699894741",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.VolumeDiscountProgram.minClose",
									"value": "48h0m0s"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.VolumeDiscountProgram.minClose to 48h0m0s from the previous value",
							"title": "Update governance.proposal.VolumeDiscountProgram.minClose"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "87f6f15eb0c213cfc1dd792835c453fc31b558ea4866e53d59cfb62f2b571e4e",
							"timestamp": "1699894713356497682",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "1cc3cdda1e5b03b921b66a7b46f3dfcc989ee00cf39286a7c7db6350d6442455",
						"reference": "Xr74Ib3zIl94pVhepSOJWNeUsDVkxwbLJz0cXgO5",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894702341510856",
						"terms": {
							"closingTimestamp": "1699894726",
							"enactmentTimestamp": "1699894741",
							"updateNetworkParameter": {
								"changes": {
									"key": "governance.proposal.VolumeDiscountProgram.minEnact",
									"value": "48h0m0s"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of governance.proposal.VolumeDiscountProgram.minEnact to 48h0m0s from the previous value",
							"title": "Update governance.proposal.VolumeDiscountProgram.minEnact"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "1cc3cdda1e5b03b921b66a7b46f3dfcc989ee00cf39286a7c7db6350d6442455",
							"timestamp": "1699894713356497682",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "09a9caee2af56c26b2c95f5cd71e4e951801ed0136b259d8cf7a459e1b9e1107",
						"reference": "u4VLNJ5RZ0IocEGc0eNfnEKdgYWo03Xak8lz3gv8",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699894702969872688",
						"terms": {
							"closingTimestamp": "1699894726",
							"enactmentTimestamp": "1699894741",
							"updateNetworkParameter": {
								"changes": {
									"key": "referralProgram.maxReferralRewardFactor",
									"value": "0.2"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of referralProgram.maxReferralRewardFactor to 0.2 from the previous value",
							"title": "Update referralProgram.maxReferralRewardFactor"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "09a9caee2af56c26b2c95f5cd71e4e951801ed0136b259d8cf7a459e1b9e1107",
							"timestamp": "1699894712494706185",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "1935931527b56e3f6c0fedb81186e20180d94c5eeadfe94298ddbf43869265d1",
						"reference": "JGqjb4A3plK7FYVF1nfO2g0E1tXaSJ2hmlJWIV4X",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1699896173261983368",
						"terms": {
							"closingTimestamp": "1699896197",
							"enactmentTimestamp": "1699896212",
							"updateNetworkParameter": {
								"changes": {
									"key": "spam.protection.maxUserTransfersPerEpoch",
									"value": "803"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nChange value of spam.protection.maxUserTransfersPerEpoch to 803 from the previous value",
							"title": "Update spam.protection.maxUserTransfersPerEpoch"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "1935931527b56e3f6c0fedb81186e20180d94c5eeadfe94298ddbf43869265d1",
							"timestamp": "1699896183503319577",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "7e649b0cfc1f09d95ede5756b8951397bf95e9d13daf394574ecdafe7d6014e0",
						"reference": "Z24iNgZlVDzWLJ7iresz86zsYofHWzI8MFl8QhBy",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1703077284786817246",
						"terms": {
							"closingTimestamp": "1703079228",
							"enactmentTimestamp": "1703079328",
							"updateMarket": {
								"marketId": "40f22f1012e971d3532df04f4c9c15bbda43715dd46304612eeb6bd17d5eb3a4",
								"changes": {
									"instrument": {
										"code": "ETHBTC.QM21",
										"future": {
											"quoteName": "BTC",
											"dataSourceSpecForSettlementData": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "prices.ETH.value",
																	"type": "TYPE_INTEGER"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"oracle": {
														"signers": [
															{
																"pubKey": {
																	"key": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f"
																}
															}
														],
														"filters": [
															{
																"key": {
																	"name": "termination.ETH.value",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "1"
																	}
																]
															}
														]
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "prices.ETH.value",
												"tradingTerminationProperty": "termination.ETH.value"
											}
										}
									},
									"metadata": [
										"formerly:1F0BB6EB5703B099",
										"base:ETH",
										"quote:BTC",
										"class:fx/crypto",
										"quarterly",
										"managed:vega/ops",
										"sector:crypto",
										"auto:ethbtc",
										"some-metadata"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "600"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 10
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.01,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.3
										}
									},
									"linearSlippageFactor": "0.1",
									"liquiditySlaParameters": {
										"priceRange": "0.05",
										"commitmentMinTimeFraction": "0.95",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.9"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "New BTC market",
							"title": "New BTC market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "e4eb7a055934f23201fbf7d06cc5cd38c507ddad59699d8f93b232c3a7a23177",
							"value": "VALUE_YES",
							"proposalId": "7e649b0cfc1f09d95ede5756b8951397bf95e9d13daf394574ecdafe7d6014e0",
							"timestamp": "1703077306363982907",
							"totalGovernanceTokenBalance": "15000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0.9999999999999869"
						}
					]
				},
				{
					"proposal": {
						"id": "288adf344ca12ee2142316d142f6be068c208dc329f4798b1ac65a123250e48e",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707382772683060862",
						"terms": {
							"closingTimestamp": "1707383709",
							"enactmentTimestamp": "1707383809",
							"updateMarketState": {
								"changes": {
									"marketId": "74ea9b5f3441189dcd7b1fbff42f200bb77653cac76f14d21123a80267c2730b",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Terminate market",
							"title": "Terminate market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "288adf344ca12ee2142316d142f6be068c208dc329f4798b1ac65a123250e48e",
							"timestamp": "1707382983844690984",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "9e999f3b3153632012b554e7f8b9e0e42cd84ff1a0f0ab9bf671ef342022a8b2",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707382773628934288",
						"terms": {
							"closingTimestamp": "1707383709",
							"enactmentTimestamp": "1707383809",
							"updateMarketState": {
								"changes": {
									"marketId": "7cf3abfe9c3310de14dd613523caf8d52cd27a7bd7befb958693b4ec9abee6ae",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Terminate market",
							"title": "Terminate market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "9e999f3b3153632012b554e7f8b9e0e42cd84ff1a0f0ab9bf671ef342022a8b2",
							"timestamp": "1707382983844690984",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "bd9df32945bf886bd4242031bfea4938228a8f6f282ccb93eac5bad590d19bf4",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707382774791537287",
						"terms": {
							"closingTimestamp": "1707383709",
							"enactmentTimestamp": "1707383809",
							"updateMarketState": {
								"changes": {
									"marketId": "4ea4c96f3a498c2384fa5b3362fc97da157f5bc17260627c7d7c5ea92392da1c",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Terminate market",
							"title": "Terminate market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "bd9df32945bf886bd4242031bfea4938228a8f6f282ccb93eac5bad590d19bf4",
							"timestamp": "1707382985574975070",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "25d7d13f2123187f46381969376af022dc86f0792a7cd96400d30b090793b384",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707382775507359072",
						"terms": {
							"closingTimestamp": "1707383709",
							"enactmentTimestamp": "1707383809",
							"updateMarketState": {
								"changes": {
									"marketId": "7314703a4424f74e88869196e27beaac5bac5f4da86cdfe41b036bf7a66e9cfb",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Terminate market",
							"title": "Terminate market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "25d7d13f2123187f46381969376af022dc86f0792a7cd96400d30b090793b384",
							"timestamp": "1707382985574975070",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "718be2e694ef372a4fcf1b1431813f221eb2f940af81f2fb5f2bff4a5ffca5a1",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707382776203953923",
						"terms": {
							"closingTimestamp": "1707383709",
							"enactmentTimestamp": "1707383809",
							"updateMarketState": {
								"changes": {
									"marketId": "66aa425859126386468555ee489d9ef657e6d70061e01f70913500c9cb74a237",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Terminate market",
							"title": "Terminate market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "718be2e694ef372a4fcf1b1431813f221eb2f940af81f2fb5f2bff4a5ffca5a1",
							"timestamp": "1707382986338700765",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "040b9179f244e31574cde7907e0fa5844a05ed8910225527c9ad1903c7cabfc8",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707382776800458079",
						"terms": {
							"closingTimestamp": "1707383709",
							"enactmentTimestamp": "1707383809",
							"updateMarketState": {
								"changes": {
									"marketId": "40f22f1012e971d3532df04f4c9c15bbda43715dd46304612eeb6bd17d5eb3a4",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Terminate market",
							"title": "Terminate market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "040b9179f244e31574cde7907e0fa5844a05ed8910225527c9ad1903c7cabfc8",
							"timestamp": "1707382987201009169",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "a02ac2ae166ec22217b7d1bb973dcea6563e66b6c20766a6a4ee7c5b088d7273",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707382777431639985",
						"terms": {
							"closingTimestamp": "1707383709",
							"enactmentTimestamp": "1707383809",
							"updateMarketState": {
								"changes": {
									"marketId": "498a96130727446e28efb6df6b2da9590b369142216a31f9a844bee1ced2d3bd",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Terminate market",
							"title": "Terminate market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "a02ac2ae166ec22217b7d1bb973dcea6563e66b6c20766a6a4ee7c5b088d7273",
							"timestamp": "1707382988714082396",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "49514ae0fa09567283a3886cb8aa6ad0e0e5123c3778a654d4c1186e73fb788b",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707382778069785078",
						"terms": {
							"closingTimestamp": "1707383709",
							"enactmentTimestamp": "1707383809",
							"updateMarketState": {
								"changes": {
									"marketId": "a0b23dd16ccf749487b1c6cc2a3356edbd6638b9691250038a4805b0f15c0dc4",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Terminate market",
							"title": "Terminate market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "49514ae0fa09567283a3886cb8aa6ad0e0e5123c3778a654d4c1186e73fb788b",
							"timestamp": "1707382989565352570",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "7bcada7d263ecfdd1bf20253567900eda866821254ca3bc66c71c811c5cd7043",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707382778810065198",
						"terms": {
							"closingTimestamp": "1707383709",
							"enactmentTimestamp": "1707383809",
							"updateMarketState": {
								"changes": {
									"marketId": "1b0d91c8422f6dfb3037f0d2beea9b41a6045e42341b6b9deeefa956cead18fb",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Terminate market",
							"title": "Terminate market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "7bcada7d263ecfdd1bf20253567900eda866821254ca3bc66c71c811c5cd7043",
							"timestamp": "1707382990500772259",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "20c2007ec0f9a8c24a49469708cf3e80633b0e6ce240dcc76b053bc2ad8afe56",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707382779347803001",
						"terms": {
							"closingTimestamp": "1707383709",
							"enactmentTimestamp": "1707383809",
							"updateMarketState": {
								"changes": {
									"marketId": "c2697de7929f37adf10b4e91a99fe4133f2f1d0209c6089ff8a41b6519214238",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Terminate market",
							"title": "Terminate market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "20c2007ec0f9a8c24a49469708cf3e80633b0e6ce240dcc76b053bc2ad8afe56",
							"timestamp": "1707382990500772259",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "e331b4686c9ef8a60badcd47dcd9e628f3533dad82bd382e5fdc24bbe9e6fba3",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707382779939844996",
						"terms": {
							"closingTimestamp": "1707383709",
							"enactmentTimestamp": "1707383809",
							"updateMarketState": {
								"changes": {
									"marketId": "d5f5b8d3bdb4f918b86bc3e6609b3ded911b8b47ea4ca7170869dddee2eb2639",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Terminate market",
							"title": "Terminate market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "e331b4686c9ef8a60badcd47dcd9e628f3533dad82bd382e5fdc24bbe9e6fba3",
							"timestamp": "1707382991856526548",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "cd46f373e343af102fd40bed49f7434f55c5e4a3dcca653270518328cb17c559",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707382780551956057",
						"terms": {
							"closingTimestamp": "1707383709",
							"enactmentTimestamp": "1707383809",
							"updateMarketState": {
								"changes": {
									"marketId": "1c49476272bd5a5a3516bfecf5e12b48a121ce1061f34ba6a316397214056ab5",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Terminate market",
							"title": "Terminate market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "cd46f373e343af102fd40bed49f7434f55c5e4a3dcca653270518328cb17c559",
							"timestamp": "1707382992589530601",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "279b6e238a2af1e057e63e24bc5a503dea0f86036fb6e5a6adce7bf93dec0f64",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707387043461824855",
						"terms": {
							"closingTimestamp": "1707387627",
							"enactmentTimestamp": "1707387727",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "INJ/USDT-Perp",
										"code": "INJ/USDT-PERP",
										"perpetual": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1707387727",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x9eb2EBD260D82410592758B3091F74977E4A404c",
														"abi": "[{\"inputs\":[{\"internalType\":\"contract IUniswapV3Pool\",\"name\":\"pool\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"twapInterval\",\"type\":\"uint32\"}],\"name\":\"priceFromEthPoolInUsdt\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
														"method": "priceFromEthPoolInUsdt",
														"args": [
															{
																"stringValue": "0x6c063a6e8cd45869b5eb75291e65a3de298f3aa8"
															},
															{
																"numberValue": 300
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1707387727",
																"every": "300"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "inj.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "inj.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "11155111"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "inj.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											}
										}
									},
									"decimalPlaces": "3",
									"metadata": [
										"base:INJ",
										"quote:USDT",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-04T15:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "1",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nThis proposal requests to list INJ/USDT Perpetual as a market with USDT as a settlement asset",
							"title": "VMP-29 Create market INJ/USDT Perpetual"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "279b6e238a2af1e057e63e24bc5a503dea0f86036fb6e5a6adce7bf93dec0f64",
							"timestamp": "1707387170428409377",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "de0f57d1ec1e5f7868cc47bcab0aabf896a376e9db1c8bec83d00fda3cd53bd8",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707387043929797179",
						"terms": {
							"closingTimestamp": "1707387627",
							"enactmentTimestamp": "1707387727",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "LDO/USDT-Perp",
										"code": "LDO/USDT-PERP",
										"perpetual": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1707387727",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x9eb2EBD260D82410592758B3091F74977E4A404c",
														"abi": "[{\"inputs\":[{\"internalType\":\"contract IUniswapV3Pool\",\"name\":\"pool\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"twapInterval\",\"type\":\"uint32\"}],\"name\":\"priceFromEthPoolInUsdt\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
														"method": "priceFromEthPoolInUsdt",
														"args": [
															{
																"stringValue": "0xa3f558aebaecaf0e11ca4b2199cc5ed341edfd74"
															},
															{
																"numberValue": 300
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1707387727",
																"every": "300"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "ldo.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "ldo.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "11155111"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "ldo.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											}
										}
									},
									"decimalPlaces": "4",
									"metadata": [
										"base:LDO",
										"quote:USDT",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-04T15:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "1",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nThis proposal requests to list LDO/USDT Perpetual as a market with USDT as a settlement asset",
							"title": "VMP-28 Create market LDO/USDT Perpetual"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "de0f57d1ec1e5f7868cc47bcab0aabf896a376e9db1c8bec83d00fda3cd53bd8",
							"timestamp": "1707387171052298993",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "53d34c0b8cb45d0ff6232e605191b5d215fc8b0d949b2efb495dbf223448192d",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707387044563878593",
						"terms": {
							"closingTimestamp": "1707387627",
							"enactmentTimestamp": "1707387727",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "SNX/USDT-Perp",
										"code": "SNX/USDT-PERP",
										"perpetual": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1707387727",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x9eb2EBD260D82410592758B3091F74977E4A404c",
														"abi": "[{\"inputs\":[{\"internalType\":\"contract IUniswapV3Pool\",\"name\":\"pool\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"twapInterval\",\"type\":\"uint32\"}],\"name\":\"priceFromEthPoolInUsdt\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
														"method": "priceFromEthPoolInUsdt",
														"args": [
															{
																"stringValue": "0xede8dd046586d22625ae7ff2708f879ef7bdb8cf"
															},
															{
																"numberValue": 300
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1707387727",
																"every": "300"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "snx.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "snx.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "11155111"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "snx.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											}
										}
									},
									"decimalPlaces": "3",
									"metadata": [
										"base:SNX",
										"quote:USDT",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-03T14:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "1",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nThis proposal requests to list SNX/USDT Perpetual as a market with USDT as a settlement asset",
							"title": "VMP-26 Create market SNX/USDT Perpetual"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "53d34c0b8cb45d0ff6232e605191b5d215fc8b0d949b2efb495dbf223448192d",
							"timestamp": "1707387171793523420",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707392820351859025",
						"terms": {
							"closingTimestamp": "1707393126",
							"enactmentTimestamp": "1707393226",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "BTC/USD(USDT)-Perp",
										"code": "BTC/USD-PERP",
										"perpetual": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1707393226",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0xF4030086522a5bEEa4988F8cA5B36dbC97BeE88c",
														"abi": "[{\"inputs\":[],\"name\":\"latestAnswer\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
														"method": "latestAnswer",
														"trigger": {
															"timeTrigger": {
																"initial": "1707393226",
																"every": "300"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "8"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "11155111"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "btc.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											}
										}
									},
									"decimalPlaces": "1",
									"metadata": [
										"base:BTC",
										"quote:USDT",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2023-12-04T12:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.00000380258,
										"params": {
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "4",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nThis proposal requests to list BTC/USD Perpetual as a market with USDT as a settlement asset",
							"title": "VMP-13 Create market BTC/USD Perpetual"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
							"timestamp": "1707392899249459332",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1707392951160027077",
						"terms": {
							"closingTimestamp": "1707393226",
							"enactmentTimestamp": "1707393326",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "ETH/USD(USDT)-PERP",
										"code": "ETH/USD-PERP",
										"perpetual": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USD",
											"marginFundingFactor": "0.95",
											"interestRate": "0",
											"clampLowerBound": "0",
											"clampUpperBound": "0",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1707393326",
																"every": "1800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419",
														"abi": "[{\"inputs\":[],\"name\":\"latestAnswer\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
														"method": "latestAnswer",
														"trigger": {
															"timeTrigger": {
																"initial": "1707393326",
																"every": "30"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "8"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "11155111"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "eth.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											}
										}
									},
									"decimalPlaces": "2",
									"metadata": [
										"enactment:2023-11-19T02:00:00Z",
										"base:ETH",
										"quote:USD",
										"class:fx/crypto",
										"perpetual",
										"sector:defi"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 1
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.000009506426342,
										"params": {
											"r": 0.016,
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "3",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "This is a perpetual futures market for Ethereum (ETH) denominated in USD and settled in USDT.",
							"title": "VMP-12 - Create Market - ETH/USD(USDT)-PERP"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
							"timestamp": "1707392969213245211",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "3a662911e482d58c15b65c771fda88ecfcd41b24fe6d16477b4015aefcdc84dc",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708413923441344402",
						"terms": {
							"closingTimestamp": "1708414025",
							"enactmentTimestamp": "1708414065",
							"updateNetworkParameter": {
								"changes": {
									"key": "market.liquidity.probabilityOfTrading.tau.scaling",
									"value": "10"
								}
							}
						},
						"rationale": {
							"description": "Update market.liquidity.probabilityOfTrading.tau.scaling to 10",
							"title": "Update market.liquidity.probabilityOfTrading.tau.scaling to 10"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "3a662911e482d58c15b65c771fda88ecfcd41b24fe6d16477b4015aefcdc84dc",
							"timestamp": "1708413942509379131",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "3bf7c71497fc3fb79569a940698bc550a43ca592b3b737500a64a27896253874",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708420939898200117",
						"terms": {
							"closingTimestamp": "1708421293",
							"enactmentTimestamp": "1708421353",
							"updateMarket": {
								"marketId": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
								"changes": {
									"instrument": {
										"code": "BTCUSDT.PERP",
										"name": "Bitcoin / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708421353",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708421353",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "btc.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "1",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "btc.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "btc.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "btc.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:BTC",
										"quote:USD",
										"oracle:pyth",
										"chain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "4320",
												"probability": "0.99",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.99",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.99",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.00000380258,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.685",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "30",
										"disposalFraction": "0.1",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "5000000",
										"sourceWeights": [
											"0",
											"0",
											"0",
											"1"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "btc.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "e2bd2f75e07001b19aae12ce9b7dfb0613802c85264a3ff61fb347c23e0a0dce"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "e2bd2f75e07001b19aae12ce9b7dfb0613802c85264a3ff61fb347c23e0a0dce",
							"timestamp": "1708420977196394583",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "edbc9c6e257ccb4e987b9ae9c0935f204af8208b7aa3034a72d23bdddee01b71",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708420939898200117",
						"terms": {
							"closingTimestamp": "1708421293",
							"enactmentTimestamp": "1708421353",
							"updateMarket": {
								"marketId": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
								"changes": {
									"instrument": {
										"code": "ETHUSDT.PERP",
										"name": "Ether / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708421353",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708421353",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "eth.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "1",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "eth.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "eth.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "eth.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:ETH",
										"quote:USD",
										"oracle:pyth",
										"chain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "4320",
												"probability": "0.99",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.99",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.99",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.00000380258,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.685",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "30",
										"disposalFraction": "0.1",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "5000000",
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_MEDIAN",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "eth.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "e2bd2f75e07001b19aae12ce9b7dfb0613802c85264a3ff61fb347c23e0a0dce"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "e2bd2f75e07001b19aae12ce9b7dfb0613802c85264a3ff61fb347c23e0a0dce",
							"timestamp": "1708420977196394583",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "16d4315d4e85c3697a579f61c615c773d455735dca5e05f87cf03d1c97c1bfcf",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708420939898200117",
						"terms": {
							"closingTimestamp": "1708421293",
							"enactmentTimestamp": "1708421653",
							"updateMarketState": {
								"changes": {
									"marketId": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "e2bd2f75e07001b19aae12ce9b7dfb0613802c85264a3ff61fb347c23e0a0dce"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "e2bd2f75e07001b19aae12ce9b7dfb0613802c85264a3ff61fb347c23e0a0dce",
							"timestamp": "1708420977196394583",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "cec0929f99117d4095481320a2d961078510b1628001413fe7fe8ae3d57b4616",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708420939898200117",
						"terms": {
							"closingTimestamp": "1708421293",
							"enactmentTimestamp": "1708421653",
							"updateMarketState": {
								"changes": {
									"marketId": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "e2bd2f75e07001b19aae12ce9b7dfb0613802c85264a3ff61fb347c23e0a0dce"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "e2bd2f75e07001b19aae12ce9b7dfb0613802c85264a3ff61fb347c23e0a0dce",
							"timestamp": "1708420977196394583",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "604117a2b739c8ce0d5c593803bc427cbb390b9d5a133c5ce1de7e1004e14948",
						"reference": "some-reference1234",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708515136240038512",
						"terms": {
							"closingTimestamp": "1708515288",
							"enactmentTimestamp": "1708515388",
							"updateMarket": {
								"marketId": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
								"changes": {
									"instrument": {
										"code": "BTC/USDT",
										"name": "Bitcoin / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708515388",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708515388",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "btc.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "btc.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "btc.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "btc.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:BTC",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2023-12-01T18:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.000003995,
										"params": {
											"sigma": 1
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "50000000",
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_MEDIAN",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "btc.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c",
							"timestamp": "1708515192118400139",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "0.9998025782212833",
							"totalEquityLikeShareWeight": "0"
						}
					],
					"no": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_NO",
							"proposalId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c",
							"timestamp": "1708515214746954256",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "0.0001974217787167",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "e13ee1d32a955d2d6dc9ae27f54b3428ed9ef54964bc9c69c81f4a5af057cfc8",
						"reference": "some-reference1234",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708515136240038512",
						"terms": {
							"closingTimestamp": "1708515288",
							"enactmentTimestamp": "1708515388",
							"updateMarket": {
								"marketId": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
								"changes": {
									"instrument": {
										"code": "ETH/USDT",
										"name": "Ether / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708515388",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708515388",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "eth.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "eth.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "eth.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "eth.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:ETH",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2023-11-19T02:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.000009506426342,
										"params": {
											"r": 0.016,
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "50000000",
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_MEDIAN",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "eth.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c",
							"timestamp": "1708515192118400139",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "0.9998025782212833",
							"totalEquityLikeShareWeight": "0"
						}
					],
					"no": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_NO",
							"proposalId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c",
							"timestamp": "1708515214746954256",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "0.0001974217787167",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "eca14bf6cac541a584a0030bd785ec9bfa8c4d02b7e00eed5d9ca2549836010d",
						"reference": "some-reference1234",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708515136240038512",
						"terms": {
							"closingTimestamp": "1708515288",
							"enactmentTimestamp": "1708515388",
							"updateMarket": {
								"marketId": "279b6e238a2af1e057e63e24bc5a503dea0f86036fb6e5a6adce7bf93dec0f64",
								"changes": {
									"instrument": {
										"code": "INJ/USDT",
										"name": "Injective / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708515388",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "elvB0rVq0CkEjNY5ZLOtJ3bq34Eu3BpDoxQGy1S/9ZI="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708515388",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "inj.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "inj.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "inj.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "elvB0rVq0CkEjNY5ZLOtJ3bq34Eu3BpDoxQGy1S/9ZI="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "inj.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "inj.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "inj.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:INJ",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-04T15:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "0",
										"sourceWeights": [
											"0",
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "elvB0rVq0CkEjNY5ZLOtJ3bq34Eu3BpDoxQGy1S/9ZI="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "inj.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "inj.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "inj.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c",
							"timestamp": "1708515192118400139",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "0.9998025782212833",
							"totalEquityLikeShareWeight": "0"
						}
					],
					"no": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_NO",
							"proposalId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c",
							"timestamp": "1708515214746954256",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "0.0001974217787167",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "505f587e0fd7d50e5aa0ae4b454dd7f3a01b74c744a6a1c642304f51e3e5cc21",
						"reference": "some-reference1234",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708515136240038512",
						"terms": {
							"closingTimestamp": "1708515288",
							"enactmentTimestamp": "1708515388",
							"updateMarket": {
								"marketId": "53d34c0b8cb45d0ff6232e605191b5d215fc8b0d949b2efb495dbf223448192d",
								"changes": {
									"instrument": {
										"code": "SNX/USDT",
										"name": "Synthetix / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708515388",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "OdAg9gmC7Ykqu81KBqJ2qfm3v7zgAyBMEQtuSI9QLaM="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708515388",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "snx.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "snx.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "snx.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "OdAg9gmC7Ykqu81KBqJ2qfm3v7zgAyBMEQtuSI9QLaM="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "snx.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "snx.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "snx.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:SNX",
										"quote:USDT",
										"oracle:pyth",
										"oraclChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-03T14:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "0",
										"sourceWeights": [
											"0",
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "OdAg9gmC7Ykqu81KBqJ2qfm3v7zgAyBMEQtuSI9QLaM="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "snx.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "snx.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "snx.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c",
							"timestamp": "1708515192118400139",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "0.9998025782212833",
							"totalEquityLikeShareWeight": "0"
						}
					],
					"no": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_NO",
							"proposalId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c",
							"timestamp": "1708515214746954256",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "0.0001974217787167",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "d6672c6eb0306d87f8ac2afef8982d04997b5c3fb210de588c117124a49d0750",
						"reference": "some-reference1234",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708515136240038512",
						"terms": {
							"closingTimestamp": "1708515288",
							"enactmentTimestamp": "1708515388",
							"updateMarket": {
								"marketId": "de0f57d1ec1e5f7868cc47bcab0aabf896a376e9db1c8bec83d00fda3cd53bd8",
								"changes": {
									"instrument": {
										"code": "LDO/USDT",
										"name": "Lido / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708515388",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "xj4qfzegTl5hTAcji+2yXcw4kn+6j+iQWXpZPAsvpK0="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708515388",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "ldo.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "ldo.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "ldo.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "xj4qfzegTl5hTAcji+2yXcw4kn+6j+iQWXpZPAsvpK0="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "ldo.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "ldo.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "ldo.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:LDO",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-04T15:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "0",
										"sourceWeights": [
											"0",
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "xj4qfzegTl5hTAcji+2yXcw4kn+6j+iQWXpZPAsvpK0="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "ldo.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "ldo.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "ldo.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c",
							"timestamp": "1708515192118400139",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "0.9998025782212833",
							"totalEquityLikeShareWeight": "0"
						}
					],
					"no": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_NO",
							"proposalId": "53cb79c5b30e39ed810097bcb02940d32f36ca19af08c59eb562f973a1708a7c",
							"timestamp": "1708515214746954256",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "0.0001974217787167",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "3902a5c7c2ff151362f51814678c4d7bbb0d0381e49e56045095167f9e1b8662",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708530407237604637",
						"terms": {
							"closingTimestamp": "1708530495",
							"enactmentTimestamp": "1708530595",
							"updateNetworkParameter": {
								"changes": {
									"key": "blockchains.ethereumRpcAndEvmCompatDataSourcesConfig",
									"value": "{\"configs\":[{\"network_id\":\"100\",\"chain_id\":\"100\",\"confirmations\":3,\"name\":\"Gnosis Chain\", \"block_interval\": 4}, {\"network_id\":\"42161\",\"chain_id\":\"42161\",\"confirmations\":3,\"name\":\"Arbitrum One\", \"block_interval\": 100}]}"
								}
							}
						},
						"rationale": {
							"description": "As per title",
							"title": "Update L2 block intervals"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_YES",
							"proposalId": "3902a5c7c2ff151362f51814678c4d7bbb0d0381e49e56045095167f9e1b8662",
							"timestamp": "1708530437102588782",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "0.0001974217787167",
							"totalEquityLikeShareWeight": "0"
						},
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "3902a5c7c2ff151362f51814678c4d7bbb0d0381e49e56045095167f9e1b8662",
							"timestamp": "1708530443235537728",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "0.9998025782212833",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "ec1017a0d92a15d0e142ae036f1b327b78b240d0e48900957748f2dd223ff0fd",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708513105716700938",
						"terms": {
							"closingTimestamp": "1708534769",
							"enactmentTimestamp": "1708534869",
							"updateMarket": {
								"marketId": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
								"changes": {
									"instrument": {
										"code": "BTCUSDT.PERP",
										"name": "Bitcoin / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708534869",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708534869",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "btc.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "1",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "btc.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "btc.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "btc.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:BTC",
										"quote:USD",
										"oracle:pyth",
										"chain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "4320",
												"probability": "0.99",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.99",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.99",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.00000380258,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.685",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "30",
										"disposalFraction": "0.1",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "5000000",
										"sourceWeights": [
											"0",
											"0",
											"0",
											"1"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "btc.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "9fc3781600b7e97b42dbf00578f23811db15e89d81e4daea3a528b5e7a39947d"
					},
					"yes": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_YES",
							"proposalId": "9fc3781600b7e97b42dbf00578f23811db15e89d81e4daea3a528b5e7a39947d",
							"timestamp": "1708513484871312014",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "b9be9499eb206f333e7c06772a0db387984528496700876602a3a70a3bd96d06",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708513105716700938",
						"terms": {
							"closingTimestamp": "1708534769",
							"enactmentTimestamp": "1708534869",
							"updateMarket": {
								"marketId": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
								"changes": {
									"instrument": {
										"code": "ETHUSDT.PERP",
										"name": "Ether / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708534869",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708534869",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "eth.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "1",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "eth.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "eth.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "eth.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:ETH",
										"quote:USD",
										"oracle:pyth",
										"chain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "4320",
												"probability": "0.99",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.99",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.99",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.00000380258,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.685",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "30",
										"disposalFraction": "0.1",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "5000000",
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_MEDIAN",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\n    \"inputs\": [\n      {\n        \"internalType\": \"bytes32\",\n        \"name\": \"id\",\n        \"type\": \"bytes32\"\n      }\n    ],\n    \"name\": \"getPrice\",\n    \"outputs\": [\n      {\n        \"internalType\": \"int256\",\n        \"name\": \"\",\n        \"type\": \"int256\"\n      }\n    ],\n    \"stateMutability\": \"view\",\n    \"type\": \"function\"\n    }]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "eth.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "9fc3781600b7e97b42dbf00578f23811db15e89d81e4daea3a528b5e7a39947d"
					},
					"yes": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_YES",
							"proposalId": "9fc3781600b7e97b42dbf00578f23811db15e89d81e4daea3a528b5e7a39947d",
							"timestamp": "1708513484871312014",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "1365dc7a3f74349c477d912573a17f4ba598ee7f69baac19191ed9c5e9e9d9d2",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_FAILED",
						"timestamp": "1708513105716700938",
						"terms": {
							"closingTimestamp": "1708534769",
							"enactmentTimestamp": "1708535769",
							"updateMarketState": {
								"changes": {
									"marketId": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"reason": "PROPOSAL_ERROR_INVALID_MARKET_STATE_UPDATE",
						"errorDetails": "invalid state update request. Market for resume is not suspended",
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "9fc3781600b7e97b42dbf00578f23811db15e89d81e4daea3a528b5e7a39947d"
					},
					"yes": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_YES",
							"proposalId": "9fc3781600b7e97b42dbf00578f23811db15e89d81e4daea3a528b5e7a39947d",
							"timestamp": "1708513484871312014",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "f4df6c88858a3e203944ab40a246486ac306f481bd08a80f60169be4ece00575",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_FAILED",
						"timestamp": "1708513105716700938",
						"terms": {
							"closingTimestamp": "1708534769",
							"enactmentTimestamp": "1708535769",
							"updateMarketState": {
								"changes": {
									"marketId": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"reason": "PROPOSAL_ERROR_INVALID_MARKET_STATE_UPDATE",
						"errorDetails": "invalid state update request. Market for resume is not suspended",
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "9fc3781600b7e97b42dbf00578f23811db15e89d81e4daea3a528b5e7a39947d"
					},
					"yes": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_YES",
							"proposalId": "9fc3781600b7e97b42dbf00578f23811db15e89d81e4daea3a528b5e7a39947d",
							"timestamp": "1708513484871312014",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "64c2a4adead6637ee313efc2bb14fd693b5bd026690b81d7d293234bab1d9471",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708590252032290455",
						"terms": {
							"closingTimestamp": "1708590407",
							"enactmentTimestamp": "1708590427",
							"updateNetworkParameter": {
								"changes": {
									"key": "blockchains.ethereumRpcAndEvmCompatDataSourcesConfig",
									"value": "{\"configs\":[{\"network_id\":\"100\",\"chain_id\":\"100\",\"confirmations\":3,\"name\":\"Gnosis Chain\", \"block_interval\": 3}, {\"network_id\":\"42161\",\"chain_id\":\"42161\",\"confirmations\":3,\"name\":\"Arbitrum One\", \"block_interval\": 50}]}"
								}
							}
						},
						"rationale": {
							"description": "As per title",
							"title": "Update L2 block intervals"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "64c2a4adead6637ee313efc2bb14fd693b5bd026690b81d7d293234bab1d9471",
							"timestamp": "1708590287021703876",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "d191dc59660adec1a7353c69bdc96ab87895171d233d379632a2d31c4d1641d6",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708611156018439913",
						"terms": {
							"closingTimestamp": "1708611205",
							"enactmentTimestamp": "1708611305",
							"updateMarket": {
								"marketId": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
								"changes": {
									"instrument": {
										"code": "BTC/USDT",
										"name": "Bitcoin / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708611305",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708611305",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "btc.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "btc.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "btc.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "btc.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:BTC",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2023-12-01T18:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.000003995,
										"params": {
											"sigma": 1
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "50000000",
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_MEDIAN",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "btc.price"
											},
											{
												"priceSourceProperty": "btc.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "f6b6a87c39b18807447f8dbc33e8607e8bb43faf50e004568e08275547aef552"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "f6b6a87c39b18807447f8dbc33e8607e8bb43faf50e004568e08275547aef552",
							"timestamp": "1708611169603962615",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "e5e96711ccb0903433f8ef138a3e2ee36d76de6e849506de548386e262ca0fe9",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708611156018439913",
						"terms": {
							"closingTimestamp": "1708611205",
							"enactmentTimestamp": "1708611305",
							"updateMarket": {
								"marketId": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
								"changes": {
									"instrument": {
										"code": "ETH/USDT",
										"name": "Ether / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708611305",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708611305",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "eth.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "eth.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "eth.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "eth.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:ETH",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2023-11-19T02:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.000009506426342,
										"params": {
											"r": 0.016,
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "50000000",
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_MEDIAN",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "eth.price"
											},
											{
												"priceSourceProperty": "eth.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "f6b6a87c39b18807447f8dbc33e8607e8bb43faf50e004568e08275547aef552"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "f6b6a87c39b18807447f8dbc33e8607e8bb43faf50e004568e08275547aef552",
							"timestamp": "1708611169603962615",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "c740257efe6e930f6a79b733a066044713edaeb3f81a48782f4ac1de4fa66a10",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708611156018439913",
						"terms": {
							"closingTimestamp": "1708611205",
							"enactmentTimestamp": "1708611305",
							"updateMarket": {
								"marketId": "279b6e238a2af1e057e63e24bc5a503dea0f86036fb6e5a6adce7bf93dec0f64",
								"changes": {
									"instrument": {
										"code": "INJ/USDT",
										"name": "Injective / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708611305",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "elvB0rVq0CkEjNY5ZLOtJ3bq34Eu3BpDoxQGy1S/9ZI="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708611305",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "inj.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "inj.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "inj.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "elvB0rVq0CkEjNY5ZLOtJ3bq34Eu3BpDoxQGy1S/9ZI="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "inj.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "inj.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "inj.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:INJ",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-04T15:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "0",
										"sourceWeights": [
											"0",
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "elvB0rVq0CkEjNY5ZLOtJ3bq34Eu3BpDoxQGy1S/9ZI="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "inj.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "inj.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "inj.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "f6b6a87c39b18807447f8dbc33e8607e8bb43faf50e004568e08275547aef552"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "f6b6a87c39b18807447f8dbc33e8607e8bb43faf50e004568e08275547aef552",
							"timestamp": "1708611169603962615",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "99083ea3b150bd5e7babfcb17d86d2a30b8a5a579c4f1c4ee7ede6e5308f1297",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708611156018439913",
						"terms": {
							"closingTimestamp": "1708611205",
							"enactmentTimestamp": "1708611305",
							"updateMarket": {
								"marketId": "53d34c0b8cb45d0ff6232e605191b5d215fc8b0d949b2efb495dbf223448192d",
								"changes": {
									"instrument": {
										"code": "SNX/USDT",
										"name": "Synthetix / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708611305",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "OdAg9gmC7Ykqu81KBqJ2qfm3v7zgAyBMEQtuSI9QLaM="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708611305",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "snx.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "snx.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "snx.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "OdAg9gmC7Ykqu81KBqJ2qfm3v7zgAyBMEQtuSI9QLaM="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "snx.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "snx.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "snx.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:SNX",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-03T14:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "0",
										"sourceWeights": [
											"0",
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "OdAg9gmC7Ykqu81KBqJ2qfm3v7zgAyBMEQtuSI9QLaM="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "snx.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "snx.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "snx.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "f6b6a87c39b18807447f8dbc33e8607e8bb43faf50e004568e08275547aef552"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "f6b6a87c39b18807447f8dbc33e8607e8bb43faf50e004568e08275547aef552",
							"timestamp": "1708611169603962615",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "9bdbe5796683c35ea3d163f628d4b065002598a6f399399f905403b866d8c69a",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708611156018439913",
						"terms": {
							"closingTimestamp": "1708611205",
							"enactmentTimestamp": "1708611305",
							"updateMarket": {
								"marketId": "de0f57d1ec1e5f7868cc47bcab0aabf896a376e9db1c8bec83d00fda3cd53bd8",
								"changes": {
									"instrument": {
										"code": "LDO/USDT",
										"name": "Lido / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708611305",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "xj4qfzegTl5hTAcji+2yXcw4kn+6j+iQWXpZPAsvpK0="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708611305",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "ldo.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "ldo.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "ldo.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "xj4qfzegTl5hTAcji+2yXcw4kn+6j+iQWXpZPAsvpK0="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "ldo.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "ldo.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "ldo.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:LDO",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-04T15:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "0",
										"sourceWeights": [
											"0",
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "xj4qfzegTl5hTAcji+2yXcw4kn+6j+iQWXpZPAsvpK0="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "ldo.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "ldo.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "ldo.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "f6b6a87c39b18807447f8dbc33e8607e8bb43faf50e004568e08275547aef552"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "f6b6a87c39b18807447f8dbc33e8607e8bb43faf50e004568e08275547aef552",
							"timestamp": "1708611169603962615",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "dcfdccbd0dde103102f8c39b69546a94aa927af23ce63dca8a3d365e173ae93b",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708613489405479152",
						"terms": {
							"closingTimestamp": "1708613547",
							"enactmentTimestamp": "1708613647",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Solana / Tether USD (Perpetual)",
										"code": "SOL/USDT",
										"perpetual": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708613647",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "7w2Lb9os66QdoV1AldHaOSoNL47Qxse8D0z6yMKAtW0="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708613647",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "sol.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "sol.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "sol.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.01",
											"fundingRateUpperBound": "0.01",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "7w2Lb9os66QdoV1AldHaOSoNL47Qxse8D0z6yMKAtW0="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "sol.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "sol.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "sol.price"
													}
												]
											}
										}
									},
									"decimalPlaces": "3",
									"metadata": [
										"base:SOL",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-22T02:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.000009506426342,
										"params": {
											"r": 0.016,
											"sigma": 1.5
										}
									},
									"positionDecimalPlaces": "1",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.5"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "50000000",
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_MEDIAN",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "7w2Lb9os66QdoV1AldHaOSoNL47Qxse8D0z6yMKAtW0="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "sol.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "sol.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "7w2Lb9os66QdoV1AldHaOSoNL47Qxse8D0z6yMKAtW0="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "sol.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "sol.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "sol.price"
											},
											{
												"priceSourceProperty": "sol.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Create SOL/USDT market",
							"title": "Create SOL/USDT market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"batchId": "ba2f50e0d95e5ff77a95c84ac4c63949895f7d17078ec14602c8eca900f5cc96"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "ba2f50e0d95e5ff77a95c84ac4c63949895f7d17078ec14602c8eca900f5cc96",
							"timestamp": "1708613504511444455",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "02cf075abf8af872f05352a3606dd3772f6171231c0d60f82081e649c308fa55",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708697033649737018",
						"terms": {
							"closingTimestamp": "1708697065",
							"enactmentTimestamp": "1708697165",
							"updateMarketState": {
								"changes": {
									"marketId": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
									"updateType": "MARKET_STATE_UPDATE_TYPE_SUSPEND"
								}
							}
						},
						"rationale": {
							"description": "Suspend markets\n\t- BTC/USDT\n\t- SNX/USDT\n\t- LDO/USDT\n\t- ETH/USDT\n\t- BTC/USDT\n\t- SOL/USDT",
							"title": "Suspend markets for mainnet proposal testing"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "9244935c5c4b36751648f619dc2cbe40136af165746ff47cb8b696cbda55d92d"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "9244935c5c4b36751648f619dc2cbe40136af165746ff47cb8b696cbda55d92d",
							"timestamp": "1708697058734103887",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "96e32c128ae1d04324d841bb396a67f2244f208d4e628534086acda3de551c14",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708697033649737018",
						"terms": {
							"closingTimestamp": "1708697065",
							"enactmentTimestamp": "1708697165",
							"updateMarketState": {
								"changes": {
									"marketId": "53d34c0b8cb45d0ff6232e605191b5d215fc8b0d949b2efb495dbf223448192d",
									"updateType": "MARKET_STATE_UPDATE_TYPE_SUSPEND"
								}
							}
						},
						"rationale": {
							"description": "Suspend markets\n\t- BTC/USDT\n\t- SNX/USDT\n\t- LDO/USDT\n\t- ETH/USDT\n\t- BTC/USDT\n\t- SOL/USDT",
							"title": "Suspend markets for mainnet proposal testing"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "9244935c5c4b36751648f619dc2cbe40136af165746ff47cb8b696cbda55d92d"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "9244935c5c4b36751648f619dc2cbe40136af165746ff47cb8b696cbda55d92d",
							"timestamp": "1708697058734103887",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "97ed4683c58423a0db7426a0b729416a5059c417ac59a142d09c68c22b6bacf0",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708697033649737018",
						"terms": {
							"closingTimestamp": "1708697065",
							"enactmentTimestamp": "1708697165",
							"updateMarketState": {
								"changes": {
									"marketId": "de0f57d1ec1e5f7868cc47bcab0aabf896a376e9db1c8bec83d00fda3cd53bd8",
									"updateType": "MARKET_STATE_UPDATE_TYPE_SUSPEND"
								}
							}
						},
						"rationale": {
							"description": "Suspend markets\n\t- BTC/USDT\n\t- SNX/USDT\n\t- LDO/USDT\n\t- ETH/USDT\n\t- BTC/USDT\n\t- SOL/USDT",
							"title": "Suspend markets for mainnet proposal testing"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "9244935c5c4b36751648f619dc2cbe40136af165746ff47cb8b696cbda55d92d"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "9244935c5c4b36751648f619dc2cbe40136af165746ff47cb8b696cbda55d92d",
							"timestamp": "1708697058734103887",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "7e079038a57e6fb1f3ccf5eee56e73596b630d54a130a565e1a2deebcc95c173",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708697033649737018",
						"terms": {
							"closingTimestamp": "1708697065",
							"enactmentTimestamp": "1708697165",
							"updateMarketState": {
								"changes": {
									"marketId": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
									"updateType": "MARKET_STATE_UPDATE_TYPE_SUSPEND"
								}
							}
						},
						"rationale": {
							"description": "Suspend markets\n\t- BTC/USDT\n\t- SNX/USDT\n\t- LDO/USDT\n\t- ETH/USDT\n\t- BTC/USDT\n\t- SOL/USDT",
							"title": "Suspend markets for mainnet proposal testing"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "9244935c5c4b36751648f619dc2cbe40136af165746ff47cb8b696cbda55d92d"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "9244935c5c4b36751648f619dc2cbe40136af165746ff47cb8b696cbda55d92d",
							"timestamp": "1708697058734103887",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "c8d389067fcc34a8aa73024585a9c23af8a2a7c6e9d6546dac8107b6a76a5fef",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708697033649737018",
						"terms": {
							"closingTimestamp": "1708697065",
							"enactmentTimestamp": "1708697165",
							"updateMarketState": {
								"changes": {
									"marketId": "dcfdccbd0dde103102f8c39b69546a94aa927af23ce63dca8a3d365e173ae93b",
									"updateType": "MARKET_STATE_UPDATE_TYPE_SUSPEND"
								}
							}
						},
						"rationale": {
							"description": "Suspend markets\n\t- BTC/USDT\n\t- SNX/USDT\n\t- LDO/USDT\n\t- ETH/USDT\n\t- BTC/USDT\n\t- SOL/USDT",
							"title": "Suspend markets for mainnet proposal testing"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "9244935c5c4b36751648f619dc2cbe40136af165746ff47cb8b696cbda55d92d"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "9244935c5c4b36751648f619dc2cbe40136af165746ff47cb8b696cbda55d92d",
							"timestamp": "1708697058734103887",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "d141945ceb47cc519dc62b17bf8323911091c789f4b8601fd5ab5b8618233c60",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708698056341644342",
						"terms": {
							"closingTimestamp": "1708698554",
							"enactmentTimestamp": "1708698604",
							"updateMarket": {
								"marketId": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
								"changes": {
									"instrument": {
										"code": "BTC/USDT",
										"name": "Bitcoin / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708698604",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708698604",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "btc.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.001",
											"fundingRateUpperBound": "0.001",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "btc.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "btc.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "btc.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:BTC",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2023-12-01T18:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.000003995,
										"params": {
											"sigma": 1
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.75",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "50000000",
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_MEDIAN",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "btc.price"
											},
											{
												"priceSourceProperty": "btc.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "65cac1cf2b58f7a9d2e24cca31054c0946dfda81666876b61a33aba36ca3b9ff"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "65cac1cf2b58f7a9d2e24cca31054c0946dfda81666876b61a33aba36ca3b9ff",
							"timestamp": "1708698317624339309",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "df14158b9c28334230cedebd8975b7085268e244ec0e495441297d3269c86199",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708698056341644342",
						"terms": {
							"closingTimestamp": "1708698554",
							"enactmentTimestamp": "1708698604",
							"updateMarket": {
								"marketId": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
								"changes": {
									"instrument": {
										"code": "ETH/USDT",
										"name": "Ether / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708698604",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708698604",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "eth.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.001",
											"fundingRateUpperBound": "0.001",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "eth.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "eth.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "eth.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:ETH",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2023-11-19T02:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.000009506426342,
										"params": {
											"r": 0.016,
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.75",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "50000000",
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_MEDIAN",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "eth.price"
											},
											{
												"priceSourceProperty": "eth.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "65cac1cf2b58f7a9d2e24cca31054c0946dfda81666876b61a33aba36ca3b9ff"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "65cac1cf2b58f7a9d2e24cca31054c0946dfda81666876b61a33aba36ca3b9ff",
							"timestamp": "1708698317624339309",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "0e452adbbae3165ce4000bfce13b214deb9326ac6b97f2baa455d13a8e26842e",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708698056341644342",
						"terms": {
							"closingTimestamp": "1708698554",
							"enactmentTimestamp": "1708698604",
							"updateMarket": {
								"marketId": "279b6e238a2af1e057e63e24bc5a503dea0f86036fb6e5a6adce7bf93dec0f64",
								"changes": {
									"instrument": {
										"code": "INJ/USDT",
										"name": "Injective / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708698604",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "elvB0rVq0CkEjNY5ZLOtJ3bq34Eu3BpDoxQGy1S/9ZI="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708698604",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "inj.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "inj.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "inj.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.001",
											"fundingRateUpperBound": "0.001",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "elvB0rVq0CkEjNY5ZLOtJ3bq34Eu3BpDoxQGy1S/9ZI="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "inj.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "inj.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "inj.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:INJ",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-04T15:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.75",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "0",
										"sourceWeights": [
											"0",
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "elvB0rVq0CkEjNY5ZLOtJ3bq34Eu3BpDoxQGy1S/9ZI="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "inj.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "inj.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "inj.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "65cac1cf2b58f7a9d2e24cca31054c0946dfda81666876b61a33aba36ca3b9ff"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "65cac1cf2b58f7a9d2e24cca31054c0946dfda81666876b61a33aba36ca3b9ff",
							"timestamp": "1708698317624339309",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "06f95b6116a956b18b081f1fb84b97809736fb1bb720034544dfe3257c6732dc",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708698056341644342",
						"terms": {
							"closingTimestamp": "1708698554",
							"enactmentTimestamp": "1708698604",
							"updateMarket": {
								"marketId": "53d34c0b8cb45d0ff6232e605191b5d215fc8b0d949b2efb495dbf223448192d",
								"changes": {
									"instrument": {
										"code": "SNX/USDT",
										"name": "Synthetix / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708698604",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "OdAg9gmC7Ykqu81KBqJ2qfm3v7zgAyBMEQtuSI9QLaM="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708698604",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "snx.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "snx.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "snx.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.001",
											"fundingRateUpperBound": "0.001",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "OdAg9gmC7Ykqu81KBqJ2qfm3v7zgAyBMEQtuSI9QLaM="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "snx.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "snx.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "snx.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:SNX",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-03T14:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.75",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "0",
										"sourceWeights": [
											"0",
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "OdAg9gmC7Ykqu81KBqJ2qfm3v7zgAyBMEQtuSI9QLaM="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "snx.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "snx.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "snx.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "65cac1cf2b58f7a9d2e24cca31054c0946dfda81666876b61a33aba36ca3b9ff"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "65cac1cf2b58f7a9d2e24cca31054c0946dfda81666876b61a33aba36ca3b9ff",
							"timestamp": "1708698317624339309",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "ce885bb8c978a348ec70e870d8a2f857312640bec44ab49ce256f839bc540c04",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708698056341644342",
						"terms": {
							"closingTimestamp": "1708698554",
							"enactmentTimestamp": "1708698604",
							"updateMarket": {
								"marketId": "de0f57d1ec1e5f7868cc47bcab0aabf896a376e9db1c8bec83d00fda3cd53bd8",
								"changes": {
									"instrument": {
										"code": "LDO/USDT",
										"name": "Lido / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708698604",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "xj4qfzegTl5hTAcji+2yXcw4kn+6j+iQWXpZPAsvpK0="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708698604",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "ldo.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "ldo.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "ldo.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "0.00001",
											"fundingRateLowerBound": "-0.001",
											"fundingRateUpperBound": "0.001",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "xj4qfzegTl5hTAcji+2yXcw4kn+6j+iQWXpZPAsvpK0="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "ldo.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "ldo.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "ldo.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:LDO",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-04T15:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.75",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "0",
										"sourceWeights": [
											"0",
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "xj4qfzegTl5hTAcji+2yXcw4kn+6j+iQWXpZPAsvpK0="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "ldo.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "ldo.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "ldo.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD",
							"title": "Update Markets to support new v0.74.x features"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "65cac1cf2b58f7a9d2e24cca31054c0946dfda81666876b61a33aba36ca3b9ff"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "65cac1cf2b58f7a9d2e24cca31054c0946dfda81666876b61a33aba36ca3b9ff",
							"timestamp": "1708698317624339309",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "f869f287ab8e8fb6cf2bccb9fb17db2c19e1841f7532b29b15a769e91d7076cf",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708699084520350553",
						"terms": {
							"closingTimestamp": "1708699354",
							"enactmentTimestamp": "1708699384",
							"updateMarketState": {
								"changes": {
									"marketId": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba",
							"timestamp": "1708699106158721669",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "016e48cfd05d44d09d5b40d62dfdf8e811c0bcb62a920090fe954c7c6db03c93",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708699084520350553",
						"terms": {
							"closingTimestamp": "1708699354",
							"enactmentTimestamp": "1708699384",
							"updateMarketState": {
								"changes": {
									"marketId": "279b6e238a2af1e057e63e24bc5a503dea0f86036fb6e5a6adce7bf93dec0f64",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba",
							"timestamp": "1708699106158721669",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "70bd06b9c74cfc3c28c70afa9737671c1f2f09000de9a2701e826a3afc4b51a8",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708699084520350553",
						"terms": {
							"closingTimestamp": "1708699354",
							"enactmentTimestamp": "1708699384",
							"updateMarketState": {
								"changes": {
									"marketId": "53d34c0b8cb45d0ff6232e605191b5d215fc8b0d949b2efb495dbf223448192d",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba",
							"timestamp": "1708699106158721669",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "fd91e2a05385ac766e69c07879678cf077b5ad0b67da7ee638bb0b3b7379e369",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708699084520350553",
						"terms": {
							"closingTimestamp": "1708699354",
							"enactmentTimestamp": "1708699384",
							"updateMarketState": {
								"changes": {
									"marketId": "de0f57d1ec1e5f7868cc47bcab0aabf896a376e9db1c8bec83d00fda3cd53bd8",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba",
							"timestamp": "1708699106158721669",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "14f05595dcdeea344dcd41d840e0354b6310401dff97e3ad7cd0e8ef5140dfac",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708699084520350553",
						"terms": {
							"closingTimestamp": "1708699354",
							"enactmentTimestamp": "1708699384",
							"updateMarketState": {
								"changes": {
									"marketId": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba",
							"timestamp": "1708699106158721669",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "9347918290bb33fa1b1c162408c0713147cf81e159615e37b639a39ab6b2e9ae",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708699084520350553",
						"terms": {
							"closingTimestamp": "1708699354",
							"enactmentTimestamp": "1708699384",
							"updateMarket": {
								"marketId": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
								"changes": {
									"instrument": {
										"code": "BTC/USDT",
										"name": "Bitcoin / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708699384",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708699384",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "btc.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "1",
											"fundingRateLowerBound": "-0.001",
											"fundingRateUpperBound": "0.001",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "btc.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "btc.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "btc.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:BTC",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2023-12-01T18:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.000003995,
										"params": {
											"sigma": 1
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.75",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "50000000",
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_MEDIAN",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "5i32yLSoX+GmfbRNwS3l2zMPesZrctxliv7fD0pBW0M="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "btc.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "btc.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "btc.price"
											},
											{
												"priceSourceProperty": "btc.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba",
							"timestamp": "1708699106158721669",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "01bd99739f551dda693b7537da85cecb12230c32efc2d71461dafc15539e44e5",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708699084520350553",
						"terms": {
							"closingTimestamp": "1708699354",
							"enactmentTimestamp": "1708699384",
							"updateMarket": {
								"marketId": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
								"changes": {
									"instrument": {
										"code": "ETH/USDT",
										"name": "Ether / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708699384",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708699384",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "eth.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "1",
											"fundingRateLowerBound": "-0.001",
											"fundingRateUpperBound": "0.001",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "eth.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "eth.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "eth.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:ETH",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2023-11-19T02:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.000009506426342,
										"params": {
											"r": 0.016,
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.75",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "50000000",
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_MEDIAN",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "/2FJGpMREt3xvYFHzRtkE3X3n1glEm1mVICHRjT9Cs4="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "eth.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "eth.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "eth.price"
											},
											{
												"priceSourceProperty": "eth.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba",
							"timestamp": "1708699106158721669",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "f6c2f6697bcfb98099cc77d1d6bb56947ca1984f68ef7133c33a672e141c5f0c",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708699084520350553",
						"terms": {
							"closingTimestamp": "1708699354",
							"enactmentTimestamp": "1708699384",
							"updateMarket": {
								"marketId": "279b6e238a2af1e057e63e24bc5a503dea0f86036fb6e5a6adce7bf93dec0f64",
								"changes": {
									"instrument": {
										"code": "INJ/USDT",
										"name": "Injective / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708699384",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "elvB0rVq0CkEjNY5ZLOtJ3bq34Eu3BpDoxQGy1S/9ZI="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708699384",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "inj.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "inj.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "inj.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "1",
											"fundingRateLowerBound": "-0.001",
											"fundingRateUpperBound": "0.001",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "elvB0rVq0CkEjNY5ZLOtJ3bq34Eu3BpDoxQGy1S/9ZI="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "inj.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "inj.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "inj.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:INJ",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-04T15:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.75",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "0",
										"sourceWeights": [
											"0",
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "elvB0rVq0CkEjNY5ZLOtJ3bq34Eu3BpDoxQGy1S/9ZI="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "inj.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "inj.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "inj.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba",
							"timestamp": "1708699106158721669",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "cea4ef428dffd3a21b6ca18eb22677df5b0bcd2519fd3770b8512f0753bba854",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708699084520350553",
						"terms": {
							"closingTimestamp": "1708699354",
							"enactmentTimestamp": "1708699384",
							"updateMarket": {
								"marketId": "53d34c0b8cb45d0ff6232e605191b5d215fc8b0d949b2efb495dbf223448192d",
								"changes": {
									"instrument": {
										"code": "SNX/USDT",
										"name": "Synthetix / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708699384",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "OdAg9gmC7Ykqu81KBqJ2qfm3v7zgAyBMEQtuSI9QLaM="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708699384",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "snx.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "snx.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "snx.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "1",
											"fundingRateLowerBound": "-0.001",
											"fundingRateUpperBound": "0.001",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "OdAg9gmC7Ykqu81KBqJ2qfm3v7zgAyBMEQtuSI9QLaM="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "snx.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "snx.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "snx.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:SNX",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-03T14:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.75",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "0",
										"sourceWeights": [
											"0",
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "OdAg9gmC7Ykqu81KBqJ2qfm3v7zgAyBMEQtuSI9QLaM="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "snx.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "snx.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "snx.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba",
							"timestamp": "1708699106158721669",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "aedc2873b02d71714a852dc1a6ac91012c1a4711d4a8126ac0f7fc5de747010d",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708699084520350553",
						"terms": {
							"closingTimestamp": "1708699354",
							"enactmentTimestamp": "1708699384",
							"updateMarket": {
								"marketId": "de0f57d1ec1e5f7868cc47bcab0aabf896a376e9db1c8bec83d00fda3cd53bd8",
								"changes": {
									"instrument": {
										"code": "LDO/USDT",
										"name": "Lido / Tether USD (Perpetual)",
										"perpetual": {
											"quoteName": "USDT",
											"marginFundingFactor": "0.9",
											"interestRate": "0.1095",
											"clampLowerBound": "-0.0005",
											"clampUpperBound": "0.0005",
											"dataSourceSpecForSettlementSchedule": {
												"internal": {
													"timeTrigger": {
														"conditions": [
															{
																"operator": "OPERATOR_GREATER_THAN",
																"value": "0"
															}
														],
														"triggers": [
															{
																"initial": "1708699384",
																"every": "28800"
															}
														]
													}
												}
											},
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "xj4qfzegTl5hTAcji+2yXcw4kn+6j+iQWXpZPAsvpK0="
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708699384",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "ldo.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "ldo.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "ldo.price",
												"settlementScheduleProperty": "vegaprotocol.builtin.timetrigger"
											},
											"fundingRateScalingFactor": "1",
											"fundingRateLowerBound": "-0.001",
											"fundingRateUpperBound": "0.001",
											"internalCompositePriceConfiguration": {
												"decayWeight": "1",
												"decayPower": "1",
												"cashAmount": "50000000",
												"sourceWeights": [
													"0",
													"0.999",
													"0.001",
													"0"
												],
												"sourceStalenessTolerance": [
													"1m0s",
													"1m0s",
													"10m0s",
													"10m0s"
												],
												"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
												"dataSourcesSpec": [
													{
														"external": {
															"ethOracle": {
																"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
																"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
																"method": "getPrice",
																"args": [
																	{
																		"stringValue": "xj4qfzegTl5hTAcji+2yXcw4kn+6j+iQWXpZPAsvpK0="
																	}
																],
																"trigger": {
																	"timeTrigger": {
																		"every": "60"
																	}
																},
																"requiredConfirmations": "3",
																"filters": [
																	{
																		"key": {
																			"name": "ldo.price",
																			"type": "TYPE_INTEGER",
																			"numberDecimalPlaces": "18"
																		},
																		"conditions": [
																			{
																				"operator": "OPERATOR_GREATER_THAN",
																				"value": "0"
																			}
																		]
																	}
																],
																"normalisers": [
																	{
																		"name": "ldo.price",
																		"expression": "$[0]"
																	}
																],
																"sourceChainId": "100"
															}
														}
													}
												],
												"dataSourcesSpecBinding": [
													{
														"priceSourceProperty": "ldo.price"
													}
												]
											}
										}
									},
									"metadata": [
										"base:LDO",
										"quote:USDT",
										"oracle:pyth",
										"oracleChain:gnosis",
										"class:fx/crypto",
										"perpetual",
										"sector:defi",
										"enactment:2024-02-04T15:00:00Z"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "86400"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "180"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "120"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.000001,
										"tau": 0.0000071,
										"params": {
											"sigma": 1.5
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.03",
										"commitmentMinTimeFraction": "0.75",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "1",
										"disposalFraction": "1",
										"fullDisposalSize": "1000000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "0",
										"sourceWeights": [
											"0",
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED",
										"dataSourcesSpec": [
											{
												"external": {
													"ethOracle": {
														"address": "0x719abd606155442c21b7d561426d42bd0e40a776",
														"abi": "[{\"inputs\": [{\"internalType\": \"bytes32\", \"name\": \"id\", \"type\": \"bytes32\"}], \"name\": \"getPrice\", \"outputs\": [{\"internalType\": \"int256\", \"name\": \"\", \"type\": \"int256\" }], \"stateMutability\": \"view\", \"type\": \"function\"}]",
														"method": "getPrice",
														"args": [
															{
																"stringValue": "xj4qfzegTl5hTAcji+2yXcw4kn+6j+iQWXpZPAsvpK0="
															}
														],
														"trigger": {
															"timeTrigger": {
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "ldo.price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "ldo.price",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											}
										],
										"dataSourcesSpecBinding": [
											{
												"priceSourceProperty": "ldo.price"
											}
										]
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "29dbc8065ab426b90d481cfbbe467a52da4f0db584c2b5e84762e21e8f5823ba",
							"timestamp": "1708699106158721669",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "79bf55147ec90be6c0b1c4486dd0b045f1191f1a30d82e946bbc78044fa91d2d",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_FAILED",
						"timestamp": "1708700888539235021",
						"terms": {
							"closingTimestamp": "1708701086",
							"enactmentTimestamp": "1708701086",
							"updateMarketState": {
								"changes": {
									"marketId": "00788b763b999ef555ac5da17de155ff4237dd14aa6671a303d1285f27f094f0",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"reason": "PROPOSAL_ERROR_INVALID_MARKET_STATE_UPDATE",
						"errorDetails": "invalid state update request. Market for resume is not suspended",
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "6232d4351f88cfdafe2b211b0dfe342082526bfa9cf21f254b776565c67446a8"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "6232d4351f88cfdafe2b211b0dfe342082526bfa9cf21f254b776565c67446a8",
							"timestamp": "1708700914340081051",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "3cd2ce6615ea36e4897dff9addbb65748caed0b5413ee3399a4ff4d178f45976",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_FAILED",
						"timestamp": "1708700888539235021",
						"terms": {
							"closingTimestamp": "1708701086",
							"enactmentTimestamp": "1708701086",
							"updateMarketState": {
								"changes": {
									"marketId": "279b6e238a2af1e057e63e24bc5a503dea0f86036fb6e5a6adce7bf93dec0f64",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"reason": "PROPOSAL_ERROR_INVALID_MARKET_STATE_UPDATE",
						"errorDetails": "invalid state update request. Market for resume is not suspended",
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "6232d4351f88cfdafe2b211b0dfe342082526bfa9cf21f254b776565c67446a8"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "6232d4351f88cfdafe2b211b0dfe342082526bfa9cf21f254b776565c67446a8",
							"timestamp": "1708700914340081051",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "0d4a9c73d35b894436c61f8ccdc1503aeda76dc6fe77b49cea28f3169aceba2d",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_FAILED",
						"timestamp": "1708700888539235021",
						"terms": {
							"closingTimestamp": "1708701086",
							"enactmentTimestamp": "1708701086",
							"updateMarketState": {
								"changes": {
									"marketId": "53d34c0b8cb45d0ff6232e605191b5d215fc8b0d949b2efb495dbf223448192d",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"reason": "PROPOSAL_ERROR_INVALID_MARKET_STATE_UPDATE",
						"errorDetails": "invalid state update request. Market for resume is not suspended",
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "6232d4351f88cfdafe2b211b0dfe342082526bfa9cf21f254b776565c67446a8"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "6232d4351f88cfdafe2b211b0dfe342082526bfa9cf21f254b776565c67446a8",
							"timestamp": "1708700914340081051",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "2c62d91e07c964fba84c243e397daeeee077290791267486178d447f37a409a9",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_FAILED",
						"timestamp": "1708700888539235021",
						"terms": {
							"closingTimestamp": "1708701086",
							"enactmentTimestamp": "1708701086",
							"updateMarketState": {
								"changes": {
									"marketId": "de0f57d1ec1e5f7868cc47bcab0aabf896a376e9db1c8bec83d00fda3cd53bd8",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"reason": "PROPOSAL_ERROR_INVALID_MARKET_STATE_UPDATE",
						"errorDetails": "invalid state update request. Market for resume is not suspended",
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "6232d4351f88cfdafe2b211b0dfe342082526bfa9cf21f254b776565c67446a8"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "6232d4351f88cfdafe2b211b0dfe342082526bfa9cf21f254b776565c67446a8",
							"timestamp": "1708700914340081051",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "89169d2fed4538d12559cb9e99acb9e77c55fda0244dcfbc52816e188fa50862",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_FAILED",
						"timestamp": "1708700888539235021",
						"terms": {
							"closingTimestamp": "1708701086",
							"enactmentTimestamp": "1708701086",
							"updateMarketState": {
								"changes": {
									"marketId": "ac7fc86bddf26748c6ba32a67037fe13c623cd4b53480aac4eaf29fbbd22ac31",
									"updateType": "MARKET_STATE_UPDATE_TYPE_RESUME"
								}
							}
						},
						"reason": "PROPOSAL_ERROR_INVALID_MARKET_STATE_UPDATE",
						"errorDetails": "invalid state update request. Market for resume is not suspended",
						"rationale": {
							"description": "Update market BTC/USD and ETH/USD to set the fundingRateScalingFactor to 1.0 in both market. Also add two market state proposal to resume trading on both markets",
							"title": "Update Markets and resume trading on BTC/USD and ETH/USD"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "6232d4351f88cfdafe2b211b0dfe342082526bfa9cf21f254b776565c67446a8"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "6232d4351f88cfdafe2b211b0dfe342082526bfa9cf21f254b776565c67446a8",
							"timestamp": "1708700914340081051",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "de99c2f716f24bb3db2f79bbc9577341d7e1d5c3b08d935a9ef21acffd2fa8c9",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708703522612080032",
						"terms": {
							"closingTimestamp": "1708703672",
							"enactmentTimestamp": "1708703732",
							"updateNetworkParameter": {
								"changes": {
									"key": "ethereum.oracles.enabled",
									"value": "1"
								}
							}
						},
						"rationale": {
							"description": "## Summary\n\nThis proposal requests to change Eth oracle/.\n\n## Rationale\n\n- Reason 1.\n- Reason 2. \n- Reason 3.\n- Reason 4.",
							"title": "Eth oracle"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "de99c2f716f24bb3db2f79bbc9577341d7e1d5c3b08d935a9ef21acffd2fa8c9",
							"timestamp": "1708703600214609660",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "2723a7ada65b9637636ffd9dc5bbf1794e008463709727f21c88dff648330601",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708705195428299422",
						"terms": {
							"closingTimestamp": "1708705384",
							"enactmentTimestamp": "1708705444",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Oranges Daily UMA",
										"code": "UMA.24h",
										"future": {
											"settlementAsset": "b340c130096819428a62e5df407fd6abe66e444b89ad64f670beb98621c9c663",
											"quoteName": "tDAI",
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x9Dc85d557BF18441F20a2fC292730f956eC43466",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct SettlementOracle.Identifier\",\"components\":[{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"enactmentDate": {
																			"stringValue": "2024-02-22T02:00:00Z"
																		},
																		"marketCode": {
																			"stringValue": "SOL/USDT"
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708705444",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "price",
																"expression": "$[1]"
															},
															{
																"name": "resolved",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"ethOracle": {
														"address": "0x7D6aa06a128f161945cD6aa6b267738f3e542551",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct TerminationOracle.Identifier\",\"components\":[{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"conditionalSettlementOracle\",\"type\":\"address\",\"internalType\":\"contract SettlementOracle\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"conditionalSettlementOracle": {
																			"stringValue": "0x9Dc85d557BF18441F20a2fC292730f956eC43466"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-02-22T02:00:00Z"
																		},
																		"marketCode": {
																			"stringValue": "SOL/USDT"
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708705444",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "vegaprotocol.builtin.timestamp",
																	"type": "TYPE_TIMESTAMP"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "1708705804"
																	}
																]
															},
															{
																"key": {
																	"name": "terminated",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "resolved",
																"expression": "$[0]"
															},
															{
																"name": "terminated",
																"expression": "$[2]"
															},
															{
																"name": "terminationTimestamp",
																"expression": "$[1]"
															}
														],
														"sourceChainId": "100"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "price",
												"tradingTerminationProperty": "terminated"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"enactment:2024-02-29T17:20:46Z",
										"settlement:2024-02-28T17:20:46Z",
										"source:docs.vega.xyz"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "1"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.00001,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.15
										}
									},
									"positionDecimalPlaces": "5",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.1",
										"commitmentMinTimeFraction": "0.1",
										"performanceHysteresisEpochs": "10",
										"slaCompetitionFactor": "0.2"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_CONSTANT",
										"feeConstant": "0.00005"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "500",
										"disposalFraction": "1",
										"fullDisposalSize": "18446744073709551615",
										"maxFractionConsumed": "1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "5000000",
										"sourceWeights": [
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "UMA Oranges 35",
							"title": "UMA Orange GO Johnny GOO"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "2723a7ada65b9637636ffd9dc5bbf1794e008463709727f21c88dff648330601",
							"timestamp": "1708705205875565096",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "67761584ec650f06d47dc1af5d8c2ed6f789dc62395a1c330465153e861b5d75",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708963390186337803",
						"terms": {
							"closingTimestamp": "1708963565",
							"enactmentTimestamp": "1708963565",
							"updateNetworkParameter": {
								"changes": {
									"key": "blockchains.ethereumRpcAndEvmCompatDataSourcesConfig",
									"value": "{\"configs\":[{\"network_id\":\"100\",\"chain_id\":\"100\",\"confirmations\":3,\"name\":\"Gnosis Chain\", \"block_interval\": 3}, {\"network_id\":\"42161\",\"chain_id\":\"42161\",\"confirmations\":3,\"name\":\"Arbitrum One\", \"block_interval\": 50},{\"network_id\":\"5\",\"chain_id\":\"5\",\"confirmations\":3,\"name\":\"Goerli\", \"block_interval\": 50}]}"
								}
							}
						},
						"rationale": {
							"description": "Add goerli network",
							"title": "Add goerli network"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5",
						"batchId": "0e90cae18ba86193ec8c248dd77b564c3e68701892cfb3e2c65a3d8c4b1b7cc6"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "0e90cae18ba86193ec8c248dd77b564c3e68701892cfb3e2c65a3d8c4b1b7cc6",
							"timestamp": "1708963483063914706",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "cb89c79627e41579a7bffeb6d0e18fa0cb852019bd9f31a8b73c8a981caeebb4",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708963977883810738",
						"terms": {
							"closingTimestamp": "1708964081",
							"enactmentTimestamp": "1708964081",
							"updateNetworkParameter": {
								"changes": {
									"key": "blockchains.ethereumRpcAndEvmCompatDataSourcesConfig",
									"value": "{\"configs\":[{\"network_id\":\"100\",\"chain_id\":\"100\",\"confirmations\":3,\"name\":\"Gnosis Chain\", \"block_interval\": 3}, {\"network_id\":\"42161\",\"chain_id\":\"42161\",\"confirmations\":3,\"name\":\"Arbitrum One\", \"block_interval\": 50},{\"network_id\":\"5\",\"chain_id\":\"5\",\"confirmations\":3,\"name\":\"Goerli\", \"block_interval\": 3}]}"
								}
							}
						},
						"rationale": {
							"description": "Add goerli network",
							"title": "Add goerli network"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5",
						"batchId": "3f00da99f2bb65121bfd52070c0e8e37867a7d7665910f2953a791601332d44c"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "3f00da99f2bb65121bfd52070c0e8e37867a7d7665910f2953a791601332d44c",
							"timestamp": "1708964001596197500",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "b07b8b6ec7b65686dc5cd695a115b3197a388d5828e6e06ee2edf2c30d50a853",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708966073892953356",
						"terms": {
							"closingTimestamp": "1708966220",
							"enactmentTimestamp": "1708966320",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "UMA settlement",
										"code": "UMA.24h",
										"future": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDC",
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0xFe6Cbc9179a68f1029570269cF41333B9911b831",
														"abi": "[{\"inputs\":[{\"components\":[{\"internalType\":\"contract IERC20\",\"name\":\"bondCurrency\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"minimumBond\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maximumBond\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"liveness\",\"type\":\"uint64\"},{\"internalType\":\"string\",\"name\":\"marketCode\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"quoteName\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"enactmentDate\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"ipfsLink\",\"type\":\"string\"},{\"internalType\":\"contract SettlementOracle\",\"name\":\"conditionalSettlementOracle\",\"type\":\"address\"}],\"internalType\":\"struct TerminationOracle.Identifier\",\"name\":\"identifier\",\"type\":\"tuple\"}],\"name\":\"getData\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-02-22T02:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:foo.bar"
																		},
																		"liveness": {
																			"numberValue": 120
																		},
																		"marketCode": {
																			"stringValue": "SOL/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 1000000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708966320",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "price",
																"expression": "$[1]"
															},
															{
																"name": "resolved",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "5"
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"ethOracle": {
														"address": "0xfe6cbc9179a68f1029570269cf41333b9911b831",
														"abi": "[{\"inputs\":[{\"components\":[{\"internalType\":\"contract IERC20\",\"name\":\"bondCurrency\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"minimumBond\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"maximumBond\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"liveness\",\"type\":\"uint64\"},{\"internalType\":\"string\",\"name\":\"marketCode\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"quoteName\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"enactmentDate\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"ipfsLink\",\"type\":\"string\"},{\"internalType\":\"contract SettlementOracle\",\"name\":\"conditionalSettlementOracle\",\"type\":\"address\"}],\"internalType\":\"struct TerminationOracle.Identifier\",\"name\":\"identifier\",\"type\":\"tuple\"}],\"name\":\"getData\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-02-22T02:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:foo.bar"
																		},
																		"liveness": {
																			"numberValue": 120
																		},
																		"marketCode": {
																			"stringValue": "SOL/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 1000000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708966320",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "terminated",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "resolved",
																"expression": "$[0]"
															},
															{
																"name": "terminated",
																"expression": "$[2]"
															}
														],
														"sourceChainId": "5"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "price",
												"tradingTerminationProperty": "terminated"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"enactment:2024-02-29T17:20:46Z",
										"settlement:2024-02-28T17:20:46Z",
										"source:docs.vega.xyz"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "43200",
												"probability": "0.9999999",
												"auctionExtension": "1"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.00001,
										"tau": 0.0001140771161,
										"params": {
											"r": 0.016,
											"sigma": 0.15
										}
									},
									"positionDecimalPlaces": "5",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.1",
										"commitmentMinTimeFraction": "0.1",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.2"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_CONSTANT",
										"feeConstant": "0.00005"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "500",
										"disposalFraction": "1",
										"fullDisposalSize": "18446744073709551615",
										"maxFractionConsumed": "1"
									},
									"markPriceConfiguration": {
										"decayWeight": "1",
										"decayPower": "1",
										"cashAmount": "5000000",
										"sourceWeights": [
											"0",
											"1",
											"0"
										],
										"sourceStalenessTolerance": [
											"1m0s",
											"1m0s",
											"1m0s"
										],
										"compositePriceType": "COMPOSITE_PRICE_TYPE_WEIGHTED"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Create UMA sample market",
							"title": "Create UMA sample market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"batchId": "db1c8d942b368d7599c0ecf5eae8b591eab2f1dca579b4bdbc6d2e948429b9fa"
					},
					"yes": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_YES",
							"proposalId": "db1c8d942b368d7599c0ecf5eae8b591eab2f1dca579b4bdbc6d2e948429b9fa",
							"timestamp": "1708966096608820045",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "6951b419300086dd99a8e8183c6aa7d630c7a828fbf7c2bf4ffabeb134ea9ba4",
						"reference": "**TO_UPDATE_OR_REMOVE**",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1708967601934960698",
						"terms": {
							"closingTimestamp": "1708967727",
							"enactmentTimestamp": "1708967827",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "DogePoint / USDT (Points future market)",
										"code": "DOGEP/USDT.POINTS",
										"future": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDT",
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0xB92997f94F4F3caCe123B13627EB68AdC3B7b091",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct SettlementOracle.Identifier\",\"components\":[{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-02-22T02:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:foo.bar"
																		},
																		"liveness": {
																			"numberValue": 120
																		},
																		"marketCode": {
																			"stringValue": "SOL/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 1000000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708967827",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "price",
																"expression": "$[1]"
															},
															{
																"name": "resolved",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "5"
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"ethOracle": {
														"address": "0xfe6cbc9179a68f1029570269cf41333b9911b831",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct TerminationOracle.Identifier\",\"components\":[{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"conditionalSettlementOracle\",\"type\":\"address\",\"internalType\":\"contract SettlementOracle\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6"
																		},
																		"conditionalSettlementOracle": {
																			"stringValue": "0xB92997f94F4F3caCe123B13627EB68AdC3B7b091"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-02-22T02:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:foo.bar"
																		},
																		"liveness": {
																			"numberValue": 120
																		},
																		"marketCode": {
																			"stringValue": "SOL/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 1000000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1708967827",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "terminated",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "resolved",
																"expression": "$[0]"
															},
															{
																"name": "terminated",
																"expression": "$[2]"
															}
														],
														"sourceChainId": "5"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "price",
												"tradingTerminationProperty": "terminated"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:DOGEPOINT",
										"quote:USDT",
										"enactment:2024-02-29T17:20:46Z",
										"settlement:fromOracle",
										"class:fx/crypto",
										"oracle:uma",
										"sector:defi",
										"oracleChain:arbitrum"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "172800"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "1200"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.0001,
										"tau": 0.002734,
										"params": {
											"sigma": 3
										}
									},
									"positionDecimalPlaces": "-2",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.2",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_CONSTANT",
										"feeConstant": "0.00005"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "5",
										"disposalFraction": "0.1",
										"fullDisposalSize": "100",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "0",
										"cashAmount": "0",
										"compositePriceType": "COMPOSITE_PRICE_TYPE_LAST_TRADE"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Create UMA sample market",
							"title": "Create UMA sample market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"batchId": "049c55ce0054c7e4848961a32aa6f5fda59b3d72d003b4d5b2e88cd8f5902c26"
					},
					"yes": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_YES",
							"proposalId": "049c55ce0054c7e4848961a32aa6f5fda59b3d72d003b4d5b2e88cd8f5902c26",
							"timestamp": "1708967632329647775",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "695eb069a7a5d7fa355c7ca2e4258a97d142a400b8585bb23341067144590243",
						"reference": "update-btc-b-team",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1709032580246746071",
						"terms": {
							"closingTimestamp": "1709032784",
							"enactmentTimestamp": "1709032884",
							"updateMarket": {
								"marketId": "6951b419300086dd99a8e8183c6aa7d630c7a828fbf7c2bf4ffabeb134ea9ba4",
								"changes": {
									"instrument": {
										"code": "DOGEP/USDT.POINTS",
										"name": "DogePoint / USDT (Points future market)",
										"future": {
											"quoteName": "USDT",
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0xB49281A7F7878Cdf5B6378d8c7dC211Ffc1b5B60",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct SettlementOracle.Identifier\",\"components\":[{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-02-22T02:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:foo.bar"
																		},
																		"liveness": {
																			"numberValue": 120
																		},
																		"marketCode": {
																			"stringValue": "SOL/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 1000000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1709032884",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "price",
																"expression": "$[1]"
															},
															{
																"name": "resolved",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "5"
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"ethOracle": {
														"address": "0xB92997f94F4F3caCe123B13627EB68AdC3B7b091",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct TerminationOracle.Identifier\",\"components\":[{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"conditionalSettlementOracle\",\"type\":\"address\",\"internalType\":\"contract SettlementOracle\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6"
																		},
																		"conditionalSettlementOracle": {
																			"stringValue": "0xB49281A7F7878Cdf5B6378d8c7dC211Ffc1b5B60"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-02-22T02:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:foo.bar"
																		},
																		"liveness": {
																			"numberValue": 120
																		},
																		"marketCode": {
																			"stringValue": "SOL/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 1000000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1709032884",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "terminated",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "resolved",
																"expression": "$[0]"
															},
															{
																"name": "terminated",
																"expression": "$[2]"
															}
														],
														"sourceChainId": "5"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "price",
												"tradingTerminationProperty": "terminated"
											}
										}
									},
									"metadata": [
										"base:DOGEPOINT",
										"quote:USDT",
										"enactment:2024-02-29T17:20:46Z",
										"settlement:fromOracle",
										"class:fx/crypto",
										"oracle:uma",
										"sector:defi",
										"oracleChain:arbitrum"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "172800"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "1200"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.0001,
										"tau": 0.002734,
										"params": {
											"sigma": 3
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.2",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_CONSTANT",
										"feeConstant": "0.00005"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "5",
										"disposalFraction": "0.1",
										"fullDisposalSize": "100",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "0",
										"cashAmount": "0",
										"compositePriceType": "COMPOSITE_PRICE_TYPE_LAST_TRADE"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update BTC-B-TEAM",
							"title": "Update BTC-B-TEAM"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_YES",
							"proposalId": "695eb069a7a5d7fa355c7ca2e4258a97d142a400b8585bb23341067144590243",
							"timestamp": "1709032607536490458",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "dcb23096047d592bfb1a8df19385403566ed750dad2e796ad21032ff8f785521",
						"reference": "update-btc-b-team",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1709033808726123105",
						"terms": {
							"closingTimestamp": "1709033883",
							"enactmentTimestamp": "1709033983",
							"updateMarket": {
								"marketId": "6951b419300086dd99a8e8183c6aa7d630c7a828fbf7c2bf4ffabeb134ea9ba4",
								"changes": {
									"instrument": {
										"code": "DOGEP/USDT.POINTS",
										"name": "DogePoint / USDT (Points future market)",
										"future": {
											"quoteName": "USDT",
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0xB49281A7F7878Cdf5B6378d8c7dC211Ffc1b5B60",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct SettlementOracle.Identifier\",\"components\":[{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-02-22T02:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:foo.bar"
																		},
																		"liveness": {
																			"numberValue": 120
																		},
																		"marketCode": {
																			"stringValue": "SOL/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 1000000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1709033983",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "price",
																"expression": "$[1]"
															},
															{
																"name": "resolved",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "5"
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"ethOracle": {
														"address": "0x8744F73A5b404ef843A76A927dF89FE20ab071CB",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct TerminationOracle.Identifier\",\"components\":[{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"conditionalSettlementOracle\",\"type\":\"address\",\"internalType\":\"contract SettlementOracle\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6"
																		},
																		"conditionalSettlementOracle": {
																			"stringValue": "0xB49281A7F7878Cdf5B6378d8c7dC211Ffc1b5B60"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-02-22T02:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:foo.bar"
																		},
																		"liveness": {
																			"numberValue": 120
																		},
																		"marketCode": {
																			"stringValue": "SOL/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 1000000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1709033983",
																"every": "60"
															}
														},
														"requiredConfirmations": "3",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "terminated",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "resolved",
																"expression": "$[0]"
															},
															{
																"name": "terminated",
																"expression": "$[2]"
															}
														],
														"sourceChainId": "5"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "price",
												"tradingTerminationProperty": "terminated"
											}
										}
									},
									"metadata": [
										"base:DOGEPOINT",
										"quote:USDT",
										"enactment:2024-02-29T17:20:46Z",
										"settlement:fromOracle",
										"class:fx/crypto",
										"oracle:uma",
										"sector:defi",
										"oracleChain:arbitrum"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "21600",
												"probability": "0.9999999",
												"auctionExtension": "172800"
											},
											{
												"horizon": "4320",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "1440",
												"probability": "0.9999999",
												"auctionExtension": "1200"
											},
											{
												"horizon": "360",
												"probability": "0.9999999",
												"auctionExtension": "300"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.0001,
										"tau": 0.002734,
										"params": {
											"sigma": 3
										}
									},
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.2",
										"commitmentMinTimeFraction": "0.85",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_CONSTANT",
										"feeConstant": "0.00005"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "5",
										"disposalFraction": "0.1",
										"fullDisposalSize": "100",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "0",
										"cashAmount": "0",
										"compositePriceType": "COMPOSITE_PRICE_TYPE_LAST_TRADE"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Update BTC-B-TEAM",
							"title": "Update BTC-B-TEAM"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "134e5b2fdf11352a85511ec1ae5c0f876977f4bffd52ad08aa04836bf6bf8585",
							"value": "VALUE_YES",
							"proposalId": "dcb23096047d592bfb1a8df19385403566ed750dad2e796ad21032ff8f785521",
							"timestamp": "1709033827307197153",
							"totalGovernanceTokenBalance": "9876000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "68fbf3aad272c734e936174cdb1468af40badf709075b55042eae4368fc3abb3",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1709313344922849916",
						"terms": {
							"closingTimestamp": "1709313725",
							"enactmentTimestamp": "1709313825",
							"updateMarketState": {
								"changes": {
									"marketId": "53d34c0b8cb45d0ff6232e605191b5d215fc8b0d949b2efb495dbf223448192d",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Fix stagnet1",
							"title": "Stagnet1 fix"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "b10fb7db9427d7628fba42bb7a69e12a64ac7008c1ed1a135835e52f58ea2321"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "b10fb7db9427d7628fba42bb7a69e12a64ac7008c1ed1a135835e52f58ea2321",
							"timestamp": "1709313390616941018",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "d3a87b46881314d43cd3709acf4488a5b81a0801074215452785c4a416e24076",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1709313344922849916",
						"terms": {
							"closingTimestamp": "1709313725",
							"enactmentTimestamp": "1709313825",
							"updateMarketState": {
								"changes": {
									"marketId": "de0f57d1ec1e5f7868cc47bcab0aabf896a376e9db1c8bec83d00fda3cd53bd8",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Fix stagnet1",
							"title": "Stagnet1 fix"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "b10fb7db9427d7628fba42bb7a69e12a64ac7008c1ed1a135835e52f58ea2321"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "b10fb7db9427d7628fba42bb7a69e12a64ac7008c1ed1a135835e52f58ea2321",
							"timestamp": "1709313390616941018",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "e0667591990d1acd8f77e61b13d5c9a99ab46f193cb1b34e34c945cc1aee6012",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1709313344922849916",
						"terms": {
							"closingTimestamp": "1709313725",
							"enactmentTimestamp": "1709313825",
							"updateMarketState": {
								"changes": {
									"marketId": "dcfdccbd0dde103102f8c39b69546a94aa927af23ce63dca8a3d365e173ae93b",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Fix stagnet1",
							"title": "Stagnet1 fix"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "b10fb7db9427d7628fba42bb7a69e12a64ac7008c1ed1a135835e52f58ea2321"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "b10fb7db9427d7628fba42bb7a69e12a64ac7008c1ed1a135835e52f58ea2321",
							"timestamp": "1709313390616941018",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "04a75c86239304e31f544b835d65ba23c4607ee40a321e1dbf521c95f6ca2fcc",
						"reference": "UPDATE_MARKET_LIKE_MAINNET",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1709313344922849916",
						"terms": {
							"closingTimestamp": "1709313725",
							"enactmentTimestamp": "1709313825",
							"updateMarketState": {
								"changes": {
									"marketId": "b07b8b6ec7b65686dc5cd695a115b3197a388d5828e6e06ee2edf2c30d50a853",
									"updateType": "MARKET_STATE_UPDATE_TYPE_TERMINATE",
									"price": "42377"
								}
							}
						},
						"rationale": {
							"description": "Fix stagnet1",
							"title": "Stagnet1 fix"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"requiredLiquidityProviderParticipation": "0.00001",
						"requiredLiquidityProviderMajority": "0.66",
						"batchId": "b10fb7db9427d7628fba42bb7a69e12a64ac7008c1ed1a135835e52f58ea2321"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "b10fb7db9427d7628fba42bb7a69e12a64ac7008c1ed1a135835e52f58ea2321",
							"timestamp": "1709313390616941018",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "397a7b58726b5d794a327df31b0827c85ed61733ff965dd67bdf871a3038acdc",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1709649169201167909",
						"terms": {
							"closingTimestamp": "1709649365",
							"enactmentTimestamp": "1709649465",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Doge Points 2 / USDT (Futures market)",
										"code": "DOGEP2/USDT.POINTS",
										"future": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDT",
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x302461E6dBF45e59acb3BE9a9c84C0a997779612",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct SettlementOracle.Identifier\",\"components\":[{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-03-04T16:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:foo.bar"
																		},
																		"liveness": {
																			"numberValue": 1
																		},
																		"marketCode": {
																			"stringValue": "DOGEP/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 500000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1709649465",
																"every": "3600"
															}
														},
														"requiredConfirmations": "64",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "price",
																"expression": "$[1]"
															},
															{
																"name": "resolved",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "42161"
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"ethOracle": {
														"address": "0x6d0b3a00265b8b4a1d22cf466c331014133ba614",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct TerminationOracle.Identifier\",\"components\":[{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"conditionalSettlementOracle\",\"type\":\"address\",\"internalType\":\"contract SettlementOracle\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"
																		},
																		"conditionalSettlementOracle": {
																			"stringValue": "0x302461E6dBF45e59acb3BE9a9c84C0a997779612"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-03-04T16:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:foo.bar"
																		},
																		"liveness": {
																			"numberValue": 1
																		},
																		"marketCode": {
																			"stringValue": "DOGEP/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 500000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1709649465",
																"every": "3600"
															}
														},
														"requiredConfirmations": "64",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "terminated",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "resolved",
																"expression": "$[0]"
															},
															{
																"name": "terminated",
																"expression": "$[2]"
															}
														],
														"sourceChainId": "42161"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "price",
												"tradingTerminationProperty": "terminated"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:DOGEPOINT",
										"quote:USDT",
										"enactment:2024-03-04T16:00:00Z",
										"settlement:fromOracle",
										"class:fx/crypto",
										"oracle:uma",
										"sector:defi",
										"oracleChain:arbitrum"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.0001,
										"tau": 0.001368925394,
										"params": {
											"sigma": 5
										}
									},
									"positionDecimalPlaces": "-2",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.2",
										"commitmentMinTimeFraction": "0.5",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "5",
										"disposalFraction": "0.1",
										"fullDisposalSize": "10000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "0",
										"cashAmount": "0",
										"compositePriceType": "COMPOSITE_PRICE_TYPE_LAST_TRADE"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Create UMA sample market",
							"title": "Create UMA sample market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"batchId": "651142d09c0662e8ce88686d47a62a53e8dc5663bf8847ef4429c870ab999b29"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "651142d09c0662e8ce88686d47a62a53e8dc5663bf8847ef4429c870ab999b29",
							"timestamp": "1709649193631816901",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "c3e83bf7a5378989d9eee50d20012253a8adfc5fe09580c1f63346e5e38a858a",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1709649215618805040",
						"terms": {
							"closingTimestamp": "1709649465",
							"enactmentTimestamp": "1709649495",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Doge Points / USDT (Futures market)",
										"code": "DOGEP/USDT.POINTS",
										"future": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDT",
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x302461E6dBF45e59acb3BE9a9c84C0a997779612",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct SettlementOracle.Identifier\",\"components\":[{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-03-04T16:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:foo.bar"
																		},
																		"liveness": {
																			"numberValue": 1
																		},
																		"marketCode": {
																			"stringValue": "DOGEP/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 500000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1709649495",
																"every": "3600"
															}
														},
														"requiredConfirmations": "64",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "price",
																"expression": "$[1]"
															},
															{
																"name": "resolved",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "42161"
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"ethOracle": {
														"address": "0x6d0b3a00265b8b4a1d22cf466c331014133ba614",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct TerminationOracle.Identifier\",\"components\":[{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"conditionalSettlementOracle\",\"type\":\"address\",\"internalType\":\"contract SettlementOracle\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"
																		},
																		"conditionalSettlementOracle": {
																			"stringValue": "0x302461E6dBF45e59acb3BE9a9c84C0a997779612"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-03-04T16:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:foo.bar"
																		},
																		"liveness": {
																			"numberValue": 1
																		},
																		"marketCode": {
																			"stringValue": "DOGEP/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 500000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1709649495",
																"every": "3600"
															}
														},
														"requiredConfirmations": "64",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "terminated",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "resolved",
																"expression": "$[0]"
															},
															{
																"name": "terminated",
																"expression": "$[2]"
															}
														],
														"sourceChainId": "42161"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "price",
												"tradingTerminationProperty": "terminated"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:DOGEPOINT",
										"quote:USDT",
										"enactment:2024-03-04T16:00:00Z",
										"settlement:fromOracle",
										"class:fx/crypto",
										"oracle:uma",
										"sector:defi",
										"oracleChain:arbitrum"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 1.5865e-7,
										"tau": 0.000009513,
										"params": {
											"sigma": 25
										}
									},
									"positionDecimalPlaces": "-2",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.2",
										"commitmentMinTimeFraction": "0.5",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "5",
										"disposalFraction": "0.1",
										"fullDisposalSize": "10000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "0",
										"cashAmount": "0",
										"compositePriceType": "COMPOSITE_PRICE_TYPE_LAST_TRADE"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Create UMA sample market",
							"title": "Create UMA sample market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"batchId": "e50344792a58ae2151db87ce282b58d3dc14f9ac8c3210df4f6909e09b0b5be8"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "e50344792a58ae2151db87ce282b58d3dc14f9ac8c3210df4f6909e09b0b5be8",
							"timestamp": "1709649242232206914",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "346d925ab76fca647e4dabb6f1e6ef9839e62a2b11a0f151368930613ce2108a",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1709651389423581194",
						"terms": {
							"closingTimestamp": "1709651585",
							"enactmentTimestamp": "1709651685",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Doge Points 2 / USDT (Futures market)",
										"code": "DOGEP2/USDT.POINTS",
										"future": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDT",
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x302461E6dBF45e59acb3BE9a9c84C0a997779612",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct SettlementOracle.Identifier\",\"components\":[{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-03-04T16:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:bar.foo"
																		},
																		"liveness": {
																			"numberValue": 1
																		},
																		"marketCode": {
																			"stringValue": "DOGEP/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 500000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1709651685",
																"every": "3600"
															}
														},
														"requiredConfirmations": "64",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "price",
																"expression": "$[1]"
															},
															{
																"name": "resolved",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "42161"
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"ethOracle": {
														"address": "0x6d0b3a00265b8b4a1d22cf466c331014133ba614",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct TerminationOracle.Identifier\",\"components\":[{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"conditionalSettlementOracle\",\"type\":\"address\",\"internalType\":\"contract SettlementOracle\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"
																		},
																		"conditionalSettlementOracle": {
																			"stringValue": "0x302461E6dBF45e59acb3BE9a9c84C0a997779612"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-03-04T16:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:bar.foo"
																		},
																		"liveness": {
																			"numberValue": 1
																		},
																		"marketCode": {
																			"stringValue": "DOGEP/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 500000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1709651685",
																"every": "3600"
															}
														},
														"requiredConfirmations": "64",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "terminated",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "resolved",
																"expression": "$[0]"
															},
															{
																"name": "terminated",
																"expression": "$[2]"
															}
														],
														"sourceChainId": "42161"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "price",
												"tradingTerminationProperty": "terminated"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:DOGEPOINT",
										"quote:USDT",
										"enactment:2024-03-04T16:00:00Z",
										"settlement:fromOracle",
										"class:fx/crypto",
										"oracle:uma",
										"sector:defi",
										"oracleChain:arbitrum"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 0.0001,
										"tau": 0.001368925394,
										"params": {
											"sigma": 5
										}
									},
									"positionDecimalPlaces": "-2",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.2",
										"commitmentMinTimeFraction": "0.5",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "5",
										"disposalFraction": "0.1",
										"fullDisposalSize": "10000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "0",
										"cashAmount": "0",
										"compositePriceType": "COMPOSITE_PRICE_TYPE_LAST_TRADE"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Create UMA sample market",
							"title": "Create UMA sample market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"batchId": "4c208ab9d0c5de97c098da7c3aca447f2271dca8093b6a69a7d21fd106b87c1e"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "4c208ab9d0c5de97c098da7c3aca447f2271dca8093b6a69a7d21fd106b87c1e",
							"timestamp": "1709651421234986698",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "58a3dd6e4c49997d0fc94c7a8d4d0ea5363d3f20e444859abbaa754bd3bde37d",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1709651381437596385",
						"terms": {
							"closingTimestamp": "1709651585",
							"enactmentTimestamp": "1709651685",
							"newMarket": {
								"changes": {
									"instrument": {
										"name": "Doge Points / USDT (Futures market)",
										"code": "DOGEP/USDT.POINTS",
										"future": {
											"settlementAsset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"quoteName": "USDT",
											"dataSourceSpecForSettlementData": {
												"external": {
													"ethOracle": {
														"address": "0x302461E6dBF45e59acb3BE9a9c84C0a997779612",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct SettlementOracle.Identifier\",\"components\":[{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-03-04T16:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:bar.foo"
																		},
																		"liveness": {
																			"numberValue": 1
																		},
																		"marketCode": {
																			"stringValue": "DOGEP/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 500000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1709651685",
																"every": "3600"
															}
														},
														"requiredConfirmations": "64",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "price",
																	"type": "TYPE_INTEGER",
																	"numberDecimalPlaces": "18"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_GREATER_THAN_OR_EQUAL",
																		"value": "0"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "price",
																"expression": "$[1]"
															},
															{
																"name": "resolved",
																"expression": "$[0]"
															}
														],
														"sourceChainId": "42161"
													}
												}
											},
											"dataSourceSpecForTradingTermination": {
												"external": {
													"ethOracle": {
														"address": "0x6d0b3a00265b8b4a1d22cf466c331014133ba614",
														"abi": "[{\"type\":\"function\",\"name\":\"getData\",\"inputs\":[{\"name\":\"identifier\",\"type\":\"tuple\",\"internalType\":\"struct TerminationOracle.Identifier\",\"components\":[{\"name\":\"bondCurrency\",\"type\":\"address\",\"internalType\":\"contract IERC20\"},{\"name\":\"minimumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"maximumBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"liveness\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"marketCode\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"quoteName\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"enactmentDate\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"ipfsLink\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"conditionalSettlementOracle\",\"type\":\"address\",\"internalType\":\"contract SettlementOracle\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"}]",
														"method": "getData",
														"args": [
															{
																"structValue": {
																	"fields": {
																		"bondCurrency": {
																			"stringValue": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"
																		},
																		"conditionalSettlementOracle": {
																			"stringValue": "0x302461E6dBF45e59acb3BE9a9c84C0a997779612"
																		},
																		"enactmentDate": {
																			"stringValue": "2024-03-04T16:00:00Z"
																		},
																		"ipfsLink": {
																			"stringValue": "ipfs:bar.foo"
																		},
																		"liveness": {
																			"numberValue": 1
																		},
																		"marketCode": {
																			"stringValue": "DOGEP/USDT"
																		},
																		"maximumBond": {
																			"numberValue": 100000000000
																		},
																		"minimumBond": {
																			"numberValue": 500000000
																		},
																		"quoteName": {
																			"stringValue": "USDT"
																		}
																	}
																}
															}
														],
														"trigger": {
															"timeTrigger": {
																"initial": "1709651685",
																"every": "3600"
															}
														},
														"requiredConfirmations": "64",
														"filters": [
															{
																"key": {
																	"name": "resolved",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															},
															{
																"key": {
																	"name": "terminated",
																	"type": "TYPE_BOOLEAN"
																},
																"conditions": [
																	{
																		"operator": "OPERATOR_EQUALS",
																		"value": "true"
																	}
																]
															}
														],
														"normalisers": [
															{
																"name": "resolved",
																"expression": "$[0]"
															},
															{
																"name": "terminated",
																"expression": "$[2]"
															}
														],
														"sourceChainId": "42161"
													}
												}
											},
											"dataSourceSpecBinding": {
												"settlementDataProperty": "price",
												"tradingTerminationProperty": "terminated"
											}
										}
									},
									"decimalPlaces": "5",
									"metadata": [
										"base:DOGEPOINT",
										"quote:USDT",
										"enactment:2024-03-04T16:00:00Z",
										"settlement:fromOracle",
										"class:fx/crypto",
										"oracle:uma",
										"sector:defi",
										"oracleChain:arbitrum"
									],
									"priceMonitoringParameters": {
										"triggers": [
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "120",
												"probability": "0.9999999",
												"auctionExtension": "60"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "200",
												"probability": "0.9999999",
												"auctionExtension": "300"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "400",
												"probability": "0.9999999",
												"auctionExtension": "900"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "500",
												"probability": "0.9999999",
												"auctionExtension": "1800"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "620",
												"probability": "0.9999999",
												"auctionExtension": "3600"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "750",
												"probability": "0.9999999",
												"auctionExtension": "7200"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											},
											{
												"horizon": "800",
												"probability": "0.9999999",
												"auctionExtension": "28800"
											}
										]
									},
									"liquidityMonitoringParameters": {
										"targetStakeParameters": {
											"timeWindow": "3600",
											"scalingFactor": 0.05
										}
									},
									"logNormal": {
										"riskAversionParameter": 1.5865e-7,
										"tau": 0.000009513,
										"params": {
											"sigma": 25
										}
									},
									"positionDecimalPlaces": "-2",
									"linearSlippageFactor": "0.001",
									"liquiditySlaParameters": {
										"priceRange": "0.2",
										"commitmentMinTimeFraction": "0.5",
										"performanceHysteresisEpochs": "1",
										"slaCompetitionFactor": "0.8"
									},
									"liquidityFeeSettings": {
										"method": "METHOD_MARGINAL_COST"
									},
									"liquidationStrategy": {
										"disposalTimeStep": "5",
										"disposalFraction": "0.1",
										"fullDisposalSize": "10000",
										"maxFractionConsumed": "0.1"
									},
									"markPriceConfiguration": {
										"decayWeight": "0",
										"cashAmount": "0",
										"compositePriceType": "COMPOSITE_PRICE_TYPE_LAST_TRADE"
									},
									"tickSize": "1"
								}
							}
						},
						"rationale": {
							"description": "Create UMA sample market",
							"title": "Create UMA sample market"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.66",
						"batchId": "fbf6d0a11e5835041af88af18fe1dc7894ab3ff0205d8d9d0c6df97c92338b98"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "fbf6d0a11e5835041af88af18fe1dc7894ab3ff0205d8d9d0c6df97c92338b98",
							"timestamp": "1709651434957402174",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "e1938ad258e57bb8750e904f82e762e2a46c6da10af45b10ee47e041a0053169",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1709891118784159969",
						"terms": {
							"closingTimestamp": "1709891332",
							"enactmentTimestamp": "1709891392",
							"updateNetworkParameter": {
								"changes": {
									"key": "spam.protection.max.batchSize",
									"value": "30"
								}
							}
						},
						"rationale": {
							"description": "Update spam.protection.max.batchSize to 30",
							"title": "Update spam.protection.max.batchSize to 30"
						},
						"requiredParticipation": "0",
						"requiredMajority": "0.5"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "e1938ad258e57bb8750e904f82e762e2a46c6da10af45b10ee47e041a0053169",
							"timestamp": "1709891148432937316",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "f041fd69fb0dde66a84a709954c8b721408fd1ea4888690b5ebc871be92b4072",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1710874980576138804",
						"terms": {
							"closingTimestamp": "1710875196",
							"enactmentTimestamp": "1710875196",
							"newTransfer": {
								"changes": {
									"sourceType": "ACCOUNT_TYPE_NETWORK_TREASURY",
									"transferType": "GOVERNANCE_TRANSFER_TYPE_BEST_EFFORT",
									"amount": "111000000",
									"asset": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
									"fractionOfBalance": "0.1",
									"destinationType": "ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL",
									"recurring": {
										"startEpoch": "116951",
										"endEpoch": "117051",
										"dispatchStrategy": {
											"assetForMetric": "c9fe6fc24fce121b2cc72680543a886055abb560043fda394ba5376203b7527d",
											"metric": "DISPATCH_METRIC_AVERAGE_NOTIONAL",
											"entityScope": "ENTITY_SCOPE_TEAMS",
											"nTopPerformers": "0.5",
											"notionalTimeWeightedAveragePositionRequirement": "25000000000",
											"windowLength": "1",
											"distributionStrategy": "DISTRIBUTION_STRATEGY_RANK",
											"rankTable": [
												{
													"startRank": 1,
													"shareRatio": 3
												}
											]
										}
									}
								}
							}
						},
						"rationale": {
							"description": "ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL rewards II AVG position",
							"title": "Reward for ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL II AVG position"
						},
						"requiredParticipation": "0.00001",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "f041fd69fb0dde66a84a709954c8b721408fd1ea4888690b5ebc871be92b4072",
							"timestamp": "1710875040684039445",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				},
				{
					"proposal": {
						"id": "07a52385be2dbbb74937a1cb3a69ab7c66008e8538fd409936e4f44e18815a78",
						"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
						"state": "STATE_ENACTED",
						"timestamp": "1710951358272840929",
						"terms": {
							"closingTimestamp": "1710953527",
							"enactmentTimestamp": "1710953597",
							"validationTimestamp": "1710953427",
							"newAsset": {
								"changes": {
									"name": "Axelar Wrapped aUSDC",
									"symbol": "aUSDC",
									"decimals": "6",
									"quantum": "1",
									"erc20": {
										"contractAddress": "0x254d06f33bDc5b8ee05b2ea472107E300226659A",
										"lifetimeLimit": "100000000000000000000",
										"withdrawThreshold": "1"
									}
								}
							}
						},
						"rationale": {
							"description": "TBD",
							"title": "VAP-001 - Add Asset for Jeremy"
						},
						"requiredParticipation": "0.5",
						"requiredMajority": "0.66"
					},
					"yes": [
						{
							"partyId": "69464e35bcb8e8a2900ca0f87acaf252d50cf2ab2fc73694845a16b7c8a0dc6f",
							"value": "VALUE_YES",
							"proposalId": "07a52385be2dbbb74937a1cb3a69ab7c66008e8538fd409936e4f44e18815a78",
							"timestamp": "1710951418844496879",
							"totalGovernanceTokenBalance": "50015000000000000000000000",
							"totalGovernanceTokenWeight": "1",
							"totalEquityLikeShareWeight": "0"
						}
					]
				}
			]
		}
	}`
	result, err := unmarshalGovernanceEnacted(t, jsonStr)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	for _, r := range result.Proposals {
		_, err := types.ProposalFromProto(r.Proposal)
		require.NoError(t, err, r.Proposal.Id)
	}
}
