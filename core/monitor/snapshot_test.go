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

package monitor_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/monitor"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getMarket(openingAuctionDuration *types.AuctionDuration) types.Market {
	mkt := types.Market{
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				LiquidityFee:      num.DecimalFromFloat(0.3),
				InfrastructureFee: num.DecimalFromFloat(0.001),
				MakerFee:          num.DecimalFromFloat(0.004),
			},
		},
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				ID:   "Crypto/ETHUSD/Futures/Dec19",
				Code: "CRYPTO:ETHUSD/DEC19",
				Name: "December 2019 ETH vs USD future",
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				Product: &types.InstrumentFuture{
					Future: &types.Future{
						SettlementAsset: "ETH",
						QuoteName:       "USD",
						DataSourceSpecBinding: &types.DataSourceSpecBindingForFuture{
							SettlementDataProperty:     "prices.ETH.value",
							TradingTerminationProperty: "trading.terminated",
						},
					},
				},
			},
			MarginCalculator: &types.MarginCalculator{
				ScalingFactors: &types.ScalingFactors{
					SearchLevel:       num.DecimalFromFloat(1.1),
					InitialMargin:     num.DecimalFromFloat(1.2),
					CollateralRelease: num.DecimalFromFloat(1.4),
				},
			},
			RiskModel: &types.TradableInstrumentSimpleRiskModel{
				SimpleRiskModel: &types.SimpleRiskModel{
					Params: &types.SimpleModelParams{
						FactorLong:           num.DecimalFromFloat(0.15),
						FactorShort:          num.DecimalFromFloat(0.25),
						MaxMoveUp:            num.DecimalFromFloat(100.0),
						MinMoveDown:          num.DecimalFromFloat(100.0),
						ProbabilityOfTrading: num.DecimalFromFloat(0.1),
					},
				},
			},
		},
		OpeningAuction: openingAuctionDuration,
		LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
			TargetStakeParameters: &types.TargetStakeParameters{
				TimeWindow:    3600, // seconds = 1h
				ScalingFactor: num.DecimalFromFloat(10),
			},
			TriggeringRatio: num.DecimalZero(),
		},
	}
	return mkt
}

func createAuctionState() *monitor.AuctionState {
	ad := &types.AuctionDuration{
		Duration: 100,
		Volume:   100,
	}
	mktCfg := getMarket(ad)
	return monitor.NewAuctionState(&mktCfg, time.Now())
}

func getHash(t *testing.T, as *monitor.AuctionState) []byte {
	t.Helper()
	state := as.GetState()
	pmproto := state.IntoProto()
	bytes, err := proto.Marshal(pmproto)
	require.NoError(t, err)

	// Check our change flag has been reset
	return crypto.Hash(bytes)
}

func TestEmpty(t *testing.T) {
	as := createAuctionState()

	// Get the hash and state for the empty object
	hash1 := getHash(t, as)
	state1 := as.GetState()

	// Create a new object and restore into it
	ad := &types.AuctionDuration{
		Duration: 100,
		Volume:   100,
	}
	mktCfg := getMarket(ad)
	as2 := monitor.NewAuctionStateFromSnapshot(&mktCfg, state1)

	// Check the new hash matches the old hash
	assert.Equal(t, hash1, getHash(t, as2))
}

func TestRestoreTriggerType(t *testing.T) {
	as := createAuctionState()

	// Perform some updates to the object
	as.StartPriceAuction(time.Now(), &types.AuctionDuration{
		Duration: 200,
		Volume:   200,
	})

	asNew := monitor.NewAuctionStateFromSnapshot(nil, as.GetState())
	require.Equal(t, as.IsPriceAuction(), asNew.IsPriceAuction())
}

func TestChangedState(t *testing.T) {
	as := createAuctionState()

	// Get the hash for the empty object
	original := getHash(t, as)

	// Perform some updates to the object
	as.StartPriceAuction(time.Now(), &types.AuctionDuration{
		Duration: 200,
		Volume:   200,
	})

	// Make sure we thinks things have changed
	assert.True(t, as.Changed())

	auctionStart := getHash(t, as)
	assert.NotEqual(t, original, auctionStart)

	// extend the auction
	as.ExtendAuction(types.AuctionDuration{Duration: 12, Volume: 12})
	assert.True(t, as.Changed())

	auctionExtended := getHash(t, as)
	assert.NotEqual(t, auctionStart, auctionExtended)

	// set Ready to leave
	as.SetReadyToLeave()
	assert.True(t, as.Changed())

	auctionReady := getHash(t, as)
	assert.NotEqual(t, auctionStart, auctionReady)

	// end it
	as.Left(context.Background(), time.Now())
	assert.True(t, as.Changed())

	auctionEnded := getHash(t, as)
	assert.NotEqual(t, auctionStart, auctionEnded)
}

// TestAuctionTypeChain checks that if an auction is ended, then started again it is logged as a change.
func TestAuctionEndsOpens(t *testing.T) {
	as := createAuctionState()
	now := time.Now()
	// Perform some updates to the object
	as.StartOpeningAuction(now, &types.AuctionDuration{
		Duration: 200,
		Volume:   200,
	})

	// Get the hash of a started auction
	require.True(t, as.Changed())
	original := getHash(t, as)

	// Close it down and then start exactly the same auction again
	as.Left(context.Background(), now)
	require.False(t, as.InAuction()) // definitely no auction

	as.StartOpeningAuction(now, &types.AuctionDuration{
		Duration: 200,
		Volume:   200,
	})

	require.True(t, as.Changed())
	newAuction := getHash(t, as)
	// change flagged even though hash is exactly the same (which is expected given all the state change that actually occurred)
	assert.Equal(t, original, newAuction)
}
