package assets

import (
	"sort"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

func (*Service) Name() types.CheckpointName {
	return types.AssetsCheckpoint
}

func (s *Service) Checkpoint() ([]byte, error) {
	t := &checkpoint.Assets{
		Assets: s.getEnabled(),
	}
	return proto.Marshal(t)
}

func (s *Service) Load(cp []byte) error {
	data := &checkpoint.Assets{}
	if err := proto.Unmarshal(cp, data); err != nil {
		return err
	}
	s.amu.Lock()
	s.pamu.Lock()
	s.pendingAssets = map[string]*Asset{}
	s.assets = map[string]*Asset{}
	s.pamu.Unlock()
	s.amu.Unlock()
	for _, a := range data.Assets {
		details := types.AssetDetailsFromProto(a.AssetDetails)
		id, err := s.NewAsset(a.Id, details)
		if err != nil {
			return err
		}
		pa, _ := s.Get(a.Id)
		if err := pa.Validate(); err != nil {
			return err
		}
		if err := s.Enable(id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) getEnabled() []*checkpoint.AssetEntry {
	s.amu.RLock()
	keys := make([]string, 0, len(s.assets))
	vals := make(map[string]*checkpoint.AssetEntry, len(s.assets))
	for k, a := range s.assets {
		keys = append(keys, k)
		vals[k] = &checkpoint.AssetEntry{
			Id:           k,
			AssetDetails: a.Type().Details.IntoProto(),
		}
	}
	s.amu.RUnlock()
	if len(keys) == 0 {
		return nil
	}
	ret := make([]*checkpoint.AssetEntry, 0, len(vals))
	sort.Strings(keys)
	for _, k := range keys {
		ret = append(ret, vals[k])
	}
	return ret
}
