// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
)

func addTestEpoch(t *testing.T, ctx context.Context, es *sqlstore.Epochs,
	epochID int64,
	startTime time.Time,
	expireTime time.Time,
	endTime *time.Time,
	block entities.Block,
) entities.Epoch {
	t.Helper()
	r := entities.Epoch{
		ID:         epochID,
		StartTime:  startTime,
		ExpireTime: expireTime,
		EndTime:    endTime,
		VegaTime:   block.VegaTime,
	}
	if endTime == nil {
		r.FirstBlock = &block.Height
	} else {
		r.LastBlock = &block.Height
	}
	err := es.Add(ctx, r)
	require.NoError(t, err)
	return r
}

func TestEpochs(t *testing.T) {
	ctx := tempTransaction(t)

	es := sqlstore.NewEpochs(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	epoch1Start := time.Date(2022, 1, 1, 0, 0, 0, 0, time.Local)
	epoch1Expire := epoch1Start.Add(time.Minute)
	epoch1End := epoch1Start.Add(time.Second)
	epoch2Start := epoch1End
	epoch2Expire := epoch2Start.Add(time.Minute)
	epoch2End := epoch2Start.Add(time.Second)
	epoch3Start := epoch2End
	epoch3Expire := epoch3Start.Add(time.Minute)

	block1 := addTestBlockForHeightAndTime(t, ctx, bs, 1, epoch1Start)
	block2 := addTestBlockForHeightAndTime(t, ctx, bs, 2, epoch2Start)
	block3 := addTestBlockForHeightAndTime(t, ctx, bs, 3, epoch3Start)

	// Insert one epoch that gets updated in the same block
	epoch1 := addTestEpoch(t, ctx, es, 1, epoch1Start, epoch1Expire, nil, block1)
	epoch1b := addTestEpoch(t, ctx, es, 1, epoch1Start, epoch1Expire, &epoch1End, block2)
	epoch1b.FirstBlock = epoch1.FirstBlock

	// And another which is updated in a subsequent block
	epoch2 := addTestEpoch(t, ctx, es, 2, epoch2Start, epoch2Expire, nil, block2)
	epoch2b := addTestEpoch(t, ctx, es, 2, epoch2Start, epoch2Expire, &epoch2End, block3)
	epoch2b.FirstBlock = epoch2.FirstBlock

	// And finally one which isn't updated (e.g. hasn't ended yet)
	epoch3 := addTestEpoch(t, ctx, es, 3, epoch3Start, epoch3Expire, nil, block3)

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

	t.Run("GetByBlock", func(t *testing.T) {
		actual, err := es.GetByBlock(ctx, uint64(block2.Height))
		require.NoError(t, err)
		assert.Equal(t, epoch2b, actual)
	})
}
