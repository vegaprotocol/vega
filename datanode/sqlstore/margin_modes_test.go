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
	"encoding/json"
	"strings"
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

func TestMarginModesStore(t *testing.T) {
	ctx := tempTransaction(t)

	marginModesStore := sqlstore.NewMarginModes(connectionSource)

	market1 := entities.MarketID(GenerateID())
	market2 := entities.MarketID(GenerateID())
	party1 := entities.PartyID(GenerateID())
	party2 := entities.PartyID(GenerateID())

	marginMode11 := entities.PartyMarginMode{
		MarketID:   market1,
		PartyID:    party1,
		MarginMode: vegapb.MarginMode_MARGIN_MODE_CROSS_MARGIN,
		AtEpoch:    5,
	}
	marginMode12 := entities.PartyMarginMode{
		MarketID:                   market1,
		PartyID:                    party2,
		MarginMode:                 vegapb.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
		MarginFactor:               ptr.From(num.DecimalFromFloat(1.20)),
		MinTheoreticalMarginFactor: ptr.From(num.DecimalFromFloat(1.21)),
		MaxTheoreticalLeverage:     ptr.From(num.DecimalFromFloat(1.22)),
		AtEpoch:                    6,
	}
	marginMode21 := entities.PartyMarginMode{
		MarketID:                   market2,
		PartyID:                    party1,
		MarginMode:                 vegapb.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
		MarginFactor:               ptr.From(num.DecimalFromFloat(2.10)),
		MinTheoreticalMarginFactor: ptr.From(num.DecimalFromFloat(2.11)),
		MaxTheoreticalLeverage:     ptr.From(num.DecimalFromFloat(2.12)),
		AtEpoch:                    10,
	}
	marginMode22 := entities.PartyMarginMode{
		MarketID:   market2,
		PartyID:    party2,
		MarginMode: vegapb.MarginMode_MARGIN_MODE_CROSS_MARGIN,
		AtEpoch:    12,
	}

	t.Run("Inserting brand new market/party combination", func(t *testing.T) {
		expectedMarginModes := []entities.PartyMarginMode{marginMode11, marginMode12, marginMode21, marginMode22}
		sortMarginModes(&expectedMarginModes)

		for _, mode := range expectedMarginModes {
			require.NoError(t, marginModesStore.UpdatePartyMarginMode(ctx, mode))
		}

		foundMarginModes, _, err := marginModesStore.ListPartyMarginModes(ctx, entities.DefaultCursorPagination(false), sqlstore.ListPartyMarginModesFilters{})
		require.NoError(t, err)
		expectedStatsJson, _ := json.Marshal(expectedMarginModes)
		statsJson, _ := json.Marshal(foundMarginModes)
		assert.JSONEq(t, string(expectedStatsJson), string(statsJson))
	})

	marginMode11 = entities.PartyMarginMode{
		MarketID:                   market1,
		PartyID:                    party1,
		MarginMode:                 vegapb.MarginMode_MARGIN_MODE_ISOLATED_MARGIN,
		MarginFactor:               ptr.From(num.DecimalFromFloat(3.10)),
		MinTheoreticalMarginFactor: ptr.From(num.DecimalFromFloat(3.11)),
		MaxTheoreticalLeverage:     ptr.From(num.DecimalFromFloat(3.12)),
		AtEpoch:                    6,
	}

	t.Run("Inserting brand new market/party combination", func(t *testing.T) {
		require.NoError(t, marginModesStore.UpdatePartyMarginMode(ctx, marginMode11))

		expectedMarginModes := []entities.PartyMarginMode{marginMode11, marginMode12, marginMode21, marginMode22}
		sortMarginModes(&expectedMarginModes)

		foundMarginModes, _, err := marginModesStore.ListPartyMarginModes(ctx,
			entities.DefaultCursorPagination(false),
			sqlstore.ListPartyMarginModesFilters{},
		)
		require.NoError(t, err)
		expectedStatsJson, _ := json.Marshal(expectedMarginModes)
		statsJson, _ := json.Marshal(foundMarginModes)
		assert.JSONEq(t, string(expectedStatsJson), string(statsJson))
	})

	t.Run("Inserting an update on an existing combination", func(t *testing.T) {
		expectedMarginModes := []entities.PartyMarginMode{marginMode11, marginMode12, marginMode21, marginMode22}
		sortMarginModes(&expectedMarginModes)

		for _, mode := range expectedMarginModes {
			require.NoError(t, marginModesStore.UpdatePartyMarginMode(ctx, mode))
		}

		foundMarginModes, _, err := marginModesStore.ListPartyMarginModes(ctx,
			entities.DefaultCursorPagination(false),
			sqlstore.ListPartyMarginModesFilters{},
		)
		require.NoError(t, err)
		expectedStatsJson, _ := json.Marshal(expectedMarginModes)
		statsJson, _ := json.Marshal(foundMarginModes)
		assert.JSONEq(t, string(expectedStatsJson), string(statsJson))
	})

	t.Run("Listing a margin mode for party", func(t *testing.T) {
		expectedMarginModes := []entities.PartyMarginMode{marginMode11, marginMode21}
		sortMarginModes(&expectedMarginModes)

		foundMarginModes, _, err := marginModesStore.ListPartyMarginModes(ctx,
			entities.DefaultCursorPagination(false),
			sqlstore.ListPartyMarginModesFilters{
				PartyID: ptr.From(party1),
			},
		)
		require.NoError(t, err)
		expectedStatsJson, _ := json.Marshal(expectedMarginModes)
		statsJson, _ := json.Marshal(foundMarginModes)
		assert.JSONEq(t, string(expectedStatsJson), string(statsJson))
	})

	t.Run("Listing a margin mode for market", func(t *testing.T) {
		expectedMarginModes := []entities.PartyMarginMode{marginMode11, marginMode12}
		sortMarginModes(&expectedMarginModes)

		foundMarginModes, _, err := marginModesStore.ListPartyMarginModes(ctx,
			entities.DefaultCursorPagination(false),
			sqlstore.ListPartyMarginModesFilters{
				MarketID: ptr.From(market1),
			},
		)
		require.NoError(t, err)
		expectedStatsJson, _ := json.Marshal(expectedMarginModes)
		statsJson, _ := json.Marshal(foundMarginModes)
		assert.JSONEq(t, string(expectedStatsJson), string(statsJson))
	})

	t.Run("Listing a margin mode for market and party", func(t *testing.T) {
		expectedMarginModes := []entities.PartyMarginMode{marginMode11}
		sortMarginModes(&expectedMarginModes)

		foundMarginModes, _, err := marginModesStore.ListPartyMarginModes(ctx,
			entities.DefaultCursorPagination(false),
			sqlstore.ListPartyMarginModesFilters{
				PartyID:  ptr.From(party1),
				MarketID: ptr.From(market1),
			},
		)
		require.NoError(t, err)
		expectedStatsJson, _ := json.Marshal(expectedMarginModes)
		statsJson, _ := json.Marshal(foundMarginModes)
		assert.JSONEq(t, string(expectedStatsJson), string(statsJson))
	})
}

func sortMarginModes(modes *[]entities.PartyMarginMode) {
	slices.SortStableFunc(*modes, func(a, b entities.PartyMarginMode) int {
		if a.MarketID == b.MarketID {
			return strings.Compare(a.PartyID.String(), b.PartyID.String())
		}
		return strings.Compare(a.MarketID.String(), b.MarketID.String())
	})
}
