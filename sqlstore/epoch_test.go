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

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestEpoch(t *testing.T, es *sqlstore.Epochs,
	epochID int64,
	startTime time.Time,
	expireTime time.Time,
	endTime *time.Time,
	block entities.Block,
) entities.Epoch {
	r := entities.Epoch{
		ID:         epochID,
		StartTime:  startTime,
		ExpireTime: expireTime,
		EndTime:    endTime,
		VegaTime:   block.VegaTime,
	}
	err := es.Add(context.Background(), r)
	require.NoError(t, err)
	return r
}

func TestEpochs(t *testing.T) {
	defer DeleteEverything()
	ctx := context.Background()
	es := sqlstore.NewEpochs(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)
	block1 := addTestBlock(t, bs)
	block2 := addTestBlock(t, bs)
	block3 := addTestBlock(t, bs)

	epoch1Start := time.Date(2022, 1, 1, 0, 0, 0, 0, time.Local)
	epoch1Expire := epoch1Start.Add(time.Minute)
	epoch1End := epoch1Expire.Add(time.Second)
	epoch2Start := epoch1End
	epoch2Expire := epoch2Start.Add(time.Minute)
	epoch2End := epoch1Expire.Add(time.Second)
	epoch3Start := epoch2End
	epoch3Expire := epoch3Start.Add(time.Minute)

	// Insert one epoch that gets updated in the same block
	epoch1 := addTestEpoch(t, es, 1, epoch1Start, epoch1Expire, nil, block1)
	epoch1b := addTestEpoch(t, es, 1, epoch1Start, epoch1Expire, &epoch1End, block1)

	// And another which is updated in a subsequent block
	epoch2 := addTestEpoch(t, es, 2, epoch2Start, epoch2Expire, nil, block1)
	epoch2b := addTestEpoch(t, es, 2, epoch2Start, epoch2Expire, &epoch2End, block2)

	// And finally one which isn't updated (e.g. hasn't ended yet)
	epoch3 := addTestEpoch(t, es, 3, epoch3Start, epoch3Expire, nil, block3)

	_ = epoch1
	_ = epoch2

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Epoch{epoch1b, epoch2b, epoch3}
		actual, err := es.GetAll(ctx)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("GetCurrent", func(t *testing.T) {
		actual, err := es.GetCurrent(ctx)
		require.NoError(t, err)
		assert.Equal(t, epoch3, actual)
	})

	t.Run("Get", func(t *testing.T) {
		actual, err := es.Get(ctx, 2)
		require.NoError(t, err)
		assert.Equal(t, epoch2b, actual)
	})
}
