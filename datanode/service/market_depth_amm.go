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

package service

import (
	"context"
	"errors"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrNoAMMVolumeReference = errors.New("cannot find reference price to estimate AMM volume")
	hundred                 = num.DecimalFromInt64(100)
)

// a version of entities.AMMPool that is less flat.
type ammDefn struct {
	partyID  string
	lower    *curve
	upper    *curve
	position num.Decimal // signed position in Vega-space
}

type curve struct {
	low       *num.Uint
	high      *num.Uint
	assetLow  *num.Uint
	assetHigh *num.Uint
	sqrtHigh  num.Decimal
	sqrtLow   num.Decimal
	pv        num.Decimal
	l         num.Decimal
	isLower   bool
}

type level struct {
	price      *num.Uint
	assetPrice *num.Uint
	assetSqrt  num.Decimal
	estimated  bool
}

func newLevel(price *num.Uint, estimated bool, priceFactor num.Decimal) *level {
	assetPrice, _ := num.UintFromDecimal(price.ToDecimal().Mul(priceFactor))
	return &level{
		price:      price.Clone(),
		assetPrice: assetPrice,
		assetSqrt:  num.UintZero().Sqrt(assetPrice),
		estimated:  estimated,
	}
}

func (cu *curve) impliedPosition(sqrtPrice, sqrtHigh num.Decimal) num.Decimal {
	// L * (sqrt(high) - sqrt(price))
	numer := sqrtHigh.Sub(sqrtPrice).Mul(cu.l)

	// sqrt(high) * sqrt(price)
	denom := sqrtHigh.Mul(sqrtPrice)

	// L * (sqrt(high) - sqrt(price)) / sqrt(high) * sqrt(price)
	res := numer.Div(denom)

	if cu.isLower {
		return res
	}

	// if we are in the upper curve the position of 0 in "curve-space" is -cu.pv in Vega position
	// so we need to flip the interval
	return cu.pv.Sub(res).Neg()
}

func (m *MarketDepth) getActiveAMMs(ctx context.Context) map[string][]entities.AMMPool {
	ammByMarket := map[string][]entities.AMMPool{}
	amms, err := m.ammStore.ListActive(ctx)
	if err != nil {
		m.log.Warn("unable to query AMM's for market-depth",
			logging.Error(err),
		)
	}

	for _, amm := range amms {
		marketID := string(amm.MarketID)
		if _, ok := ammByMarket[marketID]; !ok {
			ammByMarket[marketID] = []entities.AMMPool{}
		}

		ammByMarket[marketID] = append(ammByMarket[marketID], amm)
	}
	return ammByMarket
}

func (m *MarketDepth) getCalculationBounds(cache *ammCache, reference num.Decimal, priceFactor num.Decimal) []*level {
	if levels, ok := cache.levels[reference.String()]; ok {
		return levels
	}

	lowestBound := cache.lowestBound
	highestBound := cache.highestBound

	// first lets calculate the region we will expand accurately, this will be some percentage either side of the reference price
	factor := num.DecimalFromFloat(m.cfg.AmmFullExpansionPercentage).Div(hundred)

	// if someone has set the expansion to be more than 100% lets make sure it doesn't overflow
	factor = num.MinD(factor, num.DecimalOne())

	referenceU, _ := num.UintFromDecimal(reference)
	accHigh, _ := num.UintFromDecimal(reference.Mul(num.DecimalOne().Add(factor)))
	accLow, _ := num.UintFromDecimal(reference.Mul(num.DecimalOne().Sub(factor)))

	// always want some volume so if for some reason the bounds were set too low so we calculated a sub-tick expansion make it at least one
	if accHigh.EQ(referenceU) {
		accHigh.Add(referenceU, num.UintOne())
		accLow.Sub(referenceU, num.UintOne())
	}

	// this is the percentage of the reference price to take in estimated steps
	stepFactor := num.DecimalFromFloat(m.cfg.AmmEstimatedStepPercentage).Div(hundred)

	// this is how many of those steps to take
	maxEstimatedSteps := m.cfg.AmmMaxEstimatedSteps

	// and so this is the size of the estimated step
	eStep, _ := num.UintFromDecimal(reference.Mul(stepFactor))

	eRange := num.UintZero().Mul(eStep, num.NewUint(maxEstimatedSteps))
	estLow := num.UintZero().Sub(accLow, num.Min(accLow, eRange))
	estHigh := num.UintZero().Add(accHigh, eRange)

	// cap steps to the lowest/highest boundaries of all AMMs
	lowD, _ := num.UintFromDecimal(lowestBound)
	if accLow.LTE(lowD) {
		accLow = lowD.Clone()
		estLow = lowD.Clone()
	}

	highD, _ := num.UintFromDecimal(highestBound)
	if accHigh.GTE(highD) {
		accHigh = highD.Clone()
		estHigh = highD.Clone()
	}

	// need to find the first n such that
	// accLow - (n * eStep) < lowD
	// accLow
	if estLow.LT(lowD) {
		delta, _ := num.UintZero().Delta(accLow, lowD)
		delta.Div(delta, eStep)
		estLow = num.UintZero().Sub(accLow, delta.Mul(delta, eStep))
	}

	if estHigh.GT(highD) {
		delta, _ := num.UintZero().Delta(accHigh, highD)
		delta.Div(delta, eStep)
		estHigh = num.UintZero().Add(accHigh, delta.Mul(delta, eStep))
	}

	levels := []*level{}

	// we now have our four prices [estLow, accLow, accHigh, estHigh] where from
	// estLow -> accLow   : we will take big price steps
	// accLow -> accHigh  : we will take price steps of one-tick
	// accHigh -> estHigh : we will take big price steps
	price := estLow.Clone()

	// larger steps from estLow -> accHigh
	for price.LT(accLow) {
		levels = append(levels, newLevel(price, true, priceFactor))
		price = num.UintZero().Add(price, eStep)
	}

	// now smaller steps from accLow -> accHigh
	for price.LTE(accHigh) {
		levels = append(levels, newLevel(price, false, priceFactor))
		price = num.UintZero().Add(price, num.UintOne())
	}

	// now back to large steps for accHigh -> estHigh
	for price.LTE(estHigh) {
		levels = append(levels, newLevel(price, true, priceFactor))
		price = num.UintZero().Add(price, eStep)
	}

	cache.levels = map[string][]*level{
		reference.String(): levels,
	}

	return levels
}

func (m *MarketDepth) getReference(ctx context.Context, marketID string) (num.Decimal, error) {
	marketData, err := m.marketData.GetMarketDataByID(ctx, marketID)
	if err != nil {
		m.log.Warn("unable to get market-data for market",
			logging.String("market-id", marketID),
			logging.Error(err),
		)
		return num.DecimalZero(), err
	}

	reference := marketData.MidPrice
	if !marketData.IndicativePrice.IsZero() {
		reference = marketData.IndicativePrice
	}

	if reference.IsZero() {
		m.log.Warn("cannot calculate market-depth for AMM, no reference point available",
			logging.String("mid-price", marketData.MidPrice.String()),
			logging.String("indicative-price", marketData.IndicativePrice.String()),
		)
		return num.DecimalZero(), ErrNoAMMVolumeReference
	}

	return reference, nil
}

func (m *MarketDepth) expandByLevels(pool entities.AMMPool, levels []*level, priceFactor num.Decimal) ([]*types.Order, []bool, error) {
	// get positions

	pos, err := m.getAMMPosition(pool.MarketID.String(), pool.AmmPartyID.String())
	if err != nil {
		return nil, nil, err
	}

	ammDefn := definitionFromEntity(pool, pos, priceFactor)

	estimated := []bool{}
	orders := []*types.Order{}

	level1 := levels[0]
	extraVolume := int64(0)
	for i := range levels {
		if i == len(levels)-1 {
			break
		}

		// level1 := levels[i]
		level2 := levels[i+1]

		// check if the interval is fully outside of the AMM range
		if ammDefn.lower.low.GTE(level2.price) {
			level1 = level2
			continue
		}
		if ammDefn.upper.high.LTE(level1.price) {
			break
		}

		// snap to AMM boundaries
		if level1.price.LT(ammDefn.lower.low) {
			level1 = &level{
				price:      ammDefn.lower.low,
				assetPrice: ammDefn.lower.assetLow,
				assetSqrt:  ammDefn.lower.sqrtLow,
				estimated:  level1.estimated,
			}
		}

		if level2.price.GT(ammDefn.upper.high) {
			level2 = &level{
				price:      ammDefn.upper.high,
				assetPrice: ammDefn.upper.assetHigh,
				assetSqrt:  ammDefn.upper.sqrtHigh,
				estimated:  level2.estimated,
			}
		}

		// pick curve which curve we are in
		cu := ammDefn.lower
		if level1.price.GTE(ammDefn.lower.high) {
			cu = ammDefn.upper
		}

		// let calculate the volume between these two
		v1 := cu.impliedPosition(level1.assetSqrt, cu.sqrtHigh)
		v2 := cu.impliedPosition(level2.assetSqrt, cu.sqrtHigh)

		retPrice := level1.price
		side := types.SideBuy

		if v2.LessThan(ammDefn.position) {
			side = types.SideSell
			retPrice = level2.price

			// if we've stepped over the pool's position we need to split the volume and add it to the outer levels
			if v1.GreaterThan(ammDefn.position) {
				volume := v1.Sub(ammDefn.position).Abs().IntPart()

				// we want to add the volume to the previous order, because thats the price in marketDP when rounded away
				// from the fair-price
				if len(orders) != 0 {
					o := orders[len(orders)-1]
					o.Size += uint64(volume)
					o.Remaining += uint64(volume)
				}

				// we need to add this volume to the price level we step to next
				extraVolume = ammDefn.position.Sub(v2).Abs().IntPart()
				level1 = level2
				continue
			}
		}
		// calculate the volume
		volume := v1.Sub(v2).Abs().IntPart()

		// if the volume is less than zero AMM must be sparse and so we want to keep adding it up until we have at least 1 volume
		// so we'll continue and not shuffle along level1
		if volume == 0 {
			continue
		}

		// this is extra volume from when we stepped over the AMM's fair-price
		if extraVolume != 0 {
			volume += extraVolume
			extraVolume = 0
		}

		orders = append(
			orders,
			m.makeOrder(retPrice, ammDefn.partyID, uint64(volume), side),
		)
		estimated = append(estimated, level1.estimated || level2.estimated)

		// shuffle
		level1 = level2
	}
	return orders, estimated, nil
}

func (m *MarketDepth) InitialiseAMMs(ctx context.Context) {
	active := m.getActiveAMMs(ctx)
	if len(active) == 0 {
		return
	}

	// expand all these AMM's from the midpoint
	for marketID, amms := range active {
		md := m.getDepth(marketID)

		cache, err := m.getAMMCache(marketID)
		if err != nil {
			m.log.Panic("unable to expand AMM's for market",
				logging.Error(err),
				logging.String("market-id", marketID),
			)
		}

		priceFactor := cache.priceFactor

		// add it to our active list, we want to do this even if we fail to get a reference
		for _, a := range amms {
			cache.addAMM(a)
		}

		reference, err := m.getReference(ctx, marketID)
		if err != nil {
			continue
		}

		levels := m.getCalculationBounds(cache, reference, priceFactor)

		for _, amm := range amms {
			orders, estimated, err := m.expandByLevels(amm, levels, priceFactor)
			if err != nil {
				continue
			}

			if len(orders) == 0 {
				continue
			}

			// save them in the cache
			cache.ammOrders[amm.AmmPartyID.String()] = orders

			for i := range orders {
				md.AddAMMOrder(orders[i], estimated[i])
				if estimated[i] {
					cache.estimatedOrder[orders[i].ID] = struct{}{}
				}
			}
		}
	}
}

func (m *MarketDepth) ExpandAMM(ctx context.Context, pool entities.AMMPool, priceFactor num.Decimal) ([]*types.Order, []bool, error) {
	reference, err := m.getReference(ctx, pool.MarketID.String())
	if err == ErrNoAMMVolumeReference {
		// if we can't get a reference to expand from then the market must be fresh and we will just use the pool's base
		reference = pool.ParametersBase
	} else if err != nil {
		return nil, nil, err
	}

	cache, err := m.getAMMCache(string(pool.MarketID))
	if err != nil {
		return nil, nil, err
	}

	levels := m.getCalculationBounds(cache, reference, priceFactor)

	return m.expandByLevels(pool, levels, priceFactor)
}

func (m *MarketDepth) makeOrder(price *num.Uint, partyID string, volume uint64, side types.Side) *types.Order {
	return &types.Order{
		ID:               vgcrypto.RandomHash(),
		Party:            partyID,
		Price:            price,
		Status:           entities.OrderStatusActive,
		Type:             entities.OrderTypeLimit,
		TimeInForce:      entities.OrderTimeInForceGTC,
		Size:             volume,
		Remaining:        volume,
		GeneratedOffbook: true,
		Side:             side,
	}
}

// refreshAMM is used when an AMM has either traded or its definition has changed.
func (m *MarketDepth) refreshAMM(pool entities.AMMPool, depth *entities.MarketDepth) {
	marketID := pool.MarketID.String()
	ammParty := pool.AmmPartyID.String()

	// get all the AMM details from the cache
	cache, err := m.getAMMCache(marketID)
	if err != nil {
		m.log.Warn("unable to refresh AMM expansion",
			logging.Error(err),
			logging.String("market-id", marketID),
		)
	}

	// remove any expanded orders the AMM already has in the depth
	existing := cache.ammOrders[ammParty]
	for _, o := range existing {
		o.Status = entities.OrderStatusCancelled

		_, estimated := cache.estimatedOrder[o.ID]
		delete(cache.estimatedOrder, o.ID)

		depth.AddOrderUpdate(o, estimated)
	}

	if pool.Status == entities.AMMStatusCancelled || pool.Status == entities.AMMStatusStopped {
		cache.removeAMM(ammParty)
		return
	}

	cache.addAMM(pool)

	// expand it again into new orders and push them into the market depth
	orders, estimated, _ := m.ExpandAMM(context.Background(), pool, cache.priceFactor)
	for i := range orders {
		depth.AddOrderUpdate(orders[i], estimated[i])
		if estimated[i] {
			cache.estimatedOrder[orders[i].ID] = struct{}{}
		}
	}
	cache.ammOrders[ammParty] = orders
}

// refreshAMM is used when an AMM has either traded or its definition has changed.
func (m *MarketDepth) OnAMMUpdate(pool entities.AMMPool, vegaTime time.Time, seqNum uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.sequential(vegaTime, seqNum) {
		return
	}

	depth := m.getDepth(pool.MarketID.String())
	depth.SequenceNumber = m.sequenceNumber

	m.refreshAMM(pool, depth)
}

func (m *MarketDepth) onAMMTraded(ammParty, marketID string) {
	cache, err := m.getAMMCache(marketID)
	if err != nil {
		m.log.Warn("unable to refresh AMM expansion",
			logging.Error(err),
			logging.String("market-id", marketID),
		)
	}

	pool, ok := cache.activeAMMs[ammParty]
	if !ok {
		m.log.Panic("market-depth out of sync -- received trade event for AMM that doesn't exist")
	}

	depth := m.getDepth(pool.MarketID.String())
	depth.SequenceNumber = m.sequenceNumber
	m.refreshAMM(pool, depth)
}

func (m *MarketDepth) isAMMOrder(order *types.Order) bool {
	c, ok := m.ammCache[order.MarketID]
	if !ok {
		return false
	}

	_, ok = c.activeAMMs[order.Party]
	return ok
}

func (m *MarketDepth) getAMMCache(marketID string) (*ammCache, error) {
	if cache, ok := m.ammCache[marketID]; ok {
		return cache, nil
	}

	// first time we've seen this market lets get the price factor
	market, err := m.markets.GetByID(context.Background(), marketID)
	if err != nil {
		return nil, err
	}

	assetID, err := market.ToProto().GetAsset()
	if err != nil {
		return nil, err
	}

	asset, err := m.assetStore.GetByID(context.Background(), assetID)
	if err != nil {
		return nil, err
	}

	priceFactor := num.DecimalOne()
	if exp := asset.Decimals - market.DecimalPlaces; exp != 0 {
		priceFactor = num.DecimalFromInt64(10).Pow(num.DecimalFromInt64(int64(exp)))
	}

	cache := &ammCache{
		priceFactor:    priceFactor,
		ammOrders:      map[string][]*types.Order{},
		activeAMMs:     map[string]entities.AMMPool{},
		estimatedOrder: map[string]struct{}{},
		levels:         map[string][]*level{},
	}
	m.ammCache[marketID] = cache

	return cache, nil
}

func (m *MarketDepth) getAMMPosition(marketID, partyID string) (int64, error) {
	p, err := m.positions.GetByMarketAndParty(context.Background(), marketID, partyID)
	if err == nil {
		return p.OpenVolume, nil
	}

	if err == entities.ErrNotFound {
		return 0, nil
	}

	return 0, err
}

func definitionFromEntity(ent entities.AMMPool, position int64, priceFactor num.Decimal) *ammDefn {
	base, _ := num.UintFromDecimal(ent.ParametersBase)
	low := base.Clone()
	high := base.Clone()

	if ent.ParametersLowerBound != nil {
		low, _ = num.UintFromDecimal(*ent.ParametersLowerBound)
	}

	if ent.ParametersUpperBound != nil {
		high, _ = num.UintFromDecimal(*ent.ParametersUpperBound)
	}

	assetHigh, _ := num.UintFromDecimal(high.ToDecimal().Mul(priceFactor))
	assetBase, _ := num.UintFromDecimal(base.ToDecimal().Mul(priceFactor))
	assetLow, _ := num.UintFromDecimal(low.ToDecimal().Mul(priceFactor))

	return &ammDefn{
		position: num.DecimalFromInt64(position),
		lower: &curve{
			low:       low,
			high:      base,
			assetLow:  assetLow,
			assetHigh: assetBase,
			sqrtLow:   num.UintOne().Sqrt(assetLow),
			sqrtHigh:  num.UintOne().Sqrt(assetBase),
			isLower:   true,
			l:         ent.LowerVirtualLiquidity,
			pv:        ent.LowerTheoreticalPosition,
		},
		upper: &curve{
			low:       base,
			high:      high,
			assetLow:  assetBase,
			assetHigh: assetHigh,
			sqrtLow:   num.UintOne().Sqrt(assetBase),
			sqrtHigh:  num.UintOne().Sqrt(assetHigh),
			l:         ent.UpperVirtualLiquidity,
			pv:        ent.UpperTheoreticalPosition,
		},
	}
}
