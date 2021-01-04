package storage_test

import (
	"testing"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
	"code.vegaprotocol.io/vega/storage"

	"github.com/stretchr/testify/assert"
)

const (
	testMarketID = "ABC123DEF456"
)

func TestMarkets(t *testing.T) {
	dir, tidy, err := storage.TempDir("marketstore-test")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %s", err.Error())
	}
	defer tidy()

	config := storage.Config{
		Level:          encoding.LogLevel{Level: logging.DebugLevel},
		Markets:        storage.DefaultMarketStoreOptions(),
		MarketsDirPath: dir,
	}
	marketStore, err := storage.NewMarkets(logging.NewTestLogger(), config, func() {})
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
