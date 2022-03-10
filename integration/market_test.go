package integration_test

import "testing"

func TestMarkets(t *testing.T) {
	queries := map[string]string{
		"Basic":            "{ markets{ id, name, decimalPlaces, tradingMode, state } }",
		"Fees":             "{ markets{ fees { factors { makerFee, infrastructureFee, liquidityFee } } } }",
		"Instrument":       "{ markets{ tradableInstrument{ instrument{ id, code, name, metadata{ tags } } } } }",
		"MarginCalculator": "{ markets{ tradableInstrument{ marginCalculator{ scalingFactors{ searchLevel, initialMargin,collateralRelease } } } } }",
		"PriceMonitor":     "{ markets{ priceMonitoringSettings{ parameters{ triggers{ horizonSecs, probability } } } } }",
		"LiquidityMonitor": "{ markets{ liquidityMonitoringParameters{ targetStakeParameters{ timeWindow, scalingFactor } triggeringRatio} } }",
		"Proposal":         "{ markets{ proposal{ id, reference, party { id }, state, datetime, rejectionReason} } }",
		"ProposalTerms":    "{ markets{ proposal{ terms{ closingDatetime, enactmentDatetime } } } }",
		"ProposalYes":      "{ markets{ proposal{ votes{ yes{ totalNumber totalWeight totalTokens} } } } }",
		"ProposalYesVotes": "{ markets{ proposal{ votes{ yes{ votes{value, party { id }, datetime, proposalId, governanceTokenBalance, governanceTokenWeight} } } } } }",
		"ProposalNo":       "{ markets{ proposal{ votes{ no{ totalNumber totalWeight totalTokens} } } } }",
		"PropsalNoVotes":   "{ markets{ proposal{ votes{ no{ votes{ value, party { id }, datetime, proposalId, governanceTokenBalance, governanceTokenWeight } } } } } }",
		"Orders":           "{ markets{ orders{ id, price, side, timeInForce, size, remaining, status, reference, type, rejectionReason, version, party{ id }, market{id}, trades{id} createdAt, expiresAt,  updatedAt, peggedOrder { reference, offset } } } }",
		"OrderLP":          "{ markets{ orders{ liquidityProvision{ commitmentAmount, fee, status, version, reference, createdAt, updatedAt, market { id } } } } }",
		"OrderTrades":      "{ markets{ trades{ id, price, size, createdAt, market{ id }, type, buyOrder, sellOrder, buyer{id}, seller{id}, aggressor, buyerAuctionBatch, sellerAuctionBatch } } }",
		"OrderBuyFees":     "{ markets{ trades{ buyerFee { makerFee, infrastructureFee, liquidityFee } } } }",
		"OrderSellFees":    "{ markets{ trades{ sellerFee { makerFee, infrastructureFee, liquidityFee } } } }",
		"Accounts":         "{ markets{ accounts { balance, asset {id}, type, market {id}, } } }",
		// TODO: Market depth / data / candles / liquidity provisions / timestamps
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ Markets []Market }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
