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

package netparams

import (
	"context"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/protos/vega"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

func (s *Store) Name() types.CheckpointName {
	return types.NetParamsCheckpoint
}

func (s *Store) Checkpoint() ([]byte, error) {
	s.mu.RLock()
	params := snapshot.NetParams{
		Params: make([]*vega.NetworkParameter, 0, len(s.store)),
	}
	for k, v := range s.store {
		params.Params = append(params.Params, &vega.NetworkParameter{
			Key:   k,
			Value: v.String(),
		})
	}
	s.mu.RUnlock()
	// no net params, we can stop here
	if len(params.Params) == 0 {
		return nil, nil
	}
	// sort the keys
	sort.Slice(params.Params, func(i, j int) bool {
		return params.Params[i].Key < params.Params[j].Key
	})
	return proto.Marshal(&params)
}

func (s *Store) Load(ctx context.Context, data []byte) error {
	params := &snapshot.NetParams{}
	if err := proto.Unmarshal(data, params); err != nil {
		return err
	}
	np := make(map[string]string, len(params.Params))
	for _, param := range params.Params {
		if _, ok := s.checkpointOverwrites[param.Key]; ok {
			continue // skip all overwrites
		}
		np[param.Key] = param.Value
	}
	if err := s.updateBatch(ctx, np); err != nil {
		return err
	}
	// force the updates dispatch
	s.OnTick(ctx, time.Time{})
	return nil
}
