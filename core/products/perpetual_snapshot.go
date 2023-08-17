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

package products

import (
	"context"
	"time"

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
	oe OracleEngine,
	broker Broker,
	state *snapshotpb.Perps,
	assetDP uint32,
	tm time.Time,
) (*Perpetual, error) {
	// set next trigger from the settlement cue, it'll roll forward from `initial` to the next trigger time after `now`
	tt := p.DataSourceSpecForSettlementSchedule.Data.GetInternalTimeTriggerSpecConfiguration()
	tt.SetNextTrigger(tm.Truncate(time.Second))

	perps, err := NewPerpetual(ctx, log, p, marketID, oe, broker, assetDP)
	if err != nil {
		return nil, err
	}

	perps.external = make([]*dataPoint, 0, len(state.ExternalDataPoint))
	for _, v := range state.ExternalDataPoint {
		price, overflow := num.UintFromString(v.Price, 10)
		if overflow {
			log.Panic("invalid snapshot state in external data point", logging.String("price", v.Price), logging.Bool("overflow", overflow))
		}
		perps.external = append(perps.external, &dataPoint{price: price, t: v.Timestamp})
	}

	perps.internal = make([]*dataPoint, 0, len(state.InternalDataPoint))
	for _, v := range state.InternalDataPoint {
		price, overflow := num.UintFromString(v.Price, 10)
		if overflow {
			log.Panic("invalid snapshot state in internal data point", logging.String("price", v.Price), logging.Bool("overflow", overflow))
		}
		perps.internal = append(perps.internal, &dataPoint{price: price, t: v.Timestamp})
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
		ExternalDataPoint: make([]*snapshotpb.DataPoint, 0, len(p.external)),
		InternalDataPoint: make([]*snapshotpb.DataPoint, 0, len(p.internal)),
	}

	for _, v := range p.external {
		perps.ExternalDataPoint = append(perps.ExternalDataPoint, &snapshotpb.DataPoint{
			Price:     v.price.String(),
			Timestamp: v.t,
		})
	}

	for _, v := range p.internal {
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
