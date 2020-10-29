package assets

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/assets/builtin"
	"code.vegaprotocol.io/vega/assets/erc20"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	types "code.vegaprotocol.io/vega/proto"
	"golang.org/x/crypto/sha3"
)

var (
	ErrAssetInvalid      = errors.New("asset invalid")
	ErrAssetDoesNotExist = errors.New("asset does not exist")
	ErrAssetExistForID   = errors.New("an asset already exist for this ID")
	ErrUnknowAssetSource = errors.New("unknown asset source")
	ErrNoAssetForRef     = errors.New("no assets for proposal reference")
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
	assets map[string]*Asset

	// this is a list of pending asset which are currently going through
	// proposal, they can later on be promoted to the asset lists once
	// the proposal is accepted by both the nodes and the users
	pendingAssets map[string]*Asset

	// map of reference to proposal id
	// use to find back an asset when the governance process
	// is still ongoing
	refs map[string]string

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
		refs:          map[string]string{},
		nw:            nw,
	}
	ts.NotifyOnTick(s.onTick)
	return s, nil
}

// ReloadConf updates the internal configuration
func (a *Service) ReloadConf(cfg Config) {
	a.log.Info("reloading configuration")
	if a.log.GetLevel() != cfg.Level.Get() {
		a.log.Info("updating log level",
			logging.String("old", a.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		a.log.SetLevel(cfg.Level.Get())
	}

	a.cfg = cfg
}

func (a *Service) onTick(_ context.Context, t time.Time) {}

// Enable move the state of an from pending the list of valid and accepted assets
func (a *Service) Enable(assetID string) error {
	asset, ok := a.pendingAssets[assetID]
	if !ok {
		return ErrAssetDoesNotExist
	}
	if asset.IsValid() {
		a.assets[assetID] = asset
		delete(a.pendingAssets, assetID)
		return nil
	}
	return ErrAssetInvalid
}

func (a *Service) IsEnabled(assetID string) bool {
	_, ok := a.assets[assetID]
	return ok
}

// NewAsset add a new asset to the pending list of assets
// the ref is the reference of proposal which submitted the new asset
// returns the assetID and an error
func (s *Service) NewAsset(assetID string, assetSrc *types.AssetSource) (string, error) {
	src := assetSrc.Source
	switch assetSrcImpl := src.(type) {
	case *types.AssetSource_BuiltinAsset:
		s.pendingAssets[assetID] = &Asset{builtin.New(assetID, assetSrcImpl.BuiltinAsset)}
	case *types.AssetSource_Erc20:
		wal, ok := s.nw.Get(nodewallet.Ethereum)
		if !ok {
			return "", errors.New("missing wallet for ETH")
		}
		asset, err := erc20.New(assetID, assetSrcImpl.Erc20, wal)
		if err != nil {
			return "", err
		}
		s.pendingAssets[assetID] = &Asset{asset}
	default:
		return "", ErrUnknowAssetSource
	}
	// setup the ref lookup table
	s.refs[assetID] = assetID

	return assetID, nil
}

// RemovePending remove and asset from the list of pending assets
func (s *Service) RemovePending(assetID string) error {
	_, ok := s.pendingAssets[assetID]
	if !ok {
		return ErrAssetDoesNotExist
	}
	delete(s.pendingAssets, assetID)
	return nil
}

func (s *Service) assetHash(asset *Asset) []byte {
	data := asset.ProtoAsset()
	buf := fmt.Sprintf("%v%v%v%v%v",
		data.ID,
		data.Name,
		data.Symbol,
		data.TotalSupply,
		data.Decimals)
	return hash([]byte(buf))
}

func (s *Service) Get(assetID string) (*Asset, error) {
	asset, ok := s.assets[assetID]
	if ok {
		return asset, nil
	}
	asset, ok = s.pendingAssets[assetID]
	if ok {
		return asset, nil
	}
	return nil, ErrAssetDoesNotExist
}

func (s *Service) GetByRef(ref string) (*Asset, error) {
	id, ok := s.refs[ref]
	if !ok {
		return nil, ErrNoAssetForRef
	}

	return s.Get(id)
}

// GetAssetHash return an hash of the given asset to be used
// signed to validate the asset on the vega chain
func (s *Service) AssetHash(assetID string) ([]byte, error) {
	asset, ok := s.assets[assetID]
	if ok {
		return s.assetHash(asset), nil
	}
	asset, ok = s.pendingAssets[assetID]
	if ok {
		return s.assetHash(asset), nil
	}
	return nil, ErrAssetDoesNotExist
}

func hash(key []byte) []byte {
	hasher := sha3.New256()
	hasher.Write([]byte(key))
	return hasher.Sum(nil)
}
