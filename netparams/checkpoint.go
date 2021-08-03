package netparams

import (
	"context"
	"sort"

	"code.vegaprotocol.io/protos/vega"
)

const (
	SnapshotName = "netparams"
)

func (s *Store) Name() string {
	return SnapshotName
}

func (s *Store) Hash() []byte {
	return nil
}

type vidx struct {
	key string
	idx int
}

func (s *Store) Checkpoint() []byte {
	s.mu.RLock()
	params := vega.NetParams{
		Params: make([]*vega.NetworkParameter, 0, len(s.store)),
	}
	keys := make([]vidx, 0, len(s.store))
	// already convert to string when traversing store here
	// so creating the sorted output is more efficient
	vals := make([]string, 0, len(s.store))
	for k, v := range s.store {
		keys = append(keys, vidx{
			key: k,
			idx: len(vals), // len(vals) == idx of value for key k
		})
		vals = append(vals, v.String())
	}
	s.mu.RUnlock()
	// no net params, we can stop here
	if len(vals) == 0 {
		return nil
	}
	// sort the keys
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].key < keys[j].key
	})
	for _, k := range keys {
		params.Params = append(params.Params, &vega.NetworkParameter{
			Key:   k.key,
			Value: vals[k.idx],
		})
	}
	b, _ := vega.Marshal(&params)
	return b
}

func (s *Store) Load(data, _ []byte) error {
	params := &vega.NetParams{}
	if err := vega.Unmarshal(data, params); err != nil {
		return err
	}
	np := make(map[string]string, len(params.Params))
	for _, param := range params.Params {
		np[param.Key] = param.Value
	}
	if err := s.updateBatch(context.Background(), np); err != nil {
		return err
	}
	return nil
}
