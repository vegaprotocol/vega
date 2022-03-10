package integration_test

import "testing"

func TestMarkets(t *testing.T) {
	queries := map[string]string{
		"Basic":            "{ markets{ id, name, decimalPlaces, tradingMode, state } }",
		"Fees":             "{ markets{ id, fees { factors { makerFee, infrastructureFee, liquidityFee } } } }",
		"Instrument":       "{ markets{ id, tradableInstrument{ instrument{ id, code, name, metadata{ tags } } } } }",
		"MarginCalculator": "{ markets{ id, tradableInstrument{ marginCalculator{ scalingFactors{ searchLevel, initialMargin,collateralRelease } } } } }",
		"PriceMonitor":     "{ markets{ id, priceMonitoringSettings{ parameters{ triggers{ horizonSecs, probability } } } } }",
		"LiquidityMonitor": "{ markets{ id, liquidityMonitoringParameters{ targetStakeParameters{ timeWindow, scalingFactor } triggeringRatio} } }",
		"Proposal":         "{ markets{ id, proposal{ id, reference, party { id }, state, datetime, rejectionReason} } }",
		"ProposalTerms":    "{ markets{ id, proposal{ terms{ closingDatetime, enactmentDatetime } } } }",
		"ProposalYes":      "{ markets{ id, proposal{ votes{ yes{ totalNumber totalWeight totalTokens} } } } }",
		"ProposalYesVotes": "{ markets{ id, proposal{ votes{ yes{ votes{value, party { id }, datetime, proposalId, governanceTokenBalance, governanceTokenWeight} } } } } }",
		"ProposalNo":       "{ markets{ id, proposal{ votes{ no{ totalNumber totalWeight totalTokens} } } } }",
		"PropsalNoVotes":   "{ markets{ id, proposal{ votes{ no{ votes{ value, party { id }, datetime, proposalId, governanceTokenBalance, governanceTokenWeight } } } } } }",
		"Orders":           "{ markets{ id, orders{ id, price, side, timeInForce, size, remaining, status, reference, type, rejectionReason, version, party{ id }, market{id}, trades{id} createdAt, expiresAt,  updatedAt, peggedOrder { reference, offset } } } }",
		"OrderLP":          "{ markets{ id, orders{ id, liquidityProvision{ commitmentAmount, fee, status, version, reference, createdAt, updatedAt, market { id } } } } }",
		"OrderTrades":      "{ markets{ id, trades{ id, price, size, createdAt, market{ id }, type, buyOrder, sellOrder, buyer{id}, seller{id}, aggressor, buyerAuctionBatch, sellerAuctionBatch } } }",
		"OrderBuyFees":     "{ markets{ id, trades{ id, buyerFee { makerFee, infrastructureFee, liquidityFee } } } }",
		"OrderSellFees":    "{ markets{ id, trades{ id, sellerFee { makerFee, infrastructureFee, liquidityFee } } } }",
		// TODO - accounts fails, but I think it is the old API which has wrong market balances
		//"Accounts": "{ markets{ id, accounts { balance, asset {id}, type, market {id}, } } }",
		// TODO: Market depth / data / candles / liquidity provisions / timestamps
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ Markets []Market }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
