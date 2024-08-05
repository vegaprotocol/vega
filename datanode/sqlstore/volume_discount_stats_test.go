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
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

func TestVolumeDiscountStats_AddVolumeDiscountStats(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	vds := sqlstore.NewVolumeDiscountStats(connectionSource)

	t.Run("Should add stats for an epoch if it does not exist", func(t *testing.T) {
		epoch := uint64(1)
		block := addTestBlock(t, ctx, bs)

		stats := entities.VolumeDiscountStats{
			AtEpoch:                    epoch,
			PartiesVolumeDiscountStats: setupPartyVolumeDiscountStats(t, ctx, ps, bs),
			VegaTime:                   block.VegaTime,
		}

		require.NoError(t, vds.Add(ctx, &stats))

		var got entities.VolumeDiscountStats
		require.NoError(t, pgxscan.Get(ctx, connectionSource, &got, "SELECT * FROM volume_discount_stats WHERE at_epoch = $1", epoch))
		assert.Equal(t, stats, got)
	})

	t.Run("Should return an error if the stats for an epoch already exists", func(t *testing.T) {
		epoch := uint64(2)
		block := addTestBlock(t, ctx, bs)
		stats := entities.VolumeDiscountStats{
			AtEpoch:                    epoch,
			PartiesVolumeDiscountStats: setupPartyVolumeDiscountStats(t, ctx, ps, bs),
			VegaTime:                   block.VegaTime,
		}

		require.NoError(t, vds.Add(ctx, &stats))

		var got entities.VolumeDiscountStats
		require.NoError(t, pgxscan.Get(ctx, connectionSource, &got, "SELECT * FROM volume_discount_stats WHERE at_epoch = $1", epoch))
		assert.Equal(t, stats, got)

		err := vds.Add(ctx, &stats)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")
	})
}

func TestVolumeDiscountStats_GetVolumeDiscountStats(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	vds := sqlstore.NewVolumeDiscountStats(connectionSource)

	parties := make([]entities.Party, 0, 6)
	for i := 0; i < 6; i++ {
		block := addTestBlockForTime(t, ctx, bs, time.Now().Add(time.Duration(i-10)*time.Minute))
		parties = append(parties, addTestParty(t, ctx, ps, block))
	}

	flattenStats := make([]entities.FlattenVolumeDiscountStats, 0, 5*len(parties))
	lastEpoch := uint64(0)

	for i := 0; i < 5; i++ {
		block := addTestBlock(t, ctx, bs)
		lastEpoch = uint64(i + 1)

		stats := entities.VolumeDiscountStats{
			AtEpoch: lastEpoch,
			PartiesVolumeDiscountStats: setupPartyVolumeDiscountStatsMod(t, parties, func(j int, party entities.Party) *eventspb.PartyVolumeDiscountStats {
				return &eventspb.PartyVolumeDiscountStats{
					PartyId: party.ID.String(),
					DiscountFactors: &vega.DiscountFactors{
						InfrastructureDiscountFactor: fmt.Sprintf("0.%d%d", i+1, j+1),
						MakerDiscountFactor:          fmt.Sprintf("0.%d%d", i+1, j+1),
						LiquidityDiscountFactor:      fmt.Sprintf("0.%d%d", i+1, j+1),
					},
					RunningVolume: strconv.Itoa((i+1)*100 + (j+1)*10),
				}
			}),
			VegaTime: block.VegaTime,
		}

		require.NoError(t, vds.Add(ctx, &stats))

		for _, stat := range stats.PartiesVolumeDiscountStats {
			flattenStats = append(flattenStats, entities.FlattenVolumeDiscountStats{
				AtEpoch:         lastEpoch,
				VegaTime:        block.VegaTime,
				PartyID:         stat.PartyId,
				DiscountFactors: stat.DiscountFactors,
				RunningVolume:   stat.RunningVolume,
			})
		}
	}

	t.Run("Should return the stats for the most recent epoch if no epoch is provided", func(t *testing.T) {
		lastStats := flattenVolumeDiscountStatsForEpoch(flattenStats, lastEpoch)
		got, _, err := vds.Stats(ctx, nil, nil, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, lastStats, got)
	})

	t.Run("Should return the stats for the specified epoch if an epoch is provided", func(t *testing.T) {
		epoch := flattenStats[rand.Intn(len(flattenStats))].AtEpoch
		statsAtEpoch := flattenVolumeDiscountStatsForEpoch(flattenStats, epoch)
		got, _, err := vds.Stats(ctx, &epoch, nil, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, statsAtEpoch, got)
	})

	t.Run("Should return the stats for the specified party for epoch", func(t *testing.T) {
		partyID := flattenStats[rand.Intn(len(flattenStats))].PartyID
		statsAtEpoch := flattenVolumeDiscountStatsForParty(flattenStats, partyID)
		got, _, err := vds.Stats(ctx, nil, &partyID, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, statsAtEpoch, got)
	})

	t.Run("Should return the stats for the specified party and epoch", func(t *testing.T) {
		randomStats := flattenStats[rand.Intn(len(flattenStats))]
		partyID := randomStats.PartyID
		atEpoch := randomStats.AtEpoch
		statsAtEpoch := flattenVolumeDiscountStatsForParty(flattenVolumeDiscountStatsForEpoch(flattenStats, atEpoch), partyID)
		got, _, err := vds.Stats(ctx, &atEpoch, &partyID, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, statsAtEpoch, got)
	})
	t.Run("Pagination for latest epoch", func(t *testing.T) {
		lastStats := flattenVolumeDiscountStatsForEpoch(flattenStats, lastEpoch)

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
		lastStats := flattenVolumeDiscountStatsForParty(flattenStats, partyID)

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

func flattenVolumeDiscountStatsForEpoch(flattenStats []entities.FlattenVolumeDiscountStats, epoch uint64) []entities.FlattenVolumeDiscountStats {
	lastStats := []entities.FlattenVolumeDiscountStats{}

	for _, stat := range flattenStats {
		if stat.AtEpoch == epoch {
			lastStats = append(lastStats, stat)
		}
	}

	slices.SortStableFunc(lastStats, func(a, b entities.FlattenVolumeDiscountStats) int {
		if a.AtEpoch == b.AtEpoch {
			return strings.Compare(a.PartyID, b.PartyID)
		}

		return compareUint64(a.AtEpoch, b.AtEpoch)
	})

	return lastStats
}

func flattenVolumeDiscountStatsForParty(flattenStats []entities.FlattenVolumeDiscountStats, party string) []entities.FlattenVolumeDiscountStats {
	lastStats := []entities.FlattenVolumeDiscountStats{}

	for _, stat := range flattenStats {
		if stat.PartyID == party {
			lastStats = append(lastStats, stat)
		}
	}

	slices.SortStableFunc(lastStats, func(a, b entities.FlattenVolumeDiscountStats) int {
		if a.AtEpoch == b.AtEpoch {
			return strings.Compare(a.PartyID, b.PartyID)
		}

		return -compareUint64(a.AtEpoch, b.AtEpoch)
	})

	return lastStats
}

func setupPartyVolumeDiscountStats(t *testing.T, ctx context.Context, ps *sqlstore.Parties, bs *sqlstore.Blocks) []*eventspb.PartyVolumeDiscountStats {
	t.Helper()

	parties := make([]entities.Party, 0, 6)
	for i := 0; i < 6; i++ {
		block := addTestBlockForTime(t, ctx, bs, time.Now().Add(time.Duration(i-10)*time.Minute))
		parties = append(parties, addTestParty(t, ctx, ps, block))
	}

	return setupPartyVolumeDiscountStatsMod(t, parties, func(i int, party entities.Party) *eventspb.PartyVolumeDiscountStats {
		return &eventspb.PartyVolumeDiscountStats{
			PartyId: party.ID.String(),
			DiscountFactors: &vega.DiscountFactors{
				MakerDiscountFactor:          fmt.Sprintf("0.%d", i+1),
				LiquidityDiscountFactor:      fmt.Sprintf("0.%d", i+1),
				InfrastructureDiscountFactor: fmt.Sprintf("0.%d", i+1),
			},
			RunningVolume: strconv.Itoa((i + 1) * 100),
		}
	})
}

func setupPartyVolumeDiscountStatsMod(t *testing.T, parties []entities.Party, f func(i int, party entities.Party) *eventspb.PartyVolumeDiscountStats) []*eventspb.PartyVolumeDiscountStats {
	t.Helper()

	partiesStats := make([]*eventspb.PartyVolumeDiscountStats, 0, 6)
	for i, p := range parties {
		// make the last party an unqualified party
		if i == len(parties)-1 {
			partiesStats = append(partiesStats, &eventspb.PartyVolumeDiscountStats{
				PartyId: p.ID.String(),
				DiscountFactors: &vega.DiscountFactors{
					InfrastructureDiscountFactor: "0",
					LiquidityDiscountFactor:      "0",
					MakerDiscountFactor:          "0",
				},
				RunningVolume: "99",
			})
			continue
		}
		partiesStats = append(partiesStats, f(i, p))
	}

	return partiesStats
}
