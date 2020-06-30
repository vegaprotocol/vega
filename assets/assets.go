package assets

import (
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/assets/builtin"
	"code.vegaprotocol.io/vega/assets/common"
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
	SignBridgeWhitelisting() ([]byte, []byte, error)
	// ensure on the target chain that withdrawal on funds
	// happended
	ValidateWithdrawal() error // SignWithdrawal
	SignWithdrawal() ([]byte, error)
	// ensure on the target chain that a deposit really did happen
	ValidateDeposit() error

	String() string
}

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/assets TimeService
type TimeService interface {
	NotifyOnTick(f func(time.Time))
}

type NodeWallet interface {
	Get(chain nodewallet.Blockchain) (nodewallet.Wallet, bool)
}

type Service struct {
	log *logging.Logger
	cfg Config

	// id to asset
	// these assets exists and have been save
	assets map[string]Asset

	// this is a list of pending asset which are currently going through
	// proposal, they can later on be promoted to the asset lists once
	// the proposal is accepted by both the nodes and the users
	pendingAssets map[string]Asset

	// map of reference to proposal id
	// use to find back an asset when the governance process
	// is still ongoing
	refs map[string]string

	nw NodeWallet

	idgen *IDgenerator
}

func New(log *logging.Logger, cfg Config, nw NodeWallet, ts TimeService) (*Service, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	s := &Service{
		log:           log,
		cfg:           cfg,
		assets:        map[string]Asset{},
		pendingAssets: map[string]Asset{},
		refs:          map[string]string{},
		nw:            nw,
		idgen:         NewIDGen(),
	}
	ts.NotifyOnTick(s.onTick)
	return s, nil
}

func (a *Service) onTick(t time.Time) {
	// update block time on id generator
	a.idgen.NewBatch()
}

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

// NewAsset add a new asset to the pending list of assets
// the ref is the reference of proposal which submitted the new asset
// returns the assetID and an error
func (s *Service) NewAsset(ref string, assetSrc *types.AssetSource) (string, error) {
	// make a new asset id
	assetID := s.idgen.NewID()
	src := assetSrc.Source
	switch assetSrcImpl := src.(type) {
	case *types.AssetSource_BuiltinAsset:
		s.pendingAssets[assetID] = builtin.New(assetID, assetSrcImpl.BuiltinAsset)
	case *types.AssetSource_Erc20:
		wal, ok := s.nw.Get(nodewallet.Ethereum)
		if !ok {
			return "", errors.New("missing wallet for ETH")
		}
		asset, err := erc20.New(assetID, assetSrcImpl.Erc20, wal)
		if err != nil {
			return "", err
		}
		s.pendingAssets[assetID] = asset
	default:
		return "", ErrUnknowAssetSource
	}
	// setup the ref lookup table
	s.refs[ref] = assetID

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

func (s *Service) assetHash(asset Asset) []byte {
	data := asset.Data()
	buf := fmt.Sprintf("%v%v%v%v%v",
		data.ID,
		data.Name,
		data.Symbol,
		data.TotalSupply,
		data.Decimals)
	return hash([]byte(buf))
}

func (s *Service) Get(assetID string) (Asset, error) {
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

func (s *Service) GetByRef(ref string) (Asset, error) {
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
