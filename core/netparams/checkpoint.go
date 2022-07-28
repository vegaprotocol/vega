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

package netparams

import (
	"context"
	"sort"
	"time"

	"code.vegaprotocol.io/protos/vega"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/core/types"

	"code.vegaprotocol.io/vega/core/libs/proto"
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
