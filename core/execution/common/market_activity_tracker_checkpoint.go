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
		MarketActivity:                   marketTracker,
		TakerNotionalVolume:              takerNotionalToProto(mat.partyTakerNotionalVolume),
		MarketToPartyTakerNotionalVolume: marketToPartyTakerNotionalToProto(mat.marketToPartyTakerNotionalVolume),
		EpochTakerFees:                   epochTakerFeesToProto(mat.takerFeesPaidInEpoch),
		GameEligibilityTracker:           epochEligitbilityToProto(mat.eligibilityInEpoch),
	}
	ret, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (mat *MarketActivityTracker) Load(_ context.Context, data []byte) error {
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
	for _, marketToPartyStats := range b.MarketToPartyTakerNotionalVolume {
		mat.marketToPartyTakerNotionalVolume[marketToPartyStats.Market] = map[string]*num.Uint{}
		for _, partyStats := range marketToPartyStats.TakerNotionalVolume {
			if len(partyStats.Volume) > 0 {
				mat.marketToPartyTakerNotionalVolume[marketToPartyStats.Market][partyStats.Party] = num.UintFromBytes(partyStats.Volume)
			}
		}
	}
	if b.EpochTakerFees != nil {
		for _, epochData := range b.EpochTakerFees {
			epochMap := map[string]map[string]map[string]*num.Uint{}
			for _, assetMarketParty := range epochData.EpochPartyTakerFeesPaid {
				if _, ok := epochMap[assetMarketParty.Asset]; !ok {
					epochMap[assetMarketParty.Asset] = map[string]map[string]*num.Uint{}
				}
				if _, ok := epochMap[assetMarketParty.Asset][assetMarketParty.Market]; !ok {
					epochMap[assetMarketParty.Asset][assetMarketParty.Market] = map[string]*num.Uint{}
				}
				for _, tf := range assetMarketParty.TakerFees {
					epochMap[assetMarketParty.Asset][assetMarketParty.Market][tf.Party] = num.UintFromBytes(tf.TakerFees)
				}
			}
			mat.takerFeesPaidInEpoch = append(mat.takerFeesPaidInEpoch, epochMap)
		}
	}
	if b.GameEligibilityTracker != nil {
		for _, get := range b.GameEligibilityTracker {
			mat.eligibilityInEpoch[get.GameId] = make([]map[string]struct{}, len(get.EpochEligibility))
			for i, epoch := range get.EpochEligibility {
				mat.eligibilityInEpoch[get.GameId][i] = make(map[string]struct{}, len(epoch.EligibleParties))
				for _, party := range epoch.EligibleParties {
					mat.eligibilityInEpoch[get.GameId][i][party] = struct{}{}
				}
			}
		}
	}

	return nil
}
