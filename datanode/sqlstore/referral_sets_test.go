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
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

func setupReferralSetsTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.Parties, *sqlstore.ReferralSets) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	rs := sqlstore.NewReferralSets(connectionSource)

	return bs, ps, rs
}

func TestReferralSets_AddReferralSet(t *testing.T) {
	bs, ps, rs := setupReferralSetsTest(t)
	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)
	referrer := addTestParty(t, ctx, ps, block)

	set := entities.ReferralSet{
		ID:        entities.ReferralSetID(GenerateID()),
		Referrer:  referrer.ID,
		CreatedAt: block.VegaTime,
		UpdatedAt: block.VegaTime,
		VegaTime:  block.VegaTime,
	}

	t.Run("Should add the referral set if it does not already exist", func(t *testing.T) {
		err := rs.AddReferralSet(ctx, &set)
		require.NoError(t, err)

		var got entities.ReferralSet
		err = pgxscan.Get(ctx, connectionSource, &got, "SELECT * FROM referral_sets WHERE id = $1", set.ID)
		require.NoError(t, err)
		assert.Equal(t, set, got)
	})

	t.Run("Should error if referral set already exists", func(t *testing.T) {
		err := rs.AddReferralSet(ctx, &set)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")
	})
}

func TestReferralSets_RefereeJoinedReferralSet(t *testing.T) {
	bs, ps, rs := setupReferralSetsTest(t)
	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)
	referrer := addTestParty(t, ctx, ps, block)
	referee := addTestParty(t, ctx, ps, block)

	set := entities.ReferralSet{
		ID:        entities.ReferralSetID(GenerateID()),
		Referrer:  referrer.ID,
		CreatedAt: block.VegaTime,
		UpdatedAt: block.VegaTime,
		VegaTime:  block.VegaTime,
	}

	block2 := addTestBlock(t, ctx, bs)
	setReferee := entities.ReferralSetReferee{
		ReferralSetID: set.ID,
		Referee:       referee.ID,
		JoinedAt:      block2.VegaTime,
		AtEpoch:       uint64(block2.Height),
		VegaTime:      block2.VegaTime,
	}

	err := rs.AddReferralSet(ctx, &set)
	require.NoError(t, err)

	t.Run("Should add a new referral set referee if it does not already exist", func(t *testing.T) {
		err = rs.RefereeJoinedReferralSet(ctx, &setReferee)
		require.NoError(t, err)

		var got entities.ReferralSetReferee
		err = pgxscan.Get(ctx, connectionSource, &got, "SELECT * FROM referral_set_referees WHERE referral_set_id = $1 AND referee = $2", set.ID, referee.ID)
		require.NoError(t, err)
		assert.Equal(t, setReferee, got)
	})

	t.Run("Should error if referral set referee already exists", func(t *testing.T) {
		err = rs.RefereeJoinedReferralSet(ctx, &setReferee)
		require.Error(t, err)
	})
}

func setupReferralSetsAndReferees(t *testing.T, ctx context.Context, bs *sqlstore.Blocks, ps *sqlstore.Parties, rs *sqlstore.ReferralSets, createStats bool) (
	[]entities.ReferralSet, map[string][]entities.ReferralSetRefereeStats,
) {
	t.Helper()

	sets := make([]entities.ReferralSet, 0)
	referees := make(map[string][]entities.ReferralSetRefereeStats, 0)
	es := sqlstore.NewEpochs(connectionSource)
	fs := sqlstore.NewFeesStats(connectionSource)

	for i := 0; i < 10; i++ {
		block := addTestBlockForTime(t, ctx, bs, time.Now().Add(time.Duration(i-10)*time.Minute))
		endTime := block.VegaTime.Add(time.Minute)
		addTestEpoch(t, ctx, es, int64(i), block.VegaTime, endTime, &endTime, block)
		referrer := addTestParty(t, ctx, ps, block)
		set := entities.ReferralSet{
			ID:           entities.ReferralSetID(GenerateID()),
			Referrer:     referrer.ID,
			TotalMembers: 1,
			CreatedAt:    block.VegaTime,
			UpdatedAt:    block.VegaTime,
			VegaTime:     block.VegaTime,
		}
		err := rs.AddReferralSet(ctx, &set)
		require.NoError(t, err)

		setID := set.ID.String()
		referees[setID] = make([]entities.ReferralSetRefereeStats, 0)

		for j := 0; j < 10; j++ {
			block = addTestBlockForTime(t, ctx, bs, block.VegaTime.Add(5*time.Second))
			referee := addTestParty(t, ctx, ps, block)
			setReferee := entities.ReferralSetRefereeStats{
				ReferralSetReferee: entities.ReferralSetReferee{
					ReferralSetID: set.ID,
					Referee:       referee.ID,
					JoinedAt:      block.VegaTime,
					AtEpoch:       uint64(block.Height),
					VegaTime:      block.VegaTime,
				},
				PeriodVolume:      num.DecimalFromInt64(10),
				PeriodRewardsPaid: num.DecimalFromInt64(10),
			}

			err := rs.RefereeJoinedReferralSet(ctx, &setReferee.ReferralSetReferee)
			require.NoError(t, err)

			set.TotalMembers += 1

			referees[setID] = append(referees[setID], setReferee)
			if createStats {
				// Add some stats for the referral sets
				stats := entities.ReferralSetStats{
					SetID:                                 set.ID,
					AtEpoch:                               uint64(block.Height),
					WasEligible:                           true,
					ReferralSetRunningNotionalTakerVolume: "10",
					ReferrerTakerVolume:                   "10",
					RefereesStats: []*eventspb.RefereeStats{
						{
							PartyId:                  referee.ID.String(),
							DiscountFactor:           "10",
							EpochNotionalTakerVolume: "10",
						},
					},
					VegaTime: block.VegaTime,
					RewardFactors: &vegapb.RewardFactors{
						InfrastructureRewardFactor: "-1",
						LiquidityRewardFactor:      "-1",
						MakerRewardFactor:          "-1",
					},
					RewardsMultiplier: "1",
					RewardsFactorsMultiplier: &vegapb.RewardFactors{
						InfrastructureRewardFactor: "-1",
						LiquidityRewardFactor:      "-1",
						MakerRewardFactor:          "-1",
					},
				}
				require.NoError(t, rs.AddReferralSetStats(ctx, &stats))
				feeStats := entities.FeesStats{
					MarketID: "deadbeef01",
					AssetID:  "cafed00d01",
					EpochSeq: uint64(block.Height),
					TotalRewardsReceived: []*eventspb.PartyAmount{
						{
							Party:         referee.ID.String(),
							Amount:        "10",
							QuantumAmount: "10",
						},
					},
					ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{
						{
							Referrer: "deadd00d01",
							GeneratedReward: []*eventspb.PartyAmount{
								{
									Party:         referee.ID.String(),
									Amount:        "10",
									QuantumAmount: "10",
								},
							},
						},
					},
					VegaTime: block.VegaTime,
				}
				require.NoError(t, fs.AddFeesStats(ctx, &feeStats))
			}
		}

		sets = append(sets, set)
	}

	sort.Slice(sets, func(i, j int) bool {
		return sets[i].CreatedAt.After(sets[j].CreatedAt)
	})

	for _, refs := range referees {
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].JoinedAt.Equal(refs[j].JoinedAt) {
				return refs[i].Referee < refs[j].Referee
			}
			return refs[i].JoinedAt.After(refs[j].JoinedAt)
		})
	}

	return sets, referees
}

func TestReferralSets_ListReferralSets(t *testing.T) {
	bs, ps, rs := setupReferralSetsTest(t)
	ctx := tempTransaction(t)

	sets, referees := setupReferralSetsAndReferees(t, ctx, bs, ps, rs, true)

	t.Run("Should return all referral sets", func(t *testing.T) {
		got, pageInfo, err := rs.ListReferralSets(ctx, nil, nil, nil, entities.DefaultCursorPagination(true))
		require.NoError(t, err)
		want := sets[:]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested referral set", func(t *testing.T) {
		src := rand.New(rand.NewSource(time.Now().UnixNano()))
		r := rand.New(src)

		want := sets[r.Intn(len(sets))]
		got, pageInfo, err := rs.ListReferralSets(ctx, &want.ID, nil, nil, entities.CursorPagination{})
		require.NoError(t, err)
		assert.Equal(t, want, got[0])
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want.Cursor().Encode(),
			EndCursor:       want.Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested referral set by referrer", func(t *testing.T) {
		src := rand.New(rand.NewSource(time.Now().UnixNano()))
		r := rand.New(src)

		want := sets[r.Intn(len(sets))]
		got, pageInfo, err := rs.ListReferralSets(ctx, nil, &want.Referrer, nil, entities.CursorPagination{})
		require.NoError(t, err)
		assert.Equal(t, want, got[0])
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want.Cursor().Encode(),
			EndCursor:       want.Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested referral set by referee", func(t *testing.T) {
		src := rand.New(rand.NewSource(time.Now().UnixNano()))
		r := rand.New(src)

		want := sets[r.Intn(len(sets))]
		refs := referees[want.ID.String()]
		wantReferee := refs[r.Intn(len(refs))]

		got, pageInfo, err := rs.ListReferralSets(ctx, nil, nil, &wantReferee.Referee, entities.CursorPagination{})
		require.NoError(t, err)
		assert.Equal(t, want, got[0])
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want.Cursor().Encode(),
			EndCursor:       want.Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return first N referral sets if first cursor is set", func(t *testing.T) {
		first := int32(3)
		cursor, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
		require.NoError(t, err)

		got, pageInfo, err := rs.ListReferralSets(ctx, nil, nil, nil, cursor)
		require.NoError(t, err)
		want := sets[:first]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return last N referral sets if last cursor is set", func(t *testing.T) {
		last := int32(3)
		cursor, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
		require.NoError(t, err)

		got, pageInfo, err := rs.ListReferralSets(ctx, nil, nil, nil, cursor)
		require.NoError(t, err)
		want := sets[len(sets)-int(last):]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested page if first and after cursor are set", func(t *testing.T) {
		first := int32(3)
		after := sets[2].Cursor().Encode()
		cursor, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)

		got, pageInfo, err := rs.ListReferralSets(ctx, nil, nil, nil, cursor)
		require.NoError(t, err)
		want := sets[3:6]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested page if last and before cursor are set", func(t *testing.T) {
		last := int32(3)
		before := sets[7].Cursor().Encode()
		cursor, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
		require.NoError(t, err)

		got, pageInfo, err := rs.ListReferralSets(ctx, nil, nil, nil, cursor)
		require.NoError(t, err)
		want := sets[4:7]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})
}

func TestReferralSets_ListReferralSetReferees(t *testing.T) {
	bs, ps, rs := setupReferralSetsTest(t)
	ctx := tempTransaction(t)

	sets, referees := setupReferralSetsAndReferees(t, ctx, bs, ps, rs, true)
	src := rand.New(rand.NewSource(time.Now().UnixNano()))
	r := rand.New(src)
	set := sets[r.Intn(len(sets))]
	setID := set.ID.String()
	refs := referees[setID]

	t.Run("Should return all referees in a set if no pagination", func(t *testing.T) {
		want := refs[:]
		got, pageInfo, err := rs.ListReferralSetReferees(ctx, &set.ID, nil, nil, entities.DefaultCursorPagination(true), 30)
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return all referees in a set by referrer if no pagination", func(t *testing.T) {
		want := refs[:]
		got, pageInfo, err := rs.ListReferralSetReferees(ctx, nil, &set.Referrer, nil, entities.DefaultCursorPagination(true), 30)
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return referee in a set", func(t *testing.T) {
		want := []entities.ReferralSetRefereeStats{refs[r.Intn(len(refs))]}

		got, pageInfo, err := rs.ListReferralSetReferees(ctx, nil, nil, &want[0].Referee, entities.DefaultCursorPagination(true), 30)
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return first N referees in a set if first cursor is set", func(t *testing.T) {
		first := int32(3)
		cursor, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
		require.NoError(t, err)

		got, pageInfo, err := rs.ListReferralSetReferees(ctx, &set.ID, nil, nil, cursor, 30)
		require.NoError(t, err)
		want := refs[:first]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return last N referees in a set if last cursor is set", func(t *testing.T) {
		last := int32(3)
		cursor, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
		require.NoError(t, err)

		got, pageInfo, err := rs.ListReferralSetReferees(ctx, &set.ID, nil, nil, cursor, 30)
		require.NoError(t, err)
		want := refs[len(refs)-int(last):]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested page if set id and first and after cursor are set", func(t *testing.T) {
		first := int32(3)
		after := refs[2].Cursor().Encode()
		cursor, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)

		got, pageInfo, err := rs.ListReferralSetReferees(ctx, &set.ID, nil, nil, cursor, 30)
		require.NoError(t, err)
		want := refs[3:6]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested page if first and after cursor are set", func(t *testing.T) {
		first := int32(3)
		after := refs[2].Cursor().Encode()
		cursor, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)

		got, pageInfo, err := rs.ListReferralSetReferees(ctx, nil, nil, nil, cursor, 30)
		require.NoError(t, err)
		want := refs[3:6]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Should return the requested page if  last and before cursor are set", func(t *testing.T) {
		last := int32(3)
		before := refs[7].Cursor().Encode()
		cursor, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
		require.NoError(t, err)

		got, pageInfo, err := rs.ListReferralSetReferees(ctx, &set.ID, nil, nil, cursor, 30)
		require.NoError(t, err)
		want := refs[4:7]
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     want[0].Cursor().Encode(),
			EndCursor:       want[len(want)-1].Cursor().Encode(),
		}, pageInfo)
	})
}

func TestReferralSets_AddReferralSetStats(t *testing.T) {
	bs, ps, rs := setupReferralSetsTest(t)

	ctx := tempTransaction(t)

	sets, referees := setupReferralSetsAndReferees(t, ctx, bs, ps, rs, false)
	src := rand.New(rand.NewSource(time.Now().UnixNano()))
	r := rand.New(src)
	set := sets[r.Intn(len(sets))]
	setID := set.ID.String()
	refs := referees[setID]

	takerVolume := "100000"

	t.Run("Should add stats for an epoch if it does not exist", func(t *testing.T) {
		epoch := uint64(1)
		block := addTestBlock(t, ctx, bs)
		stats := entities.ReferralSetStats{
			SetID:                                 set.ID,
			AtEpoch:                               epoch,
			ReferralSetRunningNotionalTakerVolume: takerVolume,
			ReferrerTakerVolume:                   "100",
			RefereesStats:                         getRefereeStats(t, refs, "0.01"),
			VegaTime:                              block.VegaTime,
			RewardFactors: &vegapb.RewardFactors{
				InfrastructureRewardFactor: "0.02",
				LiquidityRewardFactor:      "0.02",
				MakerRewardFactor:          "0.02",
			},
			RewardsMultiplier: "0.03",
			RewardsFactorsMultiplier: &vegapb.RewardFactors{
				InfrastructureRewardFactor: "0.04",
				LiquidityRewardFactor:      "0.04",
				MakerRewardFactor:          "0.04",
			},
		}

		err := rs.AddReferralSetStats(ctx, &stats)
		require.NoError(t, err)

		var got entities.ReferralSetStats
		err = pgxscan.Get(ctx, connectionSource, &got, "SELECT * FROM referral_set_stats WHERE set_id = $1 AND at_epoch = $2", set.ID, epoch)
		require.NoError(t, err)
		assert.Equal(t, stats, got)
	})

	t.Run("Should return an error if the stats for an epoch already exists", func(t *testing.T) {
		epoch := uint64(2)
		block := addTestBlock(t, ctx, bs)
		stats := entities.ReferralSetStats{
			SetID:                                 set.ID,
			AtEpoch:                               epoch,
			ReferralSetRunningNotionalTakerVolume: takerVolume,
			ReferrerTakerVolume:                   "100",
			RefereesStats:                         getRefereeStats(t, refs, "0.01"),
			VegaTime:                              block.VegaTime,
			RewardFactors: &vegapb.RewardFactors{
				InfrastructureRewardFactor: "0.02",
				LiquidityRewardFactor:      "0.02",
				MakerRewardFactor:          "0.02",
			},
			RewardsMultiplier: "0.03",
			RewardsFactorsMultiplier: &vegapb.RewardFactors{
				InfrastructureRewardFactor: "0.04",
				LiquidityRewardFactor:      "0.04",
				MakerRewardFactor:          "0.04",
			},
		}

		err := rs.AddReferralSetStats(ctx, &stats)
		require.NoError(t, err)
		var got entities.ReferralSetStats
		err = pgxscan.Get(ctx, connectionSource, &got, "SELECT * FROM referral_set_stats WHERE set_id = $1 AND at_epoch = $2", set.ID, epoch)
		require.NoError(t, err)
		assert.Equal(t, stats, got)

		err = rs.AddReferralSetStats(ctx, &stats)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")
	})
}

func getRefereeStats(t *testing.T, refs []entities.ReferralSetRefereeStats, discountFactor string) []*eventspb.RefereeStats {
	t.Helper()
	stats := make([]*eventspb.RefereeStats, len(refs))
	for i, r := range refs {
		stats[i] = &eventspb.RefereeStats{
			PartyId: r.Referee.String(),
			DiscountFactors: &vega.DiscountFactors{
				InfrastructureDiscountFactor: discountFactor,
				LiquidityDiscountFactor:      discountFactor,
				MakerDiscountFactor:          discountFactor,
			},
		}
	}
	return stats
}

func TestReferralSets_GetReferralSetStats(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	ps := sqlstore.NewParties(connectionSource)
	rs := sqlstore.NewReferralSets(connectionSource)

	parties := make([]entities.Party, 0, 5)
	for i := 0; i < 5; i++ {
		block := addTestBlockForTime(t, ctx, bs, time.Now().Add(time.Duration(i-10)*time.Minute))
		parties = append(parties, addTestParty(t, ctx, ps, block))
	}

	flattenStats := make([]entities.FlattenReferralSetStats, 0, 5*len(parties))
	lastEpoch := uint64(0)

	setID := entities.ReferralSetID(vgcrypto.RandomHash())

	for i := 0; i < 5; i++ {
		block := addTestBlock(t, ctx, bs)
		lastEpoch = uint64(i + 1)

		rf := fmt.Sprintf("0.2%d", i+1)
		rmf := fmt.Sprintf("0.4%d", i+1)

		set := entities.ReferralSetStats{
			SetID:                                 setID,
			AtEpoch:                               lastEpoch,
			ReferralSetRunningNotionalTakerVolume: fmt.Sprintf("%d000000", i+1),
			RefereesStats: setupPartyReferralSetStatsMod(t, parties, func(j int, party entities.Party) *eventspb.RefereeStats {
				return &eventspb.RefereeStats{
					PartyId: party.ID.String(),
					DiscountFactors: &vega.DiscountFactors{
						InfrastructureDiscountFactor: "0.1",
						LiquidityDiscountFactor:      "0.1",
						MakerDiscountFactor:          "0.1",
					},
					EpochNotionalTakerVolume: strconv.Itoa((i+1)*100 + (j+1)*10),
				}
			}),
			VegaTime: block.VegaTime,
			RewardFactors: &vegapb.RewardFactors{
				InfrastructureRewardFactor: rf,
				LiquidityRewardFactor:      rf,
				MakerRewardFactor:          rf,
			},
			RewardsMultiplier: fmt.Sprintf("0.3%d", i+1),
			RewardsFactorsMultiplier: &vegapb.RewardFactors{
				InfrastructureRewardFactor: rmf,
				LiquidityRewardFactor:      rmf,
				MakerRewardFactor:          rmf,
			},
		}

		require.NoError(t, rs.AddReferralSetStats(ctx, &set))

		for _, stat := range set.RefereesStats {
			flattenStats = append(flattenStats, entities.FlattenReferralSetStats{
				SetID:                                 setID,
				AtEpoch:                               lastEpoch,
				ReferralSetRunningNotionalTakerVolume: set.ReferralSetRunningNotionalTakerVolume,
				VegaTime:                              block.VegaTime,
				PartyID:                               stat.PartyId,
				DiscountFactors:                       stat.DiscountFactors,
				RewardFactors:                         set.RewardFactors,
				EpochNotionalTakerVolume:              stat.EpochNotionalTakerVolume,
				RewardsMultiplier:                     set.RewardsMultiplier,
				RewardsFactorsMultiplier:              set.RewardsFactorsMultiplier,
			})
		}
	}

	t.Run("Should return the most recent stats of the last epoch regardless the set and the party", func(t *testing.T) {
		lastStats := flattenReferralSetStatsForEpoch(flattenStats, lastEpoch)
		got, _, err := rs.GetReferralSetStats(ctx, nil, nil, nil, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, lastStats, got)
	})

	t.Run("Should return the stats for the most recent epoch if no epoch is provided", func(t *testing.T) {
		lastStats := flattenReferralSetStatsForEpoch(flattenStats, lastEpoch)
		got, _, err := rs.GetReferralSetStats(ctx, &setID, nil, nil, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, lastStats, got)
	})

	t.Run("Should return the stats for the specified epoch if an epoch is provided", func(t *testing.T) {
		epoch := flattenStats[rand.Intn(len(flattenStats))].AtEpoch
		statsAtEpoch := flattenReferralSetStatsForEpoch(flattenStats, epoch)
		got, _, err := rs.GetReferralSetStats(ctx, &setID, &epoch, nil, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, statsAtEpoch, got)
	})

	t.Run("Should return the stats for the specified party for epoch", func(t *testing.T) {
		partyIDStr := flattenStats[rand.Intn(len(flattenStats))].PartyID
		partyID := entities.PartyID(partyIDStr)
		statsAtEpoch := flattenReferralSetStatsForParty(flattenStats, partyIDStr)
		got, _, err := rs.GetReferralSetStats(ctx, &setID, nil, &partyID, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, statsAtEpoch, got)
	})

	t.Run("Should return the stats for the specified party for epoch with pagination", func(t *testing.T) {
		partyIDStr := flattenStats[rand.Intn(len(flattenStats))].PartyID
		partyID := entities.PartyID(partyIDStr)
		statsAtEpoch := flattenReferralSetStatsForParty(flattenStats, partyIDStr)

		first := int32(3)
		after := statsAtEpoch[1].Cursor().Encode()
		cursor, _ := entities.NewCursorPagination(&first, &after, nil, nil, false)

		got, _, err := rs.GetReferralSetStats(ctx, &setID, nil, &partyID, cursor)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, statsAtEpoch[2:5], got)
	})

	t.Run("Should return the stats for the specified party and epoch", func(t *testing.T) {
		randomStats := flattenStats[rand.Intn(len(flattenStats))]
		partyIDStr := randomStats.PartyID
		partyID := entities.PartyID(partyIDStr)
		atEpoch := randomStats.AtEpoch
		statsAtEpoch := flattenReferralSetStatsForParty(flattenReferralSetStatsForEpoch(flattenStats, atEpoch), partyIDStr)
		got, _, err := rs.GetReferralSetStats(ctx, &setID, &atEpoch, &partyID, entities.CursorPagination{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, statsAtEpoch, got)
	})
}

func flattenReferralSetStatsForEpoch(flattenStats []entities.FlattenReferralSetStats, epoch uint64) []entities.FlattenReferralSetStats {
	lastStats := []entities.FlattenReferralSetStats{}

	for _, stat := range flattenStats {
		if stat.AtEpoch == epoch {
			lastStats = append(lastStats, stat)
		}
	}

	slices.SortStableFunc(lastStats, func(a, b entities.FlattenReferralSetStats) int {
		if a.AtEpoch == b.AtEpoch {
			if a.SetID == b.SetID {
				return strings.Compare(a.PartyID, b.PartyID)
			}
			return strings.Compare(string(a.SetID), string(b.SetID))
		}
		return -compareUint64(a.AtEpoch, b.AtEpoch)
	})

	return lastStats
}

func compareUint64(a, b uint64) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

func flattenReferralSetStatsForParty(flattenStats []entities.FlattenReferralSetStats, party string) []entities.FlattenReferralSetStats {
	lastStats := []entities.FlattenReferralSetStats{}

	for _, stat := range flattenStats {
		if stat.PartyID == party {
			lastStats = append(lastStats, stat)
		}
	}

	slices.SortStableFunc(lastStats, func(a, b entities.FlattenReferralSetStats) int {
		if a.AtEpoch == b.AtEpoch {
			if a.SetID == b.SetID {
				return strings.Compare(a.PartyID, b.PartyID)
			}
			return strings.Compare(string(a.SetID), string(b.SetID))
		}

		return -compareUint64(a.AtEpoch, b.AtEpoch)
	})

	return lastStats
}

func setupPartyReferralSetStatsMod(t *testing.T, parties []entities.Party, f func(i int, party entities.Party) *eventspb.RefereeStats) []*eventspb.RefereeStats {
	t.Helper()

	partiesStats := make([]*eventspb.RefereeStats, 0, 5)
	for i, p := range parties {
		partiesStats = append(partiesStats, f(i, p))
	}

	return partiesStats
}
