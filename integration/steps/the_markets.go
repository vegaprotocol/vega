package steps

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/integration/steps/market"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func TheMarkets(
	config *market.Config,
	executionEngine Execution,
	collateralEngine *collateral.Engine,
	table *godog.Table,
) ([]types.Market, error) {
	rows := parseMarketsTable(table)
	markets := make([]types.Market, 0, len(rows))
	for _, row := range rows {
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

func submitMarkets(markets []types.Market, executionEngine Execution) error {
	for i := range markets {
		err := executionEngine.SubmitMarket(context.Background(), &markets[i])
		if err != nil {
			return fmt.Errorf("couldn't submit market(%s): %v", markets[i].ID, err)
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
			ID: assetToEnable,
			Details: &types.AssetDetails{
				Quantum: num.DecimalZero(),
				Symbol:  assetToEnable,
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
		ID: "VOTE",
		Details: &types.AssetDetails{
			Name:        "VOTE",
			Symbol:      "VOTE",
			Decimals:    5,
			TotalSupply: num.NewUint(1000),
			Source: &types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.NewUint(10),
				},
			},
		},
	}

	err := collateralEngine.EnableAsset(context.Background(), voteAsset)
	if err != nil {
		return fmt.Errorf("couldn't enable asset(%s): %v", voteAsset.ID, err)
	}
	return nil
}

func newMarket(config *market.Config, row marketRow) types.Market {
	fees, err := config.FeesConfig.Get(row.fees())
	if err != nil {
		panic(err)
	}

	oracleConfigForSettlement, err := config.OracleConfigs.Get(row.oracleConfig(), "settlement price")
	if err != nil {
		panic(err)
	}

	oracleConfigForTradingTermination, err := config.OracleConfigs.Get(row.oracleConfig(), "trading termination")
	if err != nil {
		panic(err)
	}

	var binding proto.OracleSpecToFutureBinding
	binding.SettlementPriceProperty = oracleConfigForSettlement.Binding.SettlementPriceProperty
	binding.TradingTerminationProperty = oracleConfigForTradingTermination.Binding.TradingTerminationProperty

	priceMonitoring, err := config.PriceMonitoring.Get(row.priceMonitoring())
	if err != nil {
		panic(err)
	}

	marginCalculator, err := config.MarginCalculators.Get(row.marginCalculator())
	if err != nil {
		panic(err)
	}

	m := types.Market{
		TradingMode:           types.MarketTradingModeContinuous,
		State:                 types.MarketStateActive,
		ID:                    row.id(),
		DecimalPlaces:         row.decimalPlaces(),
		PositionDecimalPlaces: row.positionDecimalPlaces(),
		Fees:                  types.FeesFromProto(fees),
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				ID:   fmt.Sprintf("Crypto/%s/Futures", row.id()),
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
						Maturity:                        row.maturityDate(),
						SettlementAsset:                 row.asset(),
						QuoteName:                       row.quoteName(),
						OracleSpecForSettlementPrice:    oracleConfigForSettlement.Spec,
						OracleSpecForTradingTermination: oracleConfigForTradingTermination.Spec,
						OracleSpecBinding:               types.OracleSpecToFutureBindingFromProto(&binding),
					},
				},
			},
			MarginCalculator: types.MarginCalculatorFromProto(marginCalculator),
		},
		OpeningAuction:          openingAuction(row),
		PriceMonitoringSettings: types.PriceMonitoringSettingsFromProto(priceMonitoring),
		LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
			TargetStakeParameters: &types.TargetStakeParameters{
				TimeWindow:    3600,
				ScalingFactor: num.NewDecimalFromFloat(10),
			},
			TriggeringRatio: num.NewDecimalFromFloat(0),
		},
	}

	tip := m.TradableInstrument.IntoProto()
	err = config.RiskModels.LoadModel(row.riskModel(), tip)
	m.TradableInstrument = types.TradableInstrumentFromProto(tip)
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

func parseMarketsTable(table *godog.Table) []RowWrapper {
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
		"decimal places",
		"position decimal places",
	})
}

type marketRow struct {
	row RowWrapper
}

func (r marketRow) id() string {
	return r.row.MustStr("id")
}

func (r marketRow) decimalPlaces() uint64 {
	if !r.row.HasColumn("decimal places") {
		return 0
	}
	return r.row.MustU64("decimal places")
}

func (r marketRow) positionDecimalPlaces() uint64 {
	if !r.row.HasColumn("position decimal places") {
		return 0
	}
	return r.row.MustU64("position decimal places")
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
	if timeNano := time.UnixNano(); timeNano == 0 {
		panic(fmt.Errorf("maturity date is required"))
	}

	return r.row.Str("maturity date")
}
