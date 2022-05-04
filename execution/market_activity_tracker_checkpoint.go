package execution

import (
	"context"
	"sort"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"

	"code.vegaprotocol.io/vega/types"

	"code.vegaprotocol.io/vega/libs/proto"
)

func (mat *MarketActivityTracker) Name() types.CheckpointName {
	return types.MarketActivityTrackerCheckpoint
}

func (mat *MarketActivityTracker) Checkpoint() ([]byte, error) {
	markets := make([]string, 0, len(mat.marketToTracker))
	for k := range mat.marketToTracker {
		markets = append(markets, k)
	}
	sort.Strings(markets)

	marketTracker := make([]*checkpoint.MarketActivityTracker, 0, len(markets))
	for _, market := range markets {
		marketTracker = append(marketTracker, mat.marketToTracker[market].IntoProto(market))
	}
	msg := &checkpoint.MarketTracker{
		MarketActivity: marketTracker,
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
		mat.marketToTracker[data.Market] = marketTrackerFromProto(data)
	}
	return nil
}
