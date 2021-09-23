package storage_test

import (
	"testing"

	"code.vegaprotocol.io/data-node/config/encoding"
	vgtesting "code.vegaprotocol.io/data-node/libs/testing"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/storage"
	types "code.vegaprotocol.io/protos/vega"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

const (
	testMarketID = "ABC123DEF456"
)

func TestMarkets(t *testing.T) {
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()

	st, err := storage.InitialiseStorage(vegaPaths)
	defer st.Purge()
	require.NoError(t, err)

	config := storage.Config{
		Level:          encoding.LogLevel{Level: logging.DebugLevel},
		Markets:        storage.DefaultMarketStoreOptions(),
	}
	marketStore, err := storage.NewMarkets(logging.NewTestLogger(), st.MarketsHome, config, func() {})
	assert.NoError(t, err)
	assert.NotNil(t, marketStore)
	if marketStore == nil {
		t.Fatalf("Could not create market store. Giving up.")
	}
	defer marketStore.Close()

	config.Level.Level = logging.InfoLevel
	marketStore.ReloadConf(config)

	mkt := types.Market{
		Id: testMarketID,
	}
	err = marketStore.Post(&mkt)
	assert.NoError(t, err)

	mkt2, err := marketStore.GetByID("nonexistant_market")
	assert.Equal(t, storage.ErrMarketDoNotExist, err)
	assert.Nil(t, mkt2)

	mkt3, err := marketStore.GetByID(testMarketID)
	assert.NoError(t, err)
	assert.NotNil(t, mkt3)
	assert.Equal(t, mkt.Id, mkt3.Id)

	mkts, err := marketStore.GetAll()
	assert.NoError(t, err)
	assert.NotNil(t, mkts)
	assert.Equal(t, 1, len(mkts))
	assert.Equal(t, mkt.Id, mkts[0].Id)

	err = marketStore.Post(nil)
	assert.Error(t, err)
}
