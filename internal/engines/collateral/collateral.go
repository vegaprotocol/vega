package collateral

import (
	"fmt"
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

type Engine struct {
	*Config
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

func New(conf *Config, market string, accounts Accounts) (*Engine, error) {
	// ensure market accounts are all good to go - get insurance pool initial value from config?
	if err := accounts.CreateMarketAccounts(market, 0); err != nil && err != storage.ErrMarketAccountsExist {
		return nil, err
	}
	return &Engine{
		Config:       conf,
		market:       market,
		accountStore: accounts,
	}, nil
}

func (e *Engine) Collect(positions []*types.SettlePosition) ([]*types.TransferResponse, error) {
	reference := fmt.Sprintf("%s close", e.market)
	sysAccounts, err := e.accountStore.GetMarketAccountsForOwner(e.market, storage.SystemOwner)
	if err != nil {
		e.log.Error(
			"Failed to collect buys (system accounts missing)",
			logging.Error(err),
		)
		return nil, err
	}
	var settle, insurance *types.Account
	for _, sa := range sysAccounts {
		switch sa.Type {
		case types.AccountType_INSURANCE:
			insurance = sa
		case types.AccountType_SETTLEMENT:
			settle = sa
		}
	}
	if settle == nil || insurance == nil {
		return nil, ErrSystemAccountsMissing
	}
	// assign this here, so we can set cap for sell response correctly,
	// and make an educated guess for buys
	transferCap := len(positions)
	// bit clunky, bit this ensures that we're not trying to do silly things, and the response doesn't contain
	// a useless TransferResponse object
	haveBuys, haveSells := false, false
	buyResp := types.TransferResponse{
		Transfers: make([]*types.LedgerEntry, 0, transferCap), // roughly half should be buys, but create 2 ledger entries, so that's a reasonable cap to use
		Balances: []*types.TransferBalance{
			{
				Account: settle, // settle to this account
				Balance: 0,      // current balance delta -> 0
			},
		},
	}
	sellResp := types.TransferResponse{
		// we will alloc this slice once we've processed all buys
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
	responses := make([]*types.TransferResponse, 0, 2)
	for _, p := range positions {
		if e.CreateTraderAccounts {
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
		req.Reference = reference
		res, err := e.getLedgerEntries(req)
		if err != nil {
			return nil, err
		}
		// append ledger moves
		if p.Type == types.SettleType_BUY {
			haveBuys = true
			buyResp.Transfers = append(buyResp.Transfers, res.Transfers...)
			// account balance is updated automatically
			// increment balance
			buyResp.Balances[0].Balance += res.Balances[0].Balance
			// one less cap for selling to consider
			transferCap--
		} else {
			haveSells = true
			if len(sellResp.Transfers) == 0 {
				sellResp.Transfers = make([]*types.LedgerEntry, 0, transferCap*2) // each sell pos will (roughly speaking) result in 2 ledger movements, so this cap is sensible
			}
			// there's only 1 balance account here (the ToAccount)
			if err := e.accountStore.IncrementBalance(req.ToAccount[0].Id, res.Balances[0].Balance); err != nil {
				// this account might get accessed concurrently -> use increment
				e.log.Error(
					"Failed to increment balance of general account",
					logging.String("account-id", req.ToAccount[0].Id),
					logging.Int64("increment", res.Balances[0].Balance),
					logging.Error(err),
				)
				return nil, err
			}
			sellResp.Transfers = append(sellResp.Transfers, res.Transfers...)
		}
	}
	if haveBuys {
		responses = append(responses, &buyResp)
		for _, bacc := range buyResp.Balances {
			if err := e.accountStore.IncrementBalance(bacc.Account.Id, bacc.Balance); err != nil {
				e.log.Error(
					"Failed to upadte target account",
					logging.String("target-account", bacc.Account.Id),
					logging.Int64("balance", bacc.Balance),
					logging.Error(err),
				)
				return nil, err
			}
		}
	}
	if haveSells {
		responses = append(responses, &sellResp)
		for _, b := range sellResp.Balances {
			b.Balance = b.Account.Balance
		}
	}
	return responses, nil
}

// getTransferRequest builds the request, and sets the required accounts based on the type of the SettlePosition argument
func (e *Engine) getTransferRequest(p *types.SettlePosition, settle, insurance *types.Account) (*types.TransferRequest, error) {
	if p.Type == types.SettleType_BUY {
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
			Amount:    p.Amount.Amount * p.Size,
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
		Amount:    p.Amount.Amount * p.Size,
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
		if acc.Balance >= amount || acc.Type == types.AccountType_INSURANCE {
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
			// I'm keeping these weird debug statements here, because something isn't quite right ATM
			// but I'm a bit at a loss trying to figure out what
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
			acc.Balance = 0
		}
	}
	return &ret, nil
}
