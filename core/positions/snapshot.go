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

package positions

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"golang.org/x/exp/maps"
)

type SnapshotEngine struct {
	*Engine
	pl      types.Payload
	data    []byte
	stopped bool
}

func NewSnapshotEngine(
	log *logging.Logger, config Config, marketID string, broker Broker,
) *SnapshotEngine {
	return &SnapshotEngine{
		Engine:  New(log, config, marketID, broker),
		pl:      types.Payload{},
		stopped: false,
	}
}

// StopSnapshots is called when the engines respective market no longer exists. We need to stop
// taking snapshots and communicate to the snapshot engine to remove us as a provider.
func (e *SnapshotEngine) StopSnapshots() {
	e.log.Debug("market has been cleared, stopping snapshot production", logging.String("marketid", e.marketID))
	e.stopped = true
}

func (e *SnapshotEngine) Namespace() types.SnapshotNamespace {
	return types.PositionsSnapshot
}

func (e *SnapshotEngine) Keys() []string {
	return []string{e.marketID}
}

func (e *SnapshotEngine) Stopped() bool {
	return e.stopped
}

func (e *SnapshotEngine) GetState(k string) ([]byte, []types.StateProvider, error) {
	if k != e.marketID {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	state, err := e.serialise()
	return state, nil, err
}

func (e *SnapshotEngine) LoadState(_ context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	var err error
	switch pl := payload.Data.(type) {
	case *types.PayloadMarketPositions:
		// Check the payload is for this market
		if e.marketID != pl.MarketPositions.MarketID {
			return nil, types.ErrUnknownSnapshotType
		}
		e.log.Debug("loading snapshot", logging.Int("positions", len(pl.MarketPositions.Positions)))
		for _, p := range pl.MarketPositions.Positions {
			pos := NewMarketPosition(p.PartyID)
			pos.price = p.Price
			pos.buy = p.Buy
			pos.sell = p.Sell
			pos.size = p.Size
			pos.buySumProduct = p.BuySumProduct
			pos.sellSumProduct = p.SellSumProduct
			pos.distressed = p.Distressed
			pos.averageEntryPrice = p.AverageEntryPrice
			e.positionsCpy = append(e.positionsCpy, pos)
			e.positions[p.PartyID] = pos
			if p.Distressed {
				e.distressedPos[p.PartyID] = struct{}{}
			}

			// This is for migration, on the first time we load from snapshot there won't be an average entry price
			// so take the last price as the current average
			if p.AverageEntryPrice == nil {
				if pos.size != 0 && !pos.price.IsZero() {
					pos.averageEntryPrice = pos.price.Clone()
				} else {
					pos.averageEntryPrice = num.UintZero()
				}
			}

			// ensure these exists on the first snapshot after the upgrade
			e.partiesHighestVolume[p.PartyID] = &openVolumeRecord{}
		}

		for _, v := range pl.MarketPositions.PartieRecords {
			if v.LatestOpenInterest != nil && v.LowestOpenInterest != nil {
				e.partiesHighestVolume[v.Party] = &openVolumeRecord{
					Latest:  *v.LatestOpenInterest,
					Highest: *v.LowestOpenInterest,
				}
			}

			if v.TradedVolume != nil {
				e.partiesTradedSize[v.Party] = *v.TradedVolume
			}
		}

		e.data, err = proto.Marshal(payload.IntoProto())
		return nil, err

	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

// serialise marshal the snapshot state, populating the data field
// with updated values.
func (e *SnapshotEngine) serialise() ([]byte, error) {
	if e.stopped {
		return nil, nil
	}

	e.log.Debug("serialising snapshot", logging.Int("positions", len(e.positionsCpy)))
	positions := make([]*types.MarketPosition, 0, len(e.positionsCpy))

	for _, evt := range e.positionsCpy {
		party := evt.Party()
		_, distressed := e.distressedPos[party]
		pos := &types.MarketPosition{
			PartyID:           party,
			Price:             evt.Price(),
			Buy:               evt.Buy(),
			Sell:              evt.Sell(),
			Size:              evt.Size(),
			BuySumProduct:     evt.BuySumProduct(),
			SellSumProduct:    evt.SellSumProduct(),
			Distressed:        distressed,
			AverageEntryPrice: evt.AverageEntryPrice(),
		}
		positions = append(positions, pos)
	}

	partiesRecordsMap := map[string]*snapshotpb.PartyPositionStats{}

	// now iterate over both map as some could have been remove
	// when closing positions or being closed out.
	for party, poi := range e.partiesHighestVolume {
		partiesRecordsMap[party] = &snapshotpb.PartyPositionStats{
			Party:              party,
			LowestOpenInterest: ptr.From(poi.Highest),
			LatestOpenInterest: ptr.From(poi.Latest),
		}
	}

	for party, tradedSize := range e.partiesTradedSize {
		if pr, ok := partiesRecordsMap[party]; ok {
			pr.TradedVolume = ptr.From(tradedSize)
			continue
		}

		partiesRecordsMap[party] = &snapshotpb.PartyPositionStats{
			Party:        party,
			TradedVolume: ptr.From(tradedSize),
		}
	}

	partiesRecord := maps.Values(partiesRecordsMap)
	sort.Slice(partiesRecord, func(i, j int) bool {
		return partiesRecord[i].Party < partiesRecord[j].Party
	})

	e.pl.Data = &types.PayloadMarketPositions{
		MarketPositions: &types.MarketPositions{
			MarketID:      e.marketID,
			Positions:     positions,
			PartieRecords: partiesRecord,
		},
	}

	var err error
	e.data, err = proto.Marshal(e.pl.IntoProto())
	if err != nil {
		return nil, err
	}
	return e.data, nil
}
