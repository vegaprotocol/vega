package sqlstore_test

import (
	"context"
	"math/rand"
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/datanode/sqlstore"
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
	ctx, rollback := tempTransaction(t)

	defer rollback()

	block := addTestBlock(t, ctx, bs)
	referrer := addTestParty(t, ctx, ps, block)

	set := entities.ReferralSet{
		ID:        entities.ReferralSetID(helpers.GenerateID()),
		Referrer:  referrer.ID,
		CreatedAt: block.VegaTime,
		UpdatedAt: block.VegaTime,
		VegaTime:  block.VegaTime,
	}

	t.Run("Should add a nre referral set if it does not already exist", func(t *testing.T) {
		err := rs.AddReferralSet(ctx, &set)
		require.NoError(t, err)

		var got entities.ReferralSet
		err = pgxscan.Get(ctx, connectionSource.Connection, &got, "SELECT * FROM referral_sets WHERE id = $1", set.ID)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	block := addTestBlock(t, ctx, bs)
	referrer := addTestParty(t, ctx, ps, block)
	referee := addTestParty(t, ctx, ps, block)

	set := entities.ReferralSet{
		ID:        entities.ReferralSetID(helpers.GenerateID()),
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
		err = pgxscan.Get(ctx, connectionSource.Connection, &got, "SELECT * FROM referral_set_referees WHERE referral_set_id = $1 AND referee = $2", set.ID, referee.ID)
		require.NoError(t, err)
		assert.Equal(t, setReferee, got)
	})

	t.Run("Should error if referral set referee already exists", func(t *testing.T) {
		err = rs.RefereeJoinedReferralSet(ctx, &setReferee)
		require.Error(t, err)
	})
}

func setupReferralSetsAndReferees(t *testing.T, ctx context.Context, bs *sqlstore.Blocks, ps *sqlstore.Parties, rs *sqlstore.ReferralSets) (
	[]entities.ReferralSet, map[string][]entities.ReferralSetReferee,
) {
	t.Helper()

	sets := make([]entities.ReferralSet, 0)
	referees := make(map[string][]entities.ReferralSetReferee, 0)

	for i := 0; i < 10; i++ {
		block := addTestBlockForTime(t, ctx, bs, time.Now().Add(time.Duration(i-10)*time.Minute))
		referrer := addTestParty(t, ctx, ps, block)
		set := entities.ReferralSet{
			ID:        entities.ReferralSetID(helpers.GenerateID()),
			Referrer:  referrer.ID,
			CreatedAt: block.VegaTime,
			UpdatedAt: block.VegaTime,
			VegaTime:  block.VegaTime,
		}
		err := rs.AddReferralSet(ctx, &set)
		require.NoError(t, err)
		sets = append(sets, set)

		setID := set.ID.String()
		referees[setID] = make([]entities.ReferralSetReferee, 0)

		for j := 0; j < 10; j++ {
			block = addTestBlockForTime(t, ctx, bs, block.VegaTime.Add(5*time.Second))
			referee := addTestParty(t, ctx, ps, block)
			setReferee := entities.ReferralSetReferee{
				ReferralSetID: set.ID,
				Referee:       referee.ID,
				JoinedAt:      block.VegaTime,
				AtEpoch:       uint64(block.Height),
				VegaTime:      block.VegaTime,
			}

			err := rs.RefereeJoinedReferralSet(ctx, &setReferee)
			require.NoError(t, err)
			referees[setID] = append(referees[setID], setReferee)
		}
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	sets, _ := setupReferralSetsAndReferees(t, ctx, bs, ps, rs)

	t.Run("Should return all referral sets", func(t *testing.T) {
		got, pageInfo, err := rs.ListReferralSets(ctx, nil, helpers.DefaultNoPagination())
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
		got, pageInfo, err := rs.ListReferralSets(ctx, &want.ID, entities.CursorPagination{})
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

		got, pageInfo, err := rs.ListReferralSets(ctx, nil, cursor)
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

		got, pageInfo, err := rs.ListReferralSets(ctx, nil, cursor)
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

		got, pageInfo, err := rs.ListReferralSets(ctx, nil, cursor)
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

		got, pageInfo, err := rs.ListReferralSets(ctx, nil, cursor)
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
	ctx, rollback := tempTransaction(t)
	defer rollback()

	sets, referees := setupReferralSetsAndReferees(t, ctx, bs, ps, rs)
	src := rand.New(rand.NewSource(time.Now().UnixNano()))
	r := rand.New(src)
	set := sets[r.Intn(len(sets))]
	setID := set.ID.String()
	refs := referees[setID]

	t.Run("Should return all referees in a set if no pagination", func(t *testing.T) {
		want := refs[:]
		got, pageInfo, err := rs.ListReferralSetReferees(ctx, set.ID, helpers.DefaultNoPagination())
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

		got, pageInfo, err := rs.ListReferralSetReferees(ctx, set.ID, cursor)
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

		got, pageInfo, err := rs.ListReferralSetReferees(ctx, set.ID, cursor)
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

	t.Run("Should return the requested page if first and after cursor are set", func(t *testing.T) {
		first := int32(3)
		after := refs[2].Cursor().Encode()
		cursor, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
		require.NoError(t, err)

		got, pageInfo, err := rs.ListReferralSetReferees(ctx, set.ID, cursor)
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

		got, pageInfo, err := rs.ListReferralSetReferees(ctx, set.ID, cursor)
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
