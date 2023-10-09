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

package entities

import v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

// RewardSummaryFilter is the filter for the reward summary.
type RewardSummaryFilter struct {
	AssetIDs  []AssetID
	MarketIDs []MarketID
	FromEpoch *uint64
	ToEpoch   *uint64
}

// RewardSummaryFilterFromProto converts a protobuf v2.RewardSummaryFilter to an entities.RewardSummaryFilter.
func RewardSummaryFilterFromProto(pb *v2.RewardSummaryFilter) (filter RewardSummaryFilter) {
	if pb != nil {
		filter.AssetIDs = fromStringIDs[AssetID](pb.AssetIds)
		filter.MarketIDs = fromStringIDs[MarketID](pb.MarketIds)
		filter.FromEpoch = pb.FromEpoch
		filter.ToEpoch = pb.ToEpoch
	}
	return
}

func fromStringIDs[id ID[typ], typ any](in []string) (ids []id) {
	if len(in) == 0 {
		return
	}
	ids = make([]id, len(in))
	for i, idStr := range in {
		ids[i] = id(idStr)
	}
	return
}
