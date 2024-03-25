// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package steps

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/integration/steps/market"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

func TheSpotMarketsUpdated(
	config *market.Config,
	executionEngine Execution,
	existing []types.Market,
	netparams *netparams.Store,
	table *godog.Table,
) ([]types.Market, error) {
	rows := parseMarketsUpdateTable(table)
	// existing markets to update
	validByID := make(map[string]*types.Market, len(existing))
	for i := range existing {
		m := existing[i]
		validByID[m.ID] = &existing[i]
	}
	updates := make([]types.UpdateSpotMarket, 0, len(rows))
	updated := make([]*types.Market, 0, len(rows))
	for _, row := range rows {
		upd := spotMarketUpdateRow{row: row}
		// check if market exists
		current, ok := validByID[upd.id()]
		if !ok {
			return nil, fmt.Errorf("unknown market id %s", upd.id())
		}
		updates = append(updates, spotMarketUpdate(config, current, upd))
		updated = append(updated, current)
	}
	if err := updateSpotMarkets(updated, updates, executionEngine); err != nil {
		return nil, err
	}
	// we have been using pointers internally, so we should be returning the accurate state here.
	return existing, nil
}

func updateSpotMarkets(markets []*types.Market, updates []types.UpdateSpotMarket, executionEngine Execution) error {
	for i, mkt := range markets {
		if err := executionEngine.UpdateSpotMarket(context.Background(), mkt); err != nil {
			return fmt.Errorf("couldn't update market(%s) - updates %#v: %+v", mkt.ID, updates[i], err)
		}
	}
	return nil
}

func TheSpotMarkets(config *market.Config, executionEngine Execution, collateralEngine *collateral.Engine, now time.Time, table *godog.Table) ([]types.Market, error) {
	rows := parseSpotMarketsTable(table)
	markets := make([]types.Market, 0, len(rows))

	for _, row := range rows {
		mkt := newSpotMarket(config, spotMarketRow{row: row})
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
				Quantum: num.DecimalOne(),
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

func newSpotMarket(config *market.Config, row spotMarketRow) types.Market {
	fees, err := config.FeesConfig.Get(row.fees())
	if err != nil {
		panic(err)
	}

	priceMonitoring, err := config.PriceMonitoring.Get(row.priceMonitoring())
	if err != nil {
		panic(err)
	}

	slaParams, err := config.LiquiditySLAParams.Get(row.slaParams())
	if err != nil {
		panic(err)
	}

	liqMon, err := config.LiquidityMonitoring.GetType(row.liquidityMonitoring())
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
		LiquiditySLAParams:            types.LiquiditySLAParamsFromProto(slaParams),
		TickSize:                      row.tickSize(),
	}

	tip := m.TradableInstrument.IntoProto()
	err = config.RiskModels.LoadModel(row.riskModel(), tip)
	m.TradableInstrument = types.TradableInstrumentFromProto(tip)
	if err != nil {
		panic(err)
	}

	return m
}

func spotMarketUpdate(config *market.Config, existing *types.Market, row spotMarketUpdateRow) types.UpdateSpotMarket {
	update := types.UpdateSpotMarket{
		MarketID: existing.ID,
		Changes:  &types.UpdateSpotMarketConfiguration{},
	}
	// price monitoring
	if pm, ok := row.priceMonitoring(); ok {
		priceMonitoring, err := config.PriceMonitoring.Get(pm)
		if err != nil {
			panic(err)
		}
		pmt := types.PriceMonitoringSettingsFromProto(priceMonitoring)
		// update existing
		existing.PriceMonitoringSettings.Parameters = pmt.Parameters
		update.Changes.PriceMonitoringParameters = pmt.Parameters
	}
	// liquidity monitoring
	if lm, ok := row.liquidityMonitoring(); ok {
		liqMon, err := config.LiquidityMonitoring.GetType(lm)
		if err != nil {
			panic(err)
		}
		existing.LiquidityMonitoringParameters = liqMon
		update.Changes.TargetStakeParameters = liqMon.TargetStakeParameters
	}

	if sla, ok := row.slaParams(); ok {
		slaParams, err := config.LiquiditySLAParams.Get(sla)
		if err != nil {
			panic(err)
		}
		existing.LiquiditySLAParams = types.LiquiditySLAParamsFromProto(slaParams)
		update.Changes.SLAParams = types.LiquiditySLAParamsFromProto(slaParams)
	}

	// risk model
	if rm, ok := row.riskModel(); ok {
		tip := existing.TradableInstrument.IntoProto()
		if err := config.RiskModels.LoadModel(rm, tip); err != nil {
			panic(err)
		}
		current := types.TradableInstrumentFromProto(tip)
		// find the correct params:
		switch {
		case current.GetSimpleRiskModel() != nil:
			update.Changes.RiskParameters = types.UpdateMarketConfigurationSimple{
				Simple: current.GetSimpleRiskModel().Params,
			}
		case current.GetLogNormalRiskModel() != nil:
			update.Changes.RiskParameters = types.UpdateMarketConfigurationLogNormal{
				LogNormal: current.GetLogNormalRiskModel(),
			}
		default:
			panic("Unsupported risk model parameters")
		}
		// update existing
		existing.TradableInstrument = current
	}
	update.Changes.TickSize = row.tickSize()
	return update
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
		"sla params",
	}, []string{
		"decimal places",
		"position decimal places",
		"tick size",
		"liquidity monitoring",
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

func (r spotMarketRow) tickSize() *num.Uint {
	if r.row.HasColumn("tick size") {
		return num.MustUintFromString(r.row.MustStr("tick size"), 10)
	}
	return num.UintOne()
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

func (r spotMarketRow) slaParams() string {
	return r.row.MustStr("sla params")
}

type spotMarketUpdateRow struct {
	row RowWrapper
}

func (r spotMarketUpdateRow) id() string {
	return r.row.MustStr("id")
}

func (r spotMarketUpdateRow) tickSize() *num.Uint {
	if r.row.HasColumn("tick size") {
		return num.MustUintFromString(r.row.MustStr("tick size"), 10)
	}
	return num.UintOne()
}

func (r spotMarketUpdateRow) priceMonitoring() (string, bool) {
	if r.row.HasColumn("price monitoring") {
		pm := r.row.MustStr("price monitoring")
		return pm, true
	}
	return "", false
}

func (r spotMarketUpdateRow) riskModel() (string, bool) {
	if r.row.HasColumn("risk model") {
		rm := r.row.MustStr("risk model")
		return rm, true
	}
	return "", false
}

func (r spotMarketUpdateRow) liquidityMonitoring() (string, bool) {
	if r.row.HasColumn("liquidity monitoring") {
		lm := r.row.MustStr("liquidity monitoring")
		return lm, true
	}
	return "", false
}

func (r spotMarketUpdateRow) slaParams() (string, bool) {
	if r.row.HasColumn("sla params") {
		lm := r.row.MustStr("sla params")
		return lm, true
	}
	return "", false
}
