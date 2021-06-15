package banking

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/validators"
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
	StartAggregate(resID string, kind commandspb.NodeSignatureKind) error
	SendSignature(ctx context.Context, id string, sig []byte, kind commandspb.NodeSignatureKind) error
	IsSigned(ctx context.Context, id string, kind commandspb.NodeSignatureKind) ([]commandspb.NodeSignature, bool)
}

// Collateral engine
//go:generate go run github.com/golang/mock/mockgen -destination mocks/collateral_mock.go -package mocks code.vegaprotocol.io/vega/banking Collateral
type Collateral interface {
	Deposit(ctx context.Context, partyID, asset string, amount *num.Uint) (*types.TransferResponse, error)
	Withdraw(ctx context.Context, partyID, asset string, amount *num.Uint) (*types.TransferResponse, error)
	LockFundsForWithdraw(ctx context.Context, partyID, asset string, amount *num.Uint) (*types.TransferResponse, error)
	EnableAsset(ctx context.Context, asset types.Asset) error
	HasBalance(party string) bool
}

// Witness provide foreign chain resources validations
//go:generate go run github.com/golang/mock/mockgen -destination mocks/witness_mock.go -package mocks code.vegaprotocol.io/vega/banking Witness
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
}

// TimeService provide the time of the vega node using the tm time
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/banking TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
	NotifyOnTick(func(context.Context, time.Time))
}

// Broker - the event bus
type Broker interface {
	Send(e events.Event)
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
	cfg           Config
	log           *logging.Logger
	broker        Broker
	col           Collateral
	witness       Witness
	notary        Notary
	assets        Assets
	assetActs     map[string]*assetAction
	seen          map[txRef]struct{}
	withdrawals   map[string]withdrawalRef
	withdrawalCnt *big.Int
	deposits      map[string]*types.Deposit

	currentTime time.Time
	mu          sync.RWMutex
}

type withdrawalRef struct {
	w   *types.Withdrawal
	ref *big.Int
}

func New(log *logging.Logger, cfg Config, col Collateral, witness Witness, tsvc TimeService, assets Assets, notary Notary, broker Broker) (e *Engine) {
	defer func() { tsvc.NotifyOnTick(e.OnTick) }()
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Engine{
		cfg:           cfg,
		log:           log,
		broker:        broker,
		col:           col,
		witness:       witness,
		assetActs:     map[string]*assetAction{},
		assets:        assets,
		seen:          map[txRef]struct{}{},
		notary:        notary,
		withdrawals:   map[string]withdrawalRef{},
		withdrawalCnt: big.NewInt(0),
		deposits:      map[string]*types.Deposit{},
	}
}

// ReloadConf updates the internal configuration
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

func (e *Engine) HasBalance(party string) bool {
	return e.col.HasBalance(party)
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

func (e *Engine) WithdrawalBuiltinAsset(ctx context.Context, id, party, assetID string, amnt uint64) error {
	amount := num.NewUint(amnt)
	asset, err := e.assets.Get(assetID)
	if err != nil {
		e.log.Error("unable to get asset by id",
			logging.AssetID(assetID),
			logging.Error(err))
		return err
	}
	if !asset.IsBuiltinAsset() {
		return ErrWrongAssetTypeUsedInBuiltinAssetChainEvent
	}

	w, ref, err := e.newWithdrawal(id, party, assetID, amount, time.Time{}, nil)
	if err != nil {
		return err
	}
	e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
	e.withdrawals[w.ID] = withdrawalRef{w, ref}
	res, err := e.col.LockFundsForWithdraw(ctx, party, assetID, amount)
	if err != nil {
		w.Status = types.Withdrawal_STATUS_CANCELLED
		e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
		e.withdrawals[w.ID] = withdrawalRef{w, ref}
		e.log.Error("cannot withdraw asset for party",
			logging.PartyID(party),
			logging.AssetID(assetID),
			logging.BigUint("amount", amount),
			logging.Error(err))
		return err
	}
	w.Status = types.Withdrawal_STATUS_FINALIZED
	e.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{res}))
	e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
	e.withdrawals[w.ID] = withdrawalRef{w, ref}

	return e.finalizeWithdrawal(ctx, party, assetID, amount)
}

func (e *Engine) DepositBuiltinAsset(
	ctx context.Context, d *types.BuiltinAssetDeposit, id string, nonce uint64) error {
	now := e.currentTime
	dep, err := e.newDeposit(id, d.PartyID, d.VegaAssetID, d.Amount, "") // no hash
	if err != nil {
		return err
	}
	e.broker.Send(events.NewDepositEvent(ctx, *dep))
	asset, err := e.assets.Get(d.VegaAssetID)
	if err != nil {
		dep.Status = types.Deposit_STATUS_CANCELLED
		e.broker.Send(events.NewDepositEvent(ctx, *dep))
		e.log.Error("unable to get asset by id",
			logging.AssetID(d.VegaAssetID),
			logging.Error(err))
		return err
	}
	if !asset.IsBuiltinAsset() {
		dep.Status = types.Deposit_STATUS_CANCELLED
		e.broker.Send(events.NewDepositEvent(ctx, *dep))
		return ErrWrongAssetTypeUsedInBuiltinAssetChainEvent
	}

	aa := &assetAction{
		id:       dep.ID,
		state:    pendingState,
		builtinD: d,
		asset:    asset,
	}
	e.assetActs[aa.id] = aa
	e.deposits[dep.ID] = dep
	return e.witness.StartCheck(aa, e.onCheckDone, now.Add(defaultValidationDuration))
}

func (e *Engine) EnableERC20(ctx context.Context, al *types.ERC20AssetList, blockNumber, txIndex uint64, txHash string) error {
	now := e.currentTime
	asset, _ := e.assets.Get(al.VegaAssetId)
	aa := &assetAction{
		id:          id(al, uint64(now.UnixNano())),
		state:       pendingState,
		erc20AL:     al,
		asset:       asset,
		blockNumber: blockNumber,
		txIndex:     txIndex,
		hash:        txHash,
	}
	e.assetActs[aa.id] = aa
	return e.witness.StartCheck(aa, e.onCheckDone, now.Add(defaultValidationDuration))
}

func (e *Engine) DepositERC20(ctx context.Context, d *types.ERC20Deposit, id string, blockNumber, txIndex uint64, txHash string) error {
	now := e.currentTime
	dep, err := e.newDeposit(id, d.TargetPartyID, d.VegaAssetID, d.Amount, txHash)
	if err != nil {
		return err
	}
	e.broker.Send(events.NewDepositEvent(ctx, *dep))
	asset, err := e.assets.Get(d.VegaAssetID)
	if err != nil {
		dep.Status = types.Deposit_STATUS_CANCELLED
		e.broker.Send(events.NewDepositEvent(ctx, *dep))
		e.log.Error("unable to get asset by id",
			logging.AssetID(d.VegaAssetID),
			logging.Error(err))
		return err
	}
	if !asset.IsERC20() {
		dep.Status = types.Deposit_STATUS_CANCELLED
		e.broker.Send(events.NewDepositEvent(ctx, *dep))
		return ErrWrongAssetTypeUsedInERC20ChainEvent
	}
	aa := &assetAction{
		id:          dep.ID,
		state:       pendingState,
		erc20D:      d,
		asset:       asset,
		blockNumber: blockNumber,
		txIndex:     txIndex,
		hash:        txHash,
	}
	e.assetActs[aa.id] = aa
	e.deposits[dep.ID] = dep
	return e.witness.StartCheck(aa, e.onCheckDone, now.Add(defaultValidationDuration))
}

func (e *Engine) WithdrawalERC20(ctx context.Context, w *types.ERC20Withdrawal, blockNumber, txIndex uint64, txHash string) error {
	now := e.currentTime
	asset, err := e.assets.Get(w.VegaAssetId)
	if err != nil {
		e.log.Debug("unable to get asset by id",
			logging.AssetID(w.VegaAssetId),
			logging.Error(err))
		return err
	}

	// check straight away if the withdrawal is signed
	nonce := &big.Int{}
	nonce.SetString(w.ReferenceNonce, 10)
	withd, err := e.getWithdrawalFromRef(nonce)
	if err != nil {
		return err
	}
	if withd.Status != types.Withdrawal_STATUS_OPEN {
		return ErrInvalidWithdrawalState
	}
	withd.TxHash = txHash
	if _, ok := e.notary.IsSigned(ctx, withd.ID, commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL); !ok {
		return ErrWithdrawalNotReady
	}

	aa := &assetAction{
		id:          id(w, uint64(now.UnixNano())),
		state:       pendingState,
		erc20W:      w,
		asset:       asset,
		blockNumber: blockNumber,
		txIndex:     txIndex,
		hash:        txHash,
		withdrawal: &withdrawal{
			nonce: nonce,
		},
	}
	e.assetActs[aa.id] = aa
	return e.witness.StartCheck(aa, e.onCheckDone, now.Add(defaultValidationDuration))
}

func (e *Engine) LockWithdrawalERC20(ctx context.Context, id, party, assetID string, amount *num.Uint, ext *types.Erc20WithdrawExt) error {
	asset, err := e.assets.Get(assetID)
	if err != nil {
		e.log.Debug("unable to get asset by id",
			logging.AssetID(assetID),
			logging.Error(err))
		return err
	}
	if !asset.IsERC20() {
		return ErrWrongAssetUsedForERC20Withdraw
	}

	now := e.currentTime
	expiry := now.Add(e.cfg.WithdrawalExpiry.Duration)
	wext := &types.WithdrawExt{
		Ext: &types.WithdrawExt_Erc20{
			Erc20: ext,
		},
	}
	w, ref, err := e.newWithdrawal(id, party, assetID, amount, expiry, wext)
	if err != nil {
		return err
	}
	e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
	e.withdrawals[w.ID] = withdrawalRef{w, ref}
	// try to lock the funds
	res, err := e.col.LockFundsForWithdraw(ctx, party, assetID, amount)
	if err != nil {
		w.Status = types.Withdrawal_STATUS_CANCELLED
		e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
		e.withdrawals[w.ID] = withdrawalRef{w, ref}
		e.log.Debug("cannot withdraw asset for party",
			logging.PartyID(party),
			logging.AssetID(assetID),
			logging.BigUint("amount", amount),
			logging.Error(err))
		return err
	}
	e.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{res}))

	// we were able to lock the funds, then we can send the vote through the network
	if err := e.notary.StartAggregate(w.ID, commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL); err != nil {
		w.Status = types.Withdrawal_STATUS_CANCELLED
		e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
		e.withdrawals[w.ID] = withdrawalRef{w, ref}
		e.log.Error("unable to start aggregating signature for the withdrawal",
			logging.WithdrawalID(w.ID),
			logging.PartyID(party),
			logging.AssetID(assetID),
			logging.BigUint("amount", amount),
			logging.Error(err))
		return err
	}

	// then get the signature for the withdrawal and send it
	erc20asset, _ := asset.ERC20() // no check error as we checked earlier we had an erc20 asset.
	_, sig, err := erc20asset.SignWithdrawal(amount.Uint64(), w.ExpirationDate, ext.GetReceiverAddress(), ref)
	if err != nil {
		// we don't cancel it here
		// we may not be able to sign for some reason, but other may be able
		// and we would aggregate enough signature
		e.log.Error("unable to sign withdrawal",
			logging.WithdrawalID(w.ID),
			logging.PartyID(party),
			logging.AssetID(assetID),
			logging.BigUint("amount", amount),
			logging.Error(err))
		return err
	}

	err = e.notary.SendSignature(
		ctx, w.ID, sig, commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL)
	if err != nil {
		// we don't cancel it here
		// we may not be able to sign for some reason, but other may be able
		// and we would aggregate enough signature
		e.log.Error("unable to send node signature",
			logging.WithdrawalID(w.ID),
			logging.PartyID(party),
			logging.AssetID(assetID),
			logging.BigUint("amount", amount),
			logging.Error(err))
		return err
	}

	return nil
}

func (e *Engine) OnTick(ctx context.Context, t time.Time) {
	e.mu.Lock()
	e.currentTime = t
	e.mu.Unlock()
	for k, v := range e.assetActs {
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
	}
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
		return e.finalizeAssetList(ctx, aa.erc20AL.VegaAssetId)
	case aa.IsERC20Withdrawal():
		w, err := e.getWithdrawalFromRef(aa.withdrawal.nonce)
		if err != nil {
			// Nothing to do, withdrawal does not exists
			return err
		}
		if w.Status != types.Withdrawal_STATUS_OPEN {
			// withdrawal was already canceled or finalized
			return ErrInvalidWithdrawalState
		}
		now := e.currentTime
		// update with finalize time + tx hash
		w.Status = types.Withdrawal_STATUS_FINALIZED
		w.WithdrawalDate = now.UnixNano()
		e.broker.Send(events.NewWithdrawalEvent(ctx, *w))
		e.withdrawals[w.ID] = withdrawalRef{w, aa.withdrawal.nonce}
		return e.finalizeWithdrawal(ctx, w.PartyID, w.Asset, w.Amount)
	default:
		return ErrUnknownAssetAction
	}
}

func (e *Engine) getWithdrawalFromRef(ref *big.Int) (*types.Withdrawal, error) {
	for _, v := range e.withdrawals {
		if v.ref.Cmp(ref) == 0 {

			return v.w, nil
		}
	}

	return nil, ErrNotMatchingWithdrawalForReference
}

func (e *Engine) finalizeDeposit(ctx context.Context, d *types.Deposit) error {
	d.Status = types.Deposit_STATUS_FINALIZED
	d.CreditDate = e.currentTime.UnixNano()
	e.broker.Send(events.NewDepositEvent(ctx, *d))
	res, err := e.col.Deposit(ctx, d.PartyID, d.Asset, d.Amount)
	if err != nil {
		return err
	}
	e.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{res}))
	return nil
}

func (e *Engine) finalizeWithdrawal(ctx context.Context, party, asset string, amount *num.Uint) error {
	res, err := e.col.Withdraw(ctx, party, asset, amount)
	if err != nil {
		return err
	}
	e.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{res}))
	return nil
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

func (e *Engine) newWithdrawal(
	id, partyID, asset string,
	amount *num.Uint,
	expirationDate time.Time,
	wext *types.WithdrawExt,
) (w *types.Withdrawal, ref *big.Int, err error) {
	partyID = strings.TrimPrefix(partyID, "0x")
	asset = strings.TrimPrefix(asset, "0x")
	ref = big.NewInt(0).Add(e.withdrawalCnt, big.NewInt(e.currentTime.Unix()))
	e.withdrawalCnt.Add(e.withdrawalCnt, big.NewInt(1))
	w = &types.Withdrawal{
		ID:             id,
		Status:         types.Withdrawal_STATUS_OPEN,
		PartyID:        partyID,
		Asset:          asset,
		Amount:         amount,
		ExpirationDate: expirationDate.Unix(),
		Ext:            wext,
		CreationDate:   e.currentTime.UnixNano(),
		Ref:            ref.String(),
	}
	return
}

func (e *Engine) newDeposit(
	id, partyID, asset string, amount *num.Uint, txHash string,
) (*types.Deposit, error) {
	partyID = strings.TrimPrefix(partyID, "0x")
	asset = strings.TrimPrefix(asset, "0x")
	return &types.Deposit{
		ID:           id,
		Status:       types.Deposit_STATUS_OPEN,
		PartyID:      partyID,
		Asset:        asset,
		Amount:       amount,
		CreationDate: e.currentTime.UnixNano(),
		TxHash:       txHash,
	}, nil
}

type HasVegaAssetID interface {
	GetVegaAssetID() string
}

func id(s fmt.Stringer, nonce uint64) string {
	return hex.EncodeToString(crypto.Hash([]byte(fmt.Sprintf("%v%v", s.String(), nonce))))
}
