package steps

import (
	"fmt"

	"github.com/cucumber/godog/gherkin"

	types "code.vegaprotocol.io/vega/proto"
	oraclesv1 "code.vegaprotocol.io/vega/proto/oracles/v1"
)

func TheMarkets(
	expiry string,
	table *gherkin.DataTable,
) []types.Market {
	markets := []types.Market{}

	for _, row := range TableWrapper(*table).Parse() {
		market := newMarket(expiry, marketRow{row: row})
		markets = append(markets, market)
	}

	return markets
}

func newMarket(marketExpiry string, row marketRow) types.Market {
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
				Product: &types.Instrument_Future{
					Future: &types.Future{
						Maturity:        marketExpiry,
						SettlementAsset: row.asset(),
						QuoteName:       row.quoteName(),
						OracleSpec: &oraclesv1.OracleSpec{
							PubKeys: row.oracleSpecPubKeys(),
							Filters: []*oraclesv1.Filter{
								{
									Key: &oraclesv1.PropertyKey{
										Name: row.oracleSpecProperty(),
										Type: row.oracleSpecPropertyType(),
									},
									Conditions: []*oraclesv1.Condition{},
								},
							},
						},
						OracleSpecBinding: &types.OracleSpecToFutureBinding{
							SettlementPriceProperty: row.oracleSpecBinding(),
						},
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
		LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
			TargetStakeParameters: &types.TargetStakeParameters{
				TimeWindow:    3600,
				ScalingFactor: 10,
			},
			TriggeringRatio: 0,
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

func (r marketRow) riskModel() string {
	return r.row.Str("risk model")
}

func (r marketRow) isLogNormalRiskModel() bool {
	return r.riskModel() == "forward"
}

func (r marketRow) riskAversion() float64 {
	return r.row.F64("lamd/long")
}

func (r marketRow) tau() float64 {
	return r.row.F64("tau/short")
}

func (r marketRow) mu() float64 {
	return r.row.F64("mu/max move up")
}

func (r marketRow) r() float64 {
	return r.row.F64("r/min move down")
}

func (r marketRow) sigma() float64 {
	return r.row.F64("sigma")
}

func (r marketRow) collateralReleaseFactor() float64 {
	return r.row.F64("release factor")
}

func (r marketRow) initialMarginFactor() float64 {
	return r.row.F64("initial factor")
}

func (r marketRow) searchLevelFactor() float64 {
	return r.row.F64("search factor")
}

func (r marketRow) auctionDuration() int64 {
	return r.row.I64("auction duration")
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
	return r.row.I64("p. m. update freq.")
}

func (r marketRow) priceMonitoringHorizons() []int64 {
	return r.row.I64Slice("p. m. horizons", ",")
}

func (r marketRow) priceMonitoringProbabilities() []float64 {
	return r.row.F64Slice("p. m. probs", ",")
}

func (r marketRow) priceMonitoringDurations() []int64 {
	return r.row.I64Slice("p. m. durations", ",")
}

func (r marketRow) probabilityOfTrading() float64 {
	return r.row.F64("prob. of trading")
}

func (r marketRow) oracleSpecPubKeys() []string {
	return r.row.StrSlice("oracle spec pub. keys", ",")
}

func (r marketRow) oracleSpecProperty() string {
	return r.row.Str("oracle spec property")
}

func (r marketRow) oracleSpecPropertyType() oraclesv1.PropertyKey_Type {
	return r.row.OracleSpecPropertyType("oracle spec property type")
}

func (r marketRow) oracleSpecBinding() string {
	return r.row.Str("oracle spec binding")
}
