// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package sqlstore_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkLimits(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, ctx, bs)
	block2 := addTestBlock(t, ctx, bs)
	nls := sqlstore.NewNetworkLimits(connectionSource)

	nl := entities.NetworkLimits{
		VegaTime:                 block.VegaTime,
		CanProposeMarket:         true,
		CanProposeAsset:          false,
		ProposeMarketEnabled:     false,
		ProposeAssetEnabled:      true,
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

	fetched, err := nls.GetLatest(ctx)
	require.NoError(t, err)
	assert.Equal(t, nl2, fetched)
}
