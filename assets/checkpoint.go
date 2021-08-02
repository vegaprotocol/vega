package assets

import (
	"sort"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
)

const (
	SnapshotName = "assets"
)

func (s Service) Name() string {
	return SnapshotName
}

func (s *Service) Hash() []byte {
	return nil
}

func (s *Service) Chekpoint() []byte {
	t := &vega.Assets{
		Assets: s.getEnabled(),
	}
	b, _ := vega.Marshal(t)
	return b
}

func (s *Service) Load(checkpoint, _ []byte) error {
	data := &vega.Assets{}
	if err := vega.Unmarshal(checkpoint, data); err != nil {
		return err
	}
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

func (s *Service) getEnabled() []*vega.AssetEntry {
	s.amu.RLock()
	keys := make([]string, 0, len(s.assets))
	vals := make(map[string]*vega.AssetEntry, len(s.assets))
	for k, a := range s.assets {
		keys = append(keys, k)
		vals[k] = &vega.AssetEntry{
			Id:           k,
			AssetDetails: a.Type().Details.IntoProto(),
		}
	}
	s.amu.RUnlock()
	if len(keys) == 0 {
		return nil
	}
	ret := make([]*vega.AssetEntry, 0, len(vals))
	sort.Strings(keys)
	for _, k := range keys {
		ret = append(ret, vals[k])
	}
	return ret
}
