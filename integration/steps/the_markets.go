package steps

import (
	"context"
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/steps/market"
	types "code.vegaprotocol.io/vega/proto"
)

func TheMarkets(
	config *market.Config,
	executionEngine *execution.Engine,
	collateralEngine *collateral.Engine,
	table *gherkin.DataTable,
) ([]types.Market, error) {
	var markets []types.Market
	for _, row := range parseMarketsTable(table) {
		mkt := newMarket(config, marketRow{row: row})
		markets = append(markets, mkt)
	}

	if err := enableMarketAssets(markets, collateralEngine); err != nil {
		return nil, err
	}

	if err := enableVoteAsset(collateralEngine); err != nil {
		return nil, err
	}

	if err := submitMarkets(markets, executionEngine); err != nil {
		return nil, err
	}

	return markets, nil
}

func submitMarkets(markets []types.Market, executionEngine *execution.Engine) error {
	for _, mkt := range markets {
		err := executionEngine.SubmitMarket(context.Background(), &mkt)
		if err != nil {
			return fmt.Errorf("couldn't submit market(%s): %v", mkt.Id, err)
		}
	}
	return nil
}

func enableMarketAssets(markets []types.Market, collateralEngine *collateral.Engine) error {
	assetsToEnable := map[string]struct{}{}
	for _, mkt := range markets {
		asset, _ := mkt.GetAsset()
		assetsToEnable[asset] = struct{}{}
	}
	for assetToEnable := range assetsToEnable {
		err := collateralEngine.EnableAsset(context.Background(), types.Asset{
			Id: assetToEnable,
			Details: &types.AssetDetails{
				Symbol: assetToEnable,
			},
		})
		if err != nil {
			return fmt.Errorf("couldn't enable asset(%s): %v", assetToEnable, err)
		}
	}
	return nil
}

func enableVoteAsset(collateralEngine *collateral.Engine) error {
	voteAsset := types.Asset{
		Id: "VOTE",
		Details: &types.AssetDetails{
			Name:        "VOTE",
			Symbol:      "VOTE",
			Decimals:    5,
			TotalSupply: "1000",
			Source: &types.AssetDetails_BuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: "10",
				},
			},
		},
	}

	err := collateralEngine.EnableAsset(context.Background(), voteAsset)
	if err != nil {
		return fmt.Errorf("couldn't enable asset(%s): %v", voteAsset.Id, err)
	}
	return nil
}

func newMarket(config *market.Config, row marketRow) types.Market {
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
						Maturity:          row.maturityDate(),
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

func parseMarketsTable(table *gherkin.DataTable) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"quote name",
		"asset",
		"risk model",
		"fees",
		"oracle config",
		"price monitoring",
		"margin calculator",
		"auction duration",
	}, []string{
		"maturity date",
	})
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

func (r marketRow) maturityDate() string {
	if !r.row.HasColumn("maturity date") {
		return "2019-12-31T23:59:59Z"
	}

	time := r.row.MustTime("maturity date")
	timeNano := time.UnixNano()
	if timeNano == 0 {
		panic(fmt.Errorf("maturity date is required"))
	}

	return r.row.Str("maturity date")
}
