package collateral

import (
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrSystemAccountsMissing = errors.New("system accounts missing for collateral engine to work")
	ErrTraderAccountsMissing = errors.New("trader accounts missing, cannot collect")
)

type collectCB func(p *types.Transfer) error
type setupF func(*types.Transfer) (*types.TransferResponse, error)

type Engine struct {
	Config
	log   *logging.Logger
	cfgMu sync.Mutex

	market       string
	accountStore Accounts
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/engines/collateral Accounts
type Accounts interface {
	CreateMarketAccounts(market string, insurance int64) error
	CreateTraderMarketAccounts(owner, market string) error
	UpdateBalance(id string, balance int64) error
	IncrementBalance(id string, inc int64) error
	GetMarketAccountsForOwner(market, owner string) ([]*types.Account, error)
	GetAccountsForOwnerByType(owner string, accType types.AccountType) (*types.Account, error)
}

func New(log *logging.Logger, conf Config, market string, accounts Accounts) (*Engine, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	// ensure market accounts are all good to go - get insurance pool initial value from config?
	if err := accounts.CreateMarketAccounts(market, 0); err != nil && err != storage.ErrMarketAccountsExist {
		return nil, err
	}
	return &Engine{
		log:          log,
		Config:       conf,
		market:       market,
		accountStore: accounts,
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

func (e *Engine) getSystemAccounts() (settle, insurance *types.Account, err error) {
	var sysAccounts []*types.Account
	sysAccounts, err = e.accountStore.GetMarketAccountsForOwner(e.market, storage.SystemOwner)
	if err != nil {
		e.log.Error(
			"Failed to collect loss (system accounts missing)",
			logging.Error(err),
		)
		return
	}
	for _, sa := range sysAccounts {
		switch sa.Type {
		case types.AccountType_INSURANCE:
			insurance = sa
		case types.AccountType_SETTLEMENT:
			settle = sa
		}
	}
	// if one of the required accounts is nil, set error accordingly
	if settle == nil || insurance == nil {
		err = ErrSystemAccountsMissing
	}
	return
}

func (e *Engine) MarkToMarket(positions []*types.Transfer) ([]*types.TransferResponse, error) {
	// for now, this is the same as collect, but once we finish the closing positions bit in positions/settlement
	// we'll first handle the close settlement, then the updated positions for mark-to-market
	return e.Transfer(positions)
}

func (e *Engine) Transfer(transfers []*types.Transfer) ([]*types.TransferResponse, error) {
	if len(transfers) == 0 {
		return nil, nil
	}
	if isSettle(transfers[0]) {
		return e.collect(transfers)
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
func (e *Engine) collect(positions []*types.Transfer) ([]*types.TransferResponse, error) {
	if len(positions) == 0 {
		return nil, nil
	}
	reference := fmt.Sprintf("%s close", e.market) // ledger moves need to indicate that they happened because market was closed
	settle, insurance, err := e.getSystemAccounts()
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
			if err := e.accountStore.IncrementBalance(bacc.Account.Id, bacc.Balance); err != nil {
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
	e.cfgMu.Lock()
	createTraderAccounts := e.CreateTraderAccounts
	e.cfgMu.Unlock()
	// common tasks performed for both win and loss positions
	return func(p *types.Transfer) (*types.TransferResponse, error) {
		if createTraderAccounts {
			// ignore errors, the only error ATM is the one telling us this call was redundant
			_ = e.accountStore.CreateTraderMarketAccounts(p.Owner, e.market)
		}
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
		if err := e.accountStore.IncrementBalance(res.Balances[0].Account.Id, res.Balances[0].Balance); err != nil {
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
	// final settle, or MTM settle, makes no difference, it's win/loss still
	if p.Type == types.TransferType_LOSS || p.Type == types.TransferType_MTM_LOSS {
		accounts, err := e.accountStore.GetMarketAccountsForOwner(e.market, p.Owner)
		if err != nil {
			e.log.Error(
				"could not get accounts for market",
				logging.String("account-owner", p.Owner),
				logging.String("market", e.market),
				logging.Error(err),
			)
			return nil, err
		}
		req := types.TransferRequest{
			FromAccount: []*types.Account{nil, nil, insurance}, // we'll need 3 accounts, last one is insurance
			ToAccount: []*types.Account{
				settle,
			},
			Amount:    uint64(-p.Amount.Amount) * p.Size,
			MinAmount: 0,  // default value, but keep it here explicitly
			Asset:     "", // TBC
		}
		for _, ca := range accounts {
			switch ca.Type {
			case types.AccountType_MARGIN:
				req.FromAccount[0] = ca
			case types.AccountType_MARKET:
				req.FromAccount[1] = ca
			}
		}
		if req.FromAccount[0] == nil || req.FromAccount[1] == nil {
			return nil, ErrTraderAccountsMissing
		}
		return &req, nil
	}
	gen, err := e.accountStore.GetAccountsForOwnerByType(p.Owner, types.AccountType_GENERAL)
	if err != nil {
		e.log.Error(
			"Failed to get the general account",
			logging.String("owner", p.Owner),
			logging.Error(err),
		)
		return nil, err
	}
	return &types.TransferRequest{
		FromAccount: []*types.Account{
			settle,
			insurance,
		},
		ToAccount: []*types.Account{
			gen,
		},
		Amount:    uint64(p.Amount.Amount) * p.Size,
		MinAmount: 0,  // default value, but keep it here explicitly
		Asset:     "", // TBC
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
			if err := e.accountStore.IncrementBalance(acc.Id, -amount); err != nil {
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
			if err := e.accountStore.UpdateBalance(acc.Id, 0); err != nil {
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
