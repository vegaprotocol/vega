package epochtime

import (
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

func (s *Svc) serialise() ([]byte, error) {

	pl := types.EpochState{
		Seq:                  s.epoch.Seq,
		StartTime:            s.epoch.StartTime,
		ExpireTime:           s.epoch.ExpireTime,
		Action:               s.epoch.Action,
		ReadyToStartNewEpoch: s.readyToStartNewEpoch,
		ReadyToEndEpoch:      s.readyToEndEpoch,
	}

	return proto.Marshal(pl.IntoProto())

}

func (s *Svc) Namespace() types.SnapshotNamespace {
	return types.EpochSnapshot
}

func (s *Svc) Keys() []string {
	t := &types.PayloadEpoch{}
	return []string{t.Key()}
}

func (s *Svc) GetHash(_ string) ([]byte, error) {
	data, err := s.serialise()
	if err != nil {
		return nil, err
	}
	return crypto.Hash(data), nil
}

func (s *Svc) Snapshot() (map[string][]byte, error) {
	data, err := s.serialise()
	if err != nil {
		return nil, err
	}

	t := &types.PayloadEpoch{}
	return map[string][]byte{t.Key(): data}, nil
}

func (s *Svc) GetState(_ string) ([]byte, error) {
	data, err := s.serialise()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *Svc) LoadSnapshot(payload *types.PayloadEpoch) error {

	snap := payload.EpochState

	s.epoch = types.Epoch{
		Seq:        snap.Seq,
		StartTime:  snap.StartTime,
		ExpireTime: snap.ExpireTime,
		Action:     snap.Action,
	}

	s.readyToStartNewEpoch = snap.ReadyToStartNewEpoch
	s.readyToEndEpoch = snap.ReadyToEndEpoch
	s.length = s.epoch.ExpireTime.Sub(s.epoch.StartTime)
	return nil
}
