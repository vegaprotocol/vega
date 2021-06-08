package assets

import (
	"context"
	"errors"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/assets/builtin"
	"code.vegaprotocol.io/vega/assets/erc20"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrAssetInvalid      = errors.New("asset invalid")
	ErrAssetDoesNotExist = errors.New("asset does not exist")
	ErrUnknowAssetSource = errors.New("unknown asset source")
)

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/assets TimeService
type TimeService interface {
	NotifyOnTick(f func(context.Context, time.Time))
}

type NodeWallet interface {
	Get(chain nodewallet.Blockchain) (nodewallet.Wallet, bool)
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

	nw NodeWallet
}

func New(log *logging.Logger, cfg Config, nw NodeWallet, ts TimeService) (*Service, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	s := &Service{
		log:           log,
		cfg:           cfg,
		assets:        map[string]*Asset{},
		pendingAssets: map[string]*Asset{},
		nw:            nw,
	}
	ts.NotifyOnTick(s.onTick)
	return s, nil
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

// NewAsset add a new asset to the pending list of assets
// the ref is the reference of proposal which submitted the new asset
// returns the assetID and an error
func (s *Service) NewAsset(assetID string, assetDetails *types.AssetDetails) (string, error) {
	s.pamu.Lock()
	defer s.pamu.Unlock()
	switch assetDetails.Source.(type) {
	case *types.AssetDetails_BuiltinAsset:
		s.pendingAssets[assetID] = &Asset{
			builtin.New(assetID, assetDetails),
		}
	case *types.AssetDetails_Erc20:
		wal, ok := s.nw.Get(nodewallet.Ethereum)
		if !ok {
			return "", errors.New("missing wallet for ETH")
		}
		asset, err := erc20.New(assetID, assetDetails, wal)
		if err != nil {
			return "", err
		}
		s.pendingAssets[assetID] = &Asset{asset}
	default:
		return "", ErrUnknowAssetSource
	}

	return assetID, nil
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
