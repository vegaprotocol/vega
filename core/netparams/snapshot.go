// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
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
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/types"

	"code.vegaprotocol.io/vega/libs/proto"
)

type snapState struct {
	updated bool
	data    []byte
	pl      *types.NetParams
	index   map[string]int
	t       *types.PayloadNetParams
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

func (s snapState) Namespace() types.SnapshotNamespace {
	return s.t.Namespace()
}

func (s *snapState) HasChanged(k string) bool {
	return true
	// return s.updated
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
	s.updated = false
	return nil
}

func (s *snapState) GetState(_ string) ([]byte, error) {
	if !s.HasChanged(s.t.Key()) {
		return s.data, nil
	}
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
	s.updated = true
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

func (s *Store) HasChanged(k string) bool {
	return s.state.updated
}

func (s *Store) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := s.state.GetState(k)
	return state, nil, err
}

func (s *Store) LoadState(ctx context.Context, pl *types.Payload) ([]types.StateProvider, error) {
	if pl.Namespace() != s.state.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	np, ok := pl.Data.(*types.PayloadNetParams)
	if !ok {
		return nil, types.ErrInconsistentNamespaceKeys // it's the only possible key/namespace combo here
	}

	for _, kv := range np.NetParams.Params {
		s.Update(ctx, kv.Key, kv.Value)
	}

	// Now they have been loaded, dispatch the changes so that the other engines pick them up
	for k := range s.store {
		if err := s.dispatchUpdate(ctx, k); err != nil {
			return nil, fmt.Errorf("could not propagate netparams update to listener, %v: %v", k, err)
		}
	}

	var err error
	s.state.updated = false
	s.state.data, err = proto.Marshal(pl.IntoProto())
	s.paramUpdates = map[string]struct{}{}
	return nil, err
}
