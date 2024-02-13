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

package products

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

func NewCachedTWAPFromSnapshot(
	log *logging.Logger,
	t int64,
	auctions *auctionIntervals,
	state *snapshotpb.TWAPData,
	points []*snapshotpb.DataPoint,
) *cachedTWAP {
	sum, _ := num.UintFromString(state.SumProduct, 10)
	c := &cachedTWAP{
		log:         log,
		periodStart: t,
		start:       state.Start,
		end:         state.End,
		sumProduct:  sum,
		auctions:    auctions,
	}
	c.points = make([]*dataPoint, 0, len(points))
	for _, v := range points {
		price, overflow := num.UintFromString(v.Price, 10)
		if overflow {
			log.Panic("invalid snapshot state in external data point", logging.String("price", v.Price), logging.Bool("overflow", overflow))
		}
		c.points = append(c.points, &dataPoint{price: price, t: v.Timestamp})
	}
	return c
}

func (c *cachedTWAP) serialise() *snapshotpb.TWAPData {
	return &snapshotpb.TWAPData{
		Start:      c.start,
		End:        c.end,
		SumProduct: c.sumProduct.String(),
	}
}

func NewPerpetualFromSnapshot(
	ctx context.Context,
	log *logging.Logger,
	p *types.Perps,
	marketID string,
	ts TimeService,
	oe OracleEngine,
	broker Broker,
	state *snapshotpb.Perps,
	assetDP uint32,
) (*Perpetual, error) {
	// set next trigger from the settlement cue, it'll roll forward from `initial` to the next trigger time after `now`
	tt := p.DataSourceSpecForSettlementSchedule.Data.GetInternalTimeTriggerSpecConfiguration()
	tt.SetNextTrigger(ts.GetTimeNow().Truncate(time.Second))

	perps, err := NewPerpetual(ctx, log, p, marketID, ts, oe, broker, assetDP)
	if err != nil {
		return nil, err
	}

	perps.startedAt = state.StartedAt
	perps.seq = state.Seq

	if vgcontext.InProgressUpgradeFrom(ctx, "v0.73.13") {
		// do it the old way where we'd regenerate the cached values by adding the points again
		perps.externalTWAP = NewCachedTWAP(log, state.StartedAt, perps.auctions)
		perps.internalTWAP = NewCachedTWAP(log, state.StartedAt, perps.auctions)

		for _, v := range state.ExternalDataPoint {
			price, overflow := num.UintFromString(v.Price, 10)
			if overflow {
				log.Panic("invalid snapshot state in external data point", logging.String("price", v.Price), logging.Bool("overflow", overflow))
			}
			perps.externalTWAP.addPoint(&dataPoint{price: price, t: v.Timestamp})
		}

		for _, v := range state.InternalDataPoint {
			price, overflow := num.UintFromString(v.Price, 10)
			if overflow {
				log.Panic("invalid snapshot state in internal data point", logging.String("price", v.Price), logging.Bool("overflow", overflow))
			}
			perps.internalTWAP.addPoint(&dataPoint{price: price, t: v.Timestamp})
		}
		return perps, nil
	}

	perps.auctions = &auctionIntervals{
		auctionStart: state.AuctionIntervals.AuctionStart,
		auctions:     state.AuctionIntervals.T,
		total:        state.AuctionIntervals.Total,
	}

	perps.externalTWAP = NewCachedTWAPFromSnapshot(log, state.StartedAt, perps.auctions, state.ExternalTwapData, state.ExternalDataPoint)
	perps.internalTWAP = NewCachedTWAPFromSnapshot(log, state.StartedAt, perps.auctions, state.InternalTwapData, state.InternalDataPoint)
	return perps, nil
}

func (p *Perpetual) Serialize() *snapshotpb.Product {
	perps := &snapshotpb.Perps{
		Id:                p.id,
		Seq:               p.seq,
		StartedAt:         p.startedAt,
		ExternalDataPoint: make([]*snapshotpb.DataPoint, 0, len(p.internalTWAP.points)),
		InternalDataPoint: make([]*snapshotpb.DataPoint, 0, len(p.externalTWAP.points)),
		AuctionIntervals: &snapshotpb.AuctionIntervals{
			AuctionStart: p.auctions.auctionStart,
			T:            p.auctions.auctions,
			Total:        p.auctions.total,
		},
		ExternalTwapData: p.externalTWAP.serialise(),
		InternalTwapData: p.internalTWAP.serialise(),
	}

	for _, v := range p.externalTWAP.points {
		perps.ExternalDataPoint = append(perps.ExternalDataPoint, &snapshotpb.DataPoint{
			Price:     v.price.String(),
			Timestamp: v.t,
		})
	}

	for _, v := range p.internalTWAP.points {
		perps.InternalDataPoint = append(perps.InternalDataPoint, &snapshotpb.DataPoint{
			Price:     v.price.String(),
			Timestamp: v.t,
		})
	}
	return &snapshotpb.Product{
		Type: &snapshotpb.Product_Perps{
			Perps: perps,
		},
	}
}
