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

package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	v1 "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

func TestLiquidityFeeStats(t *testing.T) {
	market := "market1"
	asset := "usdt"

	stats := types.NewLiquidityFeeStats()

	stats.RegisterTotalFeesAmountPerParty(map[string]*num.Uint{
		"a": num.UintFromUint64(2),
		"b": num.UintFromUint64(3),
		"c": num.UintFromUint64(4),
	})

	stats.RegisterTotalFeesAmountPerParty(map[string]*num.Uint{
		"a": num.UintFromUint64(1),
		"b": num.UintFromUint64(2),
		"c": num.UintFromUint64(3),
		"d": num.UintFromUint64(4),
	})

	expectedAmountsPerParty := []*v1.PartyAmount{
		{Party: "a", Amount: "3"},
		{Party: "b", Amount: "5"},
		{Party: "c", Amount: "7"},
		{Party: "d", Amount: "4"},
	}

	statsProto := stats.ToProto(market, asset, 100)
	require.Equal(t, market, statsProto.Market)
	require.Equal(t, asset, statsProto.Asset)
	require.Equal(t, uint64(100), statsProto.EpochSeq)
	require.Equal(t, expectedAmountsPerParty, statsProto.FeesPaidPerParty)
	require.Equal(t, "19", statsProto.TotalFeesPaid)

	stats = types.NewLiquidityFeeStats()

	statsProto = stats.ToProto(market, asset, 100)
	require.Equal(t, []*v1.PartyAmount{}, statsProto.FeesPaidPerParty)
	require.Equal(t, "0", statsProto.TotalFeesPaid)

	stats.RegisterTotalFeesAmountPerParty(map[string]*num.Uint{
		"a": num.UintFromUint64(11),
		"b": num.UintFromUint64(22),
		"c": num.UintFromUint64(33),
		"d": num.UintFromUint64(44),
	})

	expectedAmountsPerParty = []*v1.PartyAmount{
		{Party: "a", Amount: "11"},
		{Party: "b", Amount: "22"},
		{Party: "c", Amount: "33"},
		{Party: "d", Amount: "44"},
	}

	statsProto = stats.ToProto(market, asset, 100)
	require.Equal(t, expectedAmountsPerParty, statsProto.FeesPaidPerParty)
	require.Equal(t, "110", statsProto.TotalFeesPaid)
}
