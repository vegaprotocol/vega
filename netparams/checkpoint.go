package netparams

import (
	"context"
	"sort"

	"code.vegaprotocol.io/protos/vega"
	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
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
	return nil
}
