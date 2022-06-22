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

package steps

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/integration/steps/market"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func TheMarkets(
	config *market.Config,
	executionEngine Execution,
	collateralEngine *collateral.Engine,
	netparams *netparams.Store,
	table *godog.Table,
) ([]types.Market, error) {
	rows := parseMarketsTable(table)
	markets := make([]types.Market, 0, len(rows))

	for _, row := range rows {
		mkt := newMarket(config, netparams, marketRow{row: row})
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
		err := executionEngine.SubmitMarket(context.Background(), &markets[i], "proposerID")
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

func newMarket(config *market.Config, netparams *netparams.Store, row marketRow) types.Market {
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

	settlementPriceDecimals := config.OracleConfigs.GetSettlementPriceDP(row.oracleConfig())
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

	// the governance engine would fill in the liquidity monitor parameters from the network parameters (unless set explicitly)
	// so we need to do this here by hand. If the network parameters weren't set we use the below defaults
	timeWindow := int64(3600)
	scalingFactor := num.DecimalFromInt64(10)
	triggeringRatio := num.DecimalFromInt64(0)

	if tw, err := netparams.GetDuration("market.stake.target.timeWindow"); err == nil {
		timeWindow = int64(tw.Seconds())
	}

	if sf, err := netparams.GetDecimal("market.stake.target.scalingFactor"); err == nil {
		scalingFactor = sf
	}

	if tr, err := netparams.GetDecimal("market.liquidity.targetstake.triggering.ratio"); err == nil {
		triggeringRatio = tr
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
				Product: &types.InstrumentFuture{
					Future: &types.Future{
						SettlementAsset:                 row.asset(),
						QuoteName:                       row.quoteName(),
						OracleSpecForSettlementPrice:    types.OracleSpecFromProto(oracleConfigForSettlement.Spec),
						OracleSpecForTradingTermination: types.OracleSpecFromProto(oracleConfigForTradingTermination.Spec),
						OracleSpecBinding:               types.OracleSpecBindingForFutureFromProto(&binding),
						SettlementPriceDecimals:         settlementPriceDecimals,
					},
				},
			},
			MarginCalculator: types.MarginCalculatorFromProto(marginCalculator),
		},
		OpeningAuction:          openingAuction(row),
		PriceMonitoringSettings: types.PriceMonitoringSettingsFromProto(priceMonitoring),
		LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
			TargetStakeParameters: &types.TargetStakeParameters{
				TimeWindow:    timeWindow,
				ScalingFactor: scalingFactor,
			},
			TriggeringRatio: triggeringRatio,
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
