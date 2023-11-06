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

package common

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
)

func (mat *MarketActivityTracker) Name() types.CheckpointName {
	return types.MarketActivityTrackerCheckpoint
}

func (mat *MarketActivityTracker) Checkpoint() ([]byte, error) {
	assets := make([]string, 0, len(mat.assetToMarketTrackers))
	for k := range mat.assetToMarketTrackers {
		assets = append(assets, k)
	}
	sort.Strings(assets)

	marketTracker := []*checkpoint.MarketActivityTracker{}
	for _, asset := range assets {
		assetTrackers := mat.assetToMarketTrackers[asset]
		markets := make([]string, 0, len(assetTrackers))
		for k := range assetTrackers {
			markets = append(markets, k)
		}
		sort.Strings(markets)
		for _, market := range markets {
			mt := assetTrackers[market]
			marketTracker = append(marketTracker, mt.IntoProto(market))
		}
	}

	msg := &checkpoint.MarketTracker{
		MarketActivity:      marketTracker,
		TakerNotionalVolume: takerNotionalToProto(mat.partyTakerNotionalVolume),
	}
	ret, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (mat *MarketActivityTracker) Load(ctx context.Context, data []byte) error {
	b := checkpoint.MarketTracker{}
	if err := proto.Unmarshal(data, &b); err != nil {
		return err
	}

	for _, data := range b.MarketActivity {
		if _, ok := mat.assetToMarketTrackers[data.Asset]; !ok {
			mat.assetToMarketTrackers[data.Asset] = map[string]*marketTracker{}
		}
		mat.assetToMarketTrackers[data.Asset][data.Market] = marketTrackerFromProto(data)
	}
	for _, tnv := range b.TakerNotionalVolume {
		if len(tnv.Volume) > 0 {
			mat.partyTakerNotionalVolume[tnv.Party] = num.UintFromBytes(tnv.Volume)
		}
	}
	return nil
}
