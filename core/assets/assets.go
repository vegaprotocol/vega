// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package assets

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/assets/builtin"
	"code.vegaprotocol.io/vega/core/assets/erc20"
	"code.vegaprotocol.io/vega/core/broker"
	"code.vegaprotocol.io/vega/core/events"
	nweth "code.vegaprotocol.io/vega/core/nodewallets/eth"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrAssetDoesNotExist        = errors.New("asset does not exist")
	ErrUnknownAssetSource       = errors.New("unknown asset source")
	ErrErc20AddressAlreadyInUse = errors.New("erc20 address already in use")
	ErrUnknownChainID           = errors.New("erc20 chain-id does not correspond to a bridge")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/assets ERC20BridgeView,Notary

type ERC20BridgeView interface {
	FindAsset(asset *types.AssetDetails) error
}

type Notary interface {
	StartAggregate(resID string, kind types.NodeSignatureKind, signature []byte)
	OfferSignatures(kind types.NodeSignatureKind, f func(id string) []byte)
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

	ethWallet nweth.EthereumWallet

	primaryEthChainID string
	primaryEthClient  erc20.ETHClient
	primaryBridgeView ERC20BridgeView

	secondaryEthChainID string
	secondaryEthClient  erc20.ETHClient
	secondaryBridgeView ERC20BridgeView

	notary Notary
	ass    *assetsSnapshotState

	ethToVega   map[string]string
	isValidator bool
}

func New(
	ctx context.Context,
	log *logging.Logger,
	cfg Config,
	nw nweth.EthereumWallet,
	primaryEthClient erc20.ETHClient,
	secondaryEthClient erc20.ETHClient,
	broker broker.Interface,
	primaryBridgeView ERC20BridgeView,
	secondaryBridgeView ERC20BridgeView,
	notary Notary,
	isValidator bool,
) (*Service, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	s := &Service{
		log:                 log,
		cfg:                 cfg,
		broker:              broker,
		assets:              map[string]*Asset{},
		pendingAssets:       map[string]*Asset{},
		pendingAssetUpdates: map[string]*Asset{},
		ethWallet:           nw,
		primaryEthClient:    primaryEthClient,
		secondaryEthClient:  secondaryEthClient,
		notary:              notary,
		ass:                 &assetsSnapshotState{},
		isValidator:         isValidator,
		ethToVega:           map[string]string{},
		primaryBridgeView:   primaryBridgeView,
		secondaryBridgeView: secondaryBridgeView,
	}

	if isValidator {
		primaryChainID, err := s.primaryEthClient.ChainID(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not fetch chain ID from the primary ethereum client: %w", err)
		}
		s.primaryEthChainID = primaryChainID.String()

		secondaryChainID, err := s.secondaryEthClient.ChainID(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not fetch chain ID from the secondary ethereum client: %w", err)
		}
		s.secondaryEthChainID = secondaryChainID.String()
	}

	return s, nil
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
	s.broker.Send(events.NewAssetEvent(ctx, *asset.Type()))
	return nil
}

// EnactPendingAsset the given id for an asset has just been enacted by the governance engine so we
// now need to generate signatures so that the asset can be listed.
func (s *Service) EnactPendingAsset(id string) {
	pa, _ := s.Get(id)
	var err error
	var signature []byte
	if s.isValidator {
		switch {
		case pa.IsERC20():
			asset, _ := pa.ERC20()
			_, signature, err = asset.SignListAsset()
			if err != nil {
				s.log.Panic("couldn't to sign transaction to list asset, is the node properly configured as a validator?",
					logging.Error(err))
			}
		default:
			s.log.Panic("trying to generate signatures for an unknown asset type")
		}
	}

	s.notary.StartAggregate(id, types.NodeSignatureKindAssetNew, signature)
}

// ValidateEthereumAddress checks that the given ERC20 address and chainID corresponds to one of Vega's bridges
// and isn't the address of an asset that already exists.
func (s *Service) ValidateEthereumAddress(address, chainID string) error {
	if chainID != s.primaryEthChainID && chainID != s.secondaryEthChainID {
		return ErrUnknownChainID
	}

	for _, a := range s.assets {
		if source, ok := a.ERC20(); ok {
			if source.ChainID() != chainID {
				// asset is on a different chain, definitely is not a dupe of it
				continue
			}

			if strings.EqualFold(source.Address(), address) {
				return ErrErc20AddressAlreadyInUse
			}
		}
	}
	for _, a := range s.pendingAssets {
		if source, ok := a.ERC20(); ok {
			if source.ChainID() != chainID {
				// asset is on a different chain, definitely is not a dupe of it
				continue
			}

			if strings.EqualFold(source.Address(), address) {
				return ErrErc20AddressAlreadyInUse
			}
		}
	}

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

// SetBridgeChainID sets the chain-ids for the bridge once we have processed the network parameters
// this is necessary so that non-validator nodes (which cannot just ask the eth-client) can know what they
// are.
func (s *Service) SetBridgeChainID(chainID string, primary bool) {
	if primary {
		s.primaryEthChainID = chainID
		return
	}
	s.secondaryEthChainID = chainID
}

func (s *Service) OnTick(_ context.Context, _ time.Time) {
	s.notary.OfferSignatures(types.NodeSignatureKindAssetNew, s.offerERC20NotarySignatures)
}

func (s *Service) offerERC20NotarySignatures(id string) []byte {
	if !s.isValidator {
		return nil
	}

	pa, err := s.Get(id)
	if err != nil {
		s.log.Panic("unable to find asset", logging.AssetID(id))
	}

	asset, _ := pa.ERC20()
	_, signature, err := asset.SignListAsset()
	if err != nil {
		s.log.Panic("couldn't to sign transaction to list asset, is the node properly configured as a validator?",
			logging.Error(err))
	}

	return signature
}

func (s *Service) assetFromDetails(assetID string, assetDetails *types.AssetDetails) (*Asset, error) {
	switch assetDetails.Source.(type) {
	case *types.AssetDetailsBuiltinAsset:
		return &Asset{
			builtin.New(assetID, assetDetails),
		}, nil
	case *types.AssetDetailsErc20:
		var asset *erc20.ERC20
		if s.isValidator {
			client, err := s.ethClientByChainID(assetDetails.GetERC20().ChainID)
			if err != nil {
				return nil, err
			}
			a, err := erc20.New(assetID, assetDetails, s.ethWallet, client)
			if err != nil {
				return nil, err
			}
			asset = a
		} else {
			a, err := erc20.New(assetID, assetDetails, nil, nil)
			if err != nil {
				return nil, err
			}
			asset = a
		}
		return &Asset{asset}, nil
	default:
		return nil, ErrUnknownAssetSource
	}
}

func (s *Service) buildAssetUpdateFromProto(asset *types.Asset) (*Asset, error) {
	switch asset.Details.Source.(type) {
	case *types.AssetDetailsBuiltinAsset:
		return &Asset{
			builtin.New(asset.ID, asset.Details),
		}, nil
	case *types.AssetDetailsErc20:
		var (
			erc20Asset *erc20.ERC20
			err        error
		)
		if s.isValidator {
			client, err := s.ethClientByChainID(asset.Details.GetERC20().ChainID)
			if err != nil {
				return nil, err
			}
			a, err := erc20.New(asset.ID, asset.Details, s.ethWallet, client)
			if err != nil {
				return nil, err
			}
			erc20Asset = a
		} else {
			a, err := erc20.New(asset.ID, asset.Details, nil, nil)
			if err != nil {
				return nil, err
			}
			erc20Asset = a
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
	s.broker.Send(events.NewAssetEvent(ctx, *asset.Type()))

	return proposalID, err
}

func (s *Service) StageAssetUpdate(updatedAssetProto *types.Asset) error {
	s.pamu.Lock()
	defer s.pamu.Unlock()
	if _, ok := s.assets[updatedAssetProto.ID]; !ok {
		return ErrAssetDoesNotExist
	}

	updatedAsset, err := s.buildAssetUpdateFromProto(updatedAssetProto)
	if err != nil {
		return fmt.Errorf("couldn't update asset: %w", err)
	}
	s.pendingAssetUpdates[updatedAssetProto.ID] = updatedAsset
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
	updatedAsset.SetEnabled()
	if err := currentAsset.Update(updatedAsset); err != nil {
		s.log.Panic("couldn't update the asset", logging.Error(err))
	}

	delete(s.pendingAssetUpdates, assetID)
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
	sort.SliceStable(ret, func(i, j int) bool { return ret[i].ID < ret[j].ID })
	return ret
}

func (s *Service) getPendingAssets() []*types.Asset {
	s.pamu.RLock()
	defer s.pamu.RUnlock()
	ret := make([]*types.Asset, 0, len(s.assets))
	for _, a := range s.pendingAssets {
		ret = append(ret, a.ToAssetType())
	}
	sort.SliceStable(ret, func(i, j int) bool { return ret[i].ID < ret[j].ID })
	return ret
}

func (s *Service) getPendingAssetUpdates() []*types.Asset {
	s.pamu.RLock()
	defer s.pamu.RUnlock()
	ret := make([]*types.Asset, 0, len(s.assets))
	for _, a := range s.pendingAssetUpdates {
		ret = append(ret, a.ToAssetType())
	}
	sort.SliceStable(ret, func(i, j int) bool { return ret[i].ID < ret[j].ID })
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
	if erc20Asset, ok := a.ERC20(); ok {
		details := erc20Asset.Type().Details
		bridgeView, err := s.bridgeViewByChainID(details.GetERC20().ChainID)
		if err != nil {
			return err
		}
		if err := bridgeView.FindAsset(details); err != nil {
			return err
		}
		// no error, our asset exists on chain
		erc20Asset.SetValid()
	}
	return nil
}

func (s *Service) ethClientByChainID(chainID string) (erc20.ETHClient, error) {
	switch chainID {
	case s.primaryEthChainID:
		return s.primaryEthClient, nil
	case s.secondaryEthChainID:
		return s.secondaryEthClient, nil
	default:
		return nil, fmt.Errorf("chain id %q is not supported", chainID)
	}
}

func (s *Service) bridgeViewByChainID(chainID string) (ERC20BridgeView, error) {
	switch chainID {
	case s.primaryEthChainID:
		return s.primaryBridgeView, nil
	case s.secondaryEthChainID:
		return s.secondaryBridgeView, nil
	default:
		return nil, fmt.Errorf("chain id %q is not supported", chainID)
	}
}
