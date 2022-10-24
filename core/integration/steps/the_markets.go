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

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/integration/steps/market"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

func TheMarketsUpdated(
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
	updates := make([]types.UpdateMarket, 0, len(rows))
	updated := make([]*types.Market, 0, len(rows))
	for _, row := range rows {
		upd := marketUpdateRow{row: row}
		// check if market exists
		current, ok := validByID[upd.id()]
		if !ok {
			return nil, fmt.Errorf("unknown market id %s", upd.id())
		}
		updates = append(updates, marketUpdate(config, current, upd))
		updated = append(updated, current)
	}
	if err := updateMarkets(updated, updates, executionEngine); err != nil {
		return nil, err
	}
	// we have been using pointers internally, so we should be returning the accurate state here.
	return existing, nil
}

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
		if err := executionEngine.SubmitMarket(context.Background(), &markets[i], "proposerID"); err != nil {
			return fmt.Errorf("couldn't submit market(%s): %v", markets[i].ID, err)
		}
		if err := executionEngine.StartOpeningAuction(context.Background(), markets[i].ID); err != nil {
			return fmt.Errorf("could not start opening auction for market %s: %v", markets[i].ID, err)
		}
	}
	return nil
}

func updateMarkets(markets []*types.Market, updates []types.UpdateMarket, executionEngine Execution) error {
	for i, mkt := range markets {
		if err := executionEngine.UpdateMarket(context.Background(), mkt); err != nil {
			return fmt.Errorf("couldn't update market(%s) - updates %#v: %+v", mkt.ID, updates[i], err)
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
			Name:     "VOTE",
			Symbol:   "VOTE",
			Decimals: 5,
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

// marketUpdate return the UpdateMarket type just for clear error reporting and sanity checks ATM.
func marketUpdate(config *market.Config, existing *types.Market, row marketUpdateRow) types.UpdateMarket {
	update := types.UpdateMarket{
		MarketID: existing.ID,
		Changes:  &types.UpdateMarketConfiguration{},
	}
	// product update
	if oracle, ok := row.oracleConfig(); ok {
		oracleSettlement, err := config.OracleConfigs.Get(oracle, "settlement data")
		if err != nil {
			panic(err)
		}
		oracleTermination, err := config.OracleConfigs.Get(oracle, "trading termination")
		if err != nil {
			panic(err)
		}
		// we probably want to X-check the current spec, and make sure only filters + pubkeys are changed
		settleSpec := types.OracleSpecFromProto(oracleSettlement.Spec)
		termSpec := types.OracleSpecFromProto(oracleTermination.Spec)
		settlementDecimals := config.OracleConfigs.GetSettlementDataDP(oracle)
		// update product -> use type switch even though currently only futures exist
		switch ti := existing.TradableInstrument.Instrument.Product.(type) {
		case *types.InstrumentFuture:
			futureUp := &types.UpdateFutureProduct{
				QuoteName:              ti.Future.QuoteName,
				SettlementDataDecimals: settlementDecimals,
				OracleSpecForSettlementData: &types.OracleSpecConfiguration{
					PubKeys: settleSpec.PubKeys,
					Filters: settleSpec.Filters,
				},
				OracleSpecForTradingTermination: &types.OracleSpecConfiguration{
					PubKeys: termSpec.PubKeys,
					Filters: termSpec.Filters,
				},
				OracleSpecBinding: types.OracleSpecBindingForFutureFromProto(&proto.OracleSpecToFutureBinding{
					SettlementDataProperty:     oracleSettlement.Binding.SettlementDataProperty,
					TradingTerminationProperty: oracleTermination.Binding.TradingTerminationProperty,
				}),
			}
			ti.Future.SettlementDataDecimals = settlementDecimals
			ti.Future.OracleSpecForSettlementData = settleSpec
			ti.Future.OracleSpecBinding = futureUp.OracleSpecBinding
			ti.Future.OracleSpecForTradingTermination = termSpec
			// ensure we update the existing market
			existing.TradableInstrument.Instrument.Product = ti
			update.Changes.Instrument = &types.UpdateInstrumentConfiguration{
				Product: &types.UpdateInstrumentConfigurationFuture{
					Future: futureUp,
				},
			}
		default:
			panic("unsuported product")
		}
		update.Changes.Instrument.Code = existing.TradableInstrument.Instrument.Code
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
		update.Changes.LiquidityMonitoringParameters = liqMon
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
	return update
}

func newMarket(config *market.Config, netparams *netparams.Store, row marketRow) types.Market {
	fees, err := config.FeesConfig.Get(row.fees())
	if err != nil {
		panic(err)
	}

	oracleConfigForSettlement, err := config.OracleConfigs.Get(row.oracleConfig(), "settlement data")
	if err != nil {
		panic(err)
	}

	oracleConfigForTradingTermination, err := config.OracleConfigs.Get(row.oracleConfig(), "trading termination")
	if err != nil {
		panic(err)
	}

	settlementDataDecimals := config.OracleConfigs.GetSettlementDataDP(row.oracleConfig())
	var binding proto.OracleSpecToFutureBinding
	binding.SettlementDataProperty = oracleConfigForSettlement.Binding.SettlementDataProperty
	binding.TradingTerminationProperty = oracleConfigForTradingTermination.Binding.TradingTerminationProperty

	priceMonitoring, err := config.PriceMonitoring.Get(row.priceMonitoring())
	if err != nil {
		panic(err)
	}

	marginCalculator, err := config.MarginCalculators.Get(row.marginCalculator())
	if err != nil {
		panic(err)
	}

	liqMon, err := config.LiquidityMonitoring.GetType(row.liquidityMonitoring())
	if err != nil {
		panic(err)
	}

	// the governance engine would fill in the liquidity monitor parameters from the network parameters (unless set explicitly)
	// so we do this step here manually
	if tw, err := netparams.GetDuration("market.stake.target.timeWindow"); err == nil {
		liqMon.TargetStakeParameters.TimeWindow = int64(tw.Seconds())
	}

	if sf, err := netparams.GetDecimal("market.stake.target.scalingFactor"); err == nil {
		liqMon.TargetStakeParameters.ScalingFactor = sf
	}

	if tr, err := netparams.GetDecimal("market.liquidity.targetstake.triggering.ratio"); err == nil {
		liqMon.TriggeringRatio = tr
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
						OracleSpecForSettlementData:     types.OracleSpecFromProto(oracleConfigForSettlement.Spec),
						OracleSpecForTradingTermination: types.OracleSpecFromProto(oracleConfigForTradingTermination.Spec),
						OracleSpecBinding:               types.OracleSpecBindingForFutureFromProto(&binding),
						SettlementDataDecimals:          settlementDataDecimals,
					},
				},
			},
			MarginCalculator: types.MarginCalculatorFromProto(marginCalculator),
		},
		OpeningAuction:                openingAuction(row),
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
		"liquidity monitoring",
	})
}

func parseMarketsUpdateTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
	}, []string{
		"oracle config",         // product update
		"price monitoring",      // price monitoring update
		"risk model",            // risk model update
		"liquidity monitoring ", // liquidity monitoring update
	})
}

type marketRow struct {
	row RowWrapper
}

type marketUpdateRow struct {
	row RowWrapper
}

func (r marketUpdateRow) id() string {
	return r.row.MustStr("id")
}

func (r marketUpdateRow) oracleConfig() (string, bool) {
	if r.row.HasColumn("oracle config") {
		oc := r.row.MustStr("oracle config")
		return oc, true
	}
	return "", false
}

func (r marketUpdateRow) priceMonitoring() (string, bool) {
	if r.row.HasColumn("price monitoring") {
		pm := r.row.MustStr("price monitoring")
		return pm, true
	}
	return "", false
}

func (r marketUpdateRow) riskModel() (string, bool) {
	if r.row.HasColumn("risk model") {
		rm := r.row.MustStr("risk model")
		return rm, true
	}
	return "", false
}

func (r marketUpdateRow) liquidityMonitoring() (string, bool) {
	if r.row.HasColumn("liquidity monitoring") {
		lm := r.row.MustStr("liquidity monitoring")
		return lm, true
	}
	return "", false
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

func (r marketRow) positionDecimalPlaces() int64 {
	if !r.row.HasColumn("position decimal places") {
		return 0
	}
	return r.row.MustI64("position decimal places")
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

func (r marketRow) liquidityMonitoring() string {
	if !r.row.HasColumn("liquidity monitoring") {
		return "default-parameters"
	}
	return r.row.MustStr("liquidity monitoring")
}
