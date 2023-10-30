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

package execution

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
)

func (e *Engine) Name() types.CheckpointName {
	return types.ExecutionCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	for id, mkt := range e.futureMarkets {
		state := mkt.GetCPState()
		e.marketCPStates[id] = state
	}
	data := make([]*types.CPMarketState, 0, len(e.marketCPStates))
	for _, s := range e.marketCPStates {
		data = append(data, s)
	}
	sort.SliceStable(data, func(i, j int) bool {
		return data[i].ID < data[j].ID
	})
	wrapped := types.ExecutionState{
		Data: data,
	}
	cpData, err := proto.Marshal(wrapped.IntoProto())
	return cpData, err
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	if len(data) == 0 {
		// because the checkpoint data may be missing from older checkpoint data
		e.marketCPStates = map[string]*types.CPMarketState{}
		return nil
	}
	wrapper := checkpoint.ExecutionState{}
	if err := proto.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	e.marketCPStates = make(map[string]*types.CPMarketState, len(wrapper.Data))
	// for now, restore all pending markets as though their state is valid for the full TTL window
	for _, mcp := range wrapper.Data {
		e.marketCPStates[mcp.Id] = types.NewMarketStateFromProto(mcp)
	}
	return nil
}
