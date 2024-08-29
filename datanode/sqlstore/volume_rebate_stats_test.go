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
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

func TestVolumeRebateStats_AddVolumeRebateStats(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	vds := sqlstore.NewVolumeRebateStats(connectionSource)

	t.Run("Should add stats for an epoch if it does not exist", func(t *testing.T) {
		epoch := uint64(1)
		block := addTestBlock(t, ctx, bs)

		stats := entities.VolumeRebateStats{
			AtEpoch:                  epoch,
			PartiesVolumeRebateStats: setupPartyVolumeRebateStats(t, ctx, ps, bs),
			VegaTime:                 block.VegaTime,
		}

		require.NoError(t, vds.Add(ctx, &stats))

		var got entities.VolumeRebateStats
		require.NoError(t, pgxscan.Get(ctx, connectionSource, &got, "SELECT * FROM volume_rebate_stats WHERE at_epoch = $1", epoch))
		assert.Equal(t, stats, got)
	})

	t.Run("Should return an error if the stats for an epoch already exists", func(t *testing.T) {
		epoch := uint64(2)
		block := addTestBlock(t, ctx, bs)
		stats := entities.VolumeRebateStats{
			AtEpoch:                  epoch,
			PartiesVolumeRebateStats: setupPartyVolumeRebateStats(t, ctx, ps, bs),
			VegaTime:                 block.VegaTime,
		}

		require.NoError(t, vds.Add(ctx, &stats))

		var got entities.VolumeRebateStats
		require.NoError(t, pgxscan.Get(ctx, connectionSource, &got, "SELECT * FROM volume_rebate_stats WHERE at_epoch = $1", epoch))
		assert.Equal(t, stats, got)

		err := vds.Add(ctx, &stats)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")
	})
}

func TestVolumeRebateStats_GetVolumeRebateStats(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	vds := sqlstore.NewVolumeRebateStats(connectionSource)

	parties := make([]entities.Party, 0, 6)
	for i := 0; i < 6; i++ {
		block := addTestBlockForTime(t, ctx, bs, time.Now().Add(time.Duration(i-10)*time.Minute))
		parties = append(parties, addTestParty(t, ctx, ps, block))
	}

	flattenStats := make([]entities.FlattenVolumeRebateStats, 0, 5*len(parties))
	lastEpoch := uint64(0)

	for i := 0; i < 5; i++ {
		block := addTestBlock(t, ctx, bs)
		lastEpoch = uint64(i + 1)

		stats := entities.VolumeRebateStats{
			AtEpoch: lastEpoch,
			PartiesVolumeRebateStats: setupPartyVolumeRebateStatsMod(t, parties, func(j int, party entities.Party) *eventspb.PartyVolumeRebateStats {
				return &eventspb.PartyVolumeRebateStats{
					PartyId:             party.ID.String(),
					AdditionalRebate:    fmt.Sprintf("0.%d%d", i+1, j+1),
					MakerVolumeFraction: strconv.Itoa((i+1)*100 + (j+1)*10),
					MakerFeesReceived:   "1000",
				}
			}),
			VegaTime: block.VegaTime,
		}

		require.NoError(t, vds.Add(ctx, &stats))

		for _, stat := range stats.PartiesVolumeRebateStats {
			flattenStats = append(flattenStats, entities.FlattenVolumeRebateStats{
				AtEpoch:             lastEpoch,
				VegaTime:            block.VegaTime,
				PartyID:             stat.PartyId,
				AdditionalRebate:    stat.AdditionalRebate,
				MakerVolumeFraction: stat.MakerVolumeFraction,
				MakerFeesReceived:   "1000",
			})
		}
	}

	t.Run("Should return the stats for the most recent epoch if no epoch is provided", func(t *testing.T) {
		lastStats := flattenVolumeRebateStatsForEpoch(flattenStats, lastEpoch)
		got, _, err := vds.Stats(ctx, nil, nil, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, lastStats, got)
	})

	t.Run("Should return the stats for the specified epoch if an epoch is provided", func(t *testing.T) {
		epoch := flattenStats[rand.Intn(len(flattenStats))].AtEpoch
		statsAtEpoch := flattenVolumeRebateStatsForEpoch(flattenStats, epoch)
		got, _, err := vds.Stats(ctx, &epoch, nil, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, statsAtEpoch, got)
	})

	t.Run("Should return the stats for the specified party for epoch", func(t *testing.T) {
		partyID := flattenStats[rand.Intn(len(flattenStats))].PartyID
		statsAtEpoch := flattenVolumeRebateStatsForParty(flattenStats, partyID)
		got, _, err := vds.Stats(ctx, nil, &partyID, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, statsAtEpoch, got)
	})

	t.Run("Should return the stats for the specified party and epoch", func(t *testing.T) {
		randomStats := flattenStats[rand.Intn(len(flattenStats))]
		partyID := randomStats.PartyID
		atEpoch := randomStats.AtEpoch
		statsAtEpoch := flattenVolumeRebateStatsForParty(flattenVolumeRebateStatsForEpoch(flattenStats, atEpoch), partyID)
		got, _, err := vds.Stats(ctx, &atEpoch, &partyID, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, statsAtEpoch, got)
	})
	t.Run("Pagination for latest epoch", func(t *testing.T) {
		lastStats := flattenVolumeRebateStatsForEpoch(flattenStats, lastEpoch)

		first := int32(2)
		after := lastStats[2].Cursor().Encode()
		cursor, _ := entities.NewCursorPagination(&first, &after, nil, nil, false)

		want := lastStats[3:5]
		got, _, err := vds.Stats(ctx, nil, nil, cursor)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, want, got)
	})
	t.Run("Pagination for latest epoch with party ID", func(t *testing.T) {
		partyID := flattenStats[0].PartyID
		lastStats := flattenVolumeRebateStatsForParty(flattenStats, partyID)

		first := int32(2)
		after := lastStats[2].Cursor().Encode()
		cursor, _ := entities.NewCursorPagination(&first, &after, nil, nil, false)

		want := lastStats[3:5]
		got, _, err := vds.Stats(ctx, nil, &partyID, cursor)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, want, got)
	})
}

func flattenVolumeRebateStatsForEpoch(flattenStats []entities.FlattenVolumeRebateStats, epoch uint64) []entities.FlattenVolumeRebateStats {
	lastStats := []entities.FlattenVolumeRebateStats{}

	for _, stat := range flattenStats {
		if stat.AtEpoch == epoch {
			lastStats = append(lastStats, stat)
		}
	}

	slices.SortStableFunc(lastStats, func(a, b entities.FlattenVolumeRebateStats) int {
		if a.AtEpoch == b.AtEpoch {
			return strings.Compare(a.PartyID, b.PartyID)
		}

		return compareUint64(a.AtEpoch, b.AtEpoch)
	})

	return lastStats
}

func flattenVolumeRebateStatsForParty(flattenStats []entities.FlattenVolumeRebateStats, party string) []entities.FlattenVolumeRebateStats {
	lastStats := []entities.FlattenVolumeRebateStats{}

	for _, stat := range flattenStats {
		if stat.PartyID == party {
			lastStats = append(lastStats, stat)
		}
	}

	slices.SortStableFunc(lastStats, func(a, b entities.FlattenVolumeRebateStats) int {
		if a.AtEpoch == b.AtEpoch {
			return strings.Compare(a.PartyID, b.PartyID)
		}

		return -compareUint64(a.AtEpoch, b.AtEpoch)
	})

	return lastStats
}

func setupPartyVolumeRebateStats(t *testing.T, ctx context.Context, ps *sqlstore.Parties, bs *sqlstore.Blocks) []*eventspb.PartyVolumeRebateStats {
	t.Helper()

	parties := make([]entities.Party, 0, 6)
	for i := 0; i < 6; i++ {
		block := addTestBlockForTime(t, ctx, bs, time.Now().Add(time.Duration(i-10)*time.Minute))
		parties = append(parties, addTestParty(t, ctx, ps, block))
	}

	return setupPartyVolumeRebateStatsMod(t, parties, func(i int, party entities.Party) *eventspb.PartyVolumeRebateStats {
		return &eventspb.PartyVolumeRebateStats{
			PartyId:             party.ID.String(),
			AdditionalRebate:    fmt.Sprintf("0.%d", i+1),
			MakerVolumeFraction: strconv.Itoa((i + 1) * 100),
		}
	})
}

func setupPartyVolumeRebateStatsMod(t *testing.T, parties []entities.Party, f func(i int, party entities.Party) *eventspb.PartyVolumeRebateStats) []*eventspb.PartyVolumeRebateStats {
	t.Helper()

	partiesStats := make([]*eventspb.PartyVolumeRebateStats, 0, 6)
	for i, p := range parties {
		// make the last party an unqualified party
		if i == len(parties)-1 {
			partiesStats = append(partiesStats, &eventspb.PartyVolumeRebateStats{
				PartyId:             p.ID.String(),
				AdditionalRebate:    "0.1",
				MakerVolumeFraction: "99",
				MakerFeesReceived:   "1000",
			})
			continue
		}
		partiesStats = append(partiesStats, f(i, p))
	}

	return partiesStats
}
