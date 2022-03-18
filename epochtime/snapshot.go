package epochtime

import (
	"context"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"code.vegaprotocol.io/vega/libs/proto"
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

func (s *Svc) Stopped() bool {
	return false
}

func (s *Svc) GetHash(k string) ([]byte, error) {
	if k != s.pl.Key() {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	return s.hash, nil
}

func (s *Svc) GetState(k string) ([]byte, []types.StateProvider, error) {
	if k != s.pl.Key() {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	return s.data, nil, nil
}

func (s *Svc) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if s.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
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

		// notify all the engines that store epoch data about the current restored epoch
		s.notifyRestore(ctx, s.epoch)
		return nil, s.serialise()
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (s *Svc) notifyRestore(ctx context.Context, e types.Epoch) {
	for _, f := range s.restoreListeners {
		f(ctx, e)
	}
}
