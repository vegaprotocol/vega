package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

func TestRewardSummaryFromProto(t *testing.T) {
	type args struct {
		pb *v2.RewardSummaryFilter
	}
	tests := []struct {
		name string
		args args
		want RewardSummaryFilter
	}{
		{
			name: "nil",
			args: args{
				pb: nil,
			},
			want: RewardSummaryFilter{},
		}, {
			name: "empty",
			args: args{
				pb: &v2.RewardSummaryFilter{},
			},
			want: RewardSummaryFilter{},
		}, {
			name: "with values",
			args: args{
				pb: &v2.RewardSummaryFilter{
					AssetIds:  []string{"asset1", "asset2"},
					MarketIds: []string{"market1", "market2"},
					FromEpoch: toPtr(uint64(1)),
					ToEpoch:   toPtr(uint64(2)),
				},
			},
			want: RewardSummaryFilter{
				AssetIDs:  []AssetID{"asset1", "asset2"},
				MarketIDs: []MarketID{"market1", "market2"},
				FromEpoch: toPtr(uint64(1)),
				ToEpoch:   toPtr(uint64(2)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := RewardSummaryFilterFromProto(tt.args.pb)
			assert.Equalf(t, tt.want, filter, "RewardSummaryFilterFromProto(%v)", tt.args.pb)
		})
	}
}
