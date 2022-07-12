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

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addNetParam(t *testing.T, ns *sqlstore.NetworkParameters, key, value string, block entities.Block) entities.NetworkParameter {
	p := entities.NetworkParameter{
		Key:      key,
		Value:    value,
		VegaTime: block.VegaTime,
	}
	ns.Add(context.Background(), p)
	return p
}

func TestNetParams(t *testing.T) {
	defer DeleteEverything()
	ctx := context.Background()
	netParamStore := sqlstore.NewNetworkParameters(connectionSource)
	blockStore := sqlstore.NewBlocks(connectionSource)
	block1 := addTestBlock(t, blockStore)
	block2 := addTestBlock(t, blockStore)

	param1a := addNetParam(t, netParamStore, "foo", "bar", block1)
	param1b := addNetParam(t, netParamStore, "foo", "baz", block1)
	param2a := addNetParam(t, netParamStore, "cake", "apples", block1)
	param2b := addNetParam(t, netParamStore, "cake", "bananna", block2)

	_ = param1a
	_ = param2a

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.NetworkParameter{param2b, param1b}
		actual, err := netParamStore.GetAll(ctx)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
