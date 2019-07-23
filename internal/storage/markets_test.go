package storage_test

import (
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"
)

const (
	testMarketId   = "ABC123DEF456"
	testMarketName = "ABCUSD/DEC99"
)

func TestMarkets(t *testing.T) {
	config := storage.Config{
		Level: encoding.LogLevel{Level: logging.DebugLevel},
		// TODO: Use storage.DefaultMarketStoreOptions()
		Markets:        storage.DefaultStoreOptions(),
		MarketsDirPath: storage.TempDir(t, "markets"),
	}
	defer os.RemoveAll(config.MarketsDirPath)
	marketStore, err := storage.NewMarkets(logging.NewTestLogger(), config)
	assert.NoError(t, err)
	assert.NotNil(t, marketStore)
	if marketStore == nil {
		t.Fatalf("Could not create market store. Giving up.")
	}
	defer marketStore.Close()

	err = marketStore.Commit() // no-op for in-memory store
	assert.NoError(t, err)

	config.Level.Level = logging.InfoLevel
	marketStore.ReloadConf(config)

	mkt := types.Market{
		Id:   testMarketId,
		Name: testMarketName,
	}
	err = marketStore.Post(&mkt)
	assert.NoError(t, err)

	mkt2, err := marketStore.GetByID("nonexistant_market")
	assert.Equal(t, badger.ErrKeyNotFound, err)
	assert.Nil(t, mkt2)

	mkt3, err := marketStore.GetByID(testMarketId)
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
