// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
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
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/core/assets/builtin"
	"code.vegaprotocol.io/vega/core/assets/erc20"
	"code.vegaprotocol.io/vega/core/broker"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrAssetInvalid       = errors.New("asset invalid")
	ErrAssetDoesNotExist  = errors.New("asset does not exist")
	ErrUnknownAssetSource = errors.New("unknown asset source")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/erc20_bridge_view_mock.go -package mocks code.vegaprotocol.io/vega/core/assets ERC20BridgeView
type ERC20BridgeView interface {
	FindAsset(asset *types.AssetDetails) error
}

type Service struct {
	log *logging.Logger
	cfg Config

	broker broker.Interface

	// id to asset
	// these assets exists and have been save
	amu    sync.RWMutex
	assets map[string]*Asset

	// this is a list of pending asset which are currently going through
	// proposal, they can later on be promoted to the asset lists once
	// the proposal is accepted by both the nodes and the users
	pamu                sync.RWMutex
	pendingAssets       map[string]*Asset
	pendingAssetUpdates map[string]*Asset

	nodeWallets *nodewallets.NodeWallets
	ethClient   erc20.ETHClient
	ass         *assetsSnapshotState

	ethToVega   map[string]string
	isValidator bool

	bridgeView ERC20BridgeView
}

func New(
	log *logging.Logger,
	cfg Config,
	nw *nodewallets.NodeWallets,
	ethClient erc20.ETHClient,
	broker broker.Interface,
	bridgeView ERC20BridgeView,
	isValidator bool,
) *Service {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Service{
		log:                 log,
		cfg:                 cfg,
		broker:              broker,
		assets:              map[string]*Asset{},
		pendingAssets:       map[string]*Asset{},
		pendingAssetUpdates: map[string]*Asset{},
		nodeWallets:         nw,
		ethClient:           ethClient,
		ass: &assetsSnapshotState{
			changedActive:  true,
			changedPending: true,
		},
		isValidator: isValidator,
		ethToVega:   map[string]string{},
		bridgeView:  bridgeView,
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

// SetPendingListing update the state of an asset from proposed
// to pending listing on the bridge.
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
// to pending listing on the bridge.
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

func (s *Service) buildAssetFromProto(asset *types.Asset) (*Asset, error) {
	switch asset.Details.Source.(type) {
	case *types.AssetDetailsBuiltinAsset:
		return &Asset{
			builtin.New(asset.ID, asset.Details),
		}, nil
	case *types.AssetDetailsErc20:
		// TODO(): fix once the ethereum wallet and client are not required
		// anymore to construct assets
		var (
			erc20Asset *erc20.ERC20
			err        error
		)
		if s.isValidator {
			erc20Asset, err = erc20.New(asset.ID, asset.Details, s.nodeWallets.Ethereum, s.ethClient)
		} else {
			erc20Asset, err = erc20.New(asset.ID, asset.Details, nil, nil)
		}
		if err != nil {
			return nil, err
		}
		return &Asset{erc20Asset}, nil
	default:
		return nil, ErrUnknownAssetSource
	}
}

// NewAsset add a new asset to the pending list of assets
// the ref is the reference of proposal which submitted the new asset
// returns the assetID and an error.
func (s *Service) NewAsset(ctx context.Context, proposalID string, assetDetails *types.AssetDetails) (string, error) {
	s.pamu.Lock()
	defer s.pamu.Unlock()
	asset, err := s.assetFromDetails(proposalID, assetDetails)
	if err != nil {
		return "", err
	}
	s.pendingAssets[proposalID] = asset
	s.ass.changedPending = true
	s.broker.Send(events.NewAssetEvent(ctx, *asset.Type()))

	return proposalID, err
}

func (s *Service) StageAssetUpdate(updatedAssetProto *types.Asset) error {
	s.pamu.Lock()
	defer s.pamu.Unlock()
	if _, ok := s.assets[updatedAssetProto.ID]; !ok {
		return ErrAssetDoesNotExist
	}

	updatedAsset, err := s.buildAssetFromProto(updatedAssetProto)
	if err != nil {
		return fmt.Errorf("couldn't update asset: %w", err)
	}
	s.pendingAssetUpdates[updatedAssetProto.ID] = updatedAsset
	s.ass.changedPendingUpdates = true
	return nil
}

func (s *Service) ApplyAssetUpdate(ctx context.Context, assetID string) error {
	s.pamu.Lock()
	defer s.pamu.Unlock()

	updatedAsset, ok := s.pendingAssetUpdates[assetID]
	if !ok {
		return ErrAssetDoesNotExist
	}

	s.amu.Lock()
	defer s.amu.Unlock()

	currentAsset, ok := s.assets[assetID]
	if !ok {
		return ErrAssetDoesNotExist
	}
	if err := currentAsset.Update(updatedAsset); err != nil {
		s.log.Panic("couldn't update the asset", logging.Error(err))
	}

	delete(s.pendingAssetUpdates, assetID)
	s.ass.changedActive = true
	s.ass.changedPendingUpdates = true
	s.broker.Send(events.NewAssetEvent(ctx, *updatedAsset.Type()))
	return nil
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

func (s *Service) getPendingAssetUpdates() []*types.Asset {
	s.pamu.RLock()
	defer s.pamu.RUnlock()
	ret := make([]*types.Asset, 0, len(s.assets))
	for _, a := range s.pendingAssetUpdates {
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

// ValidateAssetNonValidator is only to be used by non-validators
// at startup when loading genesis file. We just assume assets are
// valid.
func (s *Service) ValidateAssetNonValidator(assetID string) error {
	// get the asset to validate from the assets pool
	asset, err := s.Get(assetID)
	// if we get an error here, we'll never change the state of the proposal,
	// so it will be dismissed later on by all the whole network
	if err != nil || asset == nil {
		s.log.Error("Validating asset, unable to get the asset",
			logging.AssetID(assetID),
			logging.Error(err),
		)
		return errors.New("invalid asset ID")
	}

	asset.SetValid()
	return nil
}

func (s *Service) ValidateAsset(assetID string) error {
	// get the asset to validate from the assets pool
	asset, err := s.Get(assetID)
	// if we get an error here, we'll never change the state of the proposal,
	// so it will be dismissed later on by all the whole network
	if err != nil || asset == nil {
		s.log.Error("Validating asset, unable to get the asset",
			logging.AssetID(assetID),
			logging.Error(err),
		)
		return errors.New("invalid asset ID")
	}

	return s.validateAsset(asset)
}

func (s *Service) validateAsset(a *Asset) error {
	var err error
	if erc20, ok := a.ERC20(); ok {
		err = s.bridgeView.FindAsset(erc20.Type().Details.DeepClone())
		// no error, our asset exists on chain
		if err == nil {
			erc20.SetValid()
		}
	}

	return err
}
