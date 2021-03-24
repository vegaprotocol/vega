package steps

import (
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/steps/market"
	types "code.vegaprotocol.io/vega/proto"
)

func TheMarkets(
	config *market.Config,
	expiry string,
	table *gherkin.DataTable,
) []types.Market {
	markets := []types.Market{}

	for _, row := range TableWrapper(*table).Parse() {
		m := newMarket(config, expiry, marketRow{row: row})
		markets = append(markets, m)
	}

	return markets
}

func newMarket(config *market.Config, marketExpiry string, row marketRow) types.Market {
	fees, err := config.FeesConfig.Get(row.fees())
	if err != nil {
		panic(err)
	}

	oracleConfig, err := config.OracleConfigs.Get(row.oracleConfig())
	if err != nil {
		panic(err)
	}

	priceMonitoring, err := config.PriceMonitoring.Get(row.priceMonitoring())
	if err != nil {
		panic(err)
	}

	marginCalculator, err := config.MarginCalculators.Get(row.marginCalculator())
	if err != nil {
		panic(err)
	}

	m := types.Market{
		TradingMode:   types.Market_TRADING_MODE_CONTINUOUS,
		State:         types.Market_STATE_ACTIVE,
		Id:            row.id(),
		DecimalPlaces: 2,
		Fees:          fees,
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				Id:   fmt.Sprintf("Crypto/%s/Futures", row.id()),
				Code: fmt.Sprintf("CRYPTO/%v", row.id()),
				Name: fmt.Sprintf("%s future", row.id()),
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				Product: &types.Instrument_Future{
					Future: &types.Future{
						Maturity:          marketExpiry,
						SettlementAsset:   row.asset(),
						QuoteName:         row.quoteName(),
						OracleSpec:        oracleConfig.Spec,
						OracleSpecBinding: oracleConfig.Binding,
					},
				},
			},
			MarginCalculator: marginCalculator,
		},
		OpeningAuction: openingAuction(row),
		TradingModeConfig: &types.Market_Continuous{
			Continuous: &types.ContinuousTrading{},
		},
		PriceMonitoringSettings: priceMonitoring,
		LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
			TargetStakeParameters: &types.TargetStakeParameters{
				TimeWindow:    3600,
				ScalingFactor: 10,
			},
			TriggeringRatio: 0,
		},
	}

	err = config.RiskModels.LoadModel(row.riskModel(), m.TradableInstrument)
	if err != nil {
		panic(err)
	}

	return m

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

type marketRow struct {
	row RowWrapper
}

func (r marketRow) id() string {
	return r.row.MustStr("id")
}

func (r marketRow) quoteName() string {
	return r.row.MustStr("quote name")
}

func (r marketRow) asset() string {
	return r.row.MustStr("asset")
}

func (r marketRow) riskModel() string {
	return r.row.MustStr("risk model")
}

func (r marketRow) fees() string {
	return r.row.MustStr("fees")
}

func (r marketRow) oracleConfig() string {
	return r.row.MustStr("oracle config")
}

func (r marketRow) priceMonitoring() string {
	return r.row.MustStr("price monitoring")
}

func (r marketRow) marginCalculator() string {
	return r.row.MustStr("margin calculator")
}

func (r marketRow) auctionDuration() int64 {
	return r.row.MustI64("auction duration")
}

