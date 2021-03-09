package core_test

import (
	"fmt"
	"time"

	"github.com/cucumber/godog/gherkin"

	types "code.vegaprotocol.io/vega/proto"
)

func TheMarket(table *gherkin.DataTable) error {
	markets := []types.Market{}

	for _, row := range TableWrapper(*table).Parse() {
		market := newMarket(marketRow{row: row})
		markets = append(markets, market)
	}

	t, _ := time.Parse("2006-01-02T15:04:05Z", marketStart)
	execsetup = getExecutionTestSetup(t, markets)

	// reset market start time and expiry for next run
	marketExpiry = defaultMarketExpiry
	marketStart = defaultMarketStart

	return nil
}

func newMarket(row marketRow) types.Market {
	market := types.Market{
		TradingMode:   types.Market_TRADING_MODE_CONTINUOUS,
		State:         types.Market_STATE_ACTIVE,
		Id:            row.name(),
		DecimalPlaces: 2,
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				LiquidityFee:      row.liquidityFee(),
				InfrastructureFee: row.infrastructureFee(),
				MakerFee:          row.makerFee(),
			},
		},
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				Id:   fmt.Sprintf("Crypto/%s/Futures", row.name()),
				Code: fmt.Sprintf("CRYPTO/%v", row.name()),
				Name: fmt.Sprintf("%s future", row.name()),
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				InitialMarkPrice: row.markPrice(),
				Product: &types.Instrument_Future{
					Future: &types.Future{
						Maturity: marketExpiry,
						Oracle: &types.Future_EthereumEvent{
							EthereumEvent: &types.EthereumEvent{
								ContractId: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
								Event:      "price_changed",
								Value:      row.settlementPrice(),
							},
						},
						SettlementAsset: row.asset(),
						QuoteName:       row.quoteName(),
					},
				},
			},
			MarginCalculator: &types.MarginCalculator{
				ScalingFactors: &types.ScalingFactors{
					SearchLevel:       row.searchLevelFactor(),
					InitialMargin:     row.initialMarginFactor(),
					CollateralRelease: row.collateralReleaseFactor(),
				},
			},
		},
		OpeningAuction: openingAuction(row),
		TradingModeConfig: &types.Market_Continuous{
			Continuous: &types.ContinuousTrading{},
		},
		PriceMonitoringSettings: &types.PriceMonitoringSettings{
			Parameters: &types.PriceMonitoringParameters{
				Triggers: priceMonitoringTriggers(row),
			},
			UpdateFrequency: row.priceMonitoringUpdateFrequency(),
		},
		TargetStakeParameters: &types.TargetStakeParameters{
			TimeWindow:    3600,
			ScalingFactor: 10,
		},
	}

	if row.isLogNormalRiskModel() {
		market.TradableInstrument.RiskModel = logNormalRiskModel(row)
	} else {
		market.TradableInstrument.RiskModel = simpleRiskModel(row)
	}

	return market

}

func simpleRiskModel(row marketRow) *types.TradableInstrument_SimpleRiskModel {
	return &types.TradableInstrument_SimpleRiskModel{
		SimpleRiskModel: &types.SimpleRiskModel{
			Params: &types.SimpleModelParams{
				FactorLong:           row.riskAversion(),
				FactorShort:          row.tau(),
				MaxMoveUp:            row.mu(),
				MinMoveDown:          row.r(),
				ProbabilityOfTrading: row.probabilityOfTrading(),
			},
		},
	}
}

func logNormalRiskModel(row marketRow) *types.TradableInstrument_LogNormalRiskModel {
	return &types.TradableInstrument_LogNormalRiskModel{
		LogNormalRiskModel: &types.LogNormalRiskModel{
			RiskAversionParameter: row.riskAversion(),
			Tau:                   row.tau(),
			Params: &types.LogNormalModelParams{
				Mu:    row.mu(),
				R:     row.r(),
				Sigma: row.sigma(),
			},
		},
	}
}

func openingAuction(row marketRow) *types.AuctionDuration {
	auction := &types.AuctionDuration{
		Duration: row.auctionDuration(),
	}

	if auction.Duration <= 0 {
		auction = nil
	}
	return auction
}

func priceMonitoringTriggers(row marketRow) []*types.PriceMonitoringTrigger {
	horizons := row.priceMonitoringHorizons()
	probabilities := row.priceMonitoringProbabilities()
	durations := row.priceMonitoringDurations()

	if len(horizons) != len(probabilities) || len(horizons) != len(durations) {
		panic(fmt.Sprintf(
			"horizons (%v), probabilities (%v) and durations (%v) need to have the same number of elements",
			len(horizons),
			len(probabilities),
			len(durations),
		))
	}

	triggers := make([]*types.PriceMonitoringTrigger, 0, len(horizons))
	for i := 0; i < len(horizons); i++ {
		p := &types.PriceMonitoringTrigger{
			Horizon:          horizons[i],
			Probability:      probabilities[i],
			AuctionExtension: durations[i],
		}
		triggers = append(triggers, p)
	}
	return triggers
}

// marketRow wraps the declaration of the properties of an oracle data
type marketRow struct {
	row RowWrapper
}

func (r marketRow) name() string {
	return r.row.Str("name")
}

func (r marketRow) quoteName() string {
	return r.row.Str("quote name")
}

func (r marketRow) asset() string {
	return r.row.Str("asset")
}

func (r marketRow) markPrice() uint64 {
	value, err := r.row.U64("mark price")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) riskModel() string {
	return r.row.Str("risk model")
}

func (r marketRow) isLogNormalRiskModel() bool {
	return r.riskModel() == "forward"
}

func (r marketRow) riskAversion() float64 {
	value, err := r.row.F64("lamd/long")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) tau() float64 {
	value, err := r.row.F64("tau/short")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) mu() float64 {
	value, err := r.row.F64("mu/max move up")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) r() float64 {
	value, err := r.row.F64("r/min move down")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) sigma() float64 {
	value, err := r.row.F64("sigma")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) collateralReleaseFactor() float64 {
	value, err := r.row.F64("release factor")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) initialMarginFactor() float64 {
	value, err := r.row.F64("initial factor")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) searchLevelFactor() float64 {
	value, err := r.row.F64("search factor")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) settlementPrice() uint64 {
	value, err := r.row.U64("settlement price")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) auctionDuration() int64 {
	value, err := r.row.I64("auction duration")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) makerFee() string {
	return r.row.Str("maker fee")
}

func (r marketRow) infrastructureFee() string {
	return r.row.Str("infrastructure fee")
}

func (r marketRow) liquidityFee() string {
	return r.row.Str("liquidity fee")
}

func (r marketRow) priceMonitoringUpdateFrequency() int64 {
	value, err := r.row.I64("p. m. update freq.")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) priceMonitoringHorizons() []int64 {
	value, err := r.row.I64Slice("p. m. horizons", ",")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) priceMonitoringProbabilities() []float64 {
	value, err := r.row.F64Slice("p. m. probs", ",")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) priceMonitoringDurations() []int64 {
	value, err := r.row.I64Slice("p. m. durations", ",")
	if err != nil {
		panic(err)
	}
	return value
}

func (r marketRow) probabilityOfTrading() float64 {
	value, err := r.row.F64("prob. of trading")
	if err != nil {
		panic(err)
	}
	return value
}
