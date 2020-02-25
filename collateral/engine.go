package collateral

import (
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

const (
	initialAccountSize = 4096
	// use weird character here, maybe non-displayable ones in the future
	// if needed
	systemOwner = "*"
	noMarket    = "!"
)

var (
	// ErrSystemAccountsMissing signals that a system account is missing, which may means that the
	// collateral engine have not been initialised properly
	ErrSystemAccountsMissing = errors.New("system accounts missing for collateral engine to work")
	// ErrTraderAccountsMissing signals that the accounts for this trader do not exists
	ErrTraderAccountsMissing = errors.New("trader accounts missing, cannot collect")
	// ErrAccountDoesNotExist signals that an account par of a transfer do not exists
	ErrAccountDoesNotExist                     = errors.New("account do not exists")
	ErrNoGeneralAccountWhenCreateMarginAccount = errors.New("party general account missing when trying to create a margin account")
	ErrMinAmountNotReached                     = errors.New("unable to reach minimum amount transfer")
	ErrInvalidTransfersBatch                   = errors.New("Number of debit and credit transfers do not match up")
)

// AccountBuffer ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_buffer_mock.go -package mocks code.vegaprotocol.io/vega/collateral AccountBuffer
type AccountBuffer interface {
	Add(types.Account)
}

// LossSocializationBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/loss_socialization_buf_mock.go -package mocks code.vegaprotocol.io/vega/collateral LossSocializationBuf
type LossSocializationBuf interface {
	Add([]events.LossSocialization)
	Flush()
}

// Engine is handling the power of the collateral
type Engine struct {
	Config
	log   *logging.Logger
	cfgMu sync.Mutex

	accs       map[string]*types.Account
	buf        AccountBuffer
	lossSocBuf LossSocializationBuf
	// could be a unix.Time but storing it like this allow us to now time.UnixNano() all the time
	currentTime int64

	idbuf []byte
}

// New instantiates a new collateral engine
func New(log *logging.Logger, conf Config, buf AccountBuffer, lossSocBuf LossSocializationBuf, now time.Time) (*Engine, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())
	return &Engine{
		log:         log,
		Config:      conf,
		accs:        make(map[string]*types.Account, initialAccountSize),
		buf:         buf,
		currentTime: now.UnixNano(),
		idbuf:       make([]byte, 256),
		lossSocBuf:  lossSocBuf,
	}, nil
}

// OnChainTimeUpdate is used to be specified as a callback in over services
// in order to be called when the chain time is updated (basically EndBlock)
func (e *Engine) OnChainTimeUpdate(t time.Time) {
	e.currentTime = t.UnixNano()
}

// ReloadConf updates the internal configuration of the collateral engine
func (e *Engine) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.cfgMu.Lock()
	e.Config = cfg
	e.cfgMu.Unlock()
}

// this func uses named returns because it makes body of the func look clearer
func (e *Engine) getSystemAccounts(marketID, asset string) (settle, insurance *types.Account, err error) {

	insID := e.accountID(marketID, systemOwner, asset, types.AccountType_INSURANCE)
	setID := e.accountID(marketID, systemOwner, asset, types.AccountType_SETTLEMENT)

	if insurance, err = e.GetAccountByID(insID); err != nil {
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("missing system account",
				logging.String("asset", asset),
				logging.String("id", insID),
				logging.String("market", marketID),
				logging.Error(err),
			)
		}
		err = ErrSystemAccountsMissing
		return
	}

	if settle, err = e.GetAccountByID(setID); err != nil {
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("missing system account",
				logging.String("asset", asset),
				logging.String("id", setID),
				logging.String("market", marketID),
				logging.Error(err),
			)
		}
		err = ErrSystemAccountsMissing
	}

	return
}

// Part of the spec says that a batch of transfers must have an equal amount of debit and credit actions
// This function will check the batch for this rule and return an error if not met
func (e *Engine) validateTransfers(transfers []*types.Transfer) error {
	var debit uint64
	var credit uint64

	for _, transfer := range transfers {
		if transfer.GetType() == types.TransferType_LOSS ||
			transfer.GetType() == types.TransferType_MTM_LOSS {
			debit += transfer.GetSize()
		} else {
			credit += transfer.GetSize()
		}
	}

	if credit != debit {
		return ErrInvalidTransfersBatch
	}
	return nil
}

// FinalSettlement will process the list of transfer instructed by other engines
// This func currently only expects TransferType_{LOSS,WIN} transfers
// other transfer types have dedicated funcs (MartToMarket, MarginUpdate)
func (e *Engine) FinalSettlement(marketID string, transfers []*types.Transfer) ([]*types.TransferResponse, error) {
	// stop immediately if there aren't any transfers, channels are closed
	if len(transfers) == 0 {
		return nil, nil
	}

	err := e.validateTransfers(transfers)
	if err != nil {
		return nil, err
	}

	responses := make([]*types.TransferResponse, 0, len(transfers))
	asset := transfers[0].Amount.Asset
	// This is where we'll implement everything
	settle, insurance, err := e.getSystemAccounts(marketID, asset)
	if err != nil {
		e.log.Error(
			"Failed to get system accounts required for final settlement",
			logging.Error(err),
		)
		return nil, err
	}
	// create this event, we're not using it, but it's required in getTransferRequests
	mevt := &marginUpdate{}
	// get the component that calculates the loss socialisation etc... if needed
	for _, transfer := range transfers {
		req, err := e.getTransferRequest(transfer, settle, insurance, mevt)
		if err != nil {
			e.log.Error(
				"Failed to build transfer request for event",
				logging.Error(err),
			)
			return nil, err
		}
		// set the amount (this can change the req.Amount value if we entered loss socialisation
		res, err := e.getLedgerEntries(req)
		if err != nil {
			e.log.Error(
				"Failed to transfer funds",
				logging.Error(err),
			)
			return nil, err
		}
		for _, bal := range res.Balances {
			if err := e.UpdateBalance(bal.Account.Id, bal.Balance); err != nil {
				e.log.Error(
					"Could not update the target account in transfer",
					logging.String("account-id", bal.Account.Id),
					logging.Error(err),
				)
				return nil, err
			}
		}
		responses = append(responses, res)
	}
	return responses, nil
}

// Part of the spec says that a batch of transfers must have an equal amount of debit and credit actions
// This function will check the batch for this rule and return an error if not met
func (e *Engine) validateEventTransfers(transfers []events.Transfer) error {
	var current int64

	e.log.Errorf("In validateEventTransfers with %d items\n", len(transfers))

	for i, transfer := range transfers {
		current += transfer.Size()
	}

	if current != 0 {
		return ErrInvalidTransfersBatch
	}
	return nil
}

// MarkToMarket will run the mark to market settlement over a given set of positions
// return ledger move stuff here, too (separate return value, because we need to stream those)
func (e *Engine) MarkToMarket(marketID string, transfers []events.Transfer, asset string) ([]events.Margin, []*types.TransferResponse, error) {
	// stop immediately if there aren't any transfers, channels are closed
	if len(transfers) == 0 {
		return nil, nil, nil
	}

	err := e.validateEventTransfers(transfers)
	if err != nil {
		return nil, nil, err
	}

	marginEvts := make([]events.Margin, 0, len(transfers))
	responses := make([]*types.TransferResponse, 0, len(transfers))

	// This is where we'll implement everything
	settle, insurance, err := e.getSystemAccounts(marketID, asset)
	if err != nil {
		e.log.Error(
			"Failed to get system accounts required for MTM settlement",
			logging.Error(err),
		)
		return nil, nil, err
	}
	// get the component that calculates the loss socialisation etc... if needed

	var (
		winidx          int
		expectCollected int64
	)

	// iterate over transfer unti we get the first win, so we need we accumulated all loss
	for i, evt := range transfers {
		transfer := evt.Transfer()

		// get the state of the accoutns before processing transfers
		// so they can be used in the marginEvt, and to calculate the missing funds
		generalAcc, err := e.GetAccountByID(e.accountID(noMarket, evt.Party(), asset, types.AccountType_GENERAL))
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "general"),
				logging.String("party-id", evt.Party()),
				logging.String("asset", asset))
		}

		marginAcc, err := e.GetAccountByID(e.accountID(settle.MarketID, evt.Party(), asset, types.AccountType_MARGIN))
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "margin"),
				logging.String("party-id", evt.Party()),
				logging.String("asset", asset),
				logging.String("market-id", settle.MarketID))
		}

		marginEvt := &marginUpdate{
			MarketPosition: evt,
			asset:          asset,
			marketID:       settle.MarketID,
		}
		// no transfer needed if transfer is nil, just build the marginUpdate
		if transfer == nil {
			marginEvt.general = generalAcc
			marginEvt.margin = marginAcc
			marginEvts = append(marginEvts, marginEvt)
			continue
		}

		if transfer.Type == types.TransferType_MTM_WIN {
			// we processed all loss break then
			winidx = i
			break
		}

		req, err := e.getTransferRequest(transfer, settle, insurance, marginEvt)
		if err != nil {
			e.log.Error(
				"Failed to build transfer request for event",
				logging.Error(err),
			)
			return nil, nil, err
		}
		// accumulate the expected transfer size
		expectCollected += int64(req.Amount)

		// set the amount (this can change the req.Amount value if we entered loss socialisation
		res, err := e.getLedgerEntries(req)
		if err != nil {
			e.log.Error(
				"Failed to transfer funds",
				logging.Error(err),
			)
			return nil, nil, err
		}

		var amountCollected int64
		// // update the to accounts now
		for _, bal := range res.Balances {
			amountCollected += bal.Balance
			if err := e.IncrementBalance(bal.Account.Id, bal.Balance); err != nil {
				e.log.Error(
					"Could not update the target account in transfer",
					logging.String("account-id", bal.Account.Id),
					logging.Error(err),
				)
				return nil, nil, err
			}
		}

		totalInAccount := marginAcc.Balance + generalAcc.Balance

		// here we check if we were able to collect all monies,
		// if not send an event to notify the plugins
		if totalInAccount < int64(req.Amount) {
			lsevt := &lossSocializationEvt{
				market:     settle.MarketID,
				party:      evt.Party(),
				amountLost: int64(req.Amount) - totalInAccount,
			}

			e.log.Warn("loss socialization missing amount to be collected or used from insurance pool",
				logging.String("party-id", lsevt.party),
				logging.Int64("amount", lsevt.amountLost),
				logging.String("market-id", lsevt.market))

			e.lossSocBuf.Add([]events.LossSocialization{lsevt})
		}

		// updating the accounts stored in the marginEvt
		marginEvt.general, err = e.GetAccountByID(marginEvt.general.GetId())
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "general"),
				logging.String("party-id", evt.Party()),
				logging.String("asset", asset))
		}
		marginEvt.margin, err = e.GetAccountByID(marginEvt.margin.GetId())
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "margin"),
				logging.String("party-id", evt.Party()),
				logging.String("asset", asset),
				logging.String("market-id", settle.MarketID))
		}

		responses = append(responses, res)
		marginEvts = append(marginEvts, marginEvt)
	}

	// if winidx is 0, this means we had now wind and loss, but may have some event which
	// needs to be propagated forward so we return now.
	if winidx == 0 {
		return marginEvts, responses, nil
	}

	// now check that what was collected is enough
	// This is where we'll implement everything
	settle, _, err = e.getSystemAccounts(marketID, asset)
	if err != nil {
		e.log.Error(
			"Failed to get system accounts required for MTM settlement",
			logging.Error(err),
		)
		return nil, nil, err
	}

	// now compare what's in the settlement account what we expect initialy to redistribute.
	// if there's not enough we enter loss socialization
	distr := simpleDistributor{
		log:             e.log,
		marketID:        settle.MarketID,
		expectCollected: expectCollected,
		collected:       settle.Balance,
		requests:        []request{},
	}

	if distr.LossSocializationEnabled() {
		e.log.Warn("Entering loss socialization",
			logging.String("market-id", marketID),
			logging.String("asset", asset),
			logging.Int64("expect-collected", expectCollected),
			logging.Int64("collected", settle.Balance))
		for _, evt := range transfers[winidx:] {
			transfer := evt.Transfer()
			if transfer != nil && transfer.Type == types.TransferType_MTM_WIN {
				distr.Add(evt.Transfer())
			}
		}
		evts := distr.Run()
		e.lossSocBuf.Add(evts)
	}

	// then we process all the wins
	for _, evt := range transfers[winidx:] {
		transfer := evt.Transfer()
		marginEvt := &marginUpdate{
			MarketPosition: evt,
			asset:          asset,
			marketID:       settle.MarketID,
		}
		// no transfer needed if transfer is nil, just build the marginUpdate
		if transfer == nil {
			marginEvt.general, err = e.GetAccountByID(e.accountID(noMarket, evt.Party(), asset, types.AccountType_GENERAL))
			if err != nil {
				e.log.Error("unable to get party account",
					logging.String("account-type", "general"),
					logging.String("party-id", evt.Party()),
					logging.String("asset", asset))
			}

			marginEvt.margin, err = e.GetAccountByID(e.accountID(settle.MarketID, evt.Party(), asset, types.AccountType_MARGIN))
			if err != nil {
				e.log.Error("unable to get party account",
					logging.String("account-type", "margin"),
					logging.String("party-id", evt.Party()),
					logging.String("asset", asset),
					logging.String("market-id", settle.MarketID))
			}

			marginEvts = append(marginEvts, marginEvt)
			continue
		}

		req, err := e.getTransferRequest(transfer, settle, insurance, marginEvt)
		if err != nil {
			e.log.Error(
				"Failed to build transfer request for event",
				logging.Error(err),
			)
			return nil, nil, err
		}

		// set the amount (this can change the req.Amount value if we entered loss socialisation
		res, err := e.getLedgerEntries(req)
		if err != nil {
			e.log.Error(
				"Failed to transfer funds",
				logging.Error(err),
			)
			return nil, nil, err
		}

		// update the to accounts now
		for _, bal := range res.Balances {
			if err := e.IncrementBalance(bal.Account.Id, bal.Balance); err != nil {
				e.log.Error(
					"Could not update the target account in transfer",
					logging.String("account-id", bal.Account.Id),
					logging.Error(err),
				)
				return nil, nil, err
			}
		}
		// updating the accounts stored in the marginEvt
		marginEvt.general, err = e.GetAccountByID(marginEvt.general.GetId())
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "general"),
				logging.String("party-id", evt.Party()),
				logging.String("asset", asset))
		}

		marginEvt.margin, err = e.GetAccountByID(marginEvt.margin.GetId())
		if err != nil {
			e.log.Error("unable to get party account",
				logging.String("account-type", "margin"),
				logging.String("party-id", evt.Party()),
				logging.String("asset", asset),
				logging.String("market-id", settle.MarketID))
		}

		responses = append(responses, res)
		marginEvts = append(marginEvts, marginEvt)
	}

	e.lossSocBuf.Flush()
	return marginEvts, responses, nil
}

// GetPartyMargin will return the current margin for a given party
func (e *Engine) GetPartyMargin(pos events.MarketPosition, asset, marketID string) (events.Margin, error) {
	genID := e.accountID("", pos.Party(), asset, types.AccountType_GENERAL)
	marginID := e.accountID(marketID, pos.Party(), asset, types.AccountType_MARGIN)
	genAcc, err := e.GetAccountByID(genID)
	if err != nil {
		e.log.Error(
			"Party doesn't have a general account somehow?",
			logging.String("party-id", pos.Party()))
		return nil, ErrTraderAccountsMissing
	}
	marAcc, err := e.GetAccountByID(marginID)
	if err != nil {
		e.log.Error(
			"Party doesn't have a margin account somehow?",
			logging.String("party-id", pos.Party()),
			logging.String("market-id", marketID))
		return nil, ErrTraderAccountsMissing
	}

	return marginUpdate{
		pos,
		marAcc,
		genAcc,
		asset,
		marketID,
	}, nil
}

// MarginUpdate will run the margin updates over a set of risk events (margin updates)
func (e *Engine) MarginUpdate(marketID string, updates []events.Risk) ([]*types.TransferResponse, []events.Margin, error) {
	response := make([]*types.TransferResponse, 0, len(updates))
	closed := make([]events.Margin, 0, len(updates)/2) // half the cap, if we have more than that, the slice will double once, and will fit all updates anyway
	// create "fake" settle account for market ID
	settle := &types.Account{
		MarketID: marketID,
	}
	for _, update := range updates {
		transfer := update.Transfer()
		// although this is mainly a duplicate event, we need to pass it to getTransferRequest
		mevt := &marginUpdate{
			MarketPosition: update,
			asset:          update.Asset(),
			marketID:       update.MarketID(),
		}

		req, err := e.getTransferRequest(transfer, settle, nil, mevt)
		if err != nil {
			return nil, nil, err
		}
		res, err := e.getLedgerEntries(req)
		if err != nil {
			return nil, nil, err
		}
		// we didn't manage to top up to even the minimum required system margin, close out trader
		// we need to be careful with this, only apply this to transfer for low margin
		// the MinAmount in the transfer is always set to 0 but in 2 case:
		// - first when a new order is created, the MinAmount is the same than Amount, which is
		//   what's required to reach the InitialMargin level
		// - second when a trader margin is under the MaintenanceLevel, the MinAmount is supposed
		//   to be at least to get back to the search level, and the amount will be enough to reach
		//   InitialMargin
		// In both case either the order will not be accepted, or the trader will be closed
		if transfer.Type == types.TransferType_MARGIN_LOW &&
			res.Balances[0].Account.Balance < (int64(update.MarginBalance())+transfer.Amount.MinAmount) {
			closed = append(closed, mevt)
		}
		response = append(response, res)
		for _, v := range res.GetTransfers() {
			// increment the to account
			if err := e.IncrementBalance(v.ToAccount, v.Amount); err != nil {
				e.log.Error(
					"Failed to increment balance for account",
					logging.String("account-id", v.ToAccount),
					logging.Int64("amount", v.Amount),
					logging.Error(err),
				)
			}
		}
	}

	return response, closed, nil
}

// MarginUpdate will run the margin updates over a set of risk events (margin updates)
func (e *Engine) MarginUpdateOnOrder(
	marketID string, update events.Risk) (*types.TransferResponse, events.Margin, error) {
	// create "fake" settle account for market ID
	settle := &types.Account{
		MarketID: marketID,
	}
	transfer := update.Transfer()
	// although this is mainly a duplicate event, we need to pass it to getTransferRequest
	mevt := &marginUpdate{
		MarketPosition: update,
		asset:          update.Asset(),
		marketID:       update.MarketID(),
	}

	req, err := e.getTransferRequest(transfer, settle, nil, mevt)
	if err != nil {
		return nil, nil, err
	}

	// we do not have enough money to get to the minimum amount,
	// we return an error.
	if mevt.GeneralBalance()+mevt.MarginBalance() < uint64(transfer.GetAmount().MinAmount) {
		return nil, mevt, ErrMinAmountNotReached
	}

	// from here we know there's enough money,
	// let get the ledger entries, return the transfers

	res, err := e.getLedgerEntries(req)
	if err != nil {
		return nil, nil, err
	}
	for _, v := range res.GetTransfers() {
		// increment the to account
		if err := e.IncrementBalance(v.ToAccount, v.Amount); err != nil {
			e.log.Error(
				"Failed to increment balance for account",
				logging.String("account-id", v.ToAccount),
				logging.Int64("amount", v.Amount),
				logging.Error(err),
			)
		}
	}

	return res, nil, nil
}

// getTransferRequest builds the request, and sets the required accounts based on the type of the Transfer argument
func (e *Engine) getTransferRequest(p *types.Transfer, settle, insurance *types.Account, mEvt *marginUpdate) (*types.TransferRequest, error) {
	asset := p.Amount.Asset

	var err error
	// the accounts for the trader we need
	mEvt.margin, err = e.GetAccountByID(e.accountID(settle.MarketID, p.Owner, asset, types.AccountType_MARGIN))
	if err != nil {
		e.log.Error(
			"Failed to get the margin trader account",
			logging.String("owner-id", p.Owner),
			logging.String("market-id", settle.MarketID),
			logging.Error(err),
		)
		return nil, err
	}
	// we'll need this account for all transfer types anyway (settlements, margin-risk updates)
	mEvt.general, err = e.GetAccountByID(e.accountID(noMarket, p.Owner, asset, types.AccountType_GENERAL))
	if err != nil {
		e.log.Error(
			"Failed to get the general trader account",
			logging.String("owner-id", p.Owner),
			logging.String("market-id", settle.MarketID),
			logging.Error(err),
		)
		return nil, err
	}
	// final settle, or MTM settle, makes no difference, it's win/loss still
	if p.Type == types.TransferType_LOSS || p.Type == types.TransferType_MTM_LOSS {
		// losses are collected first from the margin account, then the general account, and finally
		// taken out of the insurance pool
		req := types.TransferRequest{
			FromAccount: []*types.Account{
				mEvt.margin,
				mEvt.general,
				insurance,
			},
			ToAccount: []*types.Account{
				settle,
			},
			Amount:    uint64(-p.Amount.Amount) * p.Size,
			MinAmount: 0,     // default value, but keep it here explicitly
			Asset:     asset, // TBC
			Reference: p.Type.String(),
		}
		return &req, nil
	}
	if p.Type == types.TransferType_WIN || p.Type == types.TransferType_MTM_WIN {
		// the insurance pool in the FromAccount is not used ATM (losses should fully cover wins
		// or the insurance pool has already been drained).
		return &types.TransferRequest{
			FromAccount: []*types.Account{
				settle,
				insurance,
			},
			ToAccount: []*types.Account{
				mEvt.margin,
			},
			Amount:    uint64(p.Amount.Amount) * p.Size,
			MinAmount: 0,     // default value, but keep it here explicitly
			Asset:     asset, // TBC
			Reference: p.Type.String(),
		}, nil
	}

	// just in case...
	if p.Size == 0 {
		p.Size = 1
	}
	if p.Type == types.TransferType_MARGIN_LOW {
		return &types.TransferRequest{
			FromAccount: []*types.Account{
				mEvt.general,
			},
			ToAccount: []*types.Account{
				mEvt.margin,
			},
			Amount:    uint64(p.Amount.Amount) * p.Size,
			MinAmount: uint64(p.Amount.MinAmount),
			Asset:     asset,
			Reference: p.Type.String(),
		}, nil
	}
	return &types.TransferRequest{
		FromAccount: []*types.Account{
			mEvt.margin,
		},
		ToAccount: []*types.Account{
			mEvt.general,
		},
		Amount:    uint64(p.Amount.Amount) * p.Size,
		MinAmount: uint64(p.Amount.MinAmount),
		Asset:     asset,
		Reference: p.Type.String(),
	}, nil
}

// this builds a TransferResponse for a specific request, we collect all of them and aggregate
func (e *Engine) getLedgerEntries(req *types.TransferRequest) (*types.TransferResponse, error) {
	ret := types.TransferResponse{
		Transfers: []*types.LedgerEntry{},
		Balances:  make([]*types.TransferBalance, 0, len(req.ToAccount)),
	}
	for _, t := range req.ToAccount {
		ret.Balances = append(ret.Balances, &types.TransferBalance{
			Account: t,
		})
	}
	amount := int64(req.Amount)
	for _, acc := range req.FromAccount {
		// give each to account an equal share
		parts := amount / int64(len(req.ToAccount))
		// add remaining pennies to last ledger movement
		remainder := amount % int64(len(req.ToAccount))
		var (
			to *types.TransferBalance
			lm *types.LedgerEntry
		)
		// either the account contains enough, or we're having to access insurance pool money
		if acc.Balance >= amount {
			acc.Balance -= amount
			if err := e.IncrementBalance(acc.Id, -amount); err != nil {
				e.log.Error(
					"Failed to update balance for account",
					logging.String("account-id", acc.Id),
					logging.Int64("balance", acc.Balance),
					logging.Error(err),
				)
				return nil, err
			}
			for _, to = range ret.Balances {
				lm = &types.LedgerEntry{
					FromAccount: acc.Id,
					ToAccount:   to.Account.Id,
					Amount:      parts,
					Reference:   req.Reference,
					Type:        "settlement",
					Timestamp:   e.currentTime,
				}
				ret.Transfers = append(ret.Transfers, lm)
				to.Balance += parts
				to.Account.Balance += parts
			}
			// add remainder
			if remainder > 0 {
				lm.Amount += remainder
				to.Balance += remainder
				to.Account.Balance += remainder
			}
			return &ret, nil
		}
		if acc.Balance > 0 {
			amount -= acc.Balance
			// partial amount resolves differently
			parts = acc.Balance / int64(len(req.ToAccount))
			if err := e.UpdateBalance(acc.Id, 0); err != nil {
				e.log.Error(
					"Failed to set balance of account to 0",
					logging.String("account-id", acc.Id),
					logging.Error(err),
				)
				return nil, err
			}
			acc.Balance = 0
			for _, to = range ret.Balances {
				lm = &types.LedgerEntry{
					FromAccount: acc.Id,
					ToAccount:   to.Account.Id,
					Amount:      parts,
					Reference:   req.Reference,
					Type:        "settlement",
					Timestamp:   e.currentTime,
				}
				ret.Transfers = append(ret.Transfers, lm)
				to.Account.Balance += parts
				to.Balance += parts
			}
		}
		if amount == 0 {
			break
		}
	}
	return &ret, nil
}

// ClearMarket will remove all monies or accounts for parties allocated for a market (margin accounts)
// when the market reach end of life (maturity)
func (e *Engine) ClearMarket(mktID, asset string, parties []string) ([]*types.TransferResponse, error) {
	// create a transfer request that we will reuse all the time in order to make allocations smaller
	req := &types.TransferRequest{
		FromAccount: make([]*types.Account, 1),
		ToAccount:   make([]*types.Account, 1),
		Asset:       asset,
	}

	// assume we have as much transfer response than parties
	resps := make([]*types.TransferResponse, 0, len(parties))

	for _, v := range parties {
		marginAcc, err := e.GetAccountByID(e.accountID(mktID, v, asset, types.AccountType_MARGIN))
		if err != nil {
			e.log.Error(
				"Failed to get the margin account",
				logging.String("trader-id", v),
				logging.String("market-id", mktID),
				logging.String("asset", asset),
				logging.Error(err))
			// just try to do other traders
			continue
		}

		generalAcc, err := e.GetAccountByID(e.accountID("", v, asset, types.AccountType_GENERAL))
		if err != nil {
			e.log.Error(
				"Failed to get the general account",
				logging.String("trader-id", v),
				logging.String("market-id", mktID),
				logging.String("asset", asset),
				logging.Error(err))
			// just try to do other traders
			continue
		}

		req.FromAccount[0] = marginAcc
		req.ToAccount[0] = generalAcc
		req.Amount = uint64(marginAcc.Balance)

		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("Clearing trader margin account",
				logging.String("market-id", mktID),
				logging.String("asset", asset),
				logging.String("trader-id", v),
				logging.Int64("margin-before", marginAcc.Balance),
				logging.Int64("general-before", generalAcc.Balance),
				logging.Int64("general-after", generalAcc.Balance+marginAcc.Balance))
		}

		ledgerEntries, err := e.getLedgerEntries(req)
		if err != nil {
			e.log.Error(
				"Failed to move monies from margin to genral account",
				logging.String("trader-id", v),
				logging.String("market-id", mktID),
				logging.String("asset", asset),
				logging.Error(err))
			// just try to do other traders
			continue
		}

		for _, v := range ledgerEntries.Transfers {
			// increment the to account
			if err := e.IncrementBalance(v.ToAccount, v.Amount); err != nil {
				e.log.Error(
					"Failed to increment balance for account",
					logging.String("account-id", v.ToAccount),
					logging.Int64("amount", v.Amount),
					logging.Error(err),
				)
				return nil, err
			}
		}

		resps = append(resps, ledgerEntries)
	}

	return resps, nil
}

// CreatePartyMarginAccount creates a margin account if it does not exist, will return an error
// if no general account exist for the trader for the given asset
func (e *Engine) CreatePartyMarginAccount(partyID, marketID, asset string) (string, error) {
	marginID := e.accountID(marketID, partyID, asset, types.AccountType_MARGIN)
	if _, ok := e.accs[marginID]; !ok {
		// OK no margin ID, so let's try to get the general id then
		// first check if generak account exists
		generalID := e.accountID(noMarket, partyID, asset, types.AccountType_GENERAL)
		if _, ok := e.accs[generalID]; !ok {
			e.log.Error("Tried to create a margin account for a party with no general account",
				logging.String("party-id", partyID),
				logging.String("asset", asset),
				logging.String("market-id", marketID),
			)
			return "", ErrNoGeneralAccountWhenCreateMarginAccount
		}

		// general account id OK, let's create a margin account
		acc := types.Account{
			Id:       marginID,
			Asset:    asset,
			MarketID: marketID,
			Balance:  0,
			Owner:    partyID,
			Type:     types.AccountType_MARGIN,
		}
		e.accs[marginID] = &acc
		e.buf.Add(acc)
	}
	return marginID, nil
}

// CreatePartyGeneralAccount creates trader accounts for a given market
// one account per market, per asset for each trader
func (e *Engine) CreatePartyGeneralAccount(partyID, asset string) string {

	generalID := e.accountID(noMarket, partyID, asset, types.AccountType_GENERAL)
	if _, ok := e.accs[generalID]; !ok {
		acc := types.Account{
			Id:       generalID,
			Asset:    asset,
			MarketID: noMarket,
			Balance:  0,
			Owner:    partyID,
			Type:     types.AccountType_GENERAL,
		}
		e.accs[generalID] = &acc
		e.buf.Add(acc)
	}

	return generalID
}

// RemoveDistressed will remove all distressed trader in the event positions
// for a given market and asset
func (e *Engine) RemoveDistressed(traders []events.MarketPosition, marketID, asset string) (*types.TransferResponse, error) {
	tl := len(traders)
	if tl == 0 {
		return nil, nil
	}
	// insurance account is the one we're after
	_, ins, err := e.getSystemAccounts(marketID, asset)
	if err != nil {
		return nil, err
	}
	resp := types.TransferResponse{
		Transfers: make([]*types.LedgerEntry, 0, tl),
	}
	for _, trader := range traders {
		// move monies from the margin account first
		acc, err := e.GetAccountByID(e.accountID(marketID, trader.Party(), asset, types.AccountType_MARGIN))
		if err != nil {
			return nil, err
		}
		if acc.Balance > 0 {
			resp.Transfers = append(resp.Transfers, &types.LedgerEntry{
				FromAccount: acc.Id,
				ToAccount:   ins.Id,
				Amount:      acc.Balance,
				Reference:   types.TransferType_MARGIN_CONFISCATED.String(),
				Type:        "position-resolution",
				Timestamp:   e.currentTime,
			})
			if err := e.IncrementBalance(ins.Id, acc.Balance); err != nil {
				return nil, err
			}
			if err := e.UpdateBalance(acc.Id, 0); err != nil {
				return nil, err
			}
		}

		// we remove the margin account
		if err := e.removeAccount(acc.Id); err != nil {
			return nil, err
		}

	}
	return &resp, nil
}

// CreateMarketAccounts will create all required accounts for a market once
// a new market is accepted through the network
func (e *Engine) CreateMarketAccounts(marketID, asset string, insurance int64) (insuranceID, settleID string) {
	insuranceID = e.accountID(marketID, "", asset, types.AccountType_INSURANCE)
	_, ok := e.accs[insuranceID]
	if !ok {
		insAcc := &types.Account{
			Id:       insuranceID,
			Asset:    asset,
			Owner:    systemOwner,
			Balance:  insurance,
			MarketID: marketID,
			Type:     types.AccountType_INSURANCE,
		}
		e.accs[insuranceID] = insAcc
		e.buf.Add(*insAcc)

	}
	settleID = e.accountID(marketID, "", asset, types.AccountType_SETTLEMENT)
	_, ok = e.accs[settleID]
	if !ok {
		setAcc := &types.Account{
			Id:       settleID,
			Asset:    asset,
			Owner:    systemOwner,
			Balance:  0,
			MarketID: marketID,
			Type:     types.AccountType_SETTLEMENT,
		}
		e.accs[settleID] = setAcc
		e.buf.Add(*setAcc)
	}

	return
}

// Withdraw will remove the specified amount from the trader
// general account
func (e *Engine) Withdraw(partyID, asset string, amount uint64) error {
	acc, err := e.GetAccountByID(e.accountID("", partyID, asset, types.AccountType_GENERAL))
	if err != nil {
		return ErrAccountDoesNotExist
	}

	// check we have more money than required to withdraw
	if uint64(acc.Balance) < amount {
		// if we have less balance than required to withdraw, just set it to 0
		// and return an error
		if err := e.UpdateBalance(acc.Id, 0); err != nil {
			return err
		}
		return fmt.Errorf("withdraw error, required=%v, available=%v", amount, acc.Balance)
	}

	if err := e.IncrementBalance(acc.Id, -int64(amount)); err != nil {
		return err
	}
	return nil
}

// UpdateBalance will update the balance of a given account
func (e *Engine) UpdateBalance(id string, balance int64) error {
	acc, ok := e.accs[id]
	if !ok {
		return ErrAccountDoesNotExist
	}
	acc.Balance = balance
	e.buf.Add(*acc)
	return nil
}

// IncrementBalance will increment the balance of a given account
// using the given value
func (e *Engine) IncrementBalance(id string, inc int64) error {
	acc, ok := e.accs[id]
	if !ok {
		return ErrAccountDoesNotExist
	}
	acc.Balance += inc
	e.buf.Add(*acc)
	return nil
}

// GetAccountByID will return an account using the given id
func (e *Engine) GetAccountByID(id string) (*types.Account, error) {
	acc, ok := e.accs[id]
	if !ok {
		return nil, ErrAccountDoesNotExist
	}
	acccpy := *acc
	return &acccpy, nil
}

func (e *Engine) removeAccount(id string) error {
	delete(e.accs, id)
	return nil
}

// @TODO this function uses a single slice for each call. This is fine now, as we're processing
// everything sequentially, and so there's no possible data-races here. Once we start doing things
// like cleaning up expired markets asynchronously, then this func is not safe for concurrent use
func (e *Engine) accountID(marketID, partyID, asset string, ty types.AccountType) string {
	if len(marketID) <= 0 {
		marketID = noMarket
	}

	// market account
	if len(partyID) <= 0 {
		partyID = systemOwner
	}

	copy(e.idbuf, marketID)
	ln := len(marketID)
	copy(e.idbuf[ln:], partyID)
	ln += len(partyID)
	copy(e.idbuf[ln:], asset)
	ln += len(asset)
	e.idbuf[ln] = byte(ty + 48)
	return string(e.idbuf[:ln+1])
}
