package assets

import (
	"context"
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/types"
)

var (
	activeKey  = (&types.PayloadActiveAssets{}).Key()
	pendingKey = (&types.PayloadPendingAssets{}).Key()

	hashKeys = []string{
		activeKey,
		pendingKey,
	}

	ErrSnapshotKeyDoesNotExist = errors.New("unknown key for assets snapshot")
)

type assetsSnapshotState struct {
	changed    map[string]bool
	serialised map[string][]byte
}

func (s *Service) Namespace() types.SnapshotNamespace {
	return types.AssetsSnapshot
}

func (s *Service) Keys() []string {
	return hashKeys
}

func (s *Service) Stopped() bool {
	return false
}

func (s *Service) serialiseActive() ([]byte, error) {
	enabled := s.GetEnabledAssets()
	sort.SliceStable(enabled, func(i, j int) bool { return enabled[i].ID < enabled[j].ID })
	payload := types.Payload{
		Data: &types.PayloadActiveAssets{
			ActiveAssets: &types.ActiveAssets{
				Assets: enabled,
			},
		},
	}
	return proto.Marshal(payload.IntoProto())
}

func (s *Service) serialisePending() ([]byte, error) {
	pending := s.getPendingAssets()
	sort.SliceStable(pending, func(i, j int) bool { return pending[i].ID < pending[j].ID })
	payload := types.Payload{
		Data: &types.PayloadPendingAssets{
			PendingAssets: &types.PendingAssets{
				Assets: pending,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

// get the serialised form and hash of the given key.
func (s *Service) serialise(k string) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.keyToSerialiser[k]; !ok {
		return nil, ErrSnapshotKeyDoesNotExist
	}

	if !s.ass.changed[k] {
		return s.ass.serialised[k], nil
	}

	data, err := s.keyToSerialiser[k]()
	if err != nil {
		return nil, err
	}

	s.ass.serialised[k] = data
	s.ass.changed[k] = false
	return data, nil
}

func (s *Service) HasChanged(k string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ass.changed[k]
}

func (s *Service) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := s.serialise(k)
	return state, nil, err
}

func (s *Service) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if s.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadActiveAssets:
		return nil, s.restoreActive(ctx, pl.ActiveAssets, p)
	case *types.PayloadPendingAssets:
		return nil, s.restorePending(ctx, pl.PendingAssets, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (s *Service) restoreActive(ctx context.Context, active *types.ActiveAssets, p *types.Payload) error {
	var err error
	s.assets = map[string]*Asset{}
	for _, p := range active.Assets {
		if _, err = s.NewAsset(p.ID, p.Details); err != nil {
			return err
		}

		pa, _ := s.Get(p.ID)
		if s.isValidator {
			if err = pa.Validate(); err != nil {
				return err
			}
		} else {
			pa.SetValidNonValidator()
		}

		if err = s.Enable(p.ID); err != nil {
			return err
		}
	}
	s.ass.changed[activeKey] = false
	s.ass.serialised[activeKey], err = proto.Marshal(p.IntoProto())

	return err
}

func (s *Service) restorePending(ctx context.Context, pending *types.PendingAssets, p *types.Payload) error {
	var err error
	s.pendingAssets = map[string]*Asset{}
	for _, p := range pending.Assets {
		if _, err = s.NewAsset(p.ID, p.Details); err != nil {
			return err
		}
	}
	s.ass.changed[pendingKey] = false
	s.ass.serialised[pendingKey], err = proto.Marshal(p.IntoProto())

	return err
}
