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
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
)

type snapState struct {
	data  []byte
	pl    *types.NetParams
	index map[string]int
	t     *types.PayloadNetParams
}

func newSnapState(store map[string]value) *snapState {
	state := &snapState{
		pl:    &types.NetParams{},
		index: make(map[string]int, len(store)),
		t:     &types.PayloadNetParams{},
	}
	// set pointer
	state.t.NetParams = state.pl
	// set the initial state
	state.build(store)
	return state
}

func (s *snapState) build(store map[string]value) {
	params := make([]*types.NetworkParameter, 0, len(store))
	for k, v := range store {
		params = append(params, &types.NetworkParameter{
			Key:   k,
			Value: v.String(),
		})
	}
	// sort by key
	sort.SliceStable(params, func(i, j int) bool {
		return params[i].Key < params[j].Key
	})
	// build the index
	for i, p := range params {
		s.index[p.Key] = i
	}
	s.pl.Params = params
}

func (s *snapState) Keys() []string {
	return []string{
		s.t.Key(),
	}
}

func (s *snapState) Namespace() types.SnapshotNamespace {
	return s.t.Namespace()
}

func (s *snapState) hashState() error {
	// apparently the payload types can't me marshalled by themselves
	pl := types.Payload{
		Data: s.t,
	}
	data, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return err
	}
	s.data = data
	return nil
}

func (s *snapState) GetState(_ string) ([]byte, error) {
	if err := s.hashState(); err != nil {
		return nil, err
	}
	return s.data, nil
}

func (s *snapState) update(k, v string) {
	i, ok := s.index[k]
	if !ok {
		i = len(s.pl.Params)
		s.pl.Params = append(s.pl.Params, &types.NetworkParameter{
			Key: k,
		})
	}
	s.pl.Params[i].Value = v
}

// make Store implement/forward the dataprovider interface

func (s *Store) Namespace() types.SnapshotNamespace {
	return s.state.Namespace()
}

func (s *Store) Keys() []string {
	return s.state.Keys()
}

func (s *Store) Stopped() bool {
	return false
}

func (s *Store) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := s.state.GetState(k)
	return state, nil, err
}

func (s *Store) LoadState(ctx context.Context, pl *types.Payload) ([]types.StateProvider, error) {
	if pl.Namespace() != s.state.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	s.isProtocolUpgradeRunning = vgcontext.InProgressUpgrade(ctx)

	np, ok := pl.Data.(*types.PayloadNetParams)
	if !ok {
		return nil, types.ErrInconsistentNamespaceKeys // it's the only possible key/namespace combo here
	}

	for _, kv := range np.NetParams.Params {
		if err := s.UpdateOptionalValidation(ctx, kv.Key, kv.Value, false, false); err != nil {
			return nil, err
		}
	}

	if vgcontext.InProgressUpgradeFrom(ctx, "v0.74.9") {
		vgChainID, err := vgcontext.ChainIDFromContext(ctx)
		if err != nil {
			panic(fmt.Errorf("no vega chain ID found in context: %w", err))
		}
		secondaryEthConf, ok := bridgeMapping[vgChainID]
		if !ok {
			panic("Missing secondary ethereum configuration")
		}
		if err := s.UpdateOptionalValidation(ctx, BlockchainsSecondaryEthereumConfig, secondaryEthConf, false, false); err != nil {
			return nil, err
		}
	}

	// Now they have been loaded, dispatch the changes so that the other engines pick them up
	for k := range s.store {
		if err := s.dispatchUpdate(ctx, k); err != nil {
			return nil, fmt.Errorf("could not propagate netparams update to listener, %v: %v", k, err)
		}
	}

	var err error
	s.state.data, err = proto.Marshal(pl.IntoProto())
	s.paramUpdates = map[string]struct{}{}
	return nil, err
}
