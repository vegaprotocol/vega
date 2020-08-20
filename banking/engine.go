package banking

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/validators"
	"github.com/prometheus/common/log"

	"golang.org/x/crypto/sha3"
)

var (
	ErrWrongAssetTypeUsedInBuiltinAssetChainEvent = errors.New("non builtin asset used for builtin asset chain event")
	ErrWrongAssetUsedForERC20Withdraw             = errors.New("non erc20 asset used for lock withdraw")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_mock.go -package mocks code.vegaprotocol.io/vega/banking Assets
type Assets interface {
	Get(assetID string) (*assets.Asset, error)
	Enable(assetID string) error
}

// Notary ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/notary_mock.go -package mocks code.vegaprotocol.io/vega/processor Notary
type Notary interface {
	StartAggregate(resID string, kind types.NodeSignatureKind) error
	AddSig(ctx context.Context, pubKey []byte, ns types.NodeSignature) ([]types.NodeSignature, bool, error)
	IsSigned(string, types.NodeSignatureKind) ([]types.NodeSignature, bool)
}

// Collateral engine
//go:generate go run github.com/golang/mock/mockgen -destination mocks/collateral_mock.go -package mocks code.vegaprotocol.io/vega/banking Collateral
type Collateral interface {
	Deposit(ctx context.Context, partyID, asset string, amount uint64) error
	Withdraw(ctx context.Context, partyID, asset string, amount uint64) error
	LockFundsForWithdraw(ctx context.Context, partyID, asset string, amount uint64) error
	EnableAsset(ctx context.Context, asset types.Asset) error
}

// ExtResChecker provide foreign chain resources validations
//go:generate go run github.com/golang/mock/mockgen -destination mocks/ext_res_checker_mock.go -package mocks code.vegaprotocol.io/vega/banking ExtResChecker
type ExtResChecker interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
}

// TimeService provide the time of the vega node using the tm time
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/banking TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	NotifyOnTick(func(context.Context, time.Time))
}

const (
	pendingState uint32 = iota
	okState
	rejectedState
)

var (
	defaultValidationDuration = 2 * time.Hour
)

type Engine struct {
	log       *logging.Logger
	col       Collateral
	erc       ExtResChecker
	notary    Notary
	assets    Assets
	assetActs map[string]*assetAction
	tsvc      TimeService
	// top       Topology
	seen map[txRef]struct{}
}

func New(log *logging.Logger, col Collateral, erc ExtResChecker, tsvc TimeService, assets Assets, notary Notary) (e *Engine) {
	defer func() { tsvc.NotifyOnTick(e.OnTick) }()
	return &Engine{
		log:       log,
		col:       col,
		erc:       erc,
		assetActs: map[string]*assetAction{},
		tsvc:      tsvc,
		assets:    assets,
		seen:      map[txRef]struct{}{},
		notary:    notary,
	}
}

func (e *Engine) onCheckDone(i interface{}, valid bool) {
	aa, ok := i.(*assetAction)
	if !ok {
		return
	}

	var newState = rejectedState
	if valid {
		newState = okState
	}
	atomic.StoreUint32(&aa.state, newState)
}

func (e *Engine) EnableBuiltinAsset(ctx context.Context, assetID string) error {
	return e.finalizeAssetList(ctx, assetID)
}

func (e *Engine) WithdrawalBuiltinAsset(ctx context.Context, party, assetID string, amount uint64) error {
	asset, err := e.assets.Get(assetID)
	if err != nil {
		e.log.Error("unable to get asset by id",
			logging.String("asset-id", assetID),
			logging.Error(err))
		return err
	}
	if !asset.IsBuiltinAsset() {
		return ErrWrongAssetTypeUsedInBuiltinAssetChainEvent
	}
	if err := e.col.LockFundsForWithdraw(ctx, party, assetID, amount); err != nil {
		e.log.Error("cannot withdraw asset for party",
			logging.String("party-id", party),
			logging.String("asset-id", assetID),
			logging.Uint64("amount", amount),
			logging.Error(err))
		return err
	}

	return e.finalizeWithdrawal(ctx, party, assetID, amount)
}

func (e *Engine) DepositBuiltinAsset(d *types.BuiltinAssetDeposit, nonce uint64) error {
	now, _ := e.tsvc.GetTimeNow()
	asset, err := e.assets.Get(d.VegaAssetID)
	if err != nil {
		e.log.Error("unable to get asset by id",
			logging.String("asset-id", d.VegaAssetID),
			logging.Error(err))
		return err
	}
	if !asset.IsBuiltinAsset() {
		return ErrWrongAssetTypeUsedInBuiltinAssetChainEvent
	}

	aa := &assetAction{
		id:       id(d, nonce),
		state:    pendingState,
		builtinD: d,
		asset:    asset,
	}
	e.assetActs[aa.id] = aa
	return e.erc.StartCheck(aa, e.onCheckDone, now.Add(defaultValidationDuration))
}

func (e *Engine) EnableERC20(ctx context.Context, al *types.ERC20AssetList, blockNumber, txIndex uint64) error {
	now, _ := e.tsvc.GetTimeNow()
	asset, _ := e.assets.Get(al.VegaAssetID)
	aa := &assetAction{
		id:          id(al, uint64(now.UnixNano())),
		state:       pendingState,
		erc20AL:     al,
		asset:       asset,
		blockNumber: blockNumber,
		txIndex:     txIndex,
	}
	e.assetActs[aa.id] = aa
	return e.erc.StartCheck(aa, e.onCheckDone, now.Add(defaultValidationDuration))
}

func (e *Engine) DepositERC20(d *types.ERC20Deposit, blockNumber, txIndex uint64) error {
	now, _ := e.tsvc.GetTimeNow()
	asset, err := e.assets.Get(d.VegaAssetID)
	if err != nil {
		e.log.Error("unable to get asset by id",
			logging.String("asset-id", d.VegaAssetID),
			logging.Error(err))
		return err
	}
	aa := &assetAction{
		id:          id(d, uint64(now.UnixNano())),
		state:       pendingState,
		erc20D:      d,
		asset:       asset,
		blockNumber: blockNumber,
		txIndex:     txIndex,
	}
	e.assetActs[aa.id] = aa
	return e.erc.StartCheck(aa, e.onCheckDone, now.Add(defaultValidationDuration))
}

func (e *Engine) LockWithdrawalERC20(ctx context.Context, party, assetID string, amount uint64) error {
	asset, err := e.assets.Get(assetID)
	if err != nil {
		e.log.Error("unable to get asset by id",
			logging.String("asset-id", assetID),
			logging.Error(err))
		return err
	}
	if !asset.IsERC20() {
		return ErrWrongAssetUsedForERC20Withdraw
	}

	// try to lock the funds
	if err := e.col.LockFundsForWithdraw(ctx, party, assetID, amount); err != nil {
		e.log.Error("cannot withdraw asset for party",
			logging.String("party-id", party),
			logging.String("asset-id", assetID),
			logging.Uint64("amount", amount),
			logging.Error(err))
		return err
	}
	return nil
}

func (e *Engine) OnTick(ctx context.Context, t time.Time) {
	for k, v := range e.assetActs {
		state := atomic.LoadUint32(&v.state)
		if state == pendingState {
			continue
		}
		switch state {
		case okState:
			// check if this transaction have been seen before then
			if _, ok := e.seen[v.ref]; ok {
				// do nothing of this transaction, just display an error
				log.Error("chain event reference a transaction already processed",
					logging.String("asset-class", string(v.ref.asset)),
					logging.String("tx-hash", v.ref.hash),
					logging.String("action", v.String()))
			} else {
				// first time we seen this transaction, let's add iter
				e.seen[v.ref] = struct{}{}
				if err := e.finalizeAction(ctx, v); err != nil {
					e.log.Error("unable to finalize action",
						logging.String("action", v.String()),
						logging.Error(err))
				}
			}

		case rejectedState:
			e.log.Error("network rejected banking action",
				logging.String("action", v.String()))
		}
		// delete anyway the action
		// at this point the action was either rejected, so we do no need
		// need to keep waiting for its validation, or accepted. in the case
		// it's accepted it's then sent to the given collateral function
		// (deposit, withdraw, whitelist), then an error can occur down the
		// line in the collateral but if that happend there's no way for
		// us to recover for this event, so we have no real reason to keep
		// it in memory
		delete(e.assetActs, k)
	}
}

func (e *Engine) finalizeAction(ctx context.Context, aa *assetAction) error {
	switch {
	case aa.IsBuiltinAssetDeposit():
		return e.finalizeDeposit(ctx, aa.deposit)
	case aa.IsERC20Deposit():
		// here the event queue send us a 0x... pubkey
		// we do the slice operation to remove it ([2:]
		aa.deposit.partyID = aa.deposit.partyID[2:]
		return e.finalizeDeposit(ctx, aa.deposit)
	case aa.IsERC20AssetList():
		return e.finalizeAssetList(ctx, aa.erc20AL.VegaAssetID)
	default:
		return ErrUnknownAssetAction
	}
}

func (e *Engine) finalizeDeposit(ctx context.Context, d *deposit) error {
	return e.col.Deposit(ctx, d.partyID, d.assetID, d.amount)
}

func (e *Engine) finalizeWithdrawal(ctx context.Context, party, asset string, amount uint64) error {
	return e.col.Withdraw(ctx, party, asset, amount)
}

func (e *Engine) finalizeAssetList(ctx context.Context, assetID string) error {
	asset, err := e.assets.Get(assetID)
	if err != nil {
		e.log.Error("invalid asset id used to finalise asset list",
			logging.Error(err),
			logging.String("asset-id", assetID))
		return nil
	}
	if err := e.assets.Enable(assetID); err != nil {
		e.log.Error("unable to enable asset",
			logging.Error(err),
			logging.String("asset-id", assetID))
		return err
	}
	passet := asset.ProtoAsset()
	return e.col.EnableAsset(ctx, *passet)

}

type HasVegaAssetID interface {
	GetVegaAssetID() string
}

func id(s fmt.Stringer, nonce uint64) string {
	hasher := sha3.New256()
	hasher.Write([]byte(fmt.Sprintf("%v%v", s.String(), nonce)))
	return hex.EncodeToString(hasher.Sum(nil))
}
