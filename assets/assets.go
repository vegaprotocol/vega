package assets

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/assets/builtin"
	"code.vegaprotocol.io/vega/assets/common"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"golang.org/x/crypto/sha3"
)

var (
	ErrAssetInvalid      = errors.New("asset invalid")
	ErrAssetDoesNotExist = errors.New("asset does not exist")
	ErrAssetExistForID   = errors.New("an asset already exist for this ID")
	ErrUnknowAssetSource = errors.New("unknown asset source")
)

type Asset interface {
	// get informations about the asset itself
	Data() *types.Asset

	// get the internal asset class
	GetAssetClass() common.AssetClass

	// is the order valid / validated with the target chain?
	IsValid() bool

	// this is used to validate that the asset
	// exist on the target chain
	Validate() error
	// build the signature for whitelisting on the vega bridge
	SignBridgeWhitelisting() ([]byte, error)
	// ensure on the target chain that withdrawal on funds
	// happended
	ValidateWithdrawal() error // SignWithdrawal
	SignWithdrawal() ([]byte, error)
	// ensure on the target chain that a deposit really did happen
	ValidateDeposit() error
}

type NodeWallet interface {
}

type Assets struct {
	log *logging.Logger
	cfg Config

	// id to asset
	// these assets exists and have been save
	assets map[uint64]Asset

	// this is a list of pending asset which are currently going through
	// proposal, they can later on be promoted to the asset lists once
	// the proposal is accepted by both the nodes and the users
	pendingAssets map[uint64]Asset

	nw NodeWallet
}

func New(log *logging.Logger, cfg Config, nw NodeWallet) (*Assets, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	return &Assets{
		log:           log,
		cfg:           cfg,
		assets:        map[uint64]Asset{},
		pendingAssets: map[uint64]Asset{},
		nw:            nw,
	}, nil
}

// Enable move the state of an from pending the list of valid and accepted assets
func (a *Assets) Enable(assetID uint64) error {
	asset, ok := a.pendingAssets[assetID]
	if !ok {
		return ErrAssetDoesNotExist
	}
	if asset.IsValid() {
		delete(a.pendingAssets, assetID)
		a.assets[assetID] = asset
	}
	return ErrAssetInvalid
}

// NewAsset add a new asset to the pending list of assets
func (a *Assets) NewAsset(assetID uint64, assetSrc *types.AssetSource) error {
	// ensure an idea for this asset does note exists already
	_, ok := a.pendingAssets[assetID]
	if ok {
		return ErrAssetExistForID
	}
	_, ok = a.assets[assetID]
	if ok {
		return ErrAssetExistForID
	}
	src := assetSrc.Source
	switch assetSrcImpl := src.(type) {
	case *types.AssetSource_BuiltinAsset:
		a.pendingAssets[assetID] = builtin.New(assetID, assetSrcImpl.BuiltinAsset)
	case *types.AssetSource_Erc20:
	default:
		return ErrUnknowAssetSource
	}
	return nil
}

// RemovePending remove and asset from the list of pending assets
func (a *Assets) RemovePending(assetID uint64) error {
	_, ok := a.pendingAssets[assetID]
	if !ok {
		return ErrAssetDoesNotExist
	}
	delete(a.pendingAssets, assetID)
	return nil
}

func (a *Assets) assetHash(asset Asset) []byte {
	data := asset.Data()
	buf := fmt.Sprintf("%v%v%v%v%v",
		data.Id,
		data.Name,
		data.Symbol,
		data.TotalSupply,
		data.Decimals)
	return hash([]byte(buf))
}

// GetAssetHash return an hash of the given asset to be used
// signed to validate the asset on the vega chain
func (a *Assets) AssetHash(assetID uint64) ([]byte, error) {
	asset, ok := a.assets[assetID]
	if ok {
		return a.assetHash(asset), nil
	}
	asset, ok = a.pendingAssets[assetID]
	if ok {
		return a.assetHash(asset), nil
	}
	return nil, ErrAssetDoesNotExist
}

func hash(key []byte) []byte {
	hasher := sha3.New256()
	hasher.Write([]byte(key))
	return hasher.Sum(nil)
}
