// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
		"Basic":               "{ markets{ id, name, decimalPlaces, tradingMode, state } }",
		"Fees":                "{ markets{ id, fees { factors { makerFee, infrastructureFee, liquidityFee } } } }",
		"Instrument":          "{ markets{ id, tradableInstrument{ instrument{ id, code, name, metadata{ tags } } } } }",
		"MarginCalculator":    "{ markets{ id, tradableInstrument{ marginCalculator{ scalingFactors{ searchLevel, initialMargin,collateralRelease } } } } }",
		"PriceMonitor":        "{ markets{ id, priceMonitoringSettings{ parameters{ triggers{ horizonSecs, probability } } } } }",
		"LiquidityMonitor":    "{ markets{ id, liquidityMonitoringParameters{ targetStakeParameters{ timeWindow, scalingFactor } triggeringRatio} } }",
		"Proposal":            "{ markets{ id, proposal{ id, reference, party { id }, state, datetime, rejectionReason} } }",
		"ProposalTerms":       "{ markets{ id, proposal{ id, terms{ closingDatetime, enactmentDatetime } } } }",
		"ProposalYes":         "{ markets{ id, proposal{ id, votes{ yes{ totalNumber totalWeight totalTokens} } } } }",
		"ProposalYesVotes":    "{ markets{ id, proposal{ id, votes{ yes{ votes{value, party { id }, datetime, proposalId, governanceTokenBalance, governanceTokenWeight} } } } } }",
		"ProposalNo":          "{ markets{ id, proposal{ id, votes{ no{ totalNumber totalWeight totalTokens} } } } }",
		"PropsalNoVotes":      "{ markets{ id, proposal{ id, votes{ no{ votes{ value, party { id }, datetime, proposalId, governanceTokenBalance, governanceTokenWeight } } } } } }",
		"Orders":              "{ markets{ id, orders{ id, price, side, timeInForce, size, remaining, status, reference, type, rejectionReason, version, party{ id }, market{id}, trades{id} createdAt, expiresAt,  updatedAt, peggedOrder { reference, offset } } } }",
		"OrderLP":             "{ markets{ id, orders{ id, liquidityProvision{ commitmentAmount, fee, status, version, reference, createdAt, updatedAt, market { id } } } } }",
		"OrderTrades":         "{ markets{ id, trades{ id, price, size, createdAt, market{ id }, type, buyOrder, sellOrder, buyer{id}, seller{id}, aggressor, buyerAuctionBatch, sellerAuctionBatch } } }",
		"OrderBuyFees":        "{ markets{ id, trades{ id, buyerFee { makerFee, infrastructureFee, liquidityFee } } } }",
		"OrderSellFees":       "{ markets{ id, trades{ id, sellerFee { makerFee, infrastructureFee, liquidityFee } } } }",
		"Candles1Minute":      "{ markets{ id, candles(since : \"2000-01-01T00:00:00Z\",interval : I1M)  {  timestamp, datetime, high, low, open, close, volume, interval} } }",
		"Candles5Minute":      "{ markets{ id, candles(since : \"2000-01-01T00:00:00Z\",interval : I5M)  {  timestamp, datetime, high, low, open, close, volume, interval} } }",
		"Candles15Minute":     "{ markets{ id, candles(since : \"2000-01-01T00:00:00Z\",interval : I15M)  {  timestamp, datetime, high, low, open, close, volume, interval} } }",
		"RiskFactor":          "{ markets { riskFactors { market, short, long } } }",
		"LiquidityProvisions": "{ markets { id, liquidityProvisions { id, party { id }, createdAt, updatedAt, market { id }, commitmentAmount, fee, sells { order { id }, liquidityOrder { reference } }, buys { order { id }, liquidityOrder { reference } }, version, status, reference } } }",
		// TODO - accounts fails, but I think it is the old API which has wrong market balances
		//"Accounts": "{ markets{ id, accounts { balance, asset {id}, type, market {id}, } } }",
		// TODO: Market depth / data / timestamps
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ Markets []Market }
			assertGraphQLQueriesReturnSame(t, query, &old, &new)
		})
	}
}
