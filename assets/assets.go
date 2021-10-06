package assets

import (
	"context"
	"errors"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/assets/builtin"
	"code.vegaprotocol.io/vega/assets/erc20"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/types"
)

var (
	ErrAssetInvalid       = errors.New("asset invalid")
	ErrAssetDoesNotExist  = errors.New("asset does not exist")
	ErrUnknownAssetSource = errors.New("unknown asset source")
)

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/assets TimeService
type TimeService interface {
	NotifyOnTick(f func(context.Context, time.Time))
}

type Service struct {
	log *logging.Logger
	cfg Config

	// id to asset
	// these assets exists and have been save
	amu    sync.RWMutex
	assets map[string]*Asset

	// this is a list of pending asset which are currently going through
	// proposal, they can later on be promoted to the asset lists once
	// the proposal is accepted by both the nodes and the users
	pamu          sync.RWMutex
	pendingAssets map[string]*Asset

	nodeWallets     *nodewallets.NodeWallets
	ethClient       erc20.ETHClient
	dss             *assetsSnapshotState
	keyToSerialiser map[string]func() ([]byte, error)
}

func New(log *logging.Logger, cfg Config, nw *nodewallets.NodeWallets, ethClient erc20.ETHClient, ts TimeService) *Service {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	s := &Service{
		log:           log,
		cfg:           cfg,
		assets:        map[string]*Asset{},
		pendingAssets: map[string]*Asset{},
		nodeWallets:   nw,
		ethClient:     ethClient,
		dss: &assetsSnapshotState{
			changed:    map[string]bool{activeKey: true, pendingKey: true},
			hash:       map[string][]byte{},
			serialised: map[string][]byte{},
		},
		keyToSerialiser: map[string]func() ([]byte, error){},
	}

	s.keyToSerialiser[activeKey] = s.serialiseActive
	s.keyToSerialiser[pendingKey] = s.serialisePending
	ts.NotifyOnTick(s.onTick)
	return s
}

// ReloadConf updates the internal configuration
func (s *Service) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.cfg = cfg
}

func (*Service) onTick(_ context.Context, t time.Time) {}

// Enable move the state of an from pending the list of valid and accepted assets
func (s *Service) Enable(assetID string) error {
	s.pamu.Lock()
	defer s.pamu.Unlock()
	asset, ok := s.pendingAssets[assetID]
	if !ok {
		return ErrAssetDoesNotExist
	}
	if asset.IsValid() {
		s.amu.Lock()
		defer s.amu.Unlock()
		s.assets[assetID] = asset
		delete(s.pendingAssets, assetID)
		s.dss.changed[activeKey] = true
		s.dss.changed[pendingKey] = true
		return nil
	}
	return ErrAssetInvalid
}

func (s *Service) IsEnabled(assetID string) bool {
	s.amu.RLock()
	defer s.amu.RUnlock()
	_, ok := s.assets[assetID]
	return ok
}

func (s *Service) assetFromDetails(assetID string, assetDetails *types.AssetDetails) (*Asset, error) {
	switch assetDetails.Source.(type) {
	case *types.AssetDetailsBuiltinAsset:
		return &Asset{
			builtin.New(assetID, assetDetails),
		}, nil
	case *types.AssetDetailsErc20:
		asset, err := erc20.New(assetID, assetDetails, s.nodeWallets.Ethereum, s.ethClient)
		if err != nil {
			return nil, err
		}
		return &Asset{asset}, nil
	default:
		return nil, ErrUnknownAssetSource
	}
}

// NewAsset add a new asset to the pending list of assets
// the ref is the reference of proposal which submitted the new asset
// returns the assetID and an error
func (s *Service) NewAsset(assetID string, assetDetails *types.AssetDetails) (string, error) {
	s.pamu.Lock()
	defer s.pamu.Unlock()
	asset, err := s.assetFromDetails(assetID, assetDetails)
	if err != nil {
		return "", err
	}
	s.pendingAssets[assetID] = asset
	s.dss.changed[pendingKey] = true
	return assetID, err
}

func (s *Service) GetEnabledAssets() []*types.Asset {
	s.amu.RLock()
	defer s.amu.RUnlock()
	ret := make([]*types.Asset, 0, len(s.assets))
	for _, a := range s.assets {
		ret = append(ret, a.ToAssetType())
	}
	return ret
}

func (s *Service) getPendingAssets() []*types.Asset {
	s.pamu.RLock()
	defer s.pamu.RUnlock()
	ret := make([]*types.Asset, 0, len(s.assets))
	for _, a := range s.pendingAssets {
		ret = append(ret, a.ToAssetType())
	}
	return ret
}

func (s *Service) Get(assetID string) (*Asset, error) {
	s.amu.RLock()
	defer s.amu.RUnlock()
	asset, ok := s.assets[assetID]
	if ok {
		return asset, nil
	}
	s.pamu.RLock()
	defer s.pamu.RUnlock()
	asset, ok = s.pendingAssets[assetID]
	if ok {
		return asset, nil
	}
	return nil, ErrAssetDoesNotExist
}
