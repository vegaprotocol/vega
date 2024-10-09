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

package spot

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/datasource"
	dscommon "code.vegaprotocol.io/vega/core/datasource/common"
	dsdefinition "code.vegaprotocol.io/vega/core/datasource/definition"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

type ProtocolAutomatedPurchase struct {
	ID                   string
	config               *types.NewProtocolAutomatedPurchaseChanges
	nextAuctionAmount    *num.Uint
	lastOraclePrice      *num.Uint
	lastOracleUpdateTime time.Time
	priceOracle          *products.CompositePriceOracle
	scheuldingOracles    *products.AutomatedPurhcaseSchedulingOracles
	side                 types.Side
	activeOrder          string
	lock                 sync.Mutex
	readyToStop          bool
}

func (ap *ProtocolAutomatedPurchase) IntoProto() *snapshot.ProtocolAutomatedPurchase {
	apProto := &snapshot.ProtocolAutomatedPurchase{
		Id:          ap.ID,
		Config:      ap.config.IntoProto(),
		Side:        ap.side,
		ActiveOrder: ap.activeOrder,
		ReadyToStop: ap.readyToStop,
	}
	if ap.nextAuctionAmount != nil {
		apProto.NextAuctionAmount = ap.nextAuctionAmount.String()
	}
	if ap.lastOraclePrice != nil {
		apProto.LastOraclePrice = ap.lastOraclePrice.String()
		apProto.LastOracleUpdateTime = ap.lastOracleUpdateTime.UnixNano()
	}
	return apProto
}

func (m *Market) NewProtocolAutomatedPurchase(ctx context.Context, ID string, config *types.NewProtocolAutomatedPurchaseChanges, oracleEngine common.OracleEngine) error {
	if m.pap != nil {
		m.log.Panic("cannot instantiate new protocol automated purchase while there is already an active one", logging.String("active-pap", m.pap.ID))
	}
	side := types.SideUnspecified
	if config.From == m.baseAsset {
		side = types.SideSell
	} else if config.From == m.quoteAsset {
		side = types.SideBuy
	}
	if side == types.SideUnspecified {
		m.log.Panic("wrong market for automated purchase", logging.String("market-id", config.MarketID), logging.String("from", config.From), logging.String("market-base-asset", m.baseAsset), logging.String("market-quote-asset", m.quoteAsset))
	}

	pap := &ProtocolAutomatedPurchase{
		ID:          ID,
		config:      config,
		activeOrder: "",
		readyToStop: false,
		side:        side,
	}

	auctionVolumeSnapshotSchedule := datasource.SpecFromDefinition(config.AuctionVolumeSnapshotSchedule)
	auctionSchedule := datasource.SpecFromDefinition(config.AuctionSchedule)
	var err error
	pap.scheuldingOracles, err = products.NewProtocolAutomatedPurchaseScheduleOracle(ctx, oracleEngine, auctionSchedule, auctionVolumeSnapshotSchedule, datasource.SpecBindingForAutomatedPurchaseFromProto(config.AutomatedPurchaseSpecBinding), m.papAuctionSchedule, m.papAuctionVolumeSnapshot)
	if err != nil {
		return err
	}
	oracle, err := products.NewCompositePriceOracle(ctx, oracleEngine, config.PriceOracle, datasource.SpecBindingForCompositePriceFromProto(config.PriceOracleBinding), m.updatePAPPriceOracle)
	if err != nil {
		return err
	}
	pap.priceOracle = oracle

	m.pap = pap
	return nil
}

func (m *Market) NewProtocolAutomatedPurchaseFromSnapshot(ctx context.Context, oracleEngine common.OracleEngine, apProto *snapshot.ProtocolAutomatedPurchase) (*ProtocolAutomatedPurchase, error) {
	if apProto == nil {
		return nil, nil
	}
	ap := &ProtocolAutomatedPurchase{
		ID:          apProto.Id,
		config:      types.NewProtocolAutomatedPurchaseChangesFromProto(apProto.Config),
		activeOrder: apProto.ActiveOrder,
		side:        apProto.Side,
		readyToStop: apProto.ReadyToStop,
	}
	if len(apProto.LastOraclePrice) > 0 {
		ap.lastOraclePrice = num.MustUintFromString(apProto.LastOraclePrice, 10)
		ap.lastOracleUpdateTime = time.Unix(0, apProto.LastOracleUpdateTime)
	}
	if len(apProto.NextAuctionAmount) > 0 {
		ap.nextAuctionAmount = num.MustUintFromString(apProto.NextAuctionAmount, 10)
	}

	specDef, _ := dsdefinition.FromProto(apProto.Config.PriceOracle, nil)
	priceOracle := datasource.SpecFromDefinition(*dsdefinition.NewWith(specDef))

	oracle, err := products.NewCompositePriceOracle(ctx, oracleEngine, priceOracle, datasource.SpecBindingForCompositePriceFromProto(apProto.Config.PriceOracleSpecBinding), m.updatePAPPriceOracle)
	if err != nil {
		return nil, err
	}
	ap.priceOracle = oracle
	auctionSchedule := datasource.SpecFromDefinition(ap.config.AuctionSchedule)
	auctionSchedule.Data.GetInternalTimeTriggerSpecConfiguration().Triggers[0].SetNextTrigger(m.timeService.GetTimeNow().Truncate(time.Second))
	auctionVolumeSnapshotSchedule := datasource.SpecFromDefinition(ap.config.AuctionVolumeSnapshotSchedule)
	auctionVolumeSnapshotSchedule.Data.GetInternalTimeTriggerSpecConfiguration().Triggers[0].SetNextTrigger(m.timeService.GetTimeNow().Truncate(time.Second))
	ap.scheuldingOracles, err = products.NewProtocolAutomatedPurchaseScheduleOracle(ctx, oracleEngine, auctionSchedule, auctionVolumeSnapshotSchedule, datasource.SpecBindingForAutomatedPurchaseFromProto(ap.config.AutomatedPurchaseSpecBinding), m.papAuctionSchedule, m.papAuctionVolumeSnapshot)
	if err != nil {
		return nil, err
	}
	return ap, nil
}

func (m *Market) scaleOraclePriceToAssetDP(price *num.Numeric, dp int64) *num.Uint {
	if price == nil {
		return nil
	}

	if !price.SupportDecimalPlaces(int64(m.quoteAssetDP)) {
		return nil
	}

	p, err := price.ScaleTo(dp, int64(m.quoteAssetDP))
	if err != nil {
		m.log.Error(err.Error())
		return nil
	}
	return p
}

// updatePAPPriceOracle is called by the oracle to update the price in quote asset decimals.
func (m *Market) updatePAPPriceOracle(ctx context.Context, data dscommon.Data) error {
	m.log.Info("updatePAPPriceOracle", logging.String("current-time", m.timeService.GetTimeNow().String()))
	if m.pap == nil {
		m.log.Error("unexpected pap oracle price update - no active pap")
		return nil
	}
	m.pap.lock.Lock()
	defer m.pap.lock.Unlock()

	pd, err := m.pap.priceOracle.GetData(data)
	if err != nil {
		return err
	}
	p := m.scaleOraclePriceToAssetDP(pd, m.pap.priceOracle.GetDecimals())
	if p == nil || p.IsZero() {
		return nil
	}

	m.pap.lastOraclePrice = p.Clone()
	m.pap.lastOracleUpdateTime = m.timeService.GetTimeNow()
	return nil
}

// AuctionVolumeSnapshot is called from the oracle in order to take a snapshot of the source account balance in preparation for the coming auction.
func (m *Market) papAuctionVolumeSnapshot(ctx context.Context, data dscommon.Data) error {
	m.log.Info("papAuctionVolumeSnapshot", logging.String("current-time", m.timeService.GetTimeNow().String()))
	if m.pap == nil {
		m.log.Error("unexpected pap auction volume snapshot")
		return nil
	}

	m.pap.lock.Lock()
	defer m.pap.lock.Unlock()

	// if the program has been stopped, the oracles unsubscribed but this was able to sneak in, ignore it.
	if m.pap.readyToStop {
		m.log.Info("pap is ready to stop as soon as auction completes, not taking any more snapshots")
		return nil
	}

	// if we already have an order place in an auction that is waiting to be traded - do nothing
	if len(m.pap.activeOrder) > 0 {
		m.log.Info("not taking a snapshot for pap which already has an active order", logging.String("pap-id", m.pap.ID), logging.String("active-order-id", m.pap.activeOrder))
		return nil
	}

	// if we happen to have an earmarked amount that was not submitted, unearmark it first
	// this would be the case if we're seeing more than one tick from the auction volume snapshot before we see one
	// tick from the auction scheduler trigger
	if m.pap.nextAuctionAmount != nil && !m.pap.nextAuctionAmount.IsZero() {
		if err := m.collateral.UnearmarkForAutomatedPurchase(m.pap.config.From, m.pap.config.FromAccountType, m.pap.nextAuctionAmount.Clone()); err != nil {
			m.log.Panic("failed to unearmark balance for automated purchase", logging.Error(err))
		}
	}

	// earmark the amount for the next pap round
	earmarkedBalance, err := m.collateral.EarmarkForAutomatedPurchase(m.pap.config.From, m.pap.config.FromAccountType, m.pap.config.MinimumAuctionSize, m.pap.config.MaximumAuctionSize)
	if err != nil {
		m.log.Error("error in earmarking for automated purchase", logging.Error(err))
		return err
	}

	m.pap.nextAuctionAmount = earmarkedBalance
	// emit an event with the next auction balance
	m.broker.Send(events.NewProtocolAutomatedPurchaseAnnouncedEvent(ctx, m.pap.config.From, m.pap.config.FromAccountType, m.pap.config.ToAccountType, m.pap.config.MarketID, m.pap.nextAuctionAmount))
	return nil
}

// AuctionSchedule is called by the oracle to notify on a required auction.
func (m *Market) papAuctionSchedule(ctx context.Context, data dscommon.Data) error {
	m.log.Info("papAuctionSchedule", logging.String("current-time", m.timeService.GetTimeNow().String()))
	if m.pap == nil {
		m.log.Error("unexpected pap auction snapshot - no active pap")
		return nil
	}
	// at the end of this function we should unearmark and reset the next auction amount no matter if we succeeded or failed to enter an auction
	// at this point we can unearmark the amount - either because we were able to enter an auction and place an order -
	// in which case the amount has been transferred into the holding account, or because there was an error and we failed
	defer func() {
		// this should be fine as the defer happen as fifo so by the time this is called the unlock of the function locking has already taken place.
		m.pap.lock.Lock()
		defer m.pap.lock.Unlock()
		if m.pap.readyToStop {
			return
		}
		if m.pap.nextAuctionAmount != nil {
			m.collateral.UnearmarkForAutomatedPurchase(m.pap.config.From, m.pap.config.FromAccountType, m.pap.nextAuctionAmount)
		}
		m.pap.nextAuctionAmount = nil
	}()

	m.pap.lock.Lock()
	defer m.pap.lock.Unlock()

	if m.pap.readyToStop {
		return nil
	}

	// if there was nothing earmarked for next auction - return
	if m.pap.nextAuctionAmount == nil {
		return nil
	}

	// no last orace price - nothing to do here
	if m.pap.lastOraclePrice == nil {
		m.log.Warn("auction scheduled triggered but no oracle price", logging.String("marked-id", m.pap.config.MarketID), logging.String("automated-purchase-id", m.pap.ID))
		return nil
	}
	// stale orace price - nothing to do here
	if int64(m.timeService.GetTimeNow().Nanosecond())-m.pap.lastOracleUpdateTime.UnixNano() > m.pap.config.OraclePriceStalenessTolerance.Nanoseconds() {
		m.log.Warn("auction scheduled triggered but oracle price is stale", logging.String("marked-id", m.pap.config.MarketID), logging.String("automated-purchase-id", m.pap.ID), logging.String("last-oracle-update", m.pap.lastOracleUpdateTime.String()))
		return nil
	}

	// factor the last orale price by the offset
	orderPrice, overflow := num.UintFromDecimal(m.pap.lastOraclePrice.ToDecimal().Mul(m.pap.config.OracleOffsetFactor))
	if overflow || orderPrice == nil {
		m.log.Error("failed to get order price for automated purchase auction", logging.String("from", m.pap.config.From), logging.String("market-id", m.pap.config.MarketID))
		return nil
	}

	// calculate the order size
	// if the order is a sell, i.e. we're selling the base asset, we need to scale it by the base factor
	orderSize := scaleBaseAssetDPToQuantity(m.pap.nextAuctionAmount, m.baseFactor)

	// if the order is a buy, that means the auction amount is in quote asset and we need to calculate the size of base
	// while factoring in the necessary fees.
	if m.pap.side == types.SideBuy {
		feeFactor := num.DecimalOne().Add(m.mkt.Fees.Factors.InfrastructureFee).Add(m.mkt.Fees.Factors.BuyBackFee).Add(m.mkt.Fees.Factors.TreasuryFee).Add(m.fee.GetLiquidityFee())
		// this gives us a size in the quote asset decimals - we need to convert it to base asset decimals and then
		// adjust it by the position factor
		orderSizeI, _ := num.UintFromDecimal(m.pap.nextAuctionAmount.ToDecimal().Div(feeFactor.Mul(orderPrice.ToDecimal())).Mul(m.positionFactor))
		orderSize = orderSizeI.Uint64()
	}

	orderPriceInMarket := m.priceToMarketPrecision(orderPrice)
	orderID := hex.EncodeToString(crypto.Hash([]byte(m.pap.ID)))
	orderID, err := m.enterAutomatedPurchaseAuction(ctx, orderID, m.pap.side, orderPriceInMarket, orderSize, m.pap.ID, m.pap.config.AuctionDuration)
	// if there was no error save the order id as an indication that we're in an auction with active pap order
	if err == nil {
		m.pap.activeOrder = orderID
	}

	return err
}

func (m *Market) papOrderProcessingEnded(orderID string) {
	if m.pap.activeOrder == orderID {
		m.pap.activeOrder = ""
	}
}

func (ap *ProtocolAutomatedPurchase) getACcountTypesForPAP() (types.AccountType, types.AccountType, error) {
	return ap.config.FromAccountType, ap.config.ToAccountType, nil
}

func (m *Market) stopPAP(ctx context.Context) {
	m.pap.lock.Lock()
	defer m.pap.lock.Unlock()
	m.pap.readyToStop = true
	m.pap.priceOracle.UnsubAll(ctx)
	m.pap.scheuldingOracles.UnsubAll(ctx)
	m.pap.nextAuctionAmount = nil
}

// checkPAP checks if a pap has expired, if so and it.
func (m *Market) checkPAP(ctx context.Context) {
	// no pap - nothing to do
	if m.pap == nil {
		return
	}
	// pap already stopped and no active order for it - we can reset the pap
	if m.pap.readyToStop && len(m.pap.activeOrder) == 0 {
		m.pap = nil
		return
	}
	// pap has expired
	if !m.pap.readyToStop && m.pap.config.ExpiryTimestamp.Unix() > 0 && m.pap.config.ExpiryTimestamp.Before(m.timeService.GetTimeNow()) {
		m.log.Info("protocol automated purchase has expired, going to stop", logging.String("ID", m.pap.ID))
		m.stopPAP(ctx)
	}
}

func (m *Market) MarketHasActivePAP() bool {
	return m.pap != nil
}

func scaleBaseAssetDPToQuantity(assetQuantity *num.Uint, baseFactor num.Decimal) uint64 {
	sizeU, _ := num.UintFromDecimal(assetQuantity.ToDecimal().Div(baseFactor))
	return sizeU.Uint64()
}
