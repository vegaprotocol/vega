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
	"encoding/json"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/protos/vega"
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
	overrideSet := map[string]struct{}{}
	if vgcontext.InProgressUpgradeFrom(ctx, "v0.73.1") {
		// this is our list of network parameters to override
		overrides := []types.NetworkParameter{
			{
				Key:   "governance.proposal.transfer.minProposerBalance",
				Value: "20000000000000000000000",
			},
			{
				Key:   "governance.proposal.transfer.minVoterBalance",
				Value: "1000000000000000000",
			},
			{
				Key:   "governance.proposal.transfer.requiredParticipation",
				Value: "0.01",
			},
			{
				Key:   "governance.proposal.transfer.minClose",
				Value: "168h",
			},
			{
				Key:   "governance.proposal.transfer.minEnact",
				Value: "168h",
			},
		}

		// let's apply them first
		for _, v := range overrides {
			// add to the set
			overrideSet[v.Key] = struct{}{}

			// always in the override
			s.protocolUpgradeNewParameters = append(
				s.protocolUpgradeNewParameters, v.Key,
			)

			// then apply the update
			if err := s.UpdateOptionalValidation(ctx, v.Key, v.Value, false, false); err != nil {
				return nil, err
			}
		}
	}

	if pl.Namespace() != s.state.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	np, ok := pl.Data.(*types.PayloadNetParams)
	if !ok {
		return nil, types.ErrInconsistentNamespaceKeys // it's the only possible key/namespace combo here
	}

	fromSnapshot := map[string]struct{}{}
	for _, kv := range np.NetParams.Params {
		// execute this if not in the override list
		if _, ok := overrideSet[kv.Key]; !ok {
			if err := s.UpdateOptionalValidation(ctx, kv.Key, kv.Value, false, false); err != nil {
				return nil, err
			}
		}
		fromSnapshot[kv.Key] = struct{}{}
	}

	if vgcontext.InProgressUpgradeFrom(ctx, "v0.73.4") {
		k := BlockchainsEthereumConfig
		v := vega.EthereumConfig{}
		if err := s.GetJSONStruct(BlockchainsEthereumConfig, &v); err != nil {
			return nil, fmt.Errorf("could not get the ethereum config (%w)", err)
		}

		// change confirmations to 64
		v.Confirmations = 64

		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal the updated ethereum config (%w)", err)
		}

		// re-save
		if err := s.UpdateOptionalValidation(ctx, k, string(b), false, false); err != nil {
			return nil, err
		}
	}

	// Now they have been loaded, dispatch the changes so that the other engines pick them up
	for k := range s.store {
		// is this a new parameter? if yes, we are likely doing a protocol
		// upgrade, and should be sending that to the datanode please.
		if _, ok := fromSnapshot[k]; !ok {
			s.protocolUpgradeNewParameters = append(
				s.protocolUpgradeNewParameters, k,
			)
		}

		if err := s.dispatchUpdate(ctx, k); err != nil {
			return nil, fmt.Errorf("could not propagate netparams update to listener, %v: %v", k, err)
		}
	}

	var err error
	s.state.data, err = proto.Marshal(pl.IntoProto())
	s.paramUpdates = map[string]struct{}{}
	return nil, err
}
