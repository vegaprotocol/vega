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
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestDelegation(t *testing.T, ds *sqlstore.Delegations,
	party entities.Party,
	node entities.Node,
	epochID int64,
	block entities.Block,
) entities.Delegation {
	r := entities.Delegation{
		PartyID:  party.ID,
		NodeID:   node.ID,
		EpochID:  epochID,
		Amount:   decimal.NewFromInt(100),
		VegaTime: block.VegaTime,
	}
	err := ds.Add(context.Background(), r)
	require.NoError(t, err)
	return r
}

func delegationLessThan(x, y entities.Delegation) bool {
	if x.EpochID != y.EpochID {
		return x.EpochID < y.EpochID
	}
	if x.PartyID.String() != y.PartyID.String() {
		return x.PartyID.String() < y.PartyID.String()
	}
	if x.NodeID.String() != y.NodeID.String() {
		return x.NodeID.String() < y.NodeID.String()
	}
	return x.Amount.LessThan(y.Amount)
}

func assertDelegationsMatch(t *testing.T, expected, actual []entities.Delegation) {
	t.Helper()
	assert.Empty(t, cmp.Diff(expected, actual, cmpopts.SortSlices(delegationLessThan)))
}

func TestDelegations(t *testing.T) {
	defer DeleteEverything()
	ps := sqlstore.NewParties(connectionSource)
	ds := sqlstore.NewDelegations(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	block := addTestBlock(t, bs)

	node1 := addTestNode(t, ns, block)
	node2 := addTestNode(t, ns, block)

	node1ID := node1.ID.String()
	node2ID := node2.ID.String()

	party1 := addTestParty(t, ps, block)
	party2 := addTestParty(t, ps, block)

	party1ID := party1.ID.String()
	party2ID := party2.ID.String()

	delegation1 := addTestDelegation(t, ds, party1, node1, 1, block)
	delegation2 := addTestDelegation(t, ds, party1, node2, 2, block)
	delegation3 := addTestDelegation(t, ds, party2, node1, 3, block)
	delegation4 := addTestDelegation(t, ds, party2, node2, 4, block)
	delegation5 := addTestDelegation(t, ds, party2, node2, 5, block)

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Delegation{delegation1, delegation2, delegation3, delegation4, delegation5}
		actual, err := ds.GetAll(context.Background())
		require.NoError(t, err)
		assertDelegationsMatch(t, expected, actual)
	})

	t.Run("GetByParty", func(t *testing.T) {
		expected := []entities.Delegation{delegation1, delegation2}
		actual, err := ds.Get(context.Background(), &party1ID, nil, nil, nil)
		require.NoError(t, err)
		assertDelegationsMatch(t, expected, actual)
	})

	t.Run("GetByNode", func(t *testing.T) {
		expected := []entities.Delegation{delegation1, delegation3}
		actual, err := ds.Get(context.Background(), nil, &node1ID, nil, nil)
		require.NoError(t, err)
		assertDelegationsMatch(t, expected, actual)
	})

	t.Run("GetByEpoch", func(t *testing.T) {
		expected := []entities.Delegation{delegation4}
		four := int64(4)
		actual, err := ds.Get(context.Background(), nil, nil, &four, nil)
		require.NoError(t, err)
		assertDelegationsMatch(t, expected, actual)
	})

	t.Run("GetByPartyAndNode", func(t *testing.T) {
		expected := []entities.Delegation{delegation4, delegation5}
		actual, err := ds.Get(context.Background(), &party2ID, &node2ID, nil, nil)
		require.NoError(t, err)
		assertDelegationsMatch(t, expected, actual)
	})

	t.Run("GetByPartyAndNodeAndEpoch", func(t *testing.T) {
		expected := []entities.Delegation{delegation4}
		four := int64(4)
		actual, err := ds.Get(context.Background(), &party2ID, &node2ID, &four, nil)
		require.NoError(t, err)
		assertDelegationsMatch(t, expected, actual)
	})

	t.Run("GetPagination", func(t *testing.T) {
		expected := []entities.Delegation{delegation4, delegation3, delegation2}
		p := entities.OffsetPagination{Skip: 1, Limit: 3, Descending: true}
		actual, err := ds.Get(context.Background(), nil, nil, nil, &p)
		require.NoError(t, err)
		assert.Equal(t, expected, actual) // Explicitly check the order on this one
	})
}
