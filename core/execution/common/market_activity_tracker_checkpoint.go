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

package common

import (
	"context"
	"sort"

	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"

	"code.vegaprotocol.io/vega/core/types"

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
