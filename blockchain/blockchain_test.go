package blockchain

import (
	"testing"
	"vega/core"

	"github.com/stretchr/testify/assert"
	"vega/datastore"
)

func TestNewBlockchain(t *testing.T) {
	config := core.GetConfig()

	// Storage Service provides read stores for consumer VEGA API
	// Uses in memory storage (maps/slices etc), configurable in future
	storage := &datastore.MemoryStoreProvider{}
	storage.Init([]string{"market-name"}, []string{"partyA", "partyB"})

	// Vega core
	vega := core.New(config, storage)
	chain := NewBlockchain(vega)

	assert.Equal(t, chain.vega.State.Height, int64(0))
}
