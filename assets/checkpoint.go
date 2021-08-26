package assets

import (
	"sort"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

func (*Service) Name() types.CheckpointName {
	return types.AssetsCheckpoint
}

func (s *Service) Checkpoint() ([]byte, error) {
	t := &snapshot.Assets{
		Assets: s.getEnabled(),
	}
	return proto.Marshal(t)
}

func (s *Service) Load(checkpoint []byte) error {
	data := &snapshot.Assets{}
	if err := proto.Unmarshal(checkpoint, data); err != nil {
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
		if err := s.Enable(id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) getEnabled() []*snapshot.AssetEntry {
	s.amu.RLock()
	keys := make([]string, 0, len(s.assets))
	vals := make(map[string]*snapshot.AssetEntry, len(s.assets))
	for k, a := range s.assets {
		keys = append(keys, k)
		vals[k] = &snapshot.AssetEntry{
			Id:           k,
			AssetDetails: a.Type().Details.IntoProto(),
		}
	}
	s.amu.RUnlock()
	if len(keys) == 0 {
		return nil
	}
	ret := make([]*snapshot.AssetEntry, 0, len(vals))
	sort.Strings(keys)
	for _, k := range keys {
		ret = append(ret, vals[k])
	}
	return ret
}
