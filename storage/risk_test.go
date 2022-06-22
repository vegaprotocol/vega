// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package storage_test

import (
	"testing"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/storage"
	proto "code.vegaprotocol.io/protos/vega"

	"github.com/stretchr/testify/assert"
)

func TestRiskStore(t *testing.T) {
	t.Run("Get Margin Levels by party ID", testGetMarginLevelsByPartyID)
	t.Run("Get Margin Levels by party ID and Market ID", testGetMarginLevelsByPartyIDAndMarketID)
	t.Run("Get Margin levels - errors", testGetMarginLevelsErrors)
}

type testRiskStore struct {
	store *storage.Risk
}

func getTestRiskStore() *testRiskStore {
	return &testRiskStore{
		store: storage.NewRisks(logging.NewTestLogger(), storage.Config{}),
	}
}

func testGetMarginLevelsByPartyID(t *testing.T) {
	tstore := getTestRiskStore()
	tstore.store.SaveMarginLevelsBatch([]proto.MarginLevels{
		{
			PartyId:           "p1",
			MarketId:          "m1",
			MaintenanceMargin: "42",
		},
		{
			PartyId:           "p1",
			MarketId:          "m3",
			MaintenanceMargin: "84",
		},
	})
	margins, err := tstore.store.GetMarginLevelsByID("p1", "m1")
	assert.Len(t, margins, 1)
	assert.Nil(t, err)
	assert.Equal(t, margins[0].PartyId, "p1")
	assert.Equal(t, margins[0].MarketId, "m1")
	assert.Equal(t, margins[0].MaintenanceMargin, "42")
}

func testGetMarginLevelsByPartyIDAndMarketID(t *testing.T) {
	tstore := getTestRiskStore()
	tstore.store.SaveMarginLevelsBatch([]proto.MarginLevels{
		{
			PartyId:           "p1",
			MarketId:          "m1",
			MaintenanceMargin: "42",
		},
		{
			PartyId:           "p2",
			MarketId:          "m3",
			MaintenanceMargin: "84",
		},
	})
	margins, err := tstore.store.GetMarginLevelsByID("p2", "")
	assert.Len(t, margins, 1)
	assert.Nil(t, err)
	assert.Equal(t, margins[0].PartyId, "p2")
	assert.Equal(t, margins[0].MarketId, "m3")
	assert.Equal(t, margins[0].MaintenanceMargin, "84")

}

func testGetMarginLevelsErrors(t *testing.T) {
	tstore := getTestRiskStore()
	margins, err := tstore.store.GetMarginLevelsByID("", "")
	assert.Len(t, margins, 0)
	assert.Error(t, err, storage.ErrNoMarginLevelsForParty)
}
