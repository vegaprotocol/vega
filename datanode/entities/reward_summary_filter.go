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
