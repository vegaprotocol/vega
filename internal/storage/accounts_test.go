package storage_test

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"

	"github.com/stretchr/testify/assert"
)

func TestAccounts(t *testing.T) {
	dir, tidy, err := storage.TempDir("accountstore-test")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %s", err.Error())
	}
	defer tidy()

	config := storage.Config{
		Level:           encoding.LogLevel{Level: logging.DebugLevel},
		Accounts:        storage.DefaultAccountStoreOptions(),
		AccountsDirPath: dir,
	}
	accountStore, err := storage.NewAccounts(logging.NewTestLogger(), config)
	assert.NoError(t, err)
	assert.NotNil(t, accountStore)
	if accountStore == nil {
		t.Fatalf("Could not create account store. Giving up.")
	}
	defer accountStore.Close()

	config.Level.Level = logging.InfoLevel
	accountStore.ReloadConf(config)
}
