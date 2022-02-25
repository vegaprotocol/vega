package assets

import (
	"context"
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
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
	hash       map[string][]byte
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
func (s *Service) getSerialisedAndHash(k string) ([]byte, []byte, error) {
	if _, ok := s.keyToSerialiser[k]; !ok {
		return nil, nil, ErrSnapshotKeyDoesNotExist
	}

	if !s.ass.changed[k] {
		return s.ass.serialised[k], s.ass.hash[k], nil
	}

	data, err := s.keyToSerialiser[k]()
	if err != nil {
		return nil, nil, err
	}

	hash := crypto.Hash(data)
	s.ass.serialised[k] = data
	s.ass.hash[k] = hash
	s.ass.changed[k] = false
	return data, hash, nil
}

func (s *Service) GetHash(k string) ([]byte, error) {
	_, hash, err := s.getSerialisedAndHash(k)
	return hash, err
}

func (s *Service) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, _, err := s.getSerialisedAndHash(k)
	return state, nil, err
}

func (s *Service) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if s.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
	switch pl := p.Data.(type) {
	case *types.PayloadActiveAssets:
		return nil, s.restoreActive(ctx, pl.ActiveAssets)
	case *types.PayloadPendingAssets:
		return nil, s.restorePending(ctx, pl.PendingAssets)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (s *Service) restoreActive(ctx context.Context, active *types.ActiveAssets) error {
	s.assets = map[string]*Asset{}
	for _, p := range active.Assets {
		if _, err := s.NewAsset(p.ID, p.Details); err != nil {
			return err
		}

		pa, _ := s.Get(p.ID)
		if s.isValidator {
			if err := pa.Validate(); err != nil {
				return err
			}
		} else {
			pa.SetValidNonValidator()
		}

		if err := s.Enable(p.ID); err != nil {
			return err
		}
	}
	s.ass.changed[activeKey] = true
	return nil
}

func (s *Service) restorePending(ctx context.Context, pending *types.PendingAssets) error {
	s.pendingAssets = map[string]*Asset{}
	for _, p := range pending.Assets {
		if _, err := s.NewAsset(p.ID, p.Details); err != nil {
			return err
		}
	}

	// after reloading we need to set the dirty flag to true so that we know next time to recalc the hash/serialise
	s.ass.changed[pendingKey] = true
	return nil
}
