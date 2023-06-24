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

package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/integration/steps/market"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

func TheSpotMarkets(config *market.Config, executionEngine Execution, collateralEngine *collateral.Engine, netparams *netparams.Store, now time.Time, table *godog.Table) ([]types.Market, error) {
	rows := parseSpotMarketsTable(table)
	markets := make([]types.Market, 0, len(rows))

	for _, row := range rows {
		mkt := newSpotMarket(config, netparams, spotMarketRow{row: row})
		markets = append(markets, mkt)
	}

	if err := enableSpotMarketAssets(markets, collateralEngine); err != nil {
		return nil, err
	}

	if err := enableVoteAsset(collateralEngine); err != nil {
		return nil, err
	}

	if err := submitSpotMarkets(markets, executionEngine, now); err != nil {
		return nil, err
	}

	return markets, nil
}

func submitSpotMarkets(markets []types.Market, executionEngine Execution, now time.Time) error {
	for i := range markets {
		if err := executionEngine.SubmitSpotMarket(context.Background(), &markets[i], "proposerID", now); err != nil {
			return fmt.Errorf("couldn't submit market(%s): %v", markets[i].ID, err)
		}
		if err := executionEngine.StartOpeningAuction(context.Background(), markets[i].ID); err != nil {
			return fmt.Errorf("could not start opening auction for market %s: %v", markets[i].ID, err)
		}
	}
	return nil
}

func enableSpotMarketAssets(markets []types.Market, collateralEngine *collateral.Engine) error {
	assetsToEnable := map[string]struct{}{}
	for _, mkt := range markets {
		assets, _ := mkt.GetAssets()
		for _, asset := range assets {
			assetsToEnable[asset] = struct{}{}
		}
	}
	for assetToEnable := range assetsToEnable {
		err := collateralEngine.EnableAsset(context.Background(), types.Asset{
			ID: assetToEnable,
			Details: &types.AssetDetails{
				Quantum: num.DecimalZero(),
				Symbol:  assetToEnable,
			},
		})
		if err != nil && err != collateral.ErrAssetAlreadyEnabled {
			return fmt.Errorf("couldn't enable asset(%s): %v", assetToEnable, err)
		}
	}
	return nil
}

func (r spotMarketRow) liquidityMonitoring() string {
	if !r.row.HasColumn("liquidity monitoring") {
		return "default-parameters"
	}
	return r.row.MustStr("liquidity monitoring")
}

func setSpotLiquidityMonitoringNetParams(liqMon *types.LiquidityMonitoringParameters, netparams *netparams.Store) {
	// the governance engine would fill in the liquidity monitor parameters from the network parameters (unless set explicitly)
	// so we do this step here manually
	if tw, err := netparams.GetDuration("market.stake.target.timeWindow"); err == nil {
		liqMon.TargetStakeParameters.TimeWindow = int64(tw.Seconds())
	}

	if sf, err := netparams.GetDecimal("market.stake.target.scalingFactor"); err == nil {
		liqMon.TargetStakeParameters.ScalingFactor = sf
	}
}

func newSpotMarket(config *market.Config, netparams *netparams.Store, row spotMarketRow) types.Market {
	fees, err := config.FeesConfig.Get(row.fees())
	if err != nil {
		panic(err)
	}

	priceMonitoring, err := config.PriceMonitoring.Get(row.priceMonitoring())
	if err != nil {
		panic(err)
	}

	liqMon, err := config.LiquidityMonitoring.GetType(row.liquidityMonitoring())
	if err != nil {
		panic(err)
	}

	setSpotLiquidityMonitoringNetParams(liqMon, netparams)

	m := types.Market{
		TradingMode:           types.MarketTradingModeContinuous,
		State:                 types.MarketStateActive,
		ID:                    row.id(),
		DecimalPlaces:         row.decimalPlaces(),
		PositionDecimalPlaces: row.positionDecimalPlaces(),
		Fees:                  types.FeesFromProto(fees),
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				ID:   fmt.Sprintf("Crypto/%s/Spots", row.id()),
				Code: fmt.Sprintf("CRYPTO/%v", row.id()),
				Name: fmt.Sprintf("%s spot", row.id()),
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:spot/crypto",
						"product:spots",
					},
				},
				Product: &types.InstrumentSpot{
					Spot: &types.Spot{
						BaseAsset:  row.baseAsset(),
						QuoteAsset: row.quoteAsset(),
						Name:       row.name(),
					},
				},
			},
		},
		OpeningAuction:                spotOpeningAuction(row),
		PriceMonitoringSettings:       types.PriceMonitoringSettingsFromProto(priceMonitoring),
		LiquidityMonitoringParameters: liqMon,
	}

	tip := m.TradableInstrument.IntoProto()
	err = config.RiskModels.LoadModel(row.riskModel(), tip)
	m.TradableInstrument = types.TradableInstrumentFromProto(tip)
	if err != nil {
		panic(err)
	}

	return m
}

func spotOpeningAuction(row spotMarketRow) *types.AuctionDuration {
	auction := &types.AuctionDuration{
		Duration: row.auctionDuration(),
	}

	if auction.Duration <= 0 {
		auction = nil
	}
	return auction
}

func parseSpotMarketsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"name",
		"base asset",
		"quote asset",
		"risk model",
		"fees",
		"price monitoring",
		"auction duration",
	}, []string{
		"decimal places",
		"position decimal places",
	})
}

type spotMarketRow struct {
	row RowWrapper
}

func (r spotMarketRow) id() string {
	return r.row.MustStr("id")
}

func (r spotMarketRow) decimalPlaces() uint64 {
	if !r.row.HasColumn("decimal places") {
		return 0
	}
	return r.row.MustU64("decimal places")
}

func (r spotMarketRow) positionDecimalPlaces() int64 {
	if !r.row.HasColumn("position decimal places") {
		return 0
	}
	return r.row.MustI64("position decimal places")
}

func (r spotMarketRow) name() string {
	return r.row.MustStr("name")
}

func (r spotMarketRow) baseAsset() string {
	return r.row.MustStr("base asset")
}

func (r spotMarketRow) quoteAsset() string {
	return r.row.MustStr("quote asset")
}

func (r spotMarketRow) riskModel() string {
	return r.row.MustStr("risk model")
}

func (r spotMarketRow) fees() string {
	return r.row.MustStr("fees")
}

func (r spotMarketRow) priceMonitoring() string {
	return r.row.MustStr("price monitoring")
}

func (r spotMarketRow) auctionDuration() int64 {
	return r.row.MustI64("auction duration")
}
