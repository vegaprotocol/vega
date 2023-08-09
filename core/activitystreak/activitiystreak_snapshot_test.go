// Copyright (c) 2023 Gobalsky Labs Limited
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

package activitystreak_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/activitystreak"
	"code.vegaprotocol.io/vega/core/activitystreak/mocks"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
)

type testSnapshotEngine struct {
	*activitystreak.SnapshotEngine

	ctrl         *gomock.Controller
	broker       *mocks.MockBroker
	marketsStats *mocks.MockMarketsStatsAggregator
}

func getTestSnapshotEngine(t *testing.T) *testSnapshotEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	marketsStats := mocks.NewMockMarketsStatsAggregator(ctrl)
	broker := mocks.NewMockBroker(ctrl)

	return &testSnapshotEngine{
		SnapshotEngine: activitystreak.NewSnapshotEngine(
			logging.NewTestLogger(), marketsStats, broker,
		),
		ctrl:         ctrl,
		broker:       broker,
		marketsStats: marketsStats,
	}
}
