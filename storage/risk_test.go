package storage_test

import (
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"

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
			MaintenanceMargin: 42,
		},
		{
			PartyId:           "p1",
			MarketId:          "m3",
			MaintenanceMargin: 84,
		},
	})
	margins, err := tstore.store.GetMarginLevelsByID("p1", "m1")
	assert.Len(t, margins, 1)
	assert.Nil(t, err)
	assert.Equal(t, margins[0].PartyId, "p1")
	assert.Equal(t, margins[0].MarketId, "m1")
	assert.Equal(t, margins[0].MaintenanceMargin, uint64(42))
}

func testGetMarginLevelsByPartyIDAndMarketID(t *testing.T) {
	tstore := getTestRiskStore()
	tstore.store.SaveMarginLevelsBatch([]proto.MarginLevels{
		{
			PartyId:           "p1",
			MarketId:          "m1",
			MaintenanceMargin: 42,
		},
		{
			PartyId:           "p2",
			MarketId:          "m3",
			MaintenanceMargin: 84,
		},
	})
	margins, err := tstore.store.GetMarginLevelsByID("p2", "")
	assert.Len(t, margins, 1)
	assert.Nil(t, err)
	assert.Equal(t, margins[0].PartyId, "p2")
	assert.Equal(t, margins[0].MarketId, "m3")
	assert.Equal(t, margins[0].MaintenanceMargin, uint64(84))

}

func testGetMarginLevelsErrors(t *testing.T) {
	tstore := getTestRiskStore()
	margins, err := tstore.store.GetMarginLevelsByID("", "")
	assert.Len(t, margins, 0)
	assert.Error(t, err, storage.ErrNoMarginLevelsForParty)
}
