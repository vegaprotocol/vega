package epochtime

import (
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

func (s *Svc) serialise() error {

	pl := types.EpochState{
		Seq:                  s.epoch.Seq,
		StartTime:            s.epoch.StartTime,
		ExpireTime:           s.epoch.ExpireTime,
		ReadyToStartNewEpoch: s.readyToStartNewEpoch,
		ReadyToEndEpoch:      s.readyToEndEpoch,
	}

	data, err := proto.Marshal(pl.IntoProto())
	if err != nil {
		return err
	}

	s.data = data
	s.hash = crypto.Hash(data)
	return nil

}

func (s *Svc) Namespace() types.SnapshotNamespace {
	return types.EpochSnapshot
}

func (s *Svc) Keys() []string {
	t := &types.PayloadEpoch{}
	return []string{t.Key()}
}

func (s *Svc) GetHash(_ string) ([]byte, error) {
	return s.hash, nil
}

func (s *Svc) Snapshot() (map[string][]byte, error) {
	t := &types.PayloadEpoch{}
	return map[string][]byte{t.Key(): s.data}, nil
}

func (s *Svc) GetState(_ string) ([]byte, error) {
	return s.data, nil
}

func (s *Svc) LoadSnapshot(payload *types.PayloadEpoch) error {

	snap := payload.EpochState

	s.epoch = types.Epoch{
		Seq:        snap.Seq,
		StartTime:  snap.StartTime,
		ExpireTime: snap.ExpireTime,
		Action:     vega.EpochAction_EPOCH_ACTION_START,
	}

	s.readyToStartNewEpoch = snap.ReadyToStartNewEpoch
	s.readyToEndEpoch = snap.ReadyToEndEpoch
	s.length = s.epoch.ExpireTime.Sub(s.epoch.StartTime)

	// take snapshot
	s.serialise()
	return nil
}
