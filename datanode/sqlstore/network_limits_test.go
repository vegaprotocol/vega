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

package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/sqlstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkLimits(t *testing.T) {
	defer DeleteEverything()
	ctx := context.Background()
	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, bs)
	block2 := addTestBlock(t, bs)
	nls := sqlstore.NewNetworkLimits(connectionSource)

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
