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

import (
	"testing"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/stretchr/testify/assert"
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
