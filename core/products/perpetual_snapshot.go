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

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

func NewPerpetualFromSnapshot(
	ctx context.Context,
	log *logging.Logger,
	p *types.Perps,
	marketID string,
	ts common.TimeService,
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

	perps.externalTWAP = NewCachedTWAP(log, state.StartedAt)
	perps.internalTWAP = NewCachedTWAP(log, state.StartedAt)

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

	perps.startedAt = state.StartedAt
	perps.seq = state.Seq

	return perps, nil
}

func (p *Perpetual) Serialize() *snapshotpb.Product {
	perps := &snapshotpb.Perps{
		Id:                p.id,
		Seq:               p.seq,
		StartedAt:         p.startedAt,
		ExternalDataPoint: make([]*snapshotpb.DataPoint, 0, len(p.internalTWAP.points)),
		InternalDataPoint: make([]*snapshotpb.DataPoint, 0, len(p.externalTWAP.points)),
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
