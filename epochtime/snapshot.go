package epochtime

import (
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

func (s *Svc) serialise() error {

	s.state.Seq = s.epoch.Seq
	s.state.StartTime = s.epoch.StartTime
	s.state.ExpireTime = s.epoch.ExpireTime
	s.state.ReadyToStartNewEpoch = s.readyToStartNewEpoch
	s.state.ReadyToEndEpoch = s.readyToEndEpoch

	data, err := proto.Marshal(s.pl.IntoProto())
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
	return []string{s.pl.Key()}
}

func (s *Svc) GetHash(k string) ([]byte, error) {

	if k != s.pl.Key() {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
	return s.hash, nil
}

func (s *Svc) Snapshot() (map[string][]byte, error) {
	return map[string][]byte{s.pl.Key(): s.data}, nil
}

func (s *Svc) GetState(k string) ([]byte, error) {
	if k != s.pl.Key() {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
	return s.data, nil
}

func (s *Svc) LoadState(payload *types.Payload) error {

	if s.Namespace() != payload.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadEpoch:

		snap := pl.EpochState
		s.epoch = types.Epoch{
			Seq:        snap.Seq,
			StartTime:  snap.StartTime,
			ExpireTime: snap.ExpireTime,
			Action:     vega.EpochAction_EPOCH_ACTION_START,
		}

		s.readyToStartNewEpoch = snap.ReadyToStartNewEpoch
		s.readyToEndEpoch = snap.ReadyToEndEpoch
		s.length = s.epoch.ExpireTime.Sub(s.epoch.StartTime)

		return s.serialise()

	default:
		return types.ErrUnknownSnapshotType
	}

}
