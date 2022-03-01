package epochtime

import (
	"context"

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
		s.length = s.epoch.ExpireTime.Sub(s.epoch.StartTime)

		// notify everyone of the restored epoch, this will always be mid epoch since onTick only
		// happens at the start of a block and an epoch boundary is handle instantaneously. So even at
		// and epoch boundary the order of events will be
		// onEndBlock -> snapshot -> OnCommit -> onBeginBlock -> epoch-end + epoch-start events -> notify-new-epoch
		// and so we can never take a snapshot between an epoch ending and a new one starting.
		s.notify(ctx, types.Epoch{
			Seq:        snap.Seq,
			StartTime:  snap.StartTime,
			ExpireTime: snap.ExpireTime,
			Action:     vega.EpochAction_EPOCH_ACTION_RESTORED,
		})
		return nil, s.serialise()
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (s *Svc) NotifyOnRestore(f func(context.Context, types.Epoch)) {
	s.restoreListeners = append(s.listeners, f)
}
