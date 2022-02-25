package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkLimits(t *testing.T) {
	defer testStore.DeleteEverything()
	ctx := context.Background()
	bs := sqlstore.NewBlocks(testStore)
	block := addTestBlock(t, bs)
	block2 := addTestBlock(t, bs)
	nls := sqlstore.NewNetworkLimits(testStore)

	nl := entities.NetworkLimits{
		VegaTime:                 block.VegaTime,
		CanProposeMarket:         true,
		CanProposeAsset:          false,
		BootstrapFinished:        true,
		ProposeMarketEnabled:     false,
		ProposeAssetEnabled:      true,
		BootstrapBlockCount:      42,
		GenesisLoaded:            false,
		ProposeMarketEnabledFrom: time.Now().Truncate(time.Millisecond),
		ProposeAssetEnabledFrom:  time.Now().Add(time.Second * 10).Truncate(time.Millisecond),
	}

	err := nls.Add(ctx, nl)
	require.NoError(t, err)

	nl2 := nl
	nl2.VegaTime = block2.VegaTime
	err = nls.Add(ctx, nl2)
	require.NoError(t, err)

	fetched_nl, err := nls.GetLatest(ctx)
	require.NoError(t, err)
	assert.Equal(t, nl2, fetched_nl)
}
