package monitor_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/monitor"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func getMarket(closingAt time.Time, openingAuctionDuration *types.AuctionDuration) types.Market {
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
				Product: &types.Instrument_Future{
					Future: &types.Future{
						Maturity:        closingAt.Format(time.RFC3339),
						SettlementAsset: "ETH",
						QuoteName:       "USD",
						OracleSpecBinding: &types.OracleSpecToFutureBinding{
							SettlementPriceProperty:    "prices.ETH.value",
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
	mktCfg := getMarket(time.Now(), ad)
	return monitor.NewAuctionState(&mktCfg, time.Now())
}

func getHash(as *monitor.AuctionState) []byte {
	state := as.GetState()
	pmproto := state.IntoProto()
	bytes, _ := proto.Marshal(pmproto)
	return crypto.Hash(bytes)
}

func TestEmpty(t *testing.T) {
	as := createAuctionState()

	// Get the hash and state for the empty object
	hash1 := getHash(as)
	state1 := as.GetState()

	// Create a new object and restore into it
	ad := &types.AuctionDuration{
		Duration: 100,
		Volume:   100,
	}
	mktCfg := getMarket(time.Now(), ad)
	as2 := monitor.NewAuctionStateFromSnapshot(&mktCfg, state1)

	// Check the new hash matches the old hash
	hash2 := getHash(as2)
	assert.Equal(t, hash1, hash2)
}

func TestChangedState(t *testing.T) {
	as := createAuctionState()

	// Get the hash for the empty object
	hash1 := getHash(as)

	// Perform some updates to the object
	as.StartPriceAuction(time.Now(), &types.AuctionDuration{
		Duration: 200,
		Volume:   200,
	})

	// Make sure we thinks things have changed
	assert.True(t, as.Changed())

	// Get the new hash and check it's different to the original
	hash2 := getHash(as)

	assert.NotEqual(t, hash1, hash2)
}
