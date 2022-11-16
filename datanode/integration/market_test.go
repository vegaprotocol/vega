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

package integration_test

import "testing"

func TestMarkets(t *testing.T) {
	queries := map[string]string{
		"Basic":               "{ marketsConnection{ edges { node { id, decimalPlaces, tradingMode, state } } } }",
		"Fees":                "{ marketsConnection{ edges { node { id, fees { factors { makerFee, infrastructureFee, liquidityFee } } } } } }",
		"Instrument":          "{ marketsConnection{ edges { node { id, tradableInstrument{ instrument{ id, code, name, metadata{ tags } } } } } } }",
		"MarginCalculator":    "{ marketsConnection{ edges { node { id, tradableInstrument{ marginCalculator{ scalingFactors{ searchLevel, initialMargin,collateralRelease } } } } } } }",
		"PriceMonitor":        "{ marketsConnection{ edges { node { id, priceMonitoringSettings{ parameters{ triggers{ horizonSecs, probability } } } } } } }",
		"LiquidityMonitor":    "{ marketsConnection{ edges { node { id, liquidityMonitoringParameters{ targetStakeParameters{ timeWindow, scalingFactor } triggeringRatio} } } } }",
		"Proposal":            "{ marketsConnection{ edges { node { id, proposal{ id, reference, party { id }, state, datetime, rejectionReason} } } } }",
		"ProposalTerms":       "{ marketsConnection{ edges { node { id, proposal{ id, terms{ closingDatetime, enactmentDatetime } } } } } }",
		"ProposalYes":         "{ marketsConnection{ edges { node { id, proposal{ id, votes{ yes{ totalNumber totalWeight totalTokens} } } } } } }",
		"ProposalYesVotes":    "{ marketsConnection{ edges { node { id, proposal{ id, votes{ yes{ votes{value, party { id }, datetime, proposalId, governanceTokenBalance, governanceTokenWeight} } } } } } } }",
		"ProposalNo":          "{ marketsConnection{ edges { node { id, proposal{ id, votes{ no{ totalNumber totalWeight totalTokens} } } } } } }",
		"PropsalNoVotes":      "{ marketsConnection{ edges { node { id, proposal{ id, votes{ no{ votes{ value, party { id }, datetime, proposalId, governanceTokenBalance, governanceTokenWeight } } } } } } } }",
		"Orders":              "{ marketsConnection{ edges { node { id, ordersConnection{ edges { node { id, price, side, timeInForce, size, remaining, status, reference, type, rejectionReason, version, party{ id }, market{id}, tradesConnection{ edges{ node{ id } } } createdAt, expiresAt,  updatedAt, peggedOrder { reference, offset } } } } } } } }",
		"OrderLP":             "{ marketsConnection{ edges { node { id, ordersConnection{ edges { node { id, liquidityProvision{ commitmentAmount, fee, status, version, reference, createdAt, updatedAt, market { id } } } } } } } } }",
		"OrderTrades":         "{ marketsConnection{ edges { node { id, tradesConnection{ edges { node { id, price, size, createdAt, market{ id }, type, buyOrder, sellOrder, buyer{id}, seller{id}, aggressor, buyerAuctionBatch, sellerAuctionBatch } } } } } } }",
		"OrderBuyFees":        "{ marketsConnection{ edges { node { id, tradesConnection{ edges { node { id, buyerFee { makerFee, infrastructureFee, liquidityFee } } } } } } } }",
		"OrderSellFees":       "{ marketsConnection{ edges { node { id, tradesConnection{ edges { node { id, sellerFee { makerFee, infrastructureFee, liquidityFee } } } } } } } }",
		"Candles1Minute":      "{ marketsConnection{ edges { node { id, candlesConnection(since : \"2000-01-01T00:00:00Z\",interval : INTERVAL_I1M)  { edges { node { periodStart, lastUpdateInPeriod, high, low, open, close, volume } } } } } } }",
		"Candles5Minute":      "{ marketsConnection{ edges { node { id, candlesConnection(since : \"2000-01-01T00:00:00Z\",interval : INTERVAL_I5M)  { edges { node { periodStart, lastUpdateInPeriod, high, low, open, close, volume } } } } } } }",
		"Candles15Minute":     "{ marketsConnection{ edges { node { id, candlesConnection(since : \"2000-01-01T00:00:00Z\",interval : INTERVAL_I15M)  { edges { node { periodStart, lastUpdateInPeriod, high, low, open, close, volume } } } } } } }",
		"RiskFactor":          "{ marketsConnection{ edges { node { riskFactors { market, short, long } } } } }",
		"LiquidityProvisions": "{ marketsConnection{ edges { node { id, liquidityProvisionsConnection { edges { node { id, party { id }, createdAt, updatedAt, market { id }, commitmentAmount, fee, sells { liquidityOrder { reference } }, buys { liquidityOrder { reference } }, version, status, reference } } } } } } }",
		"Accounts":            "{ marketsConnection{ edges { node { id, accountsConnection { edges { node { balance, asset {id}, type, market {id} } } } } } } }",
		"MarketDepth":         "{ marketsConnection{ edges { node { id, depth{ sequenceNumber buy{ price volume numberOfOrders} sell{ price volume numberOfOrders} lastTrade{ id buyer{id} seller{id} price size } } } } } }",
	}

	assertGraphQLQueriesReturnSame(t, queries["LiquidityProvisions"])

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}
}
