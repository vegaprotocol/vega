package netparams

import (
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

type snapState struct {
	hash    []byte
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

func (s *snapState) Snapshot() (map[string][]byte, error) {
	k := s.t.Key()
	v, err := s.GetState(k)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{
		k: v,
	}, nil
}

func (s snapState) Namespace() types.SnapshotNamespace {
	return s.t.Namespace()
}

func (s *snapState) GetHash(_ string) ([]byte, error) {
	if !s.updated {
		return s.hash, nil
	}
	h, err := s.hashState()
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (s *snapState) hashState() ([]byte, error) {
	data, err := proto.Marshal(s.t.IntoProto())
	if err != nil {
		return nil, err
	}
	s.data = data
	s.hash = crypto.Hash(data)
	s.updated = false
	return s.hash, nil
}

func (s *snapState) GetState(_ string) ([]byte, error) {
	if !s.updated {
		return s.data, nil
	}
	if _, err := s.hashState(); err != nil {
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

func (s Store) Namespace() types.SnapshotNamespace {
	return s.state.Namespace()
}

func (s Store) Keys() []string {
	return s.state.Keys()
}

func (s Store) GetHash(k string) ([]byte, error) {
	return s.state.GetHash(k)
}

func (s Store) Snapshot() (map[string][]byte, error) {
	return s.state.Snapshot()
}

func (s Store) GetState(k string) ([]byte, error) {
	return s.state.GetState(k)
}
