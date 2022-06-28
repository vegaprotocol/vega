// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package assets

import (
	"context"
	"errors"
	"sync"

	"code.vegaprotocol.io/vega/assets/builtin"
	"code.vegaprotocol.io/vega/assets/erc20"
	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/vegatime"
)

var (
	ErrAssetInvalid       = errors.New("asset invalid")
	ErrAssetDoesNotExist  = errors.New("asset does not exist")
	ErrUnknownAssetSource = errors.New("unknown asset source")
)

type Service struct {
	log *logging.Logger
	cfg Config

	broker broker.BrokerI

	// id to asset
	// these assets exists and have been save
	amu    sync.RWMutex
	assets map[string]*Asset

	// this is a list of pending asset which are currently going through
	// proposal, they can later on be promoted to the asset lists once
	// the proposal is accepted by both the nodes and the users
	pamu          sync.RWMutex
	pendingAssets map[string]*Asset

	nodeWallets *nodewallets.NodeWallets
	ethClient   erc20.ETHClient
	ass         *assetsSnapshotState
	ethToVega   map[string]string

	isValidator bool
}

func New(
	log *logging.Logger,
	cfg Config,
	nw *nodewallets.NodeWallets,
	ethClient erc20.ETHClient,
	broker broker.BrokerI,
	ts vegatime.TimeService,
	isValidator bool,
) *Service {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Service{
		log:           log,
		cfg:           cfg,
		broker:        broker,
		assets:        map[string]*Asset{},
		pendingAssets: map[string]*Asset{},
		nodeWallets:   nw,
		ethClient:     ethClient,
		ass: &assetsSnapshotState{
			changedActive:  true,
			changedPending: true,
		},
		isValidator: isValidator,
		ethToVega:   map[string]string{},
	}
}

// ReloadConf updates the internal configuration.
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

// Enable move the state of an from pending the list of valid and accepted assets.
func (s *Service) Enable(ctx context.Context, assetID string) error {
	s.pamu.Lock()
	defer s.pamu.Unlock()
	asset, ok := s.pendingAssets[assetID]
	if !ok {
		return ErrAssetDoesNotExist
	}
	if asset.IsValid() {
		asset.SetEnabled()
		s.amu.Lock()
		defer s.amu.Unlock()
		s.assets[assetID] = asset
		if asset.IsERC20() {
			eth, _ := asset.ERC20()
			s.ethToVega[eth.ProtoAsset().GetDetails().GetErc20().GetContractAddress()] = assetID
		}
		delete(s.pendingAssets, assetID)
		s.ass.changedActive = true
		s.ass.changedPending = true

		s.broker.Send(events.NewAssetEvent(ctx, *asset.Type()))

		return nil
	}
	return ErrAssetInvalid
}

// SetPendingListing update the state of an asset from proposed
// to pending listing on the bridge
func (s *Service) SetPendingListing(ctx context.Context, assetID string) error {
	s.pamu.Lock()
	defer s.pamu.Unlock()
	asset, ok := s.pendingAssets[assetID]
	if !ok {
		return ErrAssetDoesNotExist
	}

	asset.SetPendingListing()
	s.broker.Send(events.NewAssetEvent(ctx, *asset.Type()))
	s.ass.changedPending = true

	return nil
}

// SetRejected update the state of an asset from proposed
// to pending listing on the bridge
func (s *Service) SetRejected(ctx context.Context, assetID string) error {
	s.pamu.Lock()
	defer s.pamu.Unlock()
	asset, ok := s.pendingAssets[assetID]
	if !ok {
		return ErrAssetDoesNotExist
	}

	asset.SetRejected()
	s.broker.Send(events.NewAssetEvent(ctx, *asset.Type()))
	delete(s.pendingAssets, assetID)
	s.ass.changedPending = true

	return nil
}

func (s *Service) GetVegaIDFromEthereumAddress(address string) string {
	s.amu.Lock()
	defer s.amu.Unlock()
	return s.ethToVega[address]
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
		// TODO(): fix once the ethereum wallet and client are not required
		// anymore to construct assets
		var (
			asset *erc20.ERC20
			err   error
		)
		if s.isValidator {
			asset, err = erc20.New(assetID, assetDetails, s.nodeWallets.Ethereum, s.ethClient)
		} else {
			asset, err = erc20.New(assetID, assetDetails, nil, nil)
		}
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
// returns the assetID and an error.
func (s *Service) NewAsset(ctx context.Context, assetID string, assetDetails *types.AssetDetails) (string, error) {
	s.pamu.Lock()
	defer s.pamu.Unlock()
	asset, err := s.assetFromDetails(assetID, assetDetails)
	if err != nil {
		return "", err
	}
	s.pendingAssets[assetID] = asset
	s.ass.changedPending = true
	s.broker.Send(events.NewAssetEvent(ctx, *asset.Type()))

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
