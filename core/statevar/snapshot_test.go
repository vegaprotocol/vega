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

package statevar_test

import (
	"bytes"
	"context"
	"testing"

	gtypes "code.vegaprotocol.io/vega/core/types"
	types "code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/stretchr/testify/require"
)

func TestSnapshot(t *testing.T) {
	engine1 := getTestEngine(t, now).engine
	engine1.RegisterStateVariable("asset1", "market1", "var1", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())
	engine1.RegisterStateVariable("asset1", "market1", "var2", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())
	engine1.RegisterStateVariable("asset1", "market2", "var1", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())
	engine1.RegisterStateVariable("asset1", "market2", "var2", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())
	engine1.RegisterStateVariable("asset2", "market1", "var1", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())
	engine1.RegisterStateVariable("asset2", "market1", "var1", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())

	engine1.ReadyForTimeTrigger("asset1", "market1")
	engine1.ReadyForTimeTrigger("asset1", "market2")

	key := (&gtypes.PayloadFloatingPointConsensus{}).Key()
	state1, _, err := engine1.GetState(key)
	require.NoError(t, err)

	engine2 := getTestEngine(t, now).engine
	engine2.RegisterStateVariable("asset1", "market1", "var1", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())
	engine2.RegisterStateVariable("asset1", "market1", "var2", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())
	engine2.RegisterStateVariable("asset1", "market2", "var1", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())
	engine2.RegisterStateVariable("asset1", "market2", "var2", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())
	engine2.RegisterStateVariable("asset2", "market1", "var1", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())
	engine2.RegisterStateVariable("asset2", "market1", "var1", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())

	pl := snapshotpb.Payload{}
	require.NoError(t, proto.Unmarshal(state1, &pl))
	engine2.LoadState(context.Background(), gtypes.PayloadFromProto(&pl))

	state2, _, err := engine2.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state1, state2))
}

func TestSnapshotChangeFlagSet(t *testing.T) {
	key := (&gtypes.PayloadFloatingPointConsensus{}).Key()
	engine1 := getTestEngine(t, now).engine

	engine1.RegisterStateVariable("asset1", "market1", "var1", converter{}, defaultStartCalc(), []types.EventType{types.EventTypeMarketEnactment, types.EventTypeTimeTrigger}, defaultResultBack())

	state1, _, err := engine1.GetState(key)
	require.NoError(t, err)

	// this should hit the change flag causing us to reserialise at the next hash
	engine1.ReadyForTimeTrigger("asset1", "market1")

	state2, _, err := engine1.GetState(key)
	require.NoError(t, err)
	require.False(t, bytes.Equal(state1, state2))
}
