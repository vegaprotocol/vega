package banking

import (
	"context"
	"errors"
	"math/big"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/validators"
)

const (
	// this is temporarily used until we remove expiry completely
	// make the expiry 2 years, which will outlive anyway any
	// vega network at first.
	// 24 hours * 365 * days * 2 years.
	withdrawalsDefaultExpiry = 24 * 365 * 2 * time.Hour
)

var (
	ErrWrongAssetTypeUsedInBuiltinAssetChainEvent = errors.New("non builtin asset used for builtin asset chain event")
	ErrWrongAssetTypeUsedInERC20ChainEvent        = errors.New("non ERC20 for ERC20 chain event")
	ErrWrongAssetUsedForERC20Withdraw             = errors.New("non erc20 asset used for lock withdraw")
	ErrInvalidWithdrawalState                     = errors.New("invalid withdrawal state")
	ErrNotMatchingWithdrawalForReference          = errors.New("invalid reference for withdrawal chain event")
	ErrWithdrawalNotReady                         = errors.New("withdrawal not ready")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/assets_mock.go -package mocks code.vegaprotocol.io/vega/banking Assets
type Assets interface {
	Get(assetID string) (*assets.Asset, error)
	Enable(assetID string) error
}

// Notary ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/notary_mock.go -package mocks code.vegaprotocol.io/vega/banking Notary
type Notary interface {
	StartAggregate(resID string, kind types.NodeSignatureKind, signature []byte)
	IsSigned(ctx context.Context, id string, kind types.NodeSignatureKind) ([]types.NodeSignature, bool)
	OfferSignatures(kind types.NodeSignatureKind, f func(resources string) []byte)
}

// Collateral engine
//go:generate go run github.com/golang/mock/mockgen -destination mocks/collateral_mock.go -package mocks code.vegaprotocol.io/vega/banking Collateral
type Collateral interface {
	Deposit(ctx context.Context, party, asset string, amount *num.Uint) (*types.TransferResponse, error)
	Withdraw(ctx context.Context, party, asset string, amount *num.Uint) (*types.TransferResponse, error)
	EnableAsset(ctx context.Context, asset types.Asset) error
	GetPartyGeneralAccount(party, asset string) (*types.Account, error)
}

// Witness provide foreign chain resources validations
//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/banking Witness
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
	RestoreResource(validators.Resource, func(interface{}, bool)) error
}

// TimeService provide the time of the vega node using the tm time
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/banking TimeService
type TimeService interface {
	NotifyOnTick(func(context.Context, time.Time))
}

// Topology ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/topology_mock.go -package mocks code.vegaprotocol.io/vega/banking Topology
type Topology interface {
	IsValidator() bool
}

// Broker - the event bus.
type Broker interface {
	Send(e events.Event)
}

const (
	pendingState uint32 = iota
	okState
	rejectedState
)

var defaultValidationDuration = 2 * time.Hour

type Engine struct {
	cfg     Config
	log     *logging.Logger
	broker  Broker
	col     Collateral
	witness Witness
	notary  Notary
	assets  Assets
	top     Topology

	assetActs     map[string]*assetAction
	seen          map[txRef]struct{}
	withdrawals   map[string]withdrawalRef
	withdrawalCnt *big.Int
	deposits      map[string]*types.Deposit

	currentTime     time.Time
	mu              sync.RWMutex
	bss             *bankingSnapshotState
	keyToSerialiser map[string]func() ([]byte, error)
}

type withdrawalRef struct {
	w   *types.Withdrawal
	ref *big.Int
}

func New(
	log *logging.Logger,
	cfg Config,
	col Collateral,
	witness Witness,
	tsvc TimeService,
	assets Assets,
	notary Notary,
	broker Broker,
	top Topology,
) (e *Engine) {
	defer func() { tsvc.NotifyOnTick(e.OnTick) }()
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	e = &Engine{
		cfg:           cfg,
		log:           log,
		broker:        broker,
		col:           col,
		witness:       witness,
		assets:        assets,
		notary:        notary,
		top:           top,
		assetActs:     map[string]*assetAction{},
		seen:          map[txRef]struct{}{},
		withdrawals:   map[string]withdrawalRef{},
		deposits:      map[string]*types.Deposit{},
		withdrawalCnt: big.NewInt(0),
		bss: &bankingSnapshotState{
			changed:    map[string]bool{withdrawalsKey: true, depositsKey: true, seenKey: true, assetActionsKey: true},
			hash:       map[string][]byte{},
			serialised: map[string][]byte{},
		},
		keyToSerialiser: map[string]func() ([]byte, error){},
	}

	e.keyToSerialiser[withdrawalsKey] = e.serialiseWithdrawals
	e.keyToSerialiser[depositsKey] = e.serialiseDeposits
	e.keyToSerialiser[seenKey] = e.serialiseSeen
	e.keyToSerialiser[assetActionsKey] = e.serialiseAssetActions
	return e
}

// ReloadConf updates the internal configuration.
func (e *Engine) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.cfg = cfg
}

func (e *Engine) OnTick(ctx context.Context, t time.Time) {
	e.mu.Lock()
	e.currentTime = t
	e.mu.Unlock()

	assetActionKeys := make([]string, 0, len(e.assetActs))
	for k := range e.assetActs {
		assetActionKeys = append(assetActionKeys, k)
	}
	sort.Strings(assetActionKeys)

	// iterate over asset actions deterministically
	for _, k := range assetActionKeys {
		v := e.assetActs[k]
		state := atomic.LoadUint32(&v.state)
		if state == pendingState {
			continue
		}

		// get the action reference to ensure it's not
		// a duplicate
		ref := v.getRef()

		switch state {
		case okState:
			// check if this transaction have been seen before then
			if _, ok := e.seen[ref]; ok {
				// do nothing of this transaction, just display an error
				e.log.Error("chain event reference a transaction already processed",
					logging.String("asset-class", string(ref.asset)),
					logging.String("tx-hash", ref.hash),
					logging.String("action", v.String()))
			} else {
				// first time we seen this transaction, let's add iter
				e.seen[ref] = struct{}{}
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
		// (deposit, withdraw, allowlist), then an error can occur down the
		// line in the collateral but if that happened there's no way for
		// us to recover for this event, so we have no real reason to keep
		// it in memory
		delete(e.assetActs, k)
		e.bss.changed[assetActionsKey] = true
	}

	// we may want a dedicated method on the snapshot engine at some
	// point but this will do for now
	// this will be restarting the signatures aggregates
	e.notary.OfferSignatures(
		types.NodeSignatureKindAssetWithdrawal, e.offerERC20NotarySignatures)
}

func (e *Engine) onCheckDone(i interface{}, valid bool) {
	aa, ok := i.(*assetAction)
	if !ok {
		return
	}

	newState := rejectedState
	if valid {
		newState = okState
	}
	atomic.StoreUint32(&aa.state, newState)
}

func (e *Engine) getWithdrawalFromRef(ref *big.Int) (*types.Withdrawal, error) {
	// sort withdraws to check deterministically
	withdrawalsK := make([]string, 0, len(e.withdrawals))
	for k := range e.withdrawals {
		withdrawalsK = append(withdrawalsK, k)
	}

	for _, k := range withdrawalsK {
		v := e.withdrawals[k]
		if v.ref.Cmp(ref) == 0 {
			return v.w, nil
		}
	}

	return nil, ErrNotMatchingWithdrawalForReference
}

func (e *Engine) finalizeAction(ctx context.Context, aa *assetAction) error {
	switch {
	case aa.IsBuiltinAssetDeposit():
		dep := e.deposits[aa.id]
		return e.finalizeDeposit(ctx, dep)
	case aa.IsERC20Deposit():
		dep := e.deposits[aa.id]
		return e.finalizeDeposit(ctx, dep)
	case aa.IsERC20AssetList():
		return e.finalizeAssetList(ctx, aa.erc20AL.VegaAssetID)
	default:
		return ErrUnknownAssetAction
	}
}

func (e *Engine) finalizeAssetList(ctx context.Context, assetID string) error {
	asset, err := e.assets.Get(assetID)
	if err != nil {
		e.log.Error("invalid asset id used to finalise asset list",
			logging.Error(err),
			logging.AssetID(assetID))
		return nil
	}
	if err := e.assets.Enable(assetID); err != nil {
		e.log.Error("unable to enable asset",
			logging.Error(err),
			logging.AssetID(assetID))
		return err
	}
	return e.col.EnableAsset(ctx, *asset.ToAssetType())
}

func (e *Engine) finalizeDeposit(ctx context.Context, d *types.Deposit) error {
	defer func() { e.broker.Send(events.NewDepositEvent(ctx, *d)) }()
	res, err := e.col.Deposit(ctx, d.PartyID, d.Asset, d.Amount)
	if err != nil {
		d.Status = types.DepositStatusCancelled
		return err
	}

	d.Status = types.DepositStatusFinalized
	d.CreditDate = e.currentTime.UnixNano()
	e.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{res}))
	e.bss.changed[depositsKey] = true
	return nil
}

func (e *Engine) finalizeWithdraw(
	ctx context.Context, w *types.Withdrawal) error {
	// always send the withdrawal event
	defer func() { e.broker.Send(events.NewWithdrawalEvent(ctx, *w)) }()

	res, err := e.col.Withdraw(ctx, w.PartyID, w.Asset, w.Amount.Clone())
	if err != nil {
		w.Status = types.WithdrawalStatusRejected
		return err
	}

	w.Status = types.WithdrawalStatusFinalized
	e.bss.changed[withdrawalsKey] = true
	e.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{res}))
	return nil
}

func (e *Engine) newWithdrawal(
	id, partyID, asset string,
	amount *num.Uint,
	expirationDate time.Time,
	wext *types.WithdrawExt,
) (w *types.Withdrawal, ref *big.Int) {
	partyID = strings.TrimPrefix(partyID, "0x")
	asset = strings.TrimPrefix(asset, "0x")
	// reference needs to be an int, deterministic for the contracts
	ref = big.NewInt(0).Add(e.withdrawalCnt, big.NewInt(e.currentTime.Unix()))
	e.withdrawalCnt.Add(e.withdrawalCnt, big.NewInt(1))
	w = &types.Withdrawal{
		ID:             id,
		Status:         types.WithdrawalStatusOpen,
		PartyID:        partyID,
		Asset:          asset,
		Amount:         amount,
		ExpirationDate: expirationDate.Unix(),
		Ext:            wext,
		CreationDate:   e.currentTime.UnixNano(),
		Ref:            ref.String(),
	}
	e.bss.changed[withdrawalsKey] = true
	return
}

func (e *Engine) newDeposit(
	id, partyID, asset string,
	amount *num.Uint,
	txHash string,
) *types.Deposit {
	partyID = strings.TrimPrefix(partyID, "0x")
	asset = strings.TrimPrefix(asset, "0x")
	e.bss.changed[depositsKey] = true
	e.bss.changed[assetActionsKey] = true
	return &types.Deposit{
		ID:           id,
		Status:       types.DepositStatusOpen,
		PartyID:      partyID,
		Asset:        asset,
		Amount:       amount,
		CreationDate: e.currentTime.UnixNano(),
		TxHash:       txHash,
	}
}
