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
	"github.com/pkg/errors"

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

func TheSuccessorMarkets(
	config *market.SuccessorConfig,
	exec Execution,
	netparams *netparams.Store,
	table *godog.Table,
) ([]types.Market, error) {
	rows := parseSuccessorMarketTable(table)
	markets := make([]types.Market, 0, len(rows))

	for _, row := range rows {
		mkt, err := newSuccessorMarket(config, exec, netparams, successorRow{row: row})
		if err != nil {
			return nil, err
		}
		markets = append(markets, mkt)
	}

	// submit the successor markets and start opening auction as we tend to do
	if err := submitMarkets(markets, exec); err != nil {
		return nil, err
	}
	return markets, nil
}

func TheSuccesorMarketIsEnacted(sID string, markets []types.Market, exec Execution) error {
	for _, mkt := range markets {
		if mkt.ID == sID {
			parent := mkt.ParentMarketID
			if err := exec.SucceedMarket(context.Background(), sID, parent, mkt.InsurancePoolFraction); err != nil {
				return fmt.Errorf("couldn't enact the successor market %s (parent: %s): %v", sID, parent, err)
			}
			return nil
		}
	}
	return fmt.Errorf("couldn't enact successor market %s - no such market ID", sID)
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
		assets, _ := mkt.GetAssets()
		assetsToEnable[assets[0]] = struct{}{}
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
			filters := settleSpec.ExternalDataSourceSpec.Spec.Data.GetFilters()
			futureUp := &types.UpdateFutureProduct{
				QuoteName: ti.Future.QuoteName,
				DataSourceSpecForSettlementData: *types.NewDataSourceDefinition(
					proto.DataSourceDefinitionTypeExt,
				).SetOracleConfig(
					&types.DataSourceSpecConfiguration{
						Signers: settleSpec.ExternalDataSourceSpec.Spec.Data.GetSigners(),
						Filters: filters,
					},
				),
				DataSourceSpecForTradingTermination: *types.NewDataSourceDefinition(
					proto.DataSourceDefinitionTypeExt,
				).SetOracleConfig(
					&types.DataSourceSpecConfiguration{
						Signers: settleSpec.ExternalDataSourceSpec.Spec.Data.GetSigners(),
						Filters: filters,
					},
				),
				DataSourceSpecBinding: types.DataSourceSpecBindingForFutureFromProto(&proto.DataSourceSpecToFutureBinding{
					SettlementDataProperty:     oracleSettlement.Binding.SettlementDataProperty,
					TradingTerminationProperty: oracleTermination.Binding.TradingTerminationProperty,
				}),
			}
			ti.Future.DataSourceSpecForSettlementData = settleSpec.ExternalDataSourceSpec.Spec.Data.SetFilterDecimals(uint64(settlementDecimals)).ToDataSourceSpec()
			ti.Future.DataSourceSpecForTradingTermination = termSpec.ExternalDataSourceSpec.Spec
			ti.Future.DataSourceSpecBinding = futureUp.DataSourceSpecBinding
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
	// lp price range
	if lppr, ok := row.tryLpPriceRange(); ok {
		lpprD := num.DecimalFromFloat(lppr)
		update.Changes.LpPriceRange = lpprD
		existing.LPPriceRange = lpprD
	}

	// linear slippage factor
	if slippage, ok := row.tryLinearSlippageFactor(); ok {
		slippageD := num.DecimalFromFloat(slippage)
		update.Changes.LinearSlippageFactor = slippageD
		existing.LinearSlippageFactor = slippageD
	}

	// quadratic slippage factor
	if slippage, ok := row.tryQuadraticSlippageFactor(); ok {
		slippageD := num.DecimalFromFloat(slippage)
		update.Changes.QuadraticSlippageFactor = slippageD
		existing.QuadraticSlippageFactor = slippageD
	}
	return update
}

func newSuccessorMarket(config *market.SuccessorConfig, exec Execution, netparams *netparams.Store, row successorRow) (types.Market, error) {
	parent, ok := exec.GetMarket(config.ParentID, true)
	if !ok {
		return types.Market{}, errors.Errorf("parent market %s does not exist", config.ParentID)
	}
	cfg := parent.DeepClone()
	cfg.ParentMarketID = config.ParentID
	cfg.InsurancePoolFraction = config.InsuranceFraction
	// @TODO create a marketRow type for successors
	cfg.ID = row.id()
	if ls, ok := row.tryLinearSlippageFactor(); ok {
		cfg.LinearSlippageFactor = num.DecimalFromFloat(ls)
	}
	if qs, ok := row.tryQuadraticSlippageFactor(); ok {
		cfg.QuadraticSlippageFactor = num.DecimalFromFloat(qs)
	}
	if pr, ok := row.tryLpPriceRange(); ok {
		cfg.LPPriceRange = num.DecimalFromFloat(pr)
	}
	if pd, ok := row.positionDecimals(); ok {
		cfg.PositionDecimalPlaces = pd
	}
	if dp, ok := row.decimals(); ok {
		cfg.DecimalPlaces = dp
	}
	if pm, ok := row.priceMonitoring(); ok {
		priceMon, err := config.PriceMonitoring.Get(pm)
		if err != nil {
			return types.Market{}, err
		}
		cfg.PriceMonitoringSettings = types.PriceMonitoringSettingsFromProto(priceMon)
	}
	if lm, ok := row.liquidityMonitoring(); ok {
		liqM, err := config.LiquidityMonitoring.GetType(lm)
		if err != nil {
			return types.Market{}, err
		}
		setLiquidityMonitoringNetParams(liqM, netparams)
		// these ought to get applied as they are specified, perhaps this is where netparams still need to be applied, though
		cfg.LiquidityMonitoringParameters = liqM
	}
	// ensure market is active
	cfg.State = types.MarketStateActive
	return *cfg, nil
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
	settlSpec := types.OracleSpecFromProto(oracleConfigForSettlement.Spec)
	var binding proto.DataSourceSpecToFutureBinding
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

	lpPriceRange := row.lpPriceRange()
	linearSlippageFactor := row.linearSlippageFactor()
	quadraticSlippageFactor := row.quadraticSlippageFactor()

	setLiquidityMonitoringNetParams(liqMon, netparams)

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
						SettlementAsset:                     row.asset(),
						QuoteName:                           row.quoteName(),
						DataSourceSpecForSettlementData:     settlSpec.ExternalDataSourceSpec.Spec.Data.SetFilterDecimals(uint64(settlementDataDecimals)).ToDataSourceSpec(),
						DataSourceSpecForTradingTermination: types.DataSourceSpecFromProto(oracleConfigForTradingTermination.Spec.ExternalDataSourceSpec.Spec),
						DataSourceSpecBinding:               types.DataSourceSpecBindingForFutureFromProto(&binding),
					},
				},
			},
			MarginCalculator: types.MarginCalculatorFromProto(marginCalculator),
		},
		OpeningAuction:                openingAuction(row),
		PriceMonitoringSettings:       types.PriceMonitoringSettingsFromProto(priceMonitoring),
		LiquidityMonitoringParameters: liqMon,
		LPPriceRange:                  num.DecimalFromFloat(lpPriceRange),
		LinearSlippageFactor:          num.DecimalFromFloat(linearSlippageFactor),
		QuadraticSlippageFactor:       num.DecimalFromFloat(quadraticSlippageFactor),
	}

	tip := m.TradableInstrument.IntoProto()
	err = config.RiskModels.LoadModel(row.riskModel(), tip)
	m.TradableInstrument = types.TradableInstrumentFromProto(tip)
	if err != nil {
		panic(err)
	}

	return m
}

func setLiquidityMonitoringNetParams(liqMon *types.LiquidityMonitoringParameters, netparams *netparams.Store) {
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
		"data source config",
		"price monitoring",
		"margin calculator",
		"auction duration",
		"linear slippage factor",
		"quadratic slippage factor",
	}, []string{
		"decimal places",
		"position decimal places",
		"liquidity monitoring",
		"lp price range",
	})
}

func parseMarketsUpdateTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"linear slippage factor", // slippage factors must be explicitly set to avoid setting them to hard-coded defaults
		"quadratic slippage factor",
	}, []string{
		"data source config",   // product update
		"price monitoring",     // price monitoring update
		"risk model",           // risk model update
		"liquidity monitoring", // liquidity monitoring update
		"lp price range",
	})
}

func parseSuccessorMarketTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"parent id",
		"linear slippage factor",
		"quadratic slippage factor",
		"insurance pool fraction",
		"risk model",
	}, []string{
		"lp price range", // we will default to parent values
		"decimal places",
		"position decimal places",
		"liquidity monitoring",
		"price monitoring",
	})
}

type marketRow struct {
	row RowWrapper
}

type marketUpdateRow struct {
	row RowWrapper
}

type successorRow struct {
	row RowWrapper
}

func (r marketUpdateRow) id() string {
	return r.row.MustStr("id")
}

func (r marketUpdateRow) oracleConfig() (string, bool) {
	if r.row.HasColumn("data source config") {
		oc := r.row.MustStr("data source config")
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
	return r.row.MustStr("data source config")
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

func (r marketRow) lpPriceRange() float64 {
	if !r.row.HasColumn("lp price range") {
		// set to 1 by default
		return 1
	}
	return r.row.MustF64("lp price range")
}

func (r marketRow) linearSlippageFactor() float64 {
	if !r.row.HasColumn("linear slippage factor") {
		// set to 0.1 by default
		return 0.001
	}
	return r.row.MustF64("linear slippage factor")
}

func (r marketRow) quadraticSlippageFactor() float64 {
	if !r.row.HasColumn("quadratic slippage factor") {
		// set to 0.1 by default
		return 0.0
	}
	return r.row.MustF64("quadratic slippage factor")
}

func (r marketUpdateRow) tryLpPriceRange() (float64, bool) {
	if r.row.HasColumn("lp price range") {
		return r.row.MustF64("lp price range"), true
	}
	return -1, false
}

func (r marketUpdateRow) tryLinearSlippageFactor() (float64, bool) {
	if r.row.HasColumn("linear slippage factor") {
		return r.row.MustF64("linear slippage factor"), true
	}
	return -1, false
}

func (r marketUpdateRow) tryQuadraticSlippageFactor() (float64, bool) {
	if r.row.HasColumn("quadratic slippage factor") {
		return r.row.MustF64("quadratic slippage factor"), true
	}
	return -1, false
}

func (s successorRow) id() string {
	return s.row.MustStr("id")
}

func (s successorRow) tryLinearSlippageFactor() (float64, bool) {
	if s.row.HasColumn("linear slippage factor") {
		return s.row.MustF64("linear slippage factor"), true
	}
	return 0, false
}

func (s successorRow) tryQuadraticSlippageFactor() (float64, bool) {
	if s.row.HasColumn("quadratic slippage factor") {
		return s.row.MustF64("quadratic slippage factor"), true
	}
	return 0, false
}

func (s successorRow) tryLpPriceRange() (float64, bool) {
	if s.row.HasColumn("lp price range") {
		return s.row.MustF64("lp price range"), true
	}
	return 0, false
}

func (s successorRow) positionDecimals() (int64, bool) {
	if s.row.HasColumn("position decimal places") {
		return s.row.MustI64("position decimal places"), true
	}
	return 0, false
}

func (s successorRow) decimals() (uint64, bool) {
	if s.row.HasColumn("decimal places") {
		return s.row.MustU64("decimal places"), true
	}
	return 0, false
}

func (s successorRow) priceMonitoring() (string, bool) {
	if s.row.HasColumn("price monitoring") {
		return s.row.MustStr("price monitoring"), true
	}
	return "", false
}

func (s successorRow) liquidityMonitoring() (string, bool) {
	if s.row.HasColumn("liquidity monitoring") {
		return s.row.MustStr("liquidity monitoring"), true
	}
	return "", false
}
