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
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/external/signedoracle"
	"code.vegaprotocol.io/vega/core/integration/steps/market"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
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

		mUpdate, err := marketUpdate(config, current, upd)
		if err != nil {
			return existing, err
		}

		updates = append(updates, mUpdate)
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
	now time.Time,
	table *godog.Table,
) ([]types.Market, error) {
	rows := parseMarketsTable(table)
	markets := make([]types.Market, 0, len(rows))

	for _, row := range rows {
		mRow := marketRow{row: row}
		isPerp := mRow.isPerp()
		if !isPerp {
			// check if we have a perp counterpart for this oracle, if so, swap to that
			if oName := mRow.oracleConfig(); oName != config.OracleConfigs.CheckName(oName) {
				isPerp = true
			}
		}
		var mkt types.Market
		if isPerp {
			mkt = newPerpMarket(config, mRow)
		} else {
			mkt = newMarket(config, mRow)
		}
		markets = append(markets, mkt)
	}

	if err := enableMarketAssets(markets, collateralEngine); err != nil {
		return nil, err
	}

	if err := enableVoteAsset(collateralEngine); err != nil {
		return nil, err
	}

	for i, row := range rows {
		if err := executionEngine.SubmitMarket(context.Background(), &markets[i], "proposerID", now); err != nil {
			return nil, fmt.Errorf("couldn't submit market(%s): %v", markets[i].ID, err)
		}
		// only start opening auction if the market is explicitly marked to leave opening auction now
		if !row.HasColumn("is passed") || row.Bool("is passed") {
			if err := executionEngine.StartOpeningAuction(context.Background(), markets[i].ID); err != nil {
				return nil, fmt.Errorf("could not start opening auction for market %s: %v", markets[i].ID, err)
			}
		}
	}
	return markets, nil
}

func TheSuccesorMarketIsEnacted(sID string, markets []types.Market, exec Execution) error {
	for _, mkt := range markets {
		if mkt.ID == sID {
			parent := mkt.ParentMarketID
			if err := exec.SucceedMarket(context.Background(), sID, parent); err != nil {
				return fmt.Errorf("couldn't enact the successor market %s (parent: %s): %v", sID, parent, err)
			}
			return nil
		}
	}
	return fmt.Errorf("couldn't enact successor market %s - no such market ID", sID)
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
func marketUpdate(config *market.Config, existing *types.Market, row marketUpdateRow) (types.UpdateMarket, error) {
	update := types.UpdateMarket{
		MarketID: existing.ID,
		Changes:  &types.UpdateMarketConfiguration{},
	}
	liqStrat := existing.LiquidationStrategy
	if ls, ok := row.liquidationStrat(); ok {
		lqs, err := config.LiquidationStrat.Get(ls)
		if err != nil {
			panic(err)
		}
		if liqStrat, err = types.LiquidationStrategyFromProto(lqs); err != nil {
			panic(err)
		}
	}
	update.Changes.LiquidationStrategy = liqStrat
	existing.LiquidationStrategy = liqStrat

	// product update
	if oracle, ok := row.oracleConfig(); ok {
		// update product -> use type switch even though currently only futures exist
		switch ti := existing.TradableInstrument.Instrument.Product.(type) {
		case *types.InstrumentFuture:
			oracleSettlement, err := config.OracleConfigs.GetFuture(oracle, "settlement data")
			if err != nil {
				panic(err)
			}
			oracleTermination, err := config.OracleConfigs.GetFuture(oracle, "trading termination")
			if err != nil {
				panic(err)
			}
			// we probably want to X-check the current spec, and make sure only filters + pubkeys are changed
			settleSpec := datasource.FromOracleSpecProto(oracleSettlement.Spec)
			termSpec := datasource.FromOracleSpecProto(oracleTermination.Spec)
			settlementDecimals := config.OracleConfigs.GetSettlementDataDP(oracle)
			filters := settleSpec.Data.GetFilters()
			futureUp := &types.UpdateFutureProduct{
				QuoteName: ti.Future.QuoteName,
				DataSourceSpecForSettlementData: *datasource.NewDefinition(
					datasource.ContentTypeOracle,
				).SetOracleConfig(
					&signedoracle.SpecConfiguration{
						Signers: settleSpec.Data.GetSigners(),
						Filters: filters,
					},
				),
				DataSourceSpecForTradingTermination: *datasource.NewDefinition(
					datasource.ContentTypeOracle,
				).SetOracleConfig(
					&signedoracle.SpecConfiguration{
						Signers: settleSpec.Data.GetSigners(),
						Filters: filters,
					},
				),
				DataSourceSpecBinding: datasource.SpecBindingForFutureFromProto(&proto.DataSourceSpecToFutureBinding{
					SettlementDataProperty:     oracleSettlement.Binding.SettlementDataProperty,
					TradingTerminationProperty: oracleTermination.Binding.TradingTerminationProperty,
				}),
			}
			ti.Future.DataSourceSpecForSettlementData = datasource.SpecFromDefinition(*settleSpec.Data.SetFilterDecimals(uint64(settlementDecimals)))
			ti.Future.DataSourceSpecForTradingTermination = termSpec
			ti.Future.DataSourceSpecBinding = futureUp.DataSourceSpecBinding
			// ensure we update the existing market
			existing.TradableInstrument.Instrument.Product = ti
			update.Changes.Instrument = &types.UpdateInstrumentConfiguration{
				Product: &types.UpdateInstrumentConfigurationFuture{
					Future: futureUp,
				},
			}
		case *types.InstrumentPerps:
			perp, err := config.OracleConfigs.GetFullPerp(oracle)
			if err != nil {
				panic(err)
			}
			pfp := types.PerpsFromProto(perp)
			if pfp.DataSourceSpecForSettlementData == nil || pfp.DataSourceSpecForSettlementData.Data == nil {
				panic("Oracle does not have a data source for settlement data")
			}
			if pfp.DataSourceSpecForSettlementSchedule == nil || pfp.DataSourceSpecForSettlementSchedule.Data == nil {
				panic("Oracle does not have a data source for settlement schedule")
			}
			update.Changes.Instrument = &types.UpdateInstrumentConfiguration{
				Product: &types.UpdateInstrumentConfigurationPerps{
					Perps: &types.UpdatePerpsProduct{
						QuoteName:                           pfp.QuoteName,
						MarginFundingFactor:                 pfp.MarginFundingFactor,
						ClampLowerBound:                     pfp.ClampLowerBound,
						ClampUpperBound:                     pfp.ClampUpperBound,
						FundingRateScalingFactor:            pfp.FundingRateScalingFactor,
						FundingRateLowerBound:               pfp.FundingRateLowerBound,
						FundingRateUpperBound:               pfp.FundingRateUpperBound,
						DataSourceSpecForSettlementData:     *pfp.DataSourceSpecForSettlementData.Data,
						DataSourceSpecForSettlementSchedule: *pfp.DataSourceSpecForSettlementSchedule.Data,
						DataSourceSpecBinding:               pfp.DataSourceSpecBinding,
					},
				},
			}
			// apply update
			ti.Perps.ClampLowerBound = pfp.ClampLowerBound
			ti.Perps.ClampUpperBound = pfp.ClampUpperBound
			ti.Perps.FundingRateScalingFactor = pfp.FundingRateScalingFactor
			ti.Perps.FundingRateUpperBound = pfp.FundingRateUpperBound
			ti.Perps.FundingRateLowerBound = pfp.FundingRateLowerBound
			ti.Perps.MarginFundingFactor = pfp.MarginFundingFactor
			ti.Perps.DataSourceSpecBinding = pfp.DataSourceSpecBinding
			ti.Perps.DataSourceSpecForSettlementData = pfp.DataSourceSpecForSettlementData
			ti.Perps.DataSourceSpecForSettlementSchedule = pfp.DataSourceSpecForSettlementSchedule
			existing.TradableInstrument.Instrument.Product = ti
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

	if liquiditySla, ok := row.tryLiquiditySLA(); ok {
		sla, err := config.LiquiditySLAParams.Get(liquiditySla)
		if err != nil {
			return update, err
		}
		slaParams := types.LiquiditySLAParamsFromProto(sla)
		// update existing
		existing.LiquiditySLAParams = slaParams
		update.Changes.LiquiditySLAParameters = slaParams
	}

	update.Changes.LiquidityFeeSettings = existing.Fees.LiquidityFeeSettings
	if liquidityFeeSettings, ok := row.tryLiquidityFeeSettings(); ok {
		settings, err := config.FeesConfig.Get(liquidityFeeSettings)
		if err != nil {
			return update, err
		}
		s := types.LiquidityFeeSettingsFromProto(settings.LiquidityFeeSettings)
		existing.Fees.LiquidityFeeSettings = s
		update.Changes.LiquidityFeeSettings = s
	}

	if existing.MarkPriceConfiguration != nil {
		markPriceConfig := existing.MarkPriceConfiguration.DeepClone()
		markPriceConfig.CompositePriceType = row.markPriceType()

		if row.row.HasColumn("decay power") {
			markPriceConfig.DecayPower = row.decayPower()
		}
		if row.row.HasColumn("decay weight") {
			markPriceConfig.DecayWeight = row.decayWeight()
		}
		if row.row.HasColumn("cash amount") {
			markPriceConfig.CashAmount = row.cashAmount()
		}
		if row.row.HasColumn("source weights") {
			markPriceConfig.SourceWeights = row.priceSourceWeights()
		}
		if row.row.HasColumn("source staleness tolerance") {
			markPriceConfig.SourceStalenessTolerance = row.priceSourceStalnessTolerance()
		}
		if row.row.HasColumn("oracle1") {
			markPriceConfig.DataSources, markPriceConfig.SpecBindingForCompositePrice = row.oracles(config)
		}
		update.Changes.MarkPriceConfiguration = markPriceConfig
		existing.MarkPriceConfiguration = markPriceConfig
	}
	return update, nil
}

func newPerpMarket(config *market.Config, row marketRow) types.Market {
	fees, err := config.FeesConfig.Get(row.fees())
	if err != nil {
		panic(err)
	}

	perp, err := config.OracleConfigs.GetFullPerp(row.oracleConfig())
	if err != nil {
		panic(err)
	}
	pfp := types.PerpsFromProto(perp)
	asset, quote := row.asset(), row.quoteName()
	// long term, this should become redundant, but for the perps flag this is useful to have
	if asset != pfp.SettlementAsset {
		pfp.SettlementAsset = asset
	}
	if quote != pfp.QuoteName {
		pfp.QuoteName = row.quoteName()
	}

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
	lqs, err := config.LiquidationStrat.Get(row.liquidationStrat())
	if err != nil {
		panic(err)
	}
	liqStrat, err := types.LiquidationStrategyFromProto(lqs)
	if err != nil {
		panic(err)
	}

	linearSlippageFactor := row.linearSlippageFactor()
	quadraticSlippageFactor := row.quadraticSlippageFactor()

	slaParams, err := config.LiquiditySLAParams.Get(row.liquiditySLA())
	if err != nil {
		panic(err)
	}

	specs, binding := row.oracles(config)
	markPriceConfig := &types.CompositePriceConfiguration{
		CompositePriceType:           row.markPriceType(),
		DecayWeight:                  row.decayWeight(),
		DecayPower:                   row.decayPower(),
		CashAmount:                   row.cashAmount(),
		SourceWeights:                row.priceSourceWeights(),
		SourceStalenessTolerance:     row.priceSourceStalnessTolerance(),
		DataSources:                  specs,
		SpecBindingForCompositePrice: binding,
	}

	m := types.Market{
		TradingMode:           types.MarketTradingModeContinuous,
		State:                 types.MarketStateActive,
		ID:                    row.id(),
		DecimalPlaces:         row.decimalPlaces(),
		PositionDecimalPlaces: row.positionDecimalPlaces(),
		Fees:                  types.FeesFromProto(fees),
		LiquidationStrategy:   liqStrat,
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				ID:   fmt.Sprintf("Crypto/%s/Perpetual", row.id()),
				Code: fmt.Sprintf("CRYPTO/%v", row.id()),
				Name: fmt.Sprintf("%s perpetual", row.id()),
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:perpetual",
					},
				},
				Product: &types.InstrumentPerps{
					Perps: pfp,
				},
			},
			MarginCalculator: types.MarginCalculatorFromProto(marginCalculator),
		},
		OpeningAuction:                openingAuction(row),
		PriceMonitoringSettings:       types.PriceMonitoringSettingsFromProto(priceMonitoring),
		LiquidityMonitoringParameters: liqMon,
		LinearSlippageFactor:          num.DecimalFromFloat(linearSlippageFactor),
		QuadraticSlippageFactor:       num.DecimalFromFloat(quadraticSlippageFactor),
		LiquiditySLAParams:            types.LiquiditySLAParamsFromProto(slaParams),
		MarkPriceConfiguration:        markPriceConfig,
	}

	if row.isSuccessor() {
		m.ParentMarketID = row.parentID()
		m.InsurancePoolFraction = row.insuranceFraction()
		// increase opening auction duration by a given amount
		m.OpeningAuction.Duration += row.successorAuction()
	}

	tip := m.TradableInstrument.IntoProto()
	err = config.RiskModels.LoadModel(row.riskModel(), tip)
	m.TradableInstrument = types.TradableInstrumentFromProto(tip)
	if err != nil {
		panic(err)
	}

	return m
}

func newMarket(config *market.Config, row marketRow) types.Market {
	fees, err := config.FeesConfig.Get(row.fees())
	if err != nil {
		panic(err)
	}

	oracleConfigForSettlement, err := config.OracleConfigs.GetFuture(row.oracleConfig(), "settlement data")
	if err != nil {
		panic(err)
	}

	oracleConfigForTradingTermination, err := config.OracleConfigs.GetFuture(row.oracleConfig(), "trading termination")
	if err != nil {
		panic(err)
	}

	settlementDataDecimals := config.OracleConfigs.GetSettlementDataDP(row.oracleConfig())
	settlSpec := datasource.FromOracleSpecProto(oracleConfigForSettlement.Spec)
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

	lqs, err := config.LiquidationStrat.Get(row.liquidationStrat())
	if err != nil {
		panic(err)
	}
	liqStrat, err := types.LiquidationStrategyFromProto(lqs)
	if err != nil {
		panic(err)
	}

	linearSlippageFactor := row.linearSlippageFactor()
	quadraticSlippageFactor := row.quadraticSlippageFactor()

	slaParams, err := config.LiquiditySLAParams.Get(row.liquiditySLA())
	if err != nil {
		panic(err)
	}

	sources, bindings := row.oracles(config)
	markPriceConfig := &types.CompositePriceConfiguration{
		CompositePriceType:           row.markPriceType(),
		DecayWeight:                  row.decayWeight(),
		DecayPower:                   row.decayPower(),
		CashAmount:                   row.cashAmount(),
		SourceWeights:                row.priceSourceWeights(),
		SourceStalenessTolerance:     row.priceSourceStalnessTolerance(),
		DataSources:                  sources,
		SpecBindingForCompositePrice: bindings,
	}

	m := types.Market{
		TradingMode:           types.MarketTradingModeContinuous,
		State:                 types.MarketStateActive,
		ID:                    row.id(),
		DecimalPlaces:         row.decimalPlaces(),
		PositionDecimalPlaces: row.positionDecimalPlaces(),
		Fees:                  types.FeesFromProto(fees),
		LiquidationStrategy:   liqStrat,
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
						DataSourceSpecForSettlementData:     datasource.SpecFromDefinition(*settlSpec.Data.SetFilterDecimals(uint64(settlementDataDecimals))),
						DataSourceSpecForTradingTermination: datasource.SpecFromProto(oracleConfigForTradingTermination.Spec.ExternalDataSourceSpec.Spec),
						DataSourceSpecBinding:               datasource.SpecBindingForFutureFromProto(&binding),
					},
				},
			},
			MarginCalculator: types.MarginCalculatorFromProto(marginCalculator),
		},
		OpeningAuction:                openingAuction(row),
		PriceMonitoringSettings:       types.PriceMonitoringSettingsFromProto(priceMonitoring),
		LiquidityMonitoringParameters: liqMon,
		LinearSlippageFactor:          num.DecimalFromFloat(linearSlippageFactor),
		QuadraticSlippageFactor:       num.DecimalFromFloat(quadraticSlippageFactor),
		LiquiditySLAParams:            types.LiquiditySLAParamsFromProto(slaParams),
		MarkPriceConfiguration:        markPriceConfig,
	}

	if row.isSuccessor() {
		m.ParentMarketID = row.parentID()
		m.InsurancePoolFraction = row.insuranceFraction()
		// increase opening auction duration by a given amount
		m.OpeningAuction.Duration += row.successorAuction()
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
		"data source config",
		"price monitoring",
		"margin calculator",
		"auction duration",
		"linear slippage factor",
		"quadratic slippage factor",
		"sla params",
	}, []string{
		"decimal places",
		"position decimal places",
		"liquidity monitoring",
		"parent market id",
		"insurance pool fraction",
		"successor auction",
		"is passed",
		"market type",
		"liquidation strategy",
		"price type",
		"decay weight",
		"decay power",
		"cash amount",
		"source weights",
		"source staleness tolerance",
		"oracle1",
		"oracle2",
		"oracle3",
		"oracle4",
		"oracle5",
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
		"sla params",
		"liquidity fee settings",
		"liquidation strategy",
		"price type",
		"decay weight",
		"decay power",
		"cash amount",
		"source weights",
		"source staleness tolerance",
		"oracle1",
		"oracle2",
		"oracle3",
		"oracle4",
		"oracle5",
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

func (r marketUpdateRow) liquidationStrat() (string, bool) {
	if r.row.HasColumn("liquidation strategy") {
		ls := r.row.MustStr("liquidation strategy")
		return ls, true
	}
	return "", false
}

func (r marketUpdateRow) priceSourceWeights() []num.Decimal {
	if !r.row.HasColumn("source weights") {
		return []num.Decimal{num.DecimalZero(), num.DecimalZero(), num.DecimalZero(), num.DecimalZero()}
	}
	weights := strings.Split(r.row.mustColumn("source weights"), ",")
	d := make([]num.Decimal, 0, len(weights))
	for _, v := range weights {
		d = append(d, num.MustDecimalFromString(v))
	}
	return d
}

func (r marketUpdateRow) compositePriceOracleFromName(config *market.Config, name string) (*datasource.Spec, *datasource.SpecBindingForCompositePrice) {
	if !r.row.HasColumn(name) {
		return nil, nil
	}

	rawSpec, binding, err := config.OracleConfigs.GetOracleDefinitionForCompositePrice(r.row.Str(name))
	if err != nil {
		return nil, nil
	}
	spec := datasource.FromOracleSpecProto(rawSpec)
	filters := spec.Data.GetFilters()
	ds := datasource.NewDefinition(datasource.ContentTypeOracle).SetOracleConfig(
		&signedoracle.SpecConfiguration{
			Signers: spec.Data.GetSigners(),
			Filters: filters,
		},
	)
	return datasource.SpecFromDefinition(*ds), &datasource.SpecBindingForCompositePrice{PriceSourceProperty: binding.PriceSourceProperty}
}

func (r marketUpdateRow) oracles(config *market.Config) ([]*datasource.Spec, []*datasource.SpecBindingForCompositePrice) {
	specs := []*datasource.Spec{}
	bindings := []*datasource.SpecBindingForCompositePrice{}
	names := []string{"oracle1", "oracle2", "oracle3", "oracle4", "oracle5"}
	for _, v := range names {
		spec, binding := r.compositePriceOracleFromName(config, v)
		if spec == nil {
			continue
		}
		specs = append(specs, spec)
		bindings = append(bindings, binding)
	}
	if len(specs) > 0 {
		return specs, bindings
	}

	return nil, nil
}

func (r marketUpdateRow) priceSourceStalnessTolerance() []time.Duration {
	if !r.row.HasColumn("source staleness tolerance") {
		return []time.Duration{1000, 1000, 1000, 1000}
	}
	durations := strings.Split(r.row.mustColumn("source staleness tolerance"), ",")
	d := make([]time.Duration, 0, len(durations))
	for _, v := range durations {
		dur, err := time.ParseDuration(v)
		if err != nil {
			panic(err)
		}
		d = append(d, dur)
	}
	return d
}

func (r marketUpdateRow) cashAmount() *num.Uint {
	if !r.row.HasColumn("cash amount") {
		return num.UintZero()
	}
	return num.MustUintFromString(r.row.mustColumn("cash amount"), 10)
}

func (r marketUpdateRow) decayPower() num.Decimal {
	if !r.row.HasColumn("decay power") {
		return num.DecimalZero()
	}
	return num.MustDecimalFromString(r.row.mustColumn("decay power"))
}

func (r marketUpdateRow) decayWeight() num.Decimal {
	if !r.row.HasColumn("decay weight") {
		return num.DecimalZero()
	}
	return num.MustDecimalFromString(r.row.mustColumn("decay weight"))
}

func (r marketUpdateRow) markPriceType() types.CompositePriceType {
	if !r.row.HasColumn("price type") {
		return types.CompositePriceTypeByLastTrade
	}
	if r.row.mustColumn("price type") == "last trade" {
		return types.CompositePriceTypeByLastTrade
	} else if r.row.mustColumn("price type") == "median" {
		return types.CompositePriceTypeByMedian
	} else if r.row.mustColumn("price type") == "weight" {
		return types.CompositePriceTypeByWeight
	} else {
		panic("invalid price type")
	}
}

func (r marketRow) id() string {
	return r.row.MustStr("id")
}

func (r marketRow) liquidationStrat() string {
	if r.row.HasColumn("liquidation strategy") {
		ls := r.row.MustStr("liquidation strategy")
		return ls
	}
	return ""
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

func (r marketRow) liquiditySLA() string {
	return r.row.MustStr("sla params")
}

func (r marketUpdateRow) tryLiquiditySLA() (string, bool) {
	if r.row.HasColumn("sla params") {
		sla := r.row.MustStr("sla params")
		return sla, true
	}
	return "", false
}

func (r marketUpdateRow) tryLiquidityFeeSettings() (string, bool) {
	if r.row.HasColumn("liquidity fee settings") {
		s := r.row.MustStr("liquidity fee settings")
		return s, true
	}
	return "", false
}

func (r marketRow) linearSlippageFactor() float64 {
	if !r.row.HasColumn("linear slippage factor") {
		// set to 0.1 by default
		return 0.001
	}
	return r.row.MustF64("linear slippage factor")
}

func (r marketRow) priceSourceWeights() []num.Decimal {
	if !r.row.HasColumn("source weights") {
		return []num.Decimal{num.DecimalZero(), num.DecimalZero(), num.DecimalZero(), num.DecimalZero()}
	}
	weights := strings.Split(r.row.mustColumn("source weights"), ",")
	d := make([]num.Decimal, 0, len(weights))
	for _, v := range weights {
		d = append(d, num.MustDecimalFromString(v))
	}
	return d
}

func (r marketRow) priceSourceStalnessTolerance() []time.Duration {
	if !r.row.HasColumn("source staleness tolerance") {
		return []time.Duration{1000, 1000, 1000, 1000}
	}
	durations := strings.Split(r.row.mustColumn("source staleness tolerance"), ",")
	d := make([]time.Duration, 0, len(durations))
	for _, v := range durations {
		dur, err := time.ParseDuration(v)
		if err != nil {
			panic(err)
		}
		d = append(d, dur)
	}
	return d
}

func (r marketRow) cashAmount() *num.Uint {
	if !r.row.HasColumn("cash amount") {
		return num.UintZero()
	}
	return num.MustUintFromString(r.row.mustColumn("cash amount"), 10)
}

func (r marketRow) decayPower() num.Decimal {
	if !r.row.HasColumn("decay power") {
		return num.DecimalZero()
	}
	return num.MustDecimalFromString(r.row.mustColumn("decay power"))
}

func (r marketRow) decayWeight() num.Decimal {
	if !r.row.HasColumn("decay weight") {
		return num.DecimalZero()
	}
	return num.MustDecimalFromString(r.row.mustColumn("decay weight"))
}

func (r marketRow) markPriceType() types.CompositePriceType {
	if !r.row.HasColumn("price type") {
		return types.CompositePriceTypeByLastTrade
	}
	if r.row.mustColumn("price type") == "last trade" {
		return types.CompositePriceTypeByLastTrade
	} else if r.row.mustColumn("price type") == "median" {
		return types.CompositePriceTypeByMedian
	} else if r.row.mustColumn("price type") == "weight" {
		return types.CompositePriceTypeByWeight
	} else {
		panic("invalid price type")
	}
}

func (r marketRow) compositePriceOracleFromName(config *market.Config, name string) (*datasource.Spec, *datasource.SpecBindingForCompositePrice) {
	if !r.row.HasColumn(name) {
		return nil, nil
	}
	rawSpec, binding, err := config.OracleConfigs.GetOracleDefinitionForCompositePrice(r.row.Str(name))
	if err != nil {
		return nil, nil
	}
	spec := datasource.FromOracleSpecProto(rawSpec)
	filters := spec.Data.GetFilters()
	ds := datasource.NewDefinition(datasource.ContentTypeOracle).SetOracleConfig(
		&signedoracle.SpecConfiguration{
			Signers: spec.Data.GetSigners(),
			Filters: filters,
		},
	)
	return datasource.SpecFromDefinition(*ds), &datasource.SpecBindingForCompositePrice{PriceSourceProperty: binding.PriceSourceProperty}
}

func (r marketRow) oracles(config *market.Config) ([]*datasource.Spec, []*datasource.SpecBindingForCompositePrice) {
	specs := []*datasource.Spec{}
	bindings := []*datasource.SpecBindingForCompositePrice{}
	names := []string{"oracle1", "oracle2", "oracle3", "oracle4", "oracle5"}
	for _, v := range names {
		spec, binding := r.compositePriceOracleFromName(config, v)
		if spec == nil {
			continue
		}
		specs = append(specs, spec)
		bindings = append(bindings, binding)
	}
	if len(specs) > 0 {
		return specs, bindings
	}

	return nil, nil
}

func (r marketRow) quadraticSlippageFactor() float64 {
	if !r.row.HasColumn("quadratic slippage factor") {
		// set to 0.1 by default
		return 0.0
	}
	return r.row.MustF64("quadratic slippage factor")
}

func (r marketRow) isSuccessor() bool {
	if pid, ok := r.row.StrB("parent market id"); !ok || len(pid) == 0 {
		return false
	}
	return true
}

func (r marketRow) isPerp() bool {
	if mt, ok := r.row.StrB("market type"); !ok || mt != "perp" {
		return false
	}
	return true
}

func (r marketRow) parentID() string {
	return r.row.MustStr("parent market id")
}

func (r marketRow) insuranceFraction() num.Decimal {
	if !r.row.HasColumn("insurance pool fraction") {
		return num.DecimalZero()
	}
	return r.row.Decimal("insurance pool fraction")
}

func (r marketRow) successorAuction() int64 {
	if !r.row.HasColumn("successor auction") {
		return 5 * r.auctionDuration() // five times auction duration
	}
	return r.row.MustI64("successor auction")
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
