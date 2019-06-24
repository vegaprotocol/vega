package collateral

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/storage"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

const (
	initialAccountSize = 4096
)

var (
	ErrSystemAccountsMissing = errors.New("system accounts missing for collateral engine to work")
	ErrTraderAccountsMissing = errors.New("trader accounts missing, cannot collect")
	ErrBalanceNotSet         = errors.New("failed to update account balance")
	ErrAccountDoNotExists    = errors.New("account do not exists")
	ErrAccountAlreadyExists  = errors.New("account already exists")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_buffer_mock.go -package mocks code.vegaprotocol.io/vega/internal/collateral AccountBuffer
type AccountBuffer interface {
	Add(types.Account)
}

type collectCB func(p *types.Transfer) error
type setupF func(*types.Transfer) (*types.TransferResponse, error)

type Engine struct {
	Config
	log   *logging.Logger
	cfgMu sync.Mutex

	// map of trader ID's to map of account types + account ID's
	// traderAccounts map[string]map[types.AccountType]map[string]string // by trader, type, and asset
	// marketAccounts map[types.AccountType]map[string]string            // by type and asset

	accs map[string]*types.Account
	buf  AccountBuffer
}

func accountID(marketID, traderID, asset string, ty types.AccountType) string {
	// if no marketID -> trader general account
	if len(marketID) <= 0 {
		marketID = storage.NoMarket
	}

	// market account
	if len(traderID) <= 0 {
		traderID = storage.SystemOwner
	}

	var b strings.Builder
	b.Grow(len(marketID) + len(traderID) + len(asset) + 1)
	b.WriteString(marketID)
	b.WriteString(traderID)
	b.WriteString(asset)
	b.WriteRune(rune(ty))
	return b.String()
}

func New(log *logging.Logger, conf Config, buf AccountBuffer) (*Engine, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())
	return &Engine{
		log:    log,
		Config: conf,
		accs:   make(map[string]*types.Account, initialAccountSize),
		buf:    buf,
	}, nil
}

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

func (e *Engine) getSystemAccounts(marketID, asset string) (settle, insurance *types.Account, err error) {
	// asset := marketID[:3]

	insID := accountID(marketID, "", asset, types.AccountType_INSURANCE)
	setID := accountID(marketID, "", asset, types.AccountType_SETTLEMENT)

	var ok bool
	insurance, ok = e.accs[insID]
	if !ok {
		fmt.Printf("asset: %s - accounts: %#v\n", asset, e.accs)
		err = ErrSystemAccountsMissing
		return
	}

	settle, ok = e.accs[setID]
	if !ok {
		fmt.Printf("asset: %s - accounts: %#v\n", asset, e.accs)
		err = ErrSystemAccountsMissing
		return
	}

	return
}

// AddTraderToMarket - when a new trader enters a market, ensure general + margin accounts both exist
func (e *Engine) AddTraderToMarket(marketID, traderID, asset string) error {
	// accountID(marketID, traderID, asset string, ty types.AccountType) accountIDT
	genID := accountID("", traderID, asset, types.AccountType_GENERAL)
	marginID := accountID(marketID, traderID, asset, types.AccountType_MARGIN)
	gen, err := e.GetAccountByID(genID)
	if err != nil {
		e.log.Error(
			"Trader doesn't have a general account somehow?",
			logging.String("trader-id", traderID))
		return ErrTraderAccountsMissing
	}
	margin, err := e.GetAccountByID(marginID)
	if err != nil {
		e.log.Error(
			"Trader doesn't have a margin account somehow?",
			logging.String("trader-id", traderID),
			logging.String("Market", marketID))
		return ErrTraderAccountsMissing
	}

	// let's get the balances we need
	e.cfgMu.Lock()
	genBal := e.Config.TraderGeneralAccountBalance
	marginBal := genBal / 100 * e.Config.TraderMarginPercent
	e.cfgMu.Unlock()
	// check to see if there's enough balance on the general account already
	// if not, add it
	if gen.Balance < genBal {
		gen.Balance = genBal
	}
	// subtract the margin from the general balance
	gen.Balance -= marginBal
	if err := e.UpdateBalance(gen.Id, gen.Balance); err != nil {
		e.log.Error(
			"Failed to set new balance for general account",
			logging.String("trader-id", traderID),
			logging.String("account-id", gen.Id),
			logging.Int64("balance", gen.Balance),
			logging.Error(err))
		return err
	}
	if err := e.UpdateBalance(margin.Id, marginBal); err != nil {
		e.log.Error(
			"Failed to set new balance for margin account",
			logging.String("trader-id", traderID),
			logging.String("account-id", margin.Id),
			logging.Int64("balance", marginBal),
			logging.Error(err))
		return err
	}
	return nil
}

func (e *Engine) MarkToMarket(marketID string, positions []events.Transfer) ([]*types.TransferResponse, error) {
	// for now, this is the same as collect, but once we finish the closing positions bit in positions/settlement
	// we'll first handle the close settlement, then the updated positions for mark-to-market
	transfers := make([]*types.Transfer, 0, len(positions))
	for _, p := range positions {
		transfers = append(transfers, p.Transfer())
	}
	return e.Transfer(marketID, transfers)
}

func (e *Engine) Transfer(marketID string, transfers []*types.Transfer) ([]*types.TransferResponse, error) {
	if len(transfers) == 0 {
		return nil, nil
	}
	if isSettle(transfers[0]) {
		return e.collect(marketID, transfers)
	}
	// this is a balance top-up or some other thing we haven't implemented yet
	return nil, nil
}

func isSettle(transfer *types.Transfer) bool {
	switch transfer.Type {
	case types.TransferType_WIN, types.TransferType_LOSS, types.TransferType_MTM_WIN, types.TransferType_MTM_LOSS:
		return true
	}
	return false
}

// collect, handles collects for both market close as mark-to-market stuff
func (e *Engine) collect(marketID string, positions []*types.Transfer) ([]*types.TransferResponse, error) {
	if len(positions) == 0 {
		return nil, nil
	}

	// FIXME(): get asset properly
	asset := positions[0].Amount.Asset

	reference := fmt.Sprintf("%s close", marketID) // ledger moves need to indicate that they happened because market was closed
	settle, insurance, err := e.getSystemAccounts(marketID, asset)
	if err != nil {
		return nil, err
	}
	// this way we know if we need to check loss response
	haveLoss := (positions[0].Type == types.TransferType_LOSS || positions[0].Type == types.TransferType_MTM_LOSS)
	// tracks delta, wins & losses and determines how to distribute losses amongst wins if needed
	distr := distributor{}
	lossResp, winResp := getTransferResponses(positions, settle, insurance)
	// get the callbacks used to process positions
	lossCB, winCB := e.getCallbacks(&distr, reference, settle, insurance, lossResp, winResp)
	// begin work, start by processing the loss positions, and get win positions while we're at it
	winPos, err := collectLoss(positions, lossCB)
	if err != nil {
		return nil, err
	}
	// process lossResp before moving on to win...
	if haveLoss {
		for _, bacc := range lossResp.Balances {
			distr.lossDelta += uint64(bacc.Balance)
			if err := e.IncrementBalance(bacc.Account.Id, bacc.Balance); err != nil {
				e.log.Error(
					"Failed to update target account",
					logging.String("target-account", bacc.Account.Id),
					logging.Int64("balance", bacc.Balance),
					logging.Error(err),
				)
				return nil, err
			}
		}
		if distr.lossDelta != distr.expLoss {
			e.log.Warn(
				"Expected to distribute and actual balance mismatch",
				logging.Uint64("expected-balance", distr.expLoss),
				logging.Uint64("actual-balance", distr.lossDelta),
			)
		}
	}
	if len(winPos) == 0 {
		return []*types.TransferResponse{
			lossResp,
		}, nil
	}
	// each position, multiplied by 2 (move from account, to account == 2 moves)
	winResp.Transfers = make([]*types.LedgerEntry, 0, len(winPos)*2)
	if err := collectWin(winPos, winCB); err != nil {
		return nil, err
	}
	// possibly verify balances?
	for _, b := range winResp.Balances {
		b.Balance = b.Account.Balance
	}
	if haveLoss {
		return []*types.TransferResponse{
			lossResp,
			winResp,
		}, nil
	}
	return []*types.TransferResponse{
		winResp,
	}, nil
}

func getTransferResponses(positions []*types.Transfer, settle, insurance *types.Account) (loss, win *types.TransferResponse) {
	loss = &types.TransferResponse{
		Transfers: make([]*types.LedgerEntry, 0, len(positions)), // roughly half should be loss, but create 2 ledger entries, so that's a reasonable cap to use
		Balances: []*types.TransferBalance{
			{
				Account: settle, // settle to this account
				Balance: 0,      // current balance delta -> 0
			},
		},
	}
	win = &types.TransferResponse{
		// we will alloc this slice once we've processed all loss
		// Transfers: make([]*types.LedgerEntry, 0, len(positions)),
		Balances: []*types.TransferBalance{
			{
				Account: settle,
			},
			{
				Account: insurance,
			},
		},
	}
	return
}

func (e *Engine) getCallbacks(distr *distributor, reference string, settle, insurance *types.Account, lossResp, winResp *types.TransferResponse) (collectCB, collectCB) {
	// this callback is internal only
	setupCB := e.getSetupCB(distr, reference, settle, insurance)
	lossCB := e.getLossCB(distr, lossResp, setupCB)
	winCB := e.getWinCB(distr, winResp, setupCB)
	return lossCB, winCB
}

func (e *Engine) getSetupCB(distr *distributor, reference string, settle, insurance *types.Account) setupF {
	// common tasks performed for both win and loss positions
	return func(p *types.Transfer) (*types.TransferResponse, error) {
		req, err := e.getTransferRequest(p, settle, insurance)
		if err != nil {
			e.log.Error(
				"Failed to create the transfer request",
				logging.String("settlement-type", p.Type.String()),
				logging.String("trader-id", p.Owner),
				logging.Error(err),
			)
			return nil, err
		}
		distr.amountCB(req)
		req.Reference = reference
		res, err := e.getLedgerEntries(req)
		if err != nil {
			return nil, err
		}
		return res, nil
	}
}

func (e *Engine) getLossCB(distr *distributor, lossResp *types.TransferResponse, setupCB setupF) collectCB {
	return func(p *types.Transfer) error {
		res, err := setupCB(p)
		if err != nil {
			return err
		}
		expAmount := uint64(-p.Amount.Amount) * p.Size
		distr.expLoss += expAmount
		// could increment distr.balanceDelta, but we're iterating over this later on anyway
		// and we might need to change this to handle multiple balances, best keep it there
		if uint64(res.Balances[0].Balance) != expAmount {
			e.log.Warn(
				"Loss trader accounts for full amount failed",
				logging.String("trader-id", p.Owner),
				logging.Uint64("expected-amount", expAmount),
				logging.Int64("actual-amount", res.Balances[0].Balance),
			)
		}
		lossResp.Transfers = append(lossResp.Transfers, res.Transfers...)
		// account balance is updated automatically
		// increment balance
		lossResp.Balances[0].Balance += res.Balances[0].Balance
		return nil
	}
}

func (e *Engine) getWinCB(distr *distributor, winResp *types.TransferResponse, setupCB setupF) collectCB {
	return func(p *types.Transfer) error {
		res, err := setupCB(p)
		if err != nil {
			return err
		}
		distr.expWin += uint64(res.Balances[0].Balance)
		// there's only 1 balance account here (the ToAccount)
		if err := e.IncrementBalance(res.Balances[0].Account.Id, res.Balances[0].Balance); err != nil {
			// this account might get accessed concurrently -> use increment
			e.log.Error(
				"Failed to increment balance of general account",
				logging.String("account-id", res.Balances[0].Account.Id),
				logging.Int64("increment", res.Balances[0].Balance),
				logging.Error(err),
			)
			return err
		}
		winResp.Transfers = append(winResp.Transfers, res.Transfers...)
		return nil
	}
}

func collectLoss(positions []*types.Transfer, cb collectCB) ([]*types.Transfer, error) {
	// collect whatever we have until we reach the DEBIT part of the positions
	for i, p := range positions {
		if p.Type == types.TransferType_WIN || p.Type == types.TransferType_MTM_WIN {
			return positions[i:], nil
		}
		if err := cb(p); err != nil {
			return nil, err
		}
	}
	// only CREDIT positions found OR positions was empty to begin with
	return nil, nil
}

func collectWin(positions []*types.Transfer, cb collectCB) error {
	// this is really simple -> just collect whatever was left
	for _, p := range positions {
		if err := cb(p); err != nil {
			return err
		}
	}
	return nil
}

// getTransferRequest builds the request, and sets the required accounts based on the type of the Transfer argument
func (e *Engine) getTransferRequest(p *types.Transfer, settle, insurance *types.Account) (*types.TransferRequest, error) {
	asset := p.Amount.Asset

	// final settle, or MTM settle, makes no difference, it's win/loss still
	if p.Type == types.TransferType_LOSS || p.Type == types.TransferType_MTM_LOSS {
		marginAcc, err := e.GetAccountByID(accountID(settle.MarketID, p.Owner, asset, types.AccountType_MARGIN))
		if err != nil {
			return nil, err
		}
		req := types.TransferRequest{
			FromAccount: []*types.Account{
				marginAcc, insurance}, // we'll need 2 accounts, last one is insurance pool
			ToAccount: []*types.Account{
				settle,
			},
			Amount:    uint64(-p.Amount.Amount) * p.Size,
			MinAmount: 0,     // default value, but keep it here explicitly
			Asset:     asset, // TBC
		}
		return &req, nil
	}

	generalAcc, err := e.GetAccountByID(accountID("", p.Owner, asset, types.AccountType_GENERAL))
	if err != nil {
		e.log.Error(
			"Failed to get the general account",
			logging.String("owner", p.Owner),
			logging.String("market", settle.MarketID),
			logging.Error(err))
		return nil, err
	}

	return &types.TransferRequest{
		FromAccount: []*types.Account{
			settle,
			insurance,
		},
		ToAccount: []*types.Account{
			generalAcc,
		},
		Amount:    uint64(p.Amount.Amount) * p.Size,
		MinAmount: 0,     // default value, but keep it here explicitly
		Asset:     asset, // TBC
	}, nil
}

/*
func (e *Engine) getTraderAccountsByAssetAndType(trader, asset string, accTypes ...types.AccountType) ([]*types.Account, error) {
	ids, err := e.getTraderAccountIDs(asset, trader, accTypes...)
	if err != nil {
		return nil, err
	}
	accounts := make([]*types.Account, 0, len(ids))
	for _, id := range ids {
		acc, err := e.GetAccountByID(id)
		if err != nil {
			e.log.Error(
				"Failed to get account by id",
				logging.String("account-id", id),
				logging.Error(err),
			)
			return nil, ErrTraderAccountsMissing
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}
*/

/*
func (e *Engine) getTraderAccountIDs(asset, trader string, types ...types.AccountType) ([]string, error) {
	taccs := e.traderAccounts[trader]
	// no specific types, get all accounts for asset
	if len(types) == 0 {
		ids := make([]string, 0, len(taccs))
		for _, accs := range taccs {
			if id, ok := accs[asset]; ok {
				ids = append(ids, id)
			}
		}
		return ids, nil
	}
	ids := make([]string, 0, len(types))
	for _, t := range types {
		accs, ok := taccs[t]
		if !ok {
			return nil, ErrTraderAccountsMissing
		}
		if id, ok := accs[asset]; ok {
			ids = append(ids, id)
		} else {
			return nil, ErrTraderAccountsMissing
		}
	}
	return ids, nil
}
*/

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
					Timestamp:   time.Now().Unix(),
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
					Timestamp:   time.Now().Unix(),
				}
				ret.Transfers = append(ret.Transfers, lm)
				to.Account.Balance += parts
				to.Balance += parts
			}
		}
	}
	return &ret, nil
}

// insert and stuff relate to accounts map

func (e *Engine) CreateMarketAccounts(marketID, asset string, insurance int64) error {
	insID := accountID(marketID, "", asset, types.AccountType_INSURANCE)
	_, ok := e.accs[insID]
	if ok {
		e.log.Error("account already exists",
			logging.String("marketID", marketID),
			logging.String("asset", asset))
		return ErrAccountAlreadyExists
	}
	setID := accountID(marketID, "", asset, types.AccountType_SETTLEMENT)
	_, ok = e.accs[setID]
	if ok {
		e.log.Error("account already exists",
			logging.String("marketID", marketID),
			logging.String("asset", asset))
		return ErrAccountAlreadyExists
	}

	// none exists, let create them
	insAcc := &types.Account{
		Id:       insID,
		Asset:    asset,
		Owner:    storage.SystemOwner,
		Balance:  insurance,
		MarketID: marketID,
		Type:     types.AccountType_INSURANCE,
	}
	e.accs[insID] = insAcc
	e.buf.Add(*insAcc)

	setAcc := &types.Account{
		Id:       setID,
		Asset:    asset,
		Owner:    storage.SystemOwner,
		Balance:  0,
		MarketID: marketID,
		Type:     types.AccountType_SETTLEMENT,
	}
	e.accs[setID] = setAcc
	e.buf.Add(*setAcc)
	return nil
}

func (e *Engine) CreateTraderAccount(traderID, marketID, asset string) (marginID, generalID string) {
	// first margin account
	marginID = accountID(marketID, traderID, asset, types.AccountType_MARGIN)
	_, ok := e.accs[marginID]
	if !ok {
		acc := &types.Account{
			Id:       marginID,
			Asset:    asset,
			MarketID: marketID,
			Balance:  0,
			Owner:    traderID,
			Type:     types.AccountType_MARGIN,
		}
		e.accs[marginID] = acc
		e.buf.Add(*acc)
	}

	generalID = accountID(storage.NoMarket, traderID, asset, types.AccountType_GENERAL)
	_, ok = e.accs[generalID]
	if !ok {
		acc := &types.Account{
			Id:       generalID,
			Asset:    asset,
			MarketID: storage.NoMarket,
			Balance:  0,
			Owner:    traderID,
			Type:     types.AccountType_GENERAL,
		}
		e.accs[generalID] = acc
		e.buf.Add(*acc)
	}

	return
}

func (e *Engine) UpdateBalance(id string, balance int64) error {
	acc, ok := e.accs[id]
	if !ok {
		return ErrAccountDoNotExists
	}
	acc.Balance = balance
	e.buf.Add(*acc)
	return nil
}

func (e *Engine) IncrementBalance(id string, inc int64) error {
	acc, ok := e.accs[id]
	if !ok {
		return ErrAccountDoNotExists
	}
	acc.Balance += inc
	e.buf.Add(*acc)
	return nil
}

func (e *Engine) GetAccountByID(id string) (*types.Account, error) {
	acc, ok := e.accs[id]
	if !ok {
		return nil, ErrAccountDoNotExists
	}
	acccpy := *acc
	return &acccpy, nil
}
